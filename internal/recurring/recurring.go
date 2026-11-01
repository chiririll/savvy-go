// Package recurring keeps the rules that make pending transactions on a
// schedule. A rule is a template (the JSON input of "create transaction",
// owned and validated by internal/ledger) and a Schedule. A rule always has
// one pending occurrence: confirming or skipping it creates the next, in the
// same database transaction, and the startup job only recovers after a crash.
// The next due date is derived from the occurrences already made, so a deleted
// occurrence is never generated again.
package recurring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	repo "savvy-go/internal/db/repo/recurring"
	"savvy-go/internal/store"
)

// Occurrences is what the service needs from internal/ledger. Each call runs
// on the handle it is given, so an occurrence and its instance row commit
// together.
type Occurrences interface {
	// Validate checks a template as the ledger would accept it as a new
	// transaction, recurring-only fields (a NULL amount, estimated) included.
	Validate(ctx context.Context, template json.RawMessage) error
	// Create makes the pending transaction of a template dated due and
	// returns its id.
	Create(ctx context.Context, template json.RawMessage, due string) (int64, error)
	// Replan rewrites a pending transaction from a changed template.
	Replan(ctx context.Context, txID int64, template json.RawMessage, due string) error
}

// Rule is a recurring rule. NextRunDate is derived: the due date of its open
// occurrence, else the next one by the schedule; nil when the rule is stopped
// or its schedule has ended.
type Rule struct {
	ID          int64
	Template    json.RawMessage
	Schedule    Schedule
	IsActive    bool
	Error       *string // why the rule was stopped
	NextRunDate *string
	LastRunDate *string // the due date of the latest confirmed occurrence
	CreatedAt   *time.Time
	UpdatedAt   *time.Time
}

// Input is a rule as entered. A nil IsActive means active.
type Input struct {
	Template json.RawMessage
	Schedule Schedule
	IsActive *bool
}

type Service struct {
	DB store.DB
	// Ledger gives the occurrence maker bound to a handle.
	Ledger func(store.DB) Occurrences
}

func (s Service) repo(d store.DB) repo.Repo { return repo.Repo{DB: d} }

func (s Service) All(ctx context.Context) ([]Rule, error) {
	rows, err := s.repo(s.DB).List(ctx, false)
	if err != nil {
		return nil, err
	}
	return ruleList(rows), nil
}

// Upcoming lists up to limit active rules by their next due date.
func (s Service) Upcoming(ctx context.Context, limit int) ([]Rule, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := s.repo(s.DB).List(ctx, true)
	if err != nil {
		return nil, err
	}
	var out []Rule
	for _, r := range ruleList(rows) {
		if r.NextRunDate != nil {
			out = append(out, r)
		}
	}
	slices.SortStableFunc(out, func(a, b Rule) int {
		if c := compareStr(*a.NextRunDate, *b.NextRunDate); c != 0 {
			return c
		}
		return int(a.ID - b.ID)
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func compareStr(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

// ByID returns nil for an unknown id.
func (s Service) ByID(ctx context.Context, id int64) (*Rule, error) {
	row, err := s.repo(s.DB).ByID(ctx, id)
	if row == nil || err != nil {
		return nil, err
	}
	r := ruleOf(*row)
	return &r, nil
}

func ruleList(rows []repo.Row) []Rule {
	out := make([]Rule, len(rows))
	for i, r := range rows {
		out[i] = ruleOf(r)
	}
	return out
}

// ruleOf reads a row. A schedule that does not parse (a rule broken outside
// this package) still lists, stopped, with the reason.
func ruleOf(row repo.Row) Rule {
	r := Rule{
		ID: row.ID, Template: json.RawMessage(row.Template), IsActive: row.IsActive, Error: row.Error,
		LastRunDate: row.LastRun, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	sched, err := ParseSchedule(row.Schedule)
	if err != nil {
		msg := err.Error()
		r.IsActive, r.Error = false, &msg
		return r
	}
	r.Schedule = sched
	if r.IsActive {
		next := sched.Next(row.LastDue)
		if row.PendingDue != nil {
			next = *row.PendingDue
		}
		if sched.Within(next) {
			r.NextRunDate = &next
		}
	}
	return r
}

// check validates what a rule is made of; the reason is what a stopped rule shows.
func (s Service) check(ctx context.Context, d store.DB, template json.RawMessage, sched Schedule) error {
	if err := sched.Validate(); err != nil {
		return err
	}
	return s.Ledger(d).Validate(ctx, template)
}

// Create saves a rule and makes its first pending occurrence.
func (s Service) Create(ctx context.Context, in Input) (*Rule, error) {
	var id int64
	err := store.Tx(ctx, s.DB, func(d store.DB) error {
		if err := s.check(ctx, d, in.Template, in.Schedule); err != nil {
			return err
		}
		var err error
		id, err = s.repo(d).Insert(ctx, fields(in), time.Now().UTC())
		if err != nil {
			return err
		}
		return s.ensure(ctx, d, id)
	})
	if err != nil {
		return nil, err
	}
	return s.ByID(ctx, id)
}

func fields(in Input) repo.Fields {
	return repo.Fields{Template: string(in.Template), Schedule: in.Schedule.JSON(), IsActive: in.IsActive == nil || *in.IsActive}
}

// Update replaces a rule and clears its error. Its pending occurrence is
// rewritten from the new template; a new start moves a pending first
// occurrence to it. Returns nil for an unknown id.
func (s Service) Update(ctx context.Context, id int64, in Input) (*Rule, error) {
	found := true
	err := store.Tx(ctx, s.DB, func(d store.DB) error {
		rp := s.repo(d)
		cur, err := rp.ByID(ctx, id)
		if err != nil {
			return err
		}
		if cur == nil {
			found = false
			return nil
		}
		if err := s.check(ctx, d, in.Template, in.Schedule); err != nil {
			return err
		}
		if err := rp.Update(ctx, id, fields(in), time.Now().UTC()); err != nil {
			return err
		}
		pending, err := rp.Pending(ctx, id)
		if err != nil {
			return err
		}
		if pending != nil {
			due := pending.DueDate
			if old, err := ParseSchedule(cur.Schedule); err == nil && old.Start != in.Schedule.Start && due == old.Start {
				if moved, err := rp.MoveInstance(ctx, id, due, in.Schedule.Start); err != nil {
					return err
				} else if moved {
					due = in.Schedule.Start
				}
			}
			return s.Ledger(d).Replan(ctx, *pending.TransactionID, in.Template, due)
		}
		return s.ensure(ctx, d, id)
	})
	if err != nil || !found {
		return nil, err
	}
	return s.ByID(ctx, id)
}

// Delete removes a rule and its instances; its transactions stay, a pending
// one becomes an ordinary pending transaction.
func (s Service) Delete(ctx context.Context, id int64) error {
	return s.repo(s.DB).Delete(ctx, id)
}

// RuleOf is the rule a transaction was made for; 0 when no rule made it.
func (s Service) RuleOf(ctx context.Context, txID int64) (int64, error) {
	rule, _, ok, err := s.repo(s.DB).OccurrenceOf(ctx, txID)
	if !ok {
		return 0, err
	}
	return rule, nil
}

// AdvanceAfterOccurrence makes the next occurrence of a rule once its pending
// one is settled (confirmed, skipped or deleted). Call it on the handle of the
// transaction that settled it, so both commit together.
func (s Service) AdvanceAfterOccurrence(ctx context.Context, d store.DB, id int64) error {
	return s.ensure(ctx, d, id)
}

// ensure makes the next occurrence of a rule when it is active, still on
// schedule and has none pending.
func (s Service) ensure(ctx context.Context, d store.DB, id int64) error {
	rp := s.repo(d)
	row, err := rp.ByID(ctx, id)
	if row == nil || err != nil {
		return err
	}
	rule := ruleOf(*row)
	if rule.NextRunDate == nil {
		return nil
	}
	if pending, err := rp.Pending(ctx, id); err != nil || pending != nil {
		return err
	}
	due := *rule.NextRunDate
	txID, err := s.Ledger(d).Create(ctx, rule.Template, due)
	if err != nil {
		return err
	}
	added, err := rp.AddInstance(ctx, id, due, txID)
	if err != nil {
		return err
	}
	if !added {
		return fmt.Errorf("recurring %d: %s was already handled", id, due)
	}
	return nil
}

// EnsureUpcoming is the startup and hourly job. It first validates every
// active rule, so a template broken outside the ledger (a restored backup, an
// old version) stops its rule with the reason instead of failing at its next
// occurrence; then it makes the missing pending occurrence of each rule that
// is still on schedule. A failing rule does not stop the others.
func (s Service) EnsureUpcoming(ctx context.Context) error {
	rows, err := s.repo(s.DB).List(ctx, true)
	if err != nil {
		return err
	}
	var errs []error
	var ok []int64
	for _, row := range rows {
		rule := ruleOf(row)
		if !rule.IsActive { // its schedule did not parse
			errs = append(errs, s.stop(ctx, row.ID, *rule.Error))
			continue
		}
		if err := s.Ledger(s.DB).Validate(ctx, rule.Template); err != nil {
			errs = append(errs, s.stop(ctx, row.ID, err.Error()))
			continue
		}
		ok = append(ok, row.ID)
	}
	for _, id := range ok {
		err := store.Tx(ctx, s.DB, func(d store.DB) error { return s.ensure(ctx, d, id) })
		if err != nil {
			errs = append(errs, fmt.Errorf("recurring %d: %w", id, err))
		}
	}
	return errors.Join(errs...)
}

func (s Service) stop(ctx context.Context, id int64, reason string) error {
	return s.repo(s.DB).Stop(ctx, id, reason, time.Now().UTC())
}
