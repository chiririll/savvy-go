// Package syncgroups serves sync groups: categories and tags kept equal across spaces.
package syncgroups

import (
	_ "embed"
	"net/http"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"slices"

	"savvy-go/internal/refsync"
)

// syncBody is a change of a sync group as the API takes it, with the decisions
// about what differs (for apply).
type syncBody struct {
	GroupID    int64   `json:"group_id"`
	AddIDs     []int64 `json:"add_space_ids"`
	RemoveIDs  []int64 `json:"remove_space_ids"`
	Categories bool    `json:"categories"`
	Tags       bool    `json:"tags"`
	Decisions  []struct {
		Kind      refsync.Kind `json:"kind"`
		Type      string       `json:"type"`
		Name      string       `json:"name"`
		Keep      bool         `json:"keep"`
		From      int64        `json:"from_space_id"`
		Successor string       `json:"successor"`
	} `json:"decisions"`
	Defaults []struct {
		Type string `json:"type"`
		Name string `json:"name"`
	} `json:"defaults"`
}

func (b syncBody) change() refsync.Change {
	return refsync.Change{
		GroupID: b.GroupID, Add: b.AddIDs, Remove: b.RemoveIDs,
		Categories: b.Categories, Tags: b.Tags,
	}
}

func (b syncBody) plan() refsync.Plan {
	p := refsync.Plan{Change: b.change()}
	for _, d := range b.Decisions {
		p.Decisions = append(p.Decisions, refsync.Decision{Kind: d.Kind, Type: d.Type, Name: d.Name, Keep: d.Keep, From: d.From, Successor: d.Successor})
	}
	for _, d := range b.Defaults {
		p.Defaults = append(p.Defaults, refsync.DefaultDecision{Type: d.Type, Name: d.Name})
	}
	return p
}

// decodeSync reads a syncBody. A space the caller does not belong to answers
// 404, so spaces cannot be probed (as for links).
func (h Handler) decodeSync(w http.ResponseWriter, r *http.Request) (syncBody, bool) {
	var body syncBody
	if !httpx.DecodeBody(w, r, &body) {
		return body, false
	}
	for _, id := range slices.Concat(body.AddIDs, body.RemoveIDs) {
		if role, _ := h.Spaces.Role(r.Context(), id, httpx.User(r).ID); role == "" {
			httpx.WriteMessage(w, http.StatusNotFound, "Space not found.")
			return body, false
		}
	}
	return body, true
}

// syncGroupShow is the sync group of the request's space: its spaces (those the
// caller does not belong to are only counted) and what it keeps identical.
func (h Handler) syncGroupShow(w http.ResponseWriter, r *http.Request) {
	g, err := h.Refsync.GroupOf(r.Context(), httpx.Scope(r).ID)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	if g == nil {
		httpx.WriteData(w, http.StatusOK, nil)
		return
	}
	members, hidden := []dto.Space{}, 0
	for _, id := range g.Members {
		role, _ := h.Spaces.Role(r.Context(), id, httpx.User(r).ID)
		sps, _ := h.Spaces.Get(r.Context(), id)
		if role == "" || sps == nil {
			hidden++
			continue
		}
		member := dto.NewSpace(*sps)
		member.Role = role
		members = append(members, member)
	}
	httpx.WriteData(w, http.StatusOK, dto.SyncGroupView{
		ID: g.ID, Categories: g.Categories, Tags: g.Tags, Members: members, Hidden: hidden,
	})
}

// syncGroupDiff previews what a change of a group has to reconcile.
func (h Handler) syncGroupDiff(w http.ResponseWriter, r *http.Request) {
	body, ok := h.decodeSync(w, r)
	if !ok {
		return
	}
	diff, err := h.Refsync.Diff(r.Context(), httpx.User(r), body.change())
	if err != nil {
		if !httpx.WriteSyncError(w, err) {
			httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewSyncDiff(diff, func(id int64) (string, bool) {
		sps, _ := h.Spaces.Get(r.Context(), id)
		if sps == nil {
			return "", false
		}
		return sps.Name, true
	}))
}

// syncGroupApply reconciles the spaces by the decisions and changes the group.
func (h Handler) syncGroupApply(w http.ResponseWriter, r *http.Request) {
	body, ok := h.decodeSync(w, r)
	if !ok {
		return
	}
	g, err := h.Refsync.Apply(r.Context(), httpx.User(r), body.plan())
	if err != nil {
		if !httpx.WriteSyncError(w, err) {
			httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	if g == nil {
		httpx.WriteData(w, http.StatusOK, nil)
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewSyncGroup(*g))
}
