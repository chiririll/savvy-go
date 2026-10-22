package dto

import (
	"savvy-go/internal/money"
	"savvy-go/internal/reports"
)

// reportCurrency is the base currency code of a report; null without one.
func reportCurrency(code string) *string {
	if code == "" {
		return nil
	}
	return &code
}

type Metric[T any] struct {
	Value     T   `json:"value"`
	Previous  *T  `json:"previous"`
	Sparkline []T `json:"sparkline"`
}

func newMetric[T any](m reports.Metric[T]) Metric[T] {
	return Metric[T]{Value: m.Value, Previous: m.Previous, Sparkline: orEmpty(m.Sparkline)}
}

// nonNil makes an empty list serialize as [].
func nonNil[T any](in []T) []T {
	if in == nil {
		return []T{}
	}
	return in
}

type Overview struct {
	Income      Metric[money.Money] `json:"income"`
	Expenses    Metric[money.Money] `json:"expenses"`
	NetCashFlow Metric[money.Money] `json:"netCashFlow"`
	SavingsRate Metric[float64]     `json:"savingsRate"`
	Currency    *string             `json:"currency"`
}

func NewOverview(r reports.OverviewReport) Overview {
	return Overview{
		Income: newMetric(r.Income), Expenses: newMetric(r.Expenses),
		NetCashFlow: newMetric(r.NetCashFlow), SavingsRate: newMetric(r.SavingsRate),
		Currency: reportCurrency(r.Currency),
	}
}

type FlowItemStyle struct {
	Color string `json:"color"`
}

type FlowNode struct {
	Name      string        `json:"name"`
	ItemStyle FlowItemStyle `json:"itemStyle"`
}

type FlowLink struct {
	Source string      `json:"source"`
	Target string      `json:"target"`
	Value  money.Money `json:"value"`
}

type FlowTotals struct {
	Income   money.Money `json:"income"`
	Expenses money.Money `json:"expenses"`
	Savings  money.Money `json:"savings"`
}

type MoneyFlow struct {
	Nodes    []FlowNode `json:"nodes"`
	Links    []FlowLink `json:"links"`
	Totals   FlowTotals `json:"totals"`
	Currency *string    `json:"currency"`
}

func NewMoneyFlow(r reports.MoneyFlowReport) MoneyFlow {
	return MoneyFlow{
		Nodes: Map(r.Nodes, func(n reports.FlowNode) FlowNode {
			return FlowNode{Name: n.Name, ItemStyle: FlowItemStyle{Color: n.Color}}
		}),
		Links: Map(r.Links, func(l reports.FlowLink) FlowLink {
			return FlowLink{Source: l.Source, Target: l.Target, Value: l.Value}
		}),
		Totals:   FlowTotals(r.Totals),
		Currency: reportCurrency(r.Currency),
	}
}

type PaceMonth struct {
	Budget        *money.Money  `json:"budget"`
	DailyExpenses []money.Money `json:"dailyExpenses"`
	CurrentDay    *int          `json:"currentDay"`
	DaysInMonth   int           `json:"daysInMonth"`
	TotalSpent    money.Money   `json:"totalSpent"`
	MonthStart    string        `json:"monthStart"`
	MonthEnd      string        `json:"monthEnd"`
}

type ExpensePace struct {
	Months   []PaceMonth `json:"months"`
	Currency *string     `json:"currency"`
}

func NewExpensePace(r reports.ExpensePaceReport) ExpensePace {
	return ExpensePace{
		Months: Map(r.Months, func(m reports.PaceMonth) PaceMonth {
			return PaceMonth{
				Budget: m.Budget, DailyExpenses: orEmpty(m.DailyExpenses), CurrentDay: m.CurrentDay,
				DaysInMonth: m.DaysInMonth, TotalSpent: m.TotalSpent, MonthStart: m.MonthStart, MonthEnd: m.MonthEnd,
			}
		}),
		Currency: reportCurrency(r.Currency),
	}
}

type CategoryExpense struct {
	ID       int64       `json:"id"`
	Name     string      `json:"name"`
	Icon     string      `json:"icon"`
	Color    string      `json:"color"`
	Current  money.Money `json:"current"`
	Previous money.Money `json:"previous"`
}

type ExpensesByCategory struct {
	Categories []CategoryExpense `json:"categories"`
	Currency   *string           `json:"currency"`
}

func NewExpensesByCategory(r reports.ExpensesByCategoryReport) ExpensesByCategory {
	return ExpensesByCategory{
		Categories: Map(r.Categories, func(c reports.CategoryExpense) CategoryExpense { return CategoryExpense(c) }),
		Currency:   reportCurrency(r.Currency),
	}
}

// CashFlowPoint leaves out the comparison fields when there is nothing to
// compare with.
type CashFlowPoint struct {
	Date         string       `json:"date"`
	Income       money.Money  `json:"income"`
	Expenses     money.Money  `json:"expenses"`
	Balance      money.Money  `json:"balance"`
	PrevIncome   *money.Money `json:"prevIncome,omitempty"`
	PrevExpenses *money.Money `json:"prevExpenses,omitempty"`
	PrevBalance  *money.Money `json:"prevBalance,omitempty"`
}

type CashFlow struct {
	Items    []CashFlowPoint `json:"items"`
	Currency *string         `json:"currency"`
}

func NewCashFlow(r reports.CashFlowReport) CashFlow {
	return CashFlow{
		Items:    Map(r.Items, func(p reports.CashFlowPoint) CashFlowPoint { return CashFlowPoint(p) }),
		Currency: reportCurrency(r.Currency),
	}
}

type HeatmapDay struct {
	Date  string      `json:"date"`
	Value money.Money `json:"value"`
	Count int         `json:"count"`
}

type Heatmap struct {
	Items    []HeatmapDay `json:"items"`
	Max      money.Money  `json:"max"`
	Currency *string      `json:"currency"`
}

func NewHeatmap(r reports.HeatmapReport) Heatmap {
	return Heatmap{
		Items:    Map(r.Items, func(d reports.HeatmapDay) HeatmapDay { return HeatmapDay(d) }),
		Max:      r.Max,
		Currency: reportCurrency(r.Currency),
	}
}

type TxSummary struct {
	Total          money.Money  `json:"total"`
	Previous       *money.Money `json:"previous"`
	AvgPerDay      money.Money  `json:"avgPerDay"`
	AvgPerWeek     money.Money  `json:"avgPerWeek"`
	PrevAvgPerDay  *money.Money `json:"prevAvgPerDay"`
	PrevAvgPerWeek *money.Money `json:"prevAvgPerWeek"`
	DaysInPeriod   int          `json:"daysInPeriod"`
	Currency       *string      `json:"currency"`
}

func NewTxSummary(r reports.TxSummaryReport) TxSummary {
	return TxSummary{
		Total: r.Total, Previous: r.Previous, AvgPerDay: r.AvgPerDay, AvgPerWeek: r.AvgPerWeek,
		PrevAvgPerDay: r.PrevAvgPerDay, PrevAvgPerWeek: r.PrevAvgPerWeek,
		DaysInPeriod: r.DaysInPeriod, Currency: reportCurrency(r.Currency),
	}
}

type CategoryShare struct {
	ID         int64       `json:"id"`
	Name       string      `json:"name"`
	Icon       string      `json:"icon"`
	Color      string      `json:"color"`
	Value      money.Money `json:"value"`
	Percentage float64     `json:"percentage"`
}

type TxByCategory struct {
	Items    []CategoryShare `json:"items"`
	Total    money.Money     `json:"total"`
	Currency *string         `json:"currency"`
}

func NewTxByCategory(r reports.TxByCategoryReport) TxByCategory {
	return TxByCategory{
		Items:    Map(r.Items, func(c reports.CategoryShare) CategoryShare { return CategoryShare(c) }),
		Total:    r.Total,
		Currency: reportCurrency(r.Currency),
	}
}

type DynamicsSeries struct {
	ID    int64         `json:"id"`
	Name  string        `json:"name"`
	Color string        `json:"color"`
	Data  []money.Money `json:"data"`
}

type TxDynamics struct {
	Dates    []string         `json:"dates"`
	Datasets []DynamicsSeries `json:"datasets"`
	Currency *string          `json:"currency"`
}

func NewTxDynamics(r reports.TxDynamicsReport) TxDynamics {
	return TxDynamics{
		Dates: orEmpty(r.Dates),
		Datasets: Map(r.Datasets, func(s reports.DynamicsSeries) DynamicsSeries {
			return DynamicsSeries{ID: s.ID, Name: s.Name, Color: s.Color, Data: orEmpty(s.Data)}
		}),
		Currency: reportCurrency(r.Currency),
	}
}

type TopCategory struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Icon  string `json:"icon"`
	Color string `json:"color"`
}

type TopAccount struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type TopTransaction struct {
	ID          int64        `json:"id"`
	Description *string      `json:"description"`
	Amount      money.Money  `json:"amount"`
	Date        string       `json:"date"`
	Category    *TopCategory `json:"category"`
	Account     TopAccount   `json:"account"`
}

type TxTop struct {
	Items    []TopTransaction `json:"items"`
	Currency *string          `json:"currency"`
}

func NewTxTop(r reports.TxTopReport) TxTop {
	return TxTop{
		Items: Map(r.Items, func(t reports.TopTransaction) TopTransaction {
			return TopTransaction{
				ID: t.ID, Description: t.Description, Amount: t.Amount, Date: t.Date,
				Category: ptr(t.Category, func(c reports.TopCategory) TopCategory { return TopCategory(c) }),
				Account:  TopAccount(t.Account),
			}
		}),
		Currency: reportCurrency(r.Currency),
	}
}

type NetWorthAccount struct {
	ID         int64       `json:"id"`
	Name       string      `json:"name"`
	Type       string      `json:"type"`
	Balance    money.Money `json:"balance"`
	Percentage float64     `json:"percentage"`
}

type NetWorth struct {
	Current       money.Money       `json:"current"`
	Previous      *money.Money      `json:"previous"`
	Change        money.Money       `json:"change"`
	ChangePercent float64           `json:"changePercent"`
	Accounts      []NetWorthAccount `json:"accounts"`
	Currency      *string           `json:"currency"`
}

func NewNetWorth(r reports.NetWorthReport) NetWorth {
	return NetWorth{
		Current: r.Current, Previous: r.Previous, Change: r.Change, ChangePercent: r.ChangePercent,
		Accounts: Map(r.Accounts, func(a reports.NetWorthAccount) NetWorthAccount { return NetWorthAccount(a) }),
		Currency: reportCurrency(r.Currency),
	}
}

type NetWorthHistory struct {
	Dates    []string      `json:"dates"`
	Values   []money.Money `json:"values"`
	Currency *string       `json:"currency"`
}

func NewNetWorthHistory(r reports.NetWorthHistoryReport) NetWorthHistory {
	return NetWorthHistory{Dates: orEmpty(r.Dates), Values: orEmpty(r.Values), Currency: reportCurrency(r.Currency)}
}
