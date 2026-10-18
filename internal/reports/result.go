package reports

import "savvy-go/internal/money"

// The results below are what each report computes. They carry no wire format:
// internal/httpserver/httpx/dto turns them into response bodies. Every amount
// is in the base currency; Currency is its code, empty while there is none.

// Metric is one figure of the overview with the same figure for the comparison
// period (nil without one) and for the last few periods.
type Metric[T any] struct {
	Value     T
	Previous  *T
	Sparkline []T
}

type OverviewReport struct {
	Income, Expenses, NetCashFlow Metric[money.Money]
	SavingsRate                   Metric[float64]
	Currency                      string
}

// FlowNode is one node of the money-flow diagram.
type FlowNode struct {
	Name  string
	Color string
}

type FlowLink struct {
	Source, Target string
	Value          money.Money
}

type FlowTotals struct {
	Income, Expenses, Savings money.Money
}

type MoneyFlowReport struct {
	Nodes    []FlowNode
	Links    []FlowLink
	Totals   FlowTotals
	Currency string
}

// PaceMonth is the spending of one month, cumulated per day.
type PaceMonth struct {
	Budget        *money.Money // nil without a monthly budget
	DailyExpenses []money.Money
	CurrentDay    *int // day of the month today is; nil for another month
	DaysInMonth   int
	TotalSpent    money.Money
	MonthStart    string
	MonthEnd      string
}

type ExpensePaceReport struct {
	Months   []PaceMonth
	Currency string
}

type CategoryExpense struct {
	ID                int64
	Name, Icon, Color string
	Current, Previous money.Money
}

type ExpensesByCategoryReport struct {
	Categories []CategoryExpense
	Currency   string
}

// CashFlowPoint is one period. The Prev fields are nil without a comparison
// period, or when it has no matching period.
type CashFlowPoint struct {
	Date                      string
	Income, Expenses, Balance money.Money
	PrevIncome, PrevExpenses  *money.Money
	PrevBalance               *money.Money
}

type CashFlowReport struct {
	Items    []CashFlowPoint
	Currency string
}

type HeatmapDay struct {
	Date  string
	Value money.Money
	Count int
}

type HeatmapReport struct {
	Items    []HeatmapDay
	Max      money.Money
	Currency string
}

// TxSummaryReport is the total of one transaction type with its averages. The
// Prev fields are nil without a comparison period.
type TxSummaryReport struct {
	Total                 money.Money
	Previous              *money.Money
	AvgPerDay, AvgPerWeek money.Money
	PrevAvgPerDay         *money.Money
	PrevAvgPerWeek        *money.Money
	DaysInPeriod          int
	Currency              string
}

type CategoryShare struct {
	ID                int64
	Name, Icon, Color string
	Value             money.Money
	Percentage        float64
}

type TxByCategoryReport struct {
	Items    []CategoryShare
	Total    money.Money
	Currency string
}

// DynamicsSeries is one line of the dynamics chart, a value per period. ID 0 is
// the total over every category.
type DynamicsSeries struct {
	ID    int64
	Name  string
	Color string
	Data  []money.Money
}

type TxDynamicsReport struct {
	Dates    []string
	Datasets []DynamicsSeries
	Currency string
}

type TopCategory struct {
	ID                int64
	Name, Icon, Color string
}

type TopAccount struct {
	ID   int64
	Name string
}

type TopTransaction struct {
	ID          int64
	Description *string
	Amount      money.Money
	Date        string
	Category    *TopCategory // nil for an uncategorised transaction
	Account     TopAccount
}

type TxTopReport struct {
	Items    []TopTransaction
	Currency string
}

type NetWorthAccount struct {
	ID         int64
	Name, Type string
	Balance    money.Money
	Percentage float64
}

type NetWorthReport struct {
	Current       money.Money
	Previous      *money.Money // nil without a comparison period
	Change        money.Money
	ChangePercent float64
	Accounts      []NetWorthAccount
	Currency      string
}

type NetWorthHistoryReport struct {
	Dates    []string
	Values   []money.Money
	Currency string
}
