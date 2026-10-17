package mcpserver

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
	Conditions     map[string]any   `json:"conditions" jsonschema:"When the rule applies: an object with match (all or any) and conditions, a list of {field, op, value}. See the tool description for fields and ops."`
	Actions        []map[string]any `json:"actions" jsonschema:"What the rule does. Each is an object with a type: set_category (category_id), add_tags (tag_ids), remove_tags (tag_ids), set_description (value), or create_transfer (from_account_id, to_account_id, amount_formula)."`
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
	Conditions     map[string]any   `json:"conditions,omitempty" jsonschema:"Replaces the conditions as a whole: an object with match (all or any) and conditions, a list of {field, op, value}. See the create_automation_rule description for fields and ops."`
	Actions        []map[string]any `json:"actions,omitempty" jsonschema:"Replaces the actions as a whole. Same shapes as create_automation_rule."`
	IsActive       *bool            `json:"is_active,omitempty"`
	StopProcessing *bool            `json:"stop_processing,omitempty"`
}

type testRuleArgs struct {
	SpaceID       int64 `json:"space_id" jsonschema:"Id of the space, from list_spaces."`
	ID            int64 `json:"id" jsonschema:"Id of the rule."`
	TransactionID int64 `json:"transaction_id" jsonschema:"The transaction to try the rule on."`
}
