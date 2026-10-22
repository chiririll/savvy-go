package reports

import (
	"context"

	"savvy-go/internal/money"
)

func (s Service) MoneyFlow(ctx context.Context, f Filter) MoneyFlowReport {
	q := s.open(ctx, f)
	r := f.Range(s.now())
	income, expenses := q.byCategory("income", r), q.byCategory("expense", r)
	fl := flow{income: income, expenses: expenses}
	fl.savingsNode = fl.nameNodes()
	fl.totalInc, fl.totalExp = q.catSum(income), q.catSum(expenses)
	fl.savings = fl.totalInc.Sub(fl.totalExp)
	return MoneyFlowReport{
		Nodes: fl.nodes(), Links: fl.links(q),
		Totals:   FlowTotals{Income: fl.totalInc, Expenses: fl.totalExp, Savings: money.Max(q.zero(), fl.savings)},
		Currency: q.code(),
	}
}

// flow is the data behind the money-flow diagram: income feeds expenses and
// whatever is left over, the savings.
type flow struct {
	income, expenses   []catTotal
	totalInc, totalExp money.Money
	savings            money.Money
	savingsNode        string
}

// catSum is the total of the categories.
func (q report) catSum(cats []catTotal) money.Money {
	total := q.zero()
	for _, x := range cats {
		total = total.Add(x.Total)
	}
	return total
}

// nameNodes gives every category and the savings a unique node name, and
// returns the savings node's.
func (fl flow) nameNodes() string {
	income, expenses := fl.income, fl.expenses
	used := map[string]bool{}
	resolve := func(name string) string {
		c := name
		for used[c] {
			c += "​"
		}
		used[c] = true
		return c
	}
	for i := range expenses {
		expenses[i].Node = resolve(expenses[i].Name)
	}
	for i := range income {
		income[i].Node = resolve(income[i].Name)
	}
	savingsNode := resolve("__savings__")
	return savingsNode
}

var (
	incomeColors  = []string{"#22c55e", "#16a34a", "#15803d", "#14532d", "#166534", "#4ade80"}
	expenseColors = []string{"#ef4444", "#f97316", "#eab308", "#ec4899", "#8b5cf6", "#06b6d4", "#f43f5e", "#a855f7"}
)

func (fl flow) nodes() []FlowNode {
	nodes := []FlowNode{}
	for i, x := range fl.income {
		nodes = append(nodes, FlowNode{Name: x.Node, Color: incomeColors[i%len(incomeColors)]})
	}
	for i, x := range fl.expenses {
		nodes = append(nodes, FlowNode{Name: x.Node, Color: expenseColors[i%len(expenseColors)]})
	}
	if fl.savings.IsPositive() {
		nodes = append(nodes, FlowNode{Name: fl.savingsNode, Color: "#3b82f6"})
	}
	return nodes
}

func (fl flow) links(q report) []FlowLink {
	links := []FlowLink{}
	if !fl.totalInc.IsPositive() || !fl.totalExp.IsPositive() {
		return links
	}
	for _, inc := range fl.income {
		for _, exp := range fl.expenses {
			if v := q.portion(exp.Total, inc.Total, fl.totalInc); v.IsPositive() {
				links = append(links, FlowLink{Source: inc.Node, Target: exp.Node, Value: v})
			}
		}
		if fl.savings.IsPositive() {
			if v := q.portion(fl.savings, inc.Total, fl.totalInc); v.IsPositive() {
				links = append(links, FlowLink{Source: inc.Node, Target: fl.savingsNode, Value: v})
			}
		}
	}
	return links
}
