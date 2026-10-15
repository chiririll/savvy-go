package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	savvyauth "savvy-go/internal/auth"
)

// The MCP write tools. Like the read tools they hold no logic of their own:
// each is a typed mapping to one REST route, called in-process with the
// caller's bearer token. The token decides: a tool needs the same
// "<resource>:write" or ":delete" grant as the route, in a space the token
// covers and with a role that may do it, and answers with the REST error
// otherwise. Money is a string (an exact decimal), never a JSON number.

const (
	amountDoc = "Exact decimal as a string, for example \"12.50\". Never a float."
	spaceDoc  = "Id of the space, from list_spaces."
)

type txItemArgs struct {
	Name         string `json:"name" jsonschema:"Name of the line item."`
	Quantity     string `json:"quantity,omitempty" jsonschema:"Exact decimal as a string (default 1)."`
	PricePerUnit string `json:"price_per_unit" jsonschema:"Price of one unit, in the transaction's currency. Exact decimal as a string."`
}

type createTxArgs struct {
	SpaceID     int64        `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	Type        string       `json:"type" jsonschema:"income, expense or transfer (between two accounts of this space)."`
	AccountID   int64        `json:"account_id" jsonschema:"The account the money leaves (expense, transfer) or arrives in (income)."`
	ToAccountID int64        `json:"to_account_id,omitempty" jsonschema:"Transfer only: the receiving account."`
	CategoryID  int64        `json:"category_id,omitempty" jsonschema:"Category id, from list_categories."`
	Amount      string       `json:"amount" jsonschema:"Amount in the currency of account_id. Exact decimal as a string, for example \"12.50\"."`
	ToAmount    string       `json:"to_amount,omitempty" jsonschema:"Transfer between accounts in different currencies: the amount that arrives, in the receiving account's currency."`
	IsEstimated bool         `json:"is_estimated,omitempty" jsonschema:"True when the amount is approximate."`
	Description string       `json:"description,omitempty"`
	Date        string       `json:"date,omitempty" jsonschema:"YYYY-MM-DD. Defaults to today."`
	TagIDs      []int64      `json:"tag_ids,omitempty" jsonschema:"Tag ids, from list_tags."`
	Items       []txItemArgs `json:"items,omitempty" jsonschema:"Optional line items."`
}

type updateTxArgs struct {
	SpaceID     int64        `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	ID          int64        `json:"id" jsonschema:"Id of the transaction."`
	Type        string       `json:"type,omitempty" jsonschema:"income, expense or transfer."`
	AccountID   int64        `json:"account_id,omitempty"`
	ToAccountID int64        `json:"to_account_id,omitempty"`
	CategoryID  int64        `json:"category_id,omitempty"`
	Amount      string       `json:"amount,omitempty" jsonschema:"Exact decimal as a string, for example \"12.50\"."`
	ToAmount    string       `json:"to_amount,omitempty"`
	IsEstimated *bool        `json:"is_estimated,omitempty"`
	Description *string      `json:"description,omitempty"`
	Date        string       `json:"date,omitempty" jsonschema:"YYYY-MM-DD."`
	TagIDs      []int64      `json:"tag_ids,omitempty" jsonschema:"Replaces the tags."`
	Items       []txItemArgs `json:"items,omitempty" jsonschema:"Replaces the line items."`
}

type createAccountArgs struct {
	SpaceID        int64  `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	Name           string `json:"name"`
	Type           string `json:"type,omitempty" jsonschema:"bank, cash or crypto (default cash)."`
	CurrencyID     int64  `json:"currency_id,omitempty" jsonschema:"Currency id, from list_currencies. Give this or currency_code."`
	CurrencyCode   string `json:"currency_code,omitempty" jsonschema:"Currency code such as EUR. Added to the space if it is not there yet."`
	InitialBalance string `json:"initial_balance,omitempty" jsonschema:"Exact decimal as a string."`
	IsActive       *bool  `json:"is_active,omitempty"`
}

type updateAccountArgs struct {
	SpaceID        int64   `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	ID             int64   `json:"id" jsonschema:"Id of the account."`
	Name           *string `json:"name,omitempty"`
	Type           *string `json:"type,omitempty" jsonschema:"bank, cash or crypto."`
	CurrencyID     int64   `json:"currency_id,omitempty"`
	CurrencyCode   string  `json:"currency_code,omitempty"`
	InitialBalance string  `json:"initial_balance,omitempty" jsonschema:"Exact decimal as a string."`
	IsActive       *bool   `json:"is_active,omitempty"`
}

type createCategoryArgs struct {
	SpaceID int64  `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	Name    string `json:"name"`
	Type    string `json:"type" jsonschema:"expense or income."`
	Icon    string `json:"icon,omitempty"`
	Color   string `json:"color,omitempty" jsonschema:"Hex color such as #3b82f6."`
}

type updateCategoryArgs struct {
	SpaceID int64   `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	ID      int64   `json:"id" jsonschema:"Id of the category."`
	Name    *string `json:"name,omitempty"`
	Icon    *string `json:"icon,omitempty"`
	Color   *string `json:"color,omitempty"`
}

type tagArgs struct {
	SpaceID int64  `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	Name    string `json:"name"`
}

type renameTagArgs struct {
	SpaceID int64  `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	ID      int64  `json:"id" jsonschema:"Id of the tag."`
	Name    string `json:"name"`
}

type createCurrencyArgs struct {
	SpaceID  int64  `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	Code     string `json:"code" jsonschema:"Currency code such as EUR or BTC."`
	Name     string `json:"name,omitempty"`
	Symbol   string `json:"symbol,omitempty"`
	Decimals *int   `json:"decimals,omitempty" jsonschema:"Digits after the decimal point (default 2, JPY 0)."`
	Rate     string `json:"rate,omitempty" jsonschema:"Value of one unit in the space's base currency. Exact decimal as a string."`
}

type updateCurrencyArgs struct {
	SpaceID  int64   `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	ID       int64   `json:"id" jsonschema:"Id of the currency."`
	Name     *string `json:"name,omitempty"`
	Symbol   *string `json:"symbol,omitempty"`
	Decimals *int    `json:"decimals,omitempty"`
	Rate     string  `json:"rate,omitempty" jsonschema:"Value of one unit in the base currency. Exact decimal as a string."`
}

type budgetFields struct {
	Name            *string `json:"name,omitempty"`
	Amount          string  `json:"amount,omitempty" jsonschema:"Limit per period, in currency_id. Exact decimal as a string."`
	CurrencyID      int64   `json:"currency_id,omitempty" jsonschema:"Currency id, from list_currencies."`
	Period          string  `json:"period,omitempty" jsonschema:"weekly, monthly, yearly or custom (with start_date and end_date)."`
	StartDate       string  `json:"start_date,omitempty" jsonschema:"YYYY-MM-DD, custom period."`
	EndDate         string  `json:"end_date,omitempty" jsonschema:"YYYY-MM-DD, custom period."`
	IsGlobal        *bool   `json:"is_global,omitempty" jsonschema:"True to count all expenses instead of chosen categories and tags."`
	NotifyAtPercent *int    `json:"notify_at_percent,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
	CategoryIDs     []int64 `json:"category_ids,omitempty"`
	TagIDs          []int64 `json:"tag_ids,omitempty"`
}

type createBudgetArgs struct {
	SpaceID int64 `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	budgetFields
}

type updateBudgetArgs struct {
	SpaceID int64 `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	ID      int64 `json:"id" jsonschema:"Id of the budget."`
	budgetFields
}

type createDebtArgs struct {
	SpaceID      int64  `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	Name         string `json:"name"`
	DebtType     string `json:"debt_type" jsonschema:"i_owe (I borrowed) or owed_to_me (I lent)."`
	CurrencyID   int64  `json:"currency_id" jsonschema:"Currency id, from list_currencies."`
	Amount       string `json:"amount" jsonschema:"Total amount. Exact decimal as a string."`
	Origin       string `json:"origin,omitempty" jsonschema:"new (money moves now, needs account_id) or existing (already happened; nothing is booked)."`
	AccountID    int64  `json:"account_id,omitempty" jsonschema:"With origin new: the account the money went from or to."`
	Date         string `json:"date,omitempty" jsonschema:"YYYY-MM-DD."`
	DueDate      string `json:"due_date,omitempty" jsonschema:"YYYY-MM-DD."`
	Counterparty string `json:"counterparty,omitempty"`
	Description  string `json:"description,omitempty"`
}

type updateDebtArgs struct {
	SpaceID      int64   `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	ID           int64   `json:"id" jsonschema:"Id of the debt."`
	Name         *string `json:"name,omitempty"`
	DebtType     *string `json:"debt_type,omitempty"`
	CurrencyID   int64   `json:"currency_id,omitempty"`
	Amount       string  `json:"amount,omitempty" jsonschema:"Exact decimal as a string."`
	DueDate      *string `json:"due_date,omitempty"`
	Counterparty *string `json:"counterparty,omitempty"`
	Description  *string `json:"description,omitempty"`
}

type debtPaymentArgs struct {
	SpaceID     int64  `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	ID          int64  `json:"id" jsonschema:"Id of the debt."`
	AccountID   int64  `json:"account_id" jsonschema:"The account the money moves through."`
	Amount      string `json:"amount" jsonschema:"Exact decimal as a string."`
	Date        string `json:"date,omitempty" jsonschema:"YYYY-MM-DD."`
	Description string `json:"description,omitempty"`
}

type createRuleArgs struct {
	SpaceID        int64            `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	Name           string           `json:"name"`
	Description    string           `json:"description,omitempty"`
	TriggerType    string           `json:"trigger_type" jsonschema:"on_transaction_create or on_transaction_update."`
	Priority       int              `json:"priority,omitempty" jsonschema:"Lower runs first."`
	Conditions     map[string]any   `json:"conditions" jsonschema:"When the rule applies. Shape: An object with match (all or any) and conditions, a list of objects with field, op and value, for example field description, op contains, value coffee. Fields: type, amount, description, date, status, account_id, to_account_id, category_id, tags. Ops: equals, not_equals, in, not_in, gt, gte, lt, lte, between, contains, not_contains, starts_with, ends_with, matches, is_null, is_not_null, has_any, has_all, has_none."`
	Actions        []map[string]any `json:"actions" jsonschema:"What the rule does. Each is an object with a type: set_category (category_id), add_tags (tag_ids), remove_tags (tag_ids), set_description (value), or create_transfer, which moves money (from_account_id, to_account_id, amount_formula)."`
	IsActive       *bool            `json:"is_active,omitempty"`
	StopProcessing *bool            `json:"stop_processing,omitempty" jsonschema:"True to skip the rules after this one when it matches."`
}

type updateRuleArgs struct {
	SpaceID        int64            `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	ID             int64            `json:"id" jsonschema:"Id of the rule."`
	Name           string           `json:"name,omitempty"`
	Description    *string          `json:"description,omitempty"`
	TriggerType    string           `json:"trigger_type,omitempty"`
	Priority       *int             `json:"priority,omitempty"`
	Conditions     map[string]any   `json:"conditions,omitempty" jsonschema:"Replaces the conditions as a whole. Shape: An object with match (all or any) and conditions, a list of objects with field, op and value, for example field description, op contains, value coffee. Fields: type, amount, description, date, status, account_id, to_account_id, category_id, tags. Ops: equals, not_equals, in, not_in, gt, gte, lt, lte, between, contains, not_contains, starts_with, ends_with, matches, is_null, is_not_null, has_any, has_all, has_none."`
	Actions        []map[string]any `json:"actions,omitempty" jsonschema:"Replaces the actions as a whole. Each is an object with a type: set_category (category_id), add_tags (tag_ids), remove_tags (tag_ids), set_description (value), or create_transfer, which moves money (from_account_id, to_account_id, amount_formula)."`
	IsActive       *bool            `json:"is_active,omitempty"`
	StopProcessing *bool            `json:"stop_processing,omitempty"`
}

type testRuleArgs struct {
	SpaceID       int64 `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	ID            int64 `json:"id" jsonschema:"Id of the rule."`
	TransactionID int64 `json:"transaction_id" jsonschema:"The transaction to try the rule on."`
}

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
func addWrite[In any](s *Server, srv *mcp.Server, t writeTool) {
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
			return s.mcpCall(ctx, req, t.method, "/api/spaces/"+strings.TrimPrefix(path, "/"), nil, body), nil, nil
		})
}

// addMCPWriteTools registers every write tool and returns what each needs.
func (s *Server) addMCPWriteTools(srv *mcp.Server) map[string]mcpNeed {
	const sp = "{space_id}/"
	needs := map[string]mcpNeed{}
	w := func(name, title, desc, method, path string) writeTool {
		// The resource is the one the REST route is guarded by.
		resource, _ := spaceResource("/spaces/0/" + path)
		needs[name] = mcpNeed{resource, savvyauth.LevelFor(method)}
		return writeTool{name: name, title: title, desc: desc, method: method, path: sp + path, idempotent: method == http.MethodPatch}
	}
	del := func(name, title, desc, path string) writeTool {
		t := w(name, title, desc, http.MethodDelete, path)
		t.destructive, t.idempotent = true, true
		return t
	}

	addWrite[createTxArgs](s, srv, w("create_transaction", "Create transaction", "Record an income, expense or transfer between two accounts. The amount is in the account's currency; a transfer between accounts of different currencies also needs to_amount. Say which currency you used.", http.MethodPost, "transactions"))
	addWrite[updateTxArgs](s, srv, w("update_transaction", "Update transaction", "Change a transaction. Send only the fields that change.", http.MethodPatch, "transactions/{id}"))
	addWrite[idArgs](s, srv, del("delete_transaction", "Delete transaction", "Permanently delete a transaction. Ask the user first.", "transactions/{id}"))
	addWrite[idArgs](s, srv, w("confirm_transaction", "Confirm transaction", "Confirm a pending (scheduled) transaction so it counts.", http.MethodPost, "transactions/{id}/confirm"))
	addWrite[idArgs](s, srv, w("skip_transaction", "Skip transaction", "Skip a pending (scheduled) transaction.", http.MethodPost, "transactions/{id}/skip"))
	addWrite[idArgs](s, srv, w("duplicate_transaction", "Duplicate transaction", "Copy a transaction.", http.MethodPost, "transactions/{id}/duplicate"))

	addWrite[createAccountArgs](s, srv, w("create_account", "Create account", "Add an account in one currency.", http.MethodPost, "accounts"))
	addWrite[updateAccountArgs](s, srv, w("update_account", "Update account", "Change an account. Send only the fields that change.", http.MethodPatch, "accounts/{id}"))
	addWrite[idArgs](s, srv, del("delete_account", "Delete account", "Permanently delete an account. Ask the user first.", "accounts/{id}"))

	addWrite[createCategoryArgs](s, srv, w("create_category", "Create category", "Add an expense or income category.", http.MethodPost, "categories"))
	addWrite[updateCategoryArgs](s, srv, w("update_category", "Update category", "Change a category. Send only the fields that change.", http.MethodPatch, "categories/{id}"))
	addWrite[idArgs](s, srv, w("set_default_category", "Set default category", "Make a category the default for its type.", http.MethodPost, "categories/{id}/set-default"))
	addWrite[idArgs](s, srv, del("delete_category", "Delete category", "Permanently delete a category. Ask the user first.", "categories/{id}"))

	addWrite[tagArgs](s, srv, w("create_tag", "Create tag", "Add a tag.", http.MethodPost, "tags"))
	addWrite[renameTagArgs](s, srv, w("rename_tag", "Rename tag", "Rename a tag.", http.MethodPatch, "tags/{id}"))
	addWrite[idArgs](s, srv, del("delete_tag", "Delete tag", "Permanently delete a tag. Ask the user first.", "tags/{id}"))

	addWrite[createCurrencyArgs](s, srv, w("create_currency", "Add currency", "Add a currency to the space, with its rate to the base currency.", http.MethodPost, "currencies"))
	addWrite[updateCurrencyArgs](s, srv, w("update_currency", "Update currency", "Change a currency's name, symbol, decimals or rate. Send only the fields that change.", http.MethodPatch, "currencies/{id}"))
	addWrite[idArgs](s, srv, w("set_base_currency", "Set base currency", "Make a currency the space's base currency. Ask the user first: every report total is in the base currency.", http.MethodPost, "currencies/{id}/set-base"))
	addWrite[idArgs](s, srv, del("delete_currency", "Delete currency", "Permanently delete a currency. Ask the user first.", "currencies/{id}"))

	addWrite[createBudgetArgs](s, srv, w("create_budget", "Create budget", "Add a spending limit per period, in one currency.", http.MethodPost, "budgets"))
	addWrite[updateBudgetArgs](s, srv, w("update_budget", "Update budget", "Change a budget. Send only the fields that change.", http.MethodPatch, "budgets/{id}"))
	addWrite[idArgs](s, srv, del("delete_budget", "Delete budget", "Permanently delete a budget. Ask the user first.", "budgets/{id}"))

	addWrite[createDebtArgs](s, srv, w("create_debt", "Create debt", "Record money you owe or are owed.", http.MethodPost, "debts"))
	addWrite[updateDebtArgs](s, srv, w("update_debt", "Update debt", "Change a debt. Send only the fields that change.", http.MethodPatch, "debts/{id}"))
	addWrite[debtPaymentArgs](s, srv, w("add_debt_payment", "Pay a debt", "Record a payment on a debt you owe (i_owe), moving money out of account_id.", http.MethodPost, "debts/{id}/payment"))
	addWrite[debtPaymentArgs](s, srv, w("collect_debt", "Collect a debt", "Record money received on a debt owed to you (owed_to_me), moving it into account_id.", http.MethodPost, "debts/{id}/collect"))
	addWrite[idArgs](s, srv, w("reopen_debt", "Reopen debt", "Reopen a settled debt.", http.MethodPost, "debts/{id}/reopen"))
	addWrite[idArgs](s, srv, del("delete_debt", "Delete debt", "Permanently delete a debt. Ask the user first.", "debts/{id}"))

	addWrite[createRuleArgs](s, srv, w("create_automation_rule", "Create automation rule", "Add a rule that changes new or edited transactions that match its conditions. Rules run by themselves from then on, so show the user what it will do and test it on a transaction first.", http.MethodPost, "automation-rules"))
	addWrite[updateRuleArgs](s, srv, w("update_automation_rule", "Update automation rule", "Change a rule. Send only the fields that change; conditions and actions are replaced whole.", http.MethodPatch, "automation-rules/{id}"))
	addWrite[idArgs](s, srv, w("toggle_automation_rule", "Turn rule on or off", "Switch a rule between active and inactive.", http.MethodPost, "automation-rules/{id}/toggle"))
	addWrite[testRuleArgs](s, srv, w("test_automation_rule", "Test automation rule", "Try a rule on one existing transaction and report what it would do.", http.MethodPost, "automation-rules/{id}/test"))
	addWrite[idArgs](s, srv, del("delete_automation_rule", "Delete automation rule", "Permanently delete a rule. Ask the user first.", "automation-rules/{id}"))
	return needs
}
