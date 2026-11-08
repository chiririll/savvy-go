package mcpserver

import (
	"context"
	"net/url"
	"strconv"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The tools for taking a change back. Like the others they hold no logic: they
// call the audit-log routes as the caller. The token sees and undoes only the
// entries it made itself, and the server refuses an entry whose entity changed
// afterwards, so an assistant cannot undo what a person did after it.

type myActionsArgs struct {
	SpaceID    int64  `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	EntityType string `json:"entity_type,omitempty" jsonschema:"Only changes to this kind of thing, e.g. transaction."`
	EntityID   int64  `json:"entity_id,omitempty" jsonschema:"Only changes to this one, e.g. a transaction id."`
	Before     int64  `json:"before,omitempty" jsonschema:"Only entries with a smaller id: the next page."`
	Limit      int    `json:"limit,omitempty" jsonschema:"At most this many, newest first (default 50, at most 200)."`
}

type rollbackArgs struct {
	SpaceID int64   `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	IDs     []int64 `json:"ids,omitempty" jsonschema:"Ids of the entries to undo, from list_my_actions. Newest are undone first. All are undone or none."`
	BatchID string  `json:"batch_id,omitempty" jsonschema:"Instead of ids: undo a whole bulk call (create_transactions), by the batchId of its entries."`
}

// addHistoryRead adds list_my_actions.
func (h Handler) addHistoryRead(srv *mcp.Server) {
	mcp.AddTool(srv, &mcp.Tool{
		Name: "list_my_actions",
		Description: "List the changes you made in a space with this connection, newest first: what was created, " +
			"changed or deleted, when, and whether it can be rolled back now (rollbackable). If it cannot, " +
			"rollbackBlockedReason says why, usually that the same thing was changed again after you. " +
			"Use it to find the id to pass to rollback_actions.",
		Annotations: ro("List my actions"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in myActionsArgs) (*mcp.CallToolResult, any, error) {
		q := url.Values{}
		if in.EntityType != "" {
			q.Set("entity_type", in.EntityType)
		}
		if in.EntityID > 0 {
			q.Set("entity_id", strconv.FormatInt(in.EntityID, 10))
		}
		if in.Before > 0 {
			q.Set("before", strconv.FormatInt(in.Before, 10))
		}
		if in.Limit > 0 {
			q.Set("limit", strconv.Itoa(in.Limit))
		}
		return h.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, "/audit-log"), q), nil, nil
	})
}

// historyTools registers rollback_actions.
func (k writeSet) historyTools() {
	t := k.post("rollback_actions", "Roll back my actions",
		"Undo changes you made yourself, found with list_my_actions: a created transaction is removed, a changed "+
			"or deleted one is put back as it was, with its exact amounts. It works without the permission to delete. "+
			"It is refused, and nothing is undone, if the same thing was changed again after your change by anyone "+
			"(a person, or you: undo the newest change first), or if it was already rolled back. Undoing a change "+
			"is itself a change you can see in list_my_actions, but not roll back; make the change again instead. "+
			"Ask the user first unless they asked you to undo.",
		"audit-log/rollback")
	t.destructive = true
	register[rollbackArgs](k, t)
}
