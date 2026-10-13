// Package refsyncrepo stores sync groups on the server: which spaces keep
// their categories and tags identical. Package refsync applies the rules.
package refsync

import (
	"context"
	"time"

	"savvy-go/internal/db"
	"savvy-go/internal/db/sqlc"
	"savvy-go/internal/store"
)

// Server reads and writes the sync group tables of the server database.
type Server struct{ DB store.DB }

func (r Server) q() *sqlc.Queries { return db.Q(r.DB) }

// Group is a sync group and what it keeps identical.
type Group struct {
	ID         int64
	Categories bool
	Tags       bool
}

// GroupOf is the group a space is in, nil when it is in none.
func (r Server) GroupOf(ctx context.Context, spaceID int64) (*Group, error) {
	row, err := r.q().GetSyncGroupOfSpace(ctx, spaceID)
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Group{ID: row.ID, Categories: row.SyncCategories != 0, Tags: row.SyncTags != 0}, nil
}

// Group is a group by its id, nil when there is none.
func (r Server) Group(ctx context.Context, id int64) (*Group, error) {
	row, err := r.q().GetSyncGroup(ctx, id)
	if err == db.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &Group{ID: row.ID, Categories: row.SyncCategories != 0, Tags: row.SyncTags != 0}, nil
}

// Groups lists every group.
func (r Server) Groups(ctx context.Context) ([]Group, error) {
	rows, err := r.q().ListSyncGroups(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Group, len(rows))
	for i, row := range rows {
		out[i] = Group{ID: row.ID, Categories: row.SyncCategories != 0, Tags: row.SyncTags != 0}
	}
	return out, nil
}

// Members lists the spaces of a group, ascending.
func (r Server) Members(ctx context.Context, groupID int64) ([]int64, error) {
	return r.q().ListSyncGroupSpaces(ctx, groupID)
}

// InsertGroup creates a group that keeps nothing identical yet.
func (r Server) InsertGroup(ctx context.Context, now time.Time) (int64, error) {
	res, err := r.q().InsertSyncGroup(ctx, sqlc.InsertSyncGroupParams{CreatedAt: db.TimeNS(now)})
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r Server) SetFlags(ctx context.Context, groupID int64, categories, tags bool) error {
	return r.q().UpdateSyncGroupFlags(ctx, sqlc.UpdateSyncGroupFlagsParams{
		SyncCategories: flag(categories), SyncTags: flag(tags), ID: groupID,
	})
}

func (r Server) DeleteGroup(ctx context.Context, groupID int64) error {
	return r.q().DeleteSyncGroup(ctx, groupID)
}

// AddSpace puts a space in a group; it fails when the space is in a group already.
func (r Server) AddSpace(ctx context.Context, groupID, spaceID int64) error {
	return r.q().InsertSyncGroupSpace(ctx, sqlc.InsertSyncGroupSpaceParams{GroupID: groupID, SpaceID: spaceID})
}

// RemoveSpace reports whether the space was in the group.
func (r Server) RemoveSpace(ctx context.Context, groupID, spaceID int64) (bool, error) {
	res, err := r.q().DeleteSyncGroupSpace(ctx, sqlc.DeleteSyncGroupSpaceParams{GroupID: groupID, SpaceID: spaceID})
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func flag(b bool) int64 {
	if b {
		return 1
	}
	return 0
}
