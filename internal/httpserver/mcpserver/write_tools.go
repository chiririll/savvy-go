package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	savvyauth "savvy-go/internal/auth"
	"savvy-go/internal/httpserver/guard"
)

// The MCP write tools. Like the read tools they hold no logic of their own:
// each is a typed mapping to one REST route, called in-process with the
// caller's bearer token. The token decides: a tool needs the same
// "<resource>:write" or ":delete" grant as the route, in a space the token
// covers and with a role that may do it, and answers with the REST error
// otherwise. Money is a string (an exact decimal), never a JSON number.

// writeTool describes one write tool: the REST route it calls and the hints
// clients use to decide whether to ask the user first.
type writeTool struct {
	name, title, desc string
	method, path      string // path may use {space_id} and {id}
	destructive       bool
	idempotent        bool
}

// addWrite registers t. The arguments become the JSON body, minus the ones the
// path already carries.
func addWrite[In any](h Handler, srv *mcp.Server, t writeTool) {
	no := false
	yes := true
	ann := &mcp.ToolAnnotations{Title: t.title, IdempotentHint: t.idempotent, OpenWorldHint: &no}
	// DestructiveHint defaults to true in MCP, so every other write says false.
	ann.DestructiveHint = &no
	if t.destructive {
		ann.DestructiveHint = &yes
	}
	mcp.AddTool(srv, &mcp.Tool{Name: t.name, Description: t.desc, Annotations: ann},
		func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, any, error) {
			raw, err := json.Marshal(in)
			if err != nil {
				return mcpFail(err.Error()), nil, nil
			}
			var args map[string]any
			_ = json.Unmarshal(raw, &args)
			path := t.path
			for _, key := range []string{"space_id", "id"} {
				if v, ok := args[key]; ok {
					path = strings.ReplaceAll(path, "{"+key+"}", strconv.FormatInt(int64(v.(float64)), 10))
					delete(args, key)
				}
			}
			var body []byte
			if t.method != http.MethodDelete {
				body, _ = json.Marshal(args)
			}
			return h.mcpCall(ctx, req, t.method, "/api/spaces/"+strings.TrimPrefix(path, "/"), nil, body), nil, nil
		})
}

// addMCPWriteTools registers every write tool and returns what each needs.
func (h Handler) addMCPWriteTools(srv *mcp.Server) map[string]mcpNeed {
	const sp = "{space_id}/"
	needs := map[string]mcpNeed{}
	w := func(name, title, desc, method, path string) writeTool {
		// The resource is the one the REST route is guarded by.
		resource, _ := guard.SpaceResource("/spaces/0/" + path)
		needs[name] = mcpNeed{resource, savvyauth.LevelFor(method)}
		return writeTool{name: name, title: title, desc: desc, method: method, path: sp + path, idempotent: method == http.MethodPatch}
	}
	del := func(name, title, desc, path string) writeTool {
		t := w(name, title, desc, http.MethodDelete, path)
		t.destructive, t.idempotent = true, true
		return t
	}

	addWrite[createTxArgs](h, srv, w("create_transaction", "Create transaction",
		"Record an income, expense or transfer between two accounts. The amount is in the account's currency; "+
			"a transfer between accounts of different currencies also needs to_amount. Say which currency you used.",
		http.MethodPost, "transactions"))
	addWrite[updateTxArgs](h, srv, w("update_transaction", "Update transaction", "Change a transaction. Send only the fields that change.", http.MethodPatch, "transactions/{id}"))
	addWrite[idArgs](h, srv, del("delete_transaction", "Delete transaction", "Permanently delete a transaction. Ask the user first.", "transactions/{id}"))
	addWrite[idArgs](h, srv, w("confirm_transaction", "Confirm transaction", "Confirm a pending (scheduled) transaction so it counts.", http.MethodPost, "transactions/{id}/confirm"))
	addWrite[idArgs](h, srv, w("skip_transaction", "Skip transaction", "Skip a pending (scheduled) transaction.", http.MethodPost, "transactions/{id}/skip"))
	addWrite[idArgs](h, srv, w("duplicate_transaction", "Duplicate transaction", "Copy a transaction.", http.MethodPost, "transactions/{id}/duplicate"))

	addWrite[createAccountArgs](h, srv, w("create_account", "Create account", "Add an account in one currency.", http.MethodPost, "accounts"))
	addWrite[updateAccountArgs](h, srv, w("update_account", "Update account", "Change an account. Send only the fields that change.", http.MethodPatch, "accounts/{id}"))
	addWrite[idArgs](h, srv, del("delete_account", "Delete account", "Permanently delete an account. Ask the user first.", "accounts/{id}"))

	addWrite[createCategoryArgs](h, srv, w("create_category", "Create category", "Add an expense or income category.", http.MethodPost, "categories"))
	addWrite[updateCategoryArgs](h, srv, w("update_category", "Update category", "Change a category. Send only the fields that change.", http.MethodPatch, "categories/{id}"))
	addWrite[idArgs](h, srv, w("set_default_category", "Set default category", "Make a category the default for its type.", http.MethodPost, "categories/{id}/set-default"))
	addWrite[idArgs](h, srv, del("delete_category", "Delete category", "Permanently delete a category. Ask the user first.", "categories/{id}"))

	addWrite[tagArgs](h, srv, w("create_tag", "Create tag", "Add a tag.", http.MethodPost, "tags"))
	addWrite[renameTagArgs](h, srv, w("rename_tag", "Rename tag", "Rename a tag.", http.MethodPatch, "tags/{id}"))
	addWrite[idArgs](h, srv, del("delete_tag", "Delete tag", "Permanently delete a tag. Ask the user first.", "tags/{id}"))

	addWrite[createCurrencyArgs](h, srv, w("create_currency", "Add currency", "Add a currency to the space, with its rate to the base currency.", http.MethodPost, "currencies"))
	addWrite[updateCurrencyArgs](h, srv, w("update_currency", "Update currency", "Change a currency's name, symbol, decimals or rate. Send only the fields that change.", http.MethodPatch, "currencies/{id}"))
	addWrite[idArgs](h, srv, w("set_base_currency", "Set base currency", "Make a currency the space's base currency. Ask the user first: every report total is in the base currency.", http.MethodPost, "currencies/{id}/set-base"))
	addWrite[idArgs](h, srv, del("delete_currency", "Delete currency", "Permanently delete a currency. Ask the user first.", "currencies/{id}"))

	addWrite[createBudgetArgs](h, srv, w("create_budget", "Create budget", "Add a spending limit per period, in one currency.", http.MethodPost, "budgets"))
	addWrite[updateBudgetArgs](h, srv, w("update_budget", "Update budget", "Change a budget. Send only the fields that change.", http.MethodPatch, "budgets/{id}"))
	addWrite[idArgs](h, srv, del("delete_budget", "Delete budget", "Permanently delete a budget. Ask the user first.", "budgets/{id}"))

	addWrite[createDebtArgs](h, srv, w("create_debt", "Create debt", "Record money you owe or are owed.", http.MethodPost, "debts"))
	addWrite[updateDebtArgs](h, srv, w("update_debt", "Update debt", "Change a debt. Send only the fields that change.", http.MethodPatch, "debts/{id}"))
	addWrite[debtPaymentArgs](h, srv, w("add_debt_payment", "Pay a debt", "Record a payment on a debt you owe (i_owe), moving money out of account_id.", http.MethodPost, "debts/{id}/payment"))
	addWrite[debtPaymentArgs](h, srv, w("collect_debt", "Collect a debt", "Record money received on a debt owed to you (owed_to_me), moving it into account_id.", http.MethodPost, "debts/{id}/collect"))
	addWrite[idArgs](h, srv, w("reopen_debt", "Reopen debt", "Reopen a settled debt.", http.MethodPost, "debts/{id}/reopen"))
	addWrite[idArgs](h, srv, del("delete_debt", "Delete debt", "Permanently delete a debt. Ask the user first.", "debts/{id}"))

	addWrite[createRuleArgs](h, srv, w("create_automation_rule", "Create automation rule",
		"Add a rule that changes new or edited transactions that match its conditions. Rules run by themselves "+
			"from then on, so show the user what it will do and test it on a transaction first. "+
			"Condition fields: type, amount, description, date, status, account_id, to_account_id, category_id, tags. "+
			"Condition ops: equals, not_equals, in, not_in, gt, gte, lt, lte, between, contains, not_contains, "+
			"starts_with, ends_with, matches, is_null, is_not_null, has_any, has_all, has_none. "+
			"Example condition: field description, op contains, value coffee.",
		http.MethodPost, "automation-rules"))
	addWrite[updateRuleArgs](h, srv, w("update_automation_rule", "Update automation rule", "Change a rule. Send only the fields that change; conditions and actions are replaced whole.", http.MethodPatch, "automation-rules/{id}"))
	addWrite[idArgs](h, srv, w("toggle_automation_rule", "Turn rule on or off", "Switch a rule between active and inactive.", http.MethodPost, "automation-rules/{id}/toggle"))
	addWrite[testRuleArgs](h, srv, w("test_automation_rule", "Test automation rule", "Try a rule on one existing transaction and report what it would do.", http.MethodPost, "automation-rules/{id}/test"))
	addWrite[idArgs](h, srv, del("delete_automation_rule", "Delete automation rule", "Permanently delete a rule. Ask the user first.", "automation-rules/{id}"))
	return needs
}
