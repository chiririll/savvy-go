package ledger

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	repo "savvy-go/internal/db/repo/ledger"
	"savvy-go/internal/store"
)

// ErrReorder is returned for a reorder the list cannot show: ids that repeat,
// do not exist or are not on the day.
var ErrReorder = errors.New("reorder: ids must be distinct transactions of that day")

// Reorder puts the transactions ids, oldest first, in that order within
// the calendar day (YYYY-MM-DD). The ids may be only some of the day's
// transactions (a filtered list): they trade the places they already hold and
// the rest stay where they are. Across days nothing moves.
func (s Transactions) Reorder(ctx context.Context, day string, ids []int64) error {
	if _, err := time.Parse("2006-01-02", day); err != nil {
		return fmt.Errorf("day must be YYYY-MM-DD")
	}
	if len(ids) == 0 {
		return nil
	}
	return store.Tx(ctx, s.DB, func(d store.DB) error {
		r := repo.Repo{DB: d}
		slots, err := r.DaySlots(ctx, ids)
		if err != nil {
			return err
		}
		if len(slots) != len(ids) {
			return ErrReorder
		}
		keys := make([]int64, 0, len(slots))
		seen := make(map[int64]bool, len(ids))
		for _, sl := range slots {
			if sl.Day != day || seen[sl.ID] {
				return ErrReorder
			}
			seen[sl.ID] = true
			keys = append(keys, sl.Key)
		}
		slices.Sort(keys)
		for i, id := range ids {
			if err := r.SetDayOrder(ctx, id, keys[i]); err != nil {
				return err
			}
		}
		return nil
	})
}
