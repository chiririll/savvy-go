package mcpserver

import (
	"context"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The reconciliation tools: points that pin an account's balance to the real
// one on a date. Like the others they call the REST routes as the caller.

type reconcileItemArgs struct {
	AccountID     int64  `json:"account_id" jsonschema:"Account id, from list_accounts."`
	ActualBalance string `json:"actual_balance" jsonschema:"Real balance at the end of the date: exact decimal string, account currency, e.g. \"1234.50\"."`
	Note          string `json:"note,omitempty" jsonschema:"Optional remark, e.g. the statement it came from."`
}

type reconcileArgs struct {
	SpaceID  int64               `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	Date     string              `json:"date" jsonschema:"YYYY-MM-DD, the day whose end balance is real. Not in the future."`
	Accounts []reconcileItemArgs `json:"accounts" jsonschema:"The accounts to reconcile on that date: one entry per account (a single account is a list of one)."`
}

type reconciliationsArgs struct {
	SpaceID    int64   `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	AccountIDs []int64 `json:"account_ids,omitempty" jsonschema:"Only points of these accounts."`
	From       string  `json:"from,omitempty" jsonschema:"YYYY-MM-DD, only points on or after this day."`
	To         string  `json:"to,omitempty" jsonschema:"YYYY-MM-DD, only points on or before this day."`
}

// reconcileBody turns the snake_case arguments into the camelCase body of
// POST /reconciliations. Balances stay the strings they came as: exact.
func reconcileBody(args map[string]any) any {
	items := []any{}
	list, _ := args["accounts"].([]any)
	for _, raw := range list {
		a, _ := raw.(map[string]any)
		it := map[string]any{"accountId": a["account_id"], "actualBalance": a["actual_balance"]}
		if note, ok := a["note"]; ok {
			it["note"] = note
		}
		items = append(items, it)
	}
	return map[string]any{"date": args["date"], "items": items}
}

// addReconciliationRead adds list_reconciliations.
func (h Handler) addReconciliationRead(srv *mcp.Server) {
	mcp.AddTool(srv, &mcp.Tool{
		Name: "list_reconciliations",
		Description: "List reconciliation points: per account and date the real balance you pinned (actualBalance), " +
			"what the ledger expected from the previous point (or the initial balance) plus the payments since " +
			"(expectedBalance), and difference = actual - expected, also in the base currency " +
			"(differenceBase, null when the currency has no rate). A nonzero difference means transactions are " +
			"missing before that date. Newest first; filter by account_ids and from/to.",
		Annotations: ro("List reconciliations"),
	}, func(ctx context.Context, req *mcp.CallToolRequest, in reconciliationsArgs) (*mcp.CallToolResult, any, error) {
		q := url.Values{}
		if err := firstErr(setDate(q, "start_date", in.From), setDate(q, "end_date", in.To)); err != nil {
			return mcpFail(err.Error()), nil, nil
		}
		setIDs(q, "account_ids", in.AccountIDs)
		return h.mcpGet(ctx, req, mcpSpacePath(in.SpaceID, "/reconciliations"), q), nil, nil
	})
}

// reconciliationTools registers reconcile_account and delete_reconciliation.
func (k writeSet) reconciliationTools() {
	t := k.post("reconcile_account", "Reconcile accounts",
		"Pin accounts to their real balance on a date, for example after reading a bank statement. From that date "+
			"the account's balance IS the actual_balance you give, like an initial balance placed on a date. "+
			"It creates no transaction and changes no later balance. Transactions added or changed before the "+
			"date only change the expected balance (previous point or initial balance plus payments since) and so the "+
			"difference. Each point of the answer has actualBalance, expectedBalance and difference = actual - "+
			"expected (and differenceBase in the base currency). A nonzero difference means transactions are "+
			"missing (or wrong) before that date: add the missing transactions afterwards with create_transaction, "+
			"do not adjust the balance. A second call for the same account and date replaces that point. Several "+
			"accounts on the same date are all stored or none. Amounts are exact decimal strings in each account's "+
			"currency. The date may not be in the future.",
		"reconciliations")
	t.bodyOf = reconcileBody
	register[reconcileArgs](k, t)
	register[idArgs](k, k.del("delete_reconciliation", "Delete reconciliation",
		"Permanently delete a reconciliation point (id from list_reconciliations). The account's balance after "+
			"its date is then computed from the previous point or the initial balance. Ask the user first.",
		"reconciliations/{id}"))
}
