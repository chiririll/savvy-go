package refsync

import (
	"context"
	"fmt"
	"slices"
	"time"

	"savvy-go/internal/auth"
	repo "savvy-go/internal/db/repo/refsync"
	"savvy-go/internal/ledger"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
)

// Change is what the actor wants of a group: spaces in or out and what it
// keeps identical afterwards. Categories and Tags are the end state, not a
// delta. GroupID 0 starts a new group from Add.
type Change struct {
	GroupID    int64
	Add        []int64
	Remove     []int64
	Categories bool
	Tags       bool
}

// Plan is a Change with the decisions about the differences it brings out.
type Plan struct {
	Change
	Decisions []Decision
	Defaults  []DefaultDecision
}

// Decision settles one item that is not the same in every space.
type Decision struct {
	Kind Kind
	Type string // categories only: "income" or "expense"
	Name string
	// Keep creates the item where it is missing; otherwise it is deleted
	// where it exists.
	Keep bool
	// From is the space whose icon and color win when Keep and they differ.
	From int64
	// Successor is the name of a category of the same type that takes the
	// transactions of a deleted one, in every space that has any.
	Successor string
}

// DefaultDecision picks the default category of a type when the spaces
// disagree about it.
type DefaultDecision struct {
	Type string
	Name string
}

// CategoryState is a category as one space has it.
type CategoryState struct {
	Space        int64
	Icon, Color  *string
	IsDefault    bool
	Transactions int
}

// CategoryConflict is a category that is missing in some spaces or looks
// different in some.
type CategoryConflict struct {
	Type, Name string
	Present    []CategoryState
	Missing    []int64
}

// TagState is a tag as one space has it.
type TagState struct {
	Space        int64
	Transactions int
}

// TagConflict is a tag that is missing in some spaces.
type TagConflict struct {
	Name    string
	Present []TagState
	Missing []int64
}

// DefaultConflict is a type whose spaces do not agree on the default.
type DefaultConflict struct {
	Type     string
	Defaults map[int64]string // space -> name of its default, "" for none
}

// Diff is everything that makes the spaces of a group differ.
type Diff struct {
	Categories []CategoryConflict
	Tags       []TagConflict
	Defaults   []DefaultConflict
}

// Empty reports whether the spaces are identical.
func (d Diff) Empty() bool {
	return len(d.Categories) == 0 && len(d.Tags) == 0 && len(d.Defaults) == 0
}

// plan is a Change checked against the world.
type resolved struct {
	group      *repo.Group // nil for a new group
	current    []int64
	members    []int64 // after the change
	removed    []int64
	added      []int64
	categories bool
	tags       bool
	reconcile  []Kind
}

func (r resolved) dissolved() bool { return len(r.members) < 2 }

func containsID(ids []int64, id int64) bool { return slices.Contains(ids, id) }

// resolve validates a Change and checks the actor's rights for it.
func (s Service) resolve(ctx context.Context, actor *auth.User, c Change) (*resolved, error) {
	r := &resolved{categories: c.Categories, tags: c.Tags}
	if err := s.loadGroup(ctx, r, c.GroupID); err != nil {
		return nil, err
	}
	if err := r.collectRemoved(c.Remove); err != nil {
		return nil, err
	}
	if err := s.collectAdded(ctx, r, c.Add); err != nil {
		return nil, err
	}
	r.members = r.membersAfter()
	if r.group == nil && r.dissolved() {
		return nil, ErrTooFew
	}
	flagsChanged := r.planReconcile(c)
	if err := s.requireAdmins(ctx, actor, r, flagsChanged); err != nil {
		return nil, err
	}
	return r, nil
}

func (s Service) loadGroup(ctx context.Context, r *resolved, groupID int64) error {
	if groupID == 0 {
		return nil
	}
	srv := s.server()
	g, err := srv.Group(ctx, groupID)
	if err != nil {
		return err
	}
	if g == nil {
		return ErrNoGroup
	}
	r.group = g
	r.current, err = srv.Members(ctx, g.ID)
	return err
}

func (r *resolved) collectRemoved(ids []int64) error {
	for _, id := range ids {
		if !containsID(r.current, id) {
			return ErrNotMember
		}
		if !containsID(r.removed, id) {
			r.removed = append(r.removed, id)
		}
	}
	return nil
}

func (s Service) collectAdded(ctx context.Context, r *resolved, ids []int64) error {
	srv := s.server()
	for _, id := range ids {
		if containsID(r.current, id) || containsID(r.added, id) {
			continue
		}
		if sp, err := s.Spaces.Get(ctx, id); err != nil {
			return err
		} else if sp == nil {
			return fmt.Errorf("space %d: %w", id, store.ErrNotFound)
		}
		if other, err := srv.GroupOf(ctx, id); err != nil {
			return err
		} else if other != nil {
			return ErrInGroup
		}
		r.added = append(r.added, id)
	}
	return nil
}

// everyone is the group's members before the change plus the added ones.
func (r *resolved) everyone() []int64 {
	return append(slices.Clone(r.current), r.added...)
}

func (r *resolved) membersAfter() []int64 {
	var members []int64
	for _, id := range r.everyone() {
		if !containsID(r.removed, id) {
			members = append(members, id)
		}
	}
	slices.Sort(members)
	return members
}

// planReconcile fills r.reconcile with the kinds that need aligning and
// reports whether the group's flags change.
func (r *resolved) planReconcile(c Change) (flagsChanged bool) {
	wasCats, wasTags := r.group != nil && r.group.Categories, r.group != nil && r.group.Tags
	switchedOn := map[Kind]bool{Categories: c.Categories && !wasCats, Tags: c.Tags && !wasTags}
	if !r.dissolved() {
		wanted := map[Kind]bool{Categories: c.Categories, Tags: c.Tags}
		for _, k := range []Kind{Categories, Tags} {
			if wanted[k] && (switchedOn[k] || len(r.added) > 0) {
				r.reconcile = append(r.reconcile, k)
			}
		}
	}
	return c.Categories != wasCats || c.Tags != wasTags
}

// requireAdmins checks the actor's rights. Taking a space out is the business
// of its admins alone; everything else touches every space of the group, so
// every admin must agree.
func (s Service) requireAdmins(ctx context.Context, actor *auth.User, r *resolved, flagsChanged bool) error {
	for _, id := range r.removed {
		if err := s.requireAdmin(ctx, actor, id); err != nil {
			return err
		}
	}
	if len(r.added) > 0 || len(r.reconcile) > 0 || flagsChanged {
		for _, id := range r.everyone() {
			if err := s.requireAdmin(ctx, actor, id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s Service) requireAdmin(ctx context.Context, actor *auth.User, id int64) error {
	role, err := s.Spaces.Role(ctx, id, actor.ID)
	if err != nil {
		return err
	}
	if role != spaces.RoleAdmin {
		return fmt.Errorf("%w (%s)", ErrNotAdmin, s.name(ctx, id))
	}
	return nil
}

// Diff previews what applying the change would have to reconcile.
func (s Service) Diff(ctx context.Context, actor *auth.User, c Change) (Diff, error) {
	r, err := s.resolve(ctx, actor, c)
	if err != nil {
		return Diff{}, err
	}
	var out Diff
	if len(r.reconcile) == 0 {
		return out, nil
	}
	err = s.st().InSpaces(ctx, r.members, func(dbs map[int64]store.DB) error {
		var err error
		out, err = diffSpaces(ctx, dbs, r.members, r.reconcile)
		return err
	})
	return out, err
}

// Apply carries out the plan: reconciles the spaces by the decisions, then
// changes the group. All of it is refused if a difference has no decision.
// It returns the group afterwards, nil when it is dissolved.
func (s Service) Apply(ctx context.Context, actor *auth.User, p Plan) (*Group, error) {
	r, err := s.resolve(ctx, actor, p.Change)
	if err != nil {
		return nil, err
	}
	if len(r.reconcile) > 0 {
		err = s.st().InSpaces(ctx, r.members, func(dbs map[int64]store.DB) error {
			return reconcileSpaces(ctx, dbs, r.members, r.reconcile, p)
		})
		if err != nil {
			return nil, err
		}
	}
	// The membership is saved after the spaces are alike: a crash in between
	// leaves identical spaces and the old group, and the same plan can simply
	// be applied again.
	var groupID int64
	err = store.Tx(ctx, s.st().Server(), func(tx store.DB) error {
		srv := repo.Server{DB: tx}
		groupID = 0
		if r.group != nil {
			groupID = r.group.ID
		}
		if r.dissolved() {
			if groupID != 0 {
				return srv.DeleteGroup(ctx, groupID) // the memberships go with it
			}
			return nil
		}
		if groupID == 0 {
			id, err := srv.InsertGroup(ctx, time.Now().UTC())
			if err != nil {
				return err
			}
			groupID = id
		}
		for _, id := range r.removed {
			if _, err := srv.RemoveSpace(ctx, groupID, id); err != nil {
				return err
			}
		}
		for _, id := range r.added {
			if err := srv.AddSpace(ctx, groupID, id); err != nil {
				return err
			}
		}
		return srv.SetFlags(ctx, groupID, r.categories, r.tags)
	})
	if err != nil || r.dissolved() {
		return nil, err
	}
	return &Group{ID: groupID, Categories: r.categories, Tags: r.tags, Members: r.members}, nil
}

// --- finding the differences -------------------------------------------------

// --- carrying out the decisions ----------------------------------------------

// reconciler carries what the phases of reconcileSpaces share.
type reconciler struct {
	dbs      map[int64]store.DB
	members  []int64
	diff     Diff
	decided  map[string]Decision
	defaults map[string]string
}

func (r reconciler) cats(id int64) ledger.Categories { return ledger.Categories{DB: r.dbs[id]} }

func (r reconciler) decision(kind Kind, typ, name string) Decision {
	return r.decided[decisionKey(kind, typ, name)]
}

// reconcileSpaces makes the spaces identical. Everything that has to exist
// is made first and only then is anything deleted, so a successor is there
// to receive the transactions of a category that goes.
func reconcileSpaces(ctx context.Context, dbs map[int64]store.DB, members []int64, kinds []Kind, p Plan) error {
	diff, err := diffSpaces(ctx, dbs, members, kinds)
	if err != nil {
		return err
	}
	r := reconciler{dbs: dbs, members: members, diff: diff, decided: map[string]Decision{}, defaults: map[string]string{}}
	for _, d := range p.Decisions {
		r.decided[decisionKey(d.Kind, d.Type, d.Name)] = d
	}
	for _, d := range p.Defaults {
		r.defaults[d.Type] = d.Name
	}
	if err := r.checkResolved(); err != nil {
		return err
	}
	steps := []func(context.Context) error{
		r.createKept,    // 1. Create what is kept and align how it looks.
		r.moveDefaults,  // 2. Move the defaults, before anything that might be one is deleted.
		r.deleteDropped, // 3. Delete what is dropped, moving transactions to the successor.
	}
	for _, step := range steps {
		if err := step(ctx); err != nil {
			return err
		}
	}

	left, err := diffSpaces(ctx, dbs, members, kinds)
	if err != nil {
		return err
	}
	if !left.Empty() {
		return fmt.Errorf("%w: %+v", ErrInvalid, left)
	}
	return nil
}

func (r reconciler) checkResolved() error {
	for _, c := range r.diff.Categories {
		if _, ok := r.decided[decisionKey(Categories, c.Type, c.Name)]; !ok {
			return fmt.Errorf("%w: category %q (%s)", ErrUnresolved, c.Name, c.Type)
		}
	}
	for _, t := range r.diff.Tags {
		if _, ok := r.decided[decisionKey(Tags, "", t.Name)]; !ok {
			return fmt.Errorf("%w: tag %q", ErrUnresolved, t.Name)
		}
	}
	for _, c := range r.diff.Defaults {
		if _, ok := r.defaults[c.Type]; !ok {
			return fmt.Errorf("%w: the default %s category", ErrUnresolved, c.Type)
		}
	}
	return nil
}

func (r reconciler) createKept(ctx context.Context) error {
	for _, c := range r.diff.Categories {
		d := r.decision(Categories, c.Type, c.Name)
		if !d.Keep {
			continue
		}
		win, err := winningLook(c, d)
		if err != nil {
			return err
		}
		for _, id := range c.Missing {
			if _,
				err := r.cats(id).Create(ctx,
				ledger.Category{Name: c.Name, Type: c.Type, Icon: win.Icon, Color: win.Color}); err != nil {
				return err
			}
		}
		for _, s := range c.Present {
			if sameText(s.Icon, win.Icon) && sameText(s.Color, win.Color) {
				continue
			}
			cur, err := findCategory(ctx, r.cats(s.Space), c.Type, c.Name)
			if err != nil {
				return err
			}
			if _,
				err := r.cats(s.Space).Update(ctx, cur.ID,
				ledger.Category{Name: cur.Name, Type: cur.Type, Icon: win.Icon, Color: win.Color,
					IsDefault: cur.IsDefault}); err != nil {
				return err
			}
		}
	}
	for _, t := range r.diff.Tags {
		if !r.decision(Tags, "", t.Name).Keep {
			continue
		}
		for _, id := range t.Missing {
			if _, err := (ledger.Tags{DB: r.dbs[id]}).Create(ctx, t.Name); err != nil {
				return err
			}
		}
	}
	return nil
}

// winningLook picks the state of a kept category whose icon and color the
// other spaces take over.
func winningLook(c CategoryConflict, d Decision) (CategoryState, error) {
	if d.From != 0 {
		i := slices.IndexFunc(c.Present, func(s CategoryState) bool { return s.Space == d.From })
		if i < 0 {
			return CategoryState{}, fmt.Errorf("%w: category %q is not in space %d", ErrInvalid, c.Name, d.From)
		}
		return c.Present[i], nil
	}
	win := c.Present[0]
	for _, s := range c.Present[1:] {
		if !sameText(s.Icon, win.Icon) || !sameText(s.Color, win.Color) {
			return CategoryState{}, fmt.Errorf(
				"%w: category %q looks different in the spaces; say which one to keep",
				ErrUnresolved,
				c.Name,
			)
		}
	}
	return win, nil
}

func (r reconciler) moveDefaults(ctx context.Context) error {
	for _, c := range r.diff.Defaults {
		name := r.defaults[c.Type]
		for _, id := range r.members {
			cur, err := findCategory(ctx, r.cats(id), c.Type, name)
			if err != nil {
				return err
			}
			if cur == nil {
				return fmt.Errorf("%w: %q is not a %s category everywhere", ErrInvalid, name, c.Type)
			}
			if _, err := r.cats(id).SetDefault(ctx, cur.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r reconciler) deleteDropped(ctx context.Context) error {
	for _, c := range r.diff.Categories {
		d := r.decision(Categories, c.Type, c.Name)
		if d.Keep {
			continue
		}
		for _, s := range c.Present {
			if err := r.deleteCategory(ctx, s.Space, c, d); err != nil {
				return err
			}
		}
	}
	for _, t := range r.diff.Tags {
		if r.decision(Tags, "", t.Name).Keep {
			continue
		}
		for _, s := range t.Present {
			tags := ledger.Tags{DB: r.dbs[s.Space]}
			cur, err := findTag(ctx, tags, t.Name)
			if err != nil {
				return err
			}
			if err := tags.Delete(ctx, cur.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r reconciler) deleteCategory(ctx context.Context, space int64, c CategoryConflict, d Decision) error {
	cats := r.cats(space)
	cur, err := findCategory(ctx, cats, c.Type, c.Name)
	if err != nil {
		return err
	}
	var succ *int64
	if d.Successor != "" {
		found, err := findCategory(ctx, cats, c.Type, d.Successor)
		if err != nil {
			return err
		}
		if found == nil {
			return fmt.Errorf("%w: the successor %q is not a %s category", ErrInvalid, d.Successor, c.Type)
		}
		succ = &found.ID
	}
	if err := cats.Delete(ctx, cur.ID, succ); err != nil {
		return fmt.Errorf("deleting %q: %w", c.Name, err)
	}
	return nil
}
