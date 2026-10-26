package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"savvy-go/internal/auth/permissions"
	"savvy-go/internal/httpserver/httpx/guard"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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
	method, path      string // path may use {space_id}, {id} and {uuid}
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
			if v, ok := args["uuid"].(string); ok {
				path = strings.ReplaceAll(path, "{uuid}", url.PathEscape(v))
				delete(args, "uuid")
			}
			var body []byte
			if t.method != http.MethodDelete {
				body, _ = json.Marshal(args)
			}
			return h.mcpCall(ctx, req, t.method, "/api/spaces/"+strings.TrimPrefix(path, "/"), nil, body), nil, nil
		})
}

// writeSet builds write tools and records the grant each one needs.
type writeSet struct {
	h     Handler
	srv   *mcp.Server
	needs map[string]mcpNeed
}

const spacePrefix = "{space_id}/"

func (k writeSet) tool(name, title, desc, method, path string) writeTool {
	// The resource is the one the REST route is guarded by.
	resource, _ := guard.SpaceResource("/spaces/0/" + path)
	k.needs[name] = mcpNeed{resource, permissions.LevelFor(method)}
	return writeTool{
		name:       name,
		title:      title,
		desc:       desc,
		method:     method,
		path:       spacePrefix + path,
		idempotent: method == http.MethodPatch,
	}
}

func (k writeSet) post(name, title, desc, path string) writeTool {
	return k.tool(name, title, desc, http.MethodPost, path)
}

func (k writeSet) patch(name, title, desc, path string) writeTool {
	return k.tool(name, title, desc, http.MethodPatch, path)
}

func (k writeSet) put(name, title, desc, path string) writeTool {
	t := k.tool(name, title, desc, http.MethodPut, path)
	t.idempotent = true
	return t
}

func (k writeSet) del(name, title, desc, path string) writeTool {
	t := k.tool(name, title, desc, http.MethodDelete, path)
	t.destructive, t.idempotent = true, true
	return t
}

// register adds t, taking In as its arguments.
func register[In any](k writeSet, t writeTool) { addWrite[In](k.h, k.srv, t) }

// addMCPWriteTools registers every write tool and returns what each needs.
func (h Handler) addMCPWriteTools(srv *mcp.Server) map[string]mcpNeed {
	k := writeSet{h: h, srv: srv, needs: map[string]mcpNeed{}}
	k.transactionTools()
	k.accountTools()
	k.categoryTools()
	k.currencyTools()
	k.budgetTools()
	k.debtTools()
	k.ruleTools()
	return k.needs
}

func (k writeSet) transactionTools() {
	register[createTxArgs](k, k.post("create_transaction", "Create transaction",
		"Record an income, expense or transfer between two accounts. The amount is in the account's currency; "+
			"a transfer between accounts of different currencies also needs to_amount. Say which currency you used.",
		"transactions"))
	register[updateTxArgs](k, k.patch("update_transaction", "Update transaction",
		"Change a transaction. Send only the fields that change.", "transactions/{id}"))
	register[idArgs](k, k.del("delete_transaction", "Delete transaction",
		"Permanently delete a transaction. Ask the user first.", "transactions/{id}"))
	register[idArgs](k, k.post("confirm_transaction", "Confirm transaction",
		"Confirm a pending (scheduled) transaction so it counts.", "transactions/{id}/confirm"))
	register[idArgs](k, k.post("skip_transaction", "Skip transaction",
		"Skip a pending (scheduled) transaction.", "transactions/{id}/skip"))
	register[idArgs](k, k.post("duplicate_transaction", "Duplicate transaction",
		"Copy a transaction.", "transactions/{id}/duplicate"))
	register[createTxGroupArgs](k, k.post("create_transaction_group", "Create split payment",
		"Record one payment made with several accounts or currencies, such as cash and card, or euros with the "+
			"change given back in dinars. Legs are written together or not at all, all on the same day. "+
			"Record the price as an expense and the money exchanged on the way as a transfer between the two "+
			"accounts with to_amount: rent of 450 EUR paid with 500 EUR and 5900 RSD back is an expense of 450 EUR "+
			"plus a transfer of 50 EUR to the dinar account with to_amount 5900. Never use a negative amount.",
		"transaction-groups"))
	register[updateTxGroupArgs](k, k.put("update_transaction_group", "Update split payment",
		"Replace all legs of a split payment. Send every leg, not only the ones that change.",
		"transaction-groups/{uuid}"))
	register[txGroupIDArgs](k, k.del("delete_transaction_group", "Delete split payment",
		"Permanently delete a split payment with all its legs. Ask the user first.", "transaction-groups/{uuid}"))
}

func (k writeSet) accountTools() {
	register[createAccountArgs](k, k.post("create_account", "Create account",
		"Add an account in one currency.", "accounts"))
	register[updateAccountArgs](k, k.patch("update_account", "Update account",
		"Change an account. Send only the fields that change.", "accounts/{id}"))
	register[idArgs](k, k.del("delete_account", "Delete account",
		"Permanently delete an account. Ask the user first.", "accounts/{id}"))
}

func (k writeSet) categoryTools() {
	register[createCategoryArgs](k, k.post("create_category", "Create category",
		"Add an expense or income category.", "categories"))
	register[updateCategoryArgs](k, k.patch("update_category", "Update category",
		"Change a category. Send only the fields that change.", "categories/{id}"))
	register[idArgs](k, k.post("set_default_category", "Set default category",
		"Make a category the default for its type.", "categories/{id}/set-default"))
	register[idArgs](k, k.del("delete_category", "Delete category",
		"Permanently delete a category. Ask the user first.", "categories/{id}"))

	register[tagArgs](k, k.post("create_tag", "Create tag", "Add a tag.", "tags"))
	register[renameTagArgs](k, k.patch("rename_tag", "Rename tag", "Rename a tag.", "tags/{id}"))
	register[idArgs](k, k.del("delete_tag", "Delete tag",
		"Permanently delete a tag. Ask the user first.", "tags/{id}"))
}

func (k writeSet) currencyTools() {
	register[createCurrencyArgs](k, k.post("create_currency", "Add currency",
		"Add a currency to the space, with its rate to the base currency.", "currencies"))
	register[updateCurrencyArgs](k, k.patch("update_currency", "Update currency",
		"Change a currency's name, symbol, decimals or rate. Send only the fields that change.",
		"currencies/{id}"))
	register[idArgs](k, k.post("set_base_currency", "Set base currency",
		"Make a currency the space's base currency. Ask the user first: every report total is in the base currency.",
		"currencies/{id}/set-base"))
	register[idArgs](k, k.del("delete_currency", "Delete currency",
		"Permanently delete a currency. Ask the user first.", "currencies/{id}"))
}

func (k writeSet) budgetTools() {
	register[createBudgetArgs](k, k.post("create_budget", "Create budget",
		"Add a spending limit per period, in one currency.", "budgets"))
	register[updateBudgetArgs](k, k.patch("update_budget", "Update budget",
		"Change a budget. Send only the fields that change.", "budgets/{id}"))
	register[idArgs](k, k.del("delete_budget", "Delete budget",
		"Permanently delete a budget. Ask the user first.", "budgets/{id}"))
}

func (k writeSet) debtTools() {
	register[createDebtArgs](k, k.post("create_debt", "Create debt",
		"Record money you owe or are owed.", "debts"))
	register[updateDebtArgs](k, k.patch("update_debt", "Update debt",
		"Change a debt. Send only the fields that change.", "debts/{id}"))
	register[debtPaymentArgs](k, k.post("add_debt_payment", "Pay a debt",
		"Record a payment on a debt you owe (i_owe), moving money out of account_id.", "debts/{id}/payment"))
	register[debtPaymentArgs](k, k.post("collect_debt", "Collect a debt",
		"Record money received on a debt owed to you (owed_to_me), moving it into account_id.",
		"debts/{id}/collect"))
	register[idArgs](k, k.post("reopen_debt", "Reopen debt", "Reopen a settled debt.", "debts/{id}/reopen"))
	register[idArgs](k, k.del("delete_debt", "Delete debt",
		"Permanently delete a debt. Ask the user first.", "debts/{id}"))
}

func (k writeSet) ruleTools() {
	register[createRuleArgs](k, k.post("create_automation_rule", "Create automation rule",
		"Add a rule that changes new or edited transactions that match its conditions. Rules run by themselves "+
			"from then on, so show the user what it will do and test it on a transaction first. "+
			"Condition fields: type, amount, description, date, status, account_id, to_account_id, category_id, tags. "+
			"Condition ops: equals, not_equals, in, not_in, gt, gte, lt, lte, between, contains, not_contains, "+
			"starts_with, ends_with, matches, is_null, is_not_null, has_any, has_all, has_none. "+
			"Conditions are an object with match (all or any) and conditions, a list of {field, op, value}. "+
			"Example condition: field description, op contains, value coffee. "+
			"Actions are objects with a type: set_category (category_id), add_tags (tag_ids), remove_tags (tag_ids), "+
			"set_description (value), or create_transfer (from_account_id, to_account_id, amount_formula).",
		"automation-rules"))
	register[updateRuleArgs](k, k.patch("update_automation_rule", "Update automation rule",
		"Change a rule. Send only the fields that change; conditions and actions are replaced whole.",
		"automation-rules/{id}"))
	register[idArgs](k, k.post("toggle_automation_rule", "Turn rule on or off",
		"Switch a rule between active and inactive.", "automation-rules/{id}/toggle"))
	register[testRuleArgs](k, k.post("test_automation_rule", "Test automation rule",
		"Try a rule on one existing transaction and report what it would do.", "automation-rules/{id}/test"))
	register[idArgs](k, k.del("delete_automation_rule", "Delete automation rule",
		"Permanently delete a rule. Ask the user first.", "automation-rules/{id}"))
}
