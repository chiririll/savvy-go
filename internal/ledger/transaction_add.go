package ledger

import (
	"context"
	"fmt"
	"time"

	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/store"
)

// Adding transactions without reading them back, one or many at a time.

// Add is Create without reading the transaction back, which loads every
// account balance and category count it touches. It returns the new id; use it
// to add many transactions, where that read costs more than the insert.
func (s Transactions) Add(ctx context.Context, in TxInput) (int64, error) {
	id, _, err := s.add(ctx, in)
	return id, err
}

// AddMany adds all of ins in one database transaction and returns their ids in
// the same order. Either all of them are stored or none: an input that does not
// validate fails the call before anything is written. It is Add for loading
// many transactions, which pays the cost of a commit once instead of for each.
func (s Transactions) AddMany(ctx context.Context, ins []TxInput) ([]int64, error) {
	ids, _, err := s.addMany(ctx, ins)
	return ids, err
}

// CreateMany is AddMany that reads the transactions back, as Create does, each
// with its difference. Use it when the caller needs to show what was stored.
func (s Transactions) CreateMany(ctx context.Context, ins []TxInput) ([]*Transaction, error) {
	ids, ps, err := s.addMany(ctx, ins)
	if err != nil {
		return nil, err
	}
	out := make([]*Transaction, len(ids))
	for i, id := range ids {
		tx, err := s.ByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if tx != nil {
			tx.Difference = s.difference(ctx, ps[i].b)
		}
		out[i] = tx
	}
	return out, nil
}

func (s Transactions) addMany(ctx context.Context, ins []TxInput) ([]int64, []prepared, error) {
	ps := make([]prepared, len(ins))
	for i, in := range ins {
		p, err := s.prepare(ctx, in)
		if err != nil {
			return nil, nil, fmt.Errorf("transaction %d: %w", i+1, err)
		}
		ps[i] = p
	}
	ids := make([]int64, len(ins))
	err := store.Tx(ctx, s.DB, func(d store.DB) error {
		for i, p := range ps {
			id, err := s.insert(ctx, d, p)
			if err != nil {
				return fmt.Errorf("transaction %d: %w", i+1, err)
			}
			ids[i] = id
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return ids, ps, nil
}

// add validates and stores a new transaction.
func (s Transactions) add(ctx context.Context, in TxInput) (int64, built, error) {
	p, err := s.prepare(ctx, in)
	if err != nil {
		return 0, built{}, err
	}
	var id int64
	err = store.Tx(ctx, s.DB, func(d store.DB) error {
		id, err = s.insert(ctx, d, p)
		return err
	})
	if err != nil {
		return 0, built{}, err
	}
	return id, p.b, nil
}

// prepared is a new transaction that passed validation and awaits its insert.
type prepared struct {
	in     TxInput
	status string
	b      built
}

// prepare settles the status of a new transaction and checks it.
func (s Transactions) prepare(ctx context.Context, in TxInput) (prepared, error) {
	status := "confirmed"
	if in.Status != nil {
		status = *in.Status
	} else if in.Date == nil || IsFuture(*in.Date) {
		status = "pending"
	}
	if status == "confirmed" {
		if in.Date == nil {
			return prepared{}, fmt.Errorf("date required")
		}
		if IsFuture(*in.Date) {
			return prepared{}, fmt.Errorf("future")
		}
	}
	b, err := s.build(ctx, in, false)
	if err != nil {
		return prepared{}, err
	}
	return prepared{in: in, status: status, b: b}, nil
}

// insert writes a prepared transaction on d and returns its id.
func (s Transactions) insert(ctx context.Context, d store.DB, p prepared) (int64, error) {
	id, err := repo.Repo{DB: d}.InsertTransaction(ctx, repo.TxFields{
		Type: p.in.Type, Status: p.status, Description: blankToNil(p.in.Description), Date: p.in.Date,
		ExternalID: p.in.ExternalID, CreatedBy: p.in.CreatedBy,
	}, time.Now().UTC())
	if err != nil {
		return 0, err
	}
	return id, s.store(ctx, d, id, p.b, p.in.TagIDs)
}
