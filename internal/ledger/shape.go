package ledger

import (
	"errors"
	"fmt"
	"slices"

	"github.com/shopspring/decimal"

	"savvy-go/internal/money"
)

// The refusals of a transaction's shape. They are about the data entered, so
// the API reports them as validation errors.
var (
	ErrTxType           = errors.New("this transaction type cannot be entered that way")
	ErrNoPayments       = errors.New("a transaction needs at least one payment")
	ErrDuplicateAccount = errors.New("an account can be paid once in a transaction")
	ErrZeroPayment      = errors.New("a payment cannot be zero")
	ErrBalancing        = errors.New("only a recurring template may leave a payment amount out")
	ErrManyBalancing    = errors.New("a template may leave out the amount of one payment only")
	ErrNetSign          = errors.New("the payments of an expense must take money and of an income must bring it")
	ErrTransferSigns    = errors.New("a transfer needs a payment that takes money and one that brings it")
	ErrNoLines          = errors.New("an income or expense needs at least one line")
	ErrLineAmount       = errors.New("a line needs an amount above zero")
	ErrLineCurrency     = errors.New("a line must be in the currency of one of the payment accounts")
	ErrLineCategory     = errors.New("the category of a line does not fit the transaction type")
	ErrDebtShape        = errors.New("a debt transaction needs one payment on the debt and one on an ordinary account")
	ErrSpaceShape       = errors.New("a payment on a space account does not fit this transaction")
)

// shapePayment is a payment with its account loaded. A Balancing payment (a
// recurring template only) has no amount yet: each occurrence gets whatever
// balances the rest.
type shapePayment struct {
	Account   Account
	Amount    money.Money
	Balancing bool
}

// shapeLine is a line with its category (nil for none) and currency loaded.
type shapeLine struct {
	Category *Category
	Currency Currency
	Amount   money.Money
}

// shape is a transaction as the ledger rules see it. Owner is the module
// writing it, which alone may use a service account; Template is true while a
// recurring rule is saved.
type shape struct {
	Type     string
	Payments []shapePayment
	Lines    []shapeLine
	Owner    Reason
	Template bool
}

// check applies the rules that need the data of several tables and so cannot
// live in the database. The soft check on amounts is difference, not here.
func (s shape) check() error {
	if err := s.checkType(); err != nil {
		return err
	}
	if len(s.Payments) == 0 {
		return ErrNoPayments
	}
	balancing, err := s.checkPayments()
	if err != nil {
		return err
	}
	switch s.Type {
	case "income", "expense":
		if len(s.Lines) == 0 {
			return ErrNoLines
		}
		if balancing == 0 && !s.netHasSign(s.Type == "income") {
			return ErrNetSign
		}
	case "transfer":
		if !s.movesBothWays() {
			return ErrTransferSigns
		}
	case "transfer_out", "transfer_in":
		if len(s.accountsOf("space")) != 1 {
			return ErrSpaceShape
		}
	default: // debt_*
		if err := s.checkDebt(); err != nil {
			return err
		}
	}
	return s.checkLines()
}

func (s shape) checkType() error {
	switch s.Type {
	case "income", "expense", "transfer":
		return nil
	case "transfer_out", "transfer_in":
		if s.Owner == ReasonSpaceTransfer {
			return nil
		}
	case "debt_lend", "debt_borrow", "debt_payment", "debt_collection":
		if s.Owner == ReasonDebt {
			return nil
		}
	}
	return fmt.Errorf("%w: %q", ErrTxType, s.Type)
}

// checkPayments checks each payment and returns how many have no amount.
func (s shape) checkPayments() (balancing int, err error) {
	seen := map[int64]bool{}
	for _, p := range s.Payments {
		if seen[p.Account.ID] {
			return 0, ErrDuplicateAccount
		}
		seen[p.Account.ID] = true
		if p.Account.Currency == nil {
			return 0, ErrUnknownCurrency
		}
		if err := s.checkServiceAccount(p.Account); err != nil {
			return 0, err
		}
		if p.Balancing {
			if !s.Template {
				return 0, ErrBalancing
			}
			balancing++
			continue
		}
		if p.Amount.IsZero() {
			return 0, ErrZeroPayment
		}
	}
	if balancing > 1 {
		return 0, ErrManyBalancing
	}
	return balancing, nil
}

// checkServiceAccount lets a debt or space account be paid only by the module
// that owns it, and a space account only in the transactions made for it.
func (s shape) checkServiceAccount(a Account) error {
	switch a.Type {
	case "debt":
		if s.Owner != ReasonDebt {
			return fmt.Errorf("%w: a payment on a debt account goes through the debt", ErrServiceAccount)
		}
	case "space":
		if s.Owner != ReasonSpaceTransfer {
			return fmt.Errorf("%w: a payment on a space account goes through a transfer", ErrServiceAccount)
		}
		if s.Template {
			return fmt.Errorf("%w: a recurring transaction cannot move money between spaces", ErrSpaceShape)
		}
		if !slices.Contains([]string{"transfer_out", "transfer_in", "expense"}, s.Type) {
			return ErrSpaceShape
		}
		if len(s.accountsOf("space")) > 1 {
			return ErrSpaceShape
		}
	}
	return nil
}

func (s shape) accountsOf(typ string) []shapePayment {
	var out []shapePayment
	for _, p := range s.Payments {
		if p.Account.Type == typ {
			out = append(out, p)
		}
	}
	return out
}

// netHasSign tells whether the payments, converted to the base currency, net
// to a gain (income) or a loss (expense). Change larger than the payment is a
// mistake, so a net of zero is refused too.
func (s shape) netHasSign(gain bool) bool {
	net := decimal.Zero
	for _, p := range s.Payments {
		net = net.Add(p.Account.Currency.ConvertToBase(p.Amount.Decimal()))
	}
	if gain {
		return net.IsPositive()
	}
	return net.IsNegative()
}

func (s shape) movesBothWays() bool {
	var out, in, unknown bool
	for _, p := range s.Payments {
		switch {
		case p.Balancing:
			unknown = true
		case p.Amount.IsNegative():
			out = true
		default:
			in = true
		}
	}
	if unknown {
		return out || in
	}
	return out && in
}

// checkDebt keeps a debt_* transaction true to its payments: one on a debt
// account and one on an ordinary account, with the signs of its type, and no
// lines, so reports skip it.
func (s shape) checkDebt() error {
	debts := s.accountsOf("debt")
	if len(s.Payments) != 2 || len(debts) != 1 || len(s.Lines) > 0 {
		return ErrDebtShape
	}
	var ordinary shapePayment
	for _, p := range s.Payments {
		if p.Account.Type != "debt" {
			ordinary = p
		}
	}
	if ordinary.Account.IsService || ordinary.Balancing {
		return ErrDebtShape
	}
	// The debt account is signed like any account: lend and payment add to it.
	debtGrows := s.Type == "debt_lend" || s.Type == "debt_payment"
	if debts[0].Amount.IsPositive() != debtGrows || ordinary.Amount.IsPositive() == debtGrows {
		return ErrDebtShape
	}
	return nil
}

// checkLines checks every line: above zero, in the currency of a payment
// account, in a category of the type of the transaction. A transfer's lines
// are fees, so they use expense categories; a debt has none.
func (s shape) checkLines() error {
	currencies := make(map[int64]bool, len(s.Payments))
	for _, p := range s.Payments {
		currencies[p.Account.Currency.ID] = true
	}
	want := lineCategoryType(s.Type)
	for _, l := range s.Lines {
		if !l.Amount.IsPositive() {
			return ErrLineAmount
		}
		if !currencies[l.Currency.ID] {
			return ErrLineCurrency
		}
		if l.Category != nil && l.Category.Type != want {
			return ErrLineCategory
		}
	}
	return nil
}

// difference is what the payments moved less what the lines account for, in
// the base currency, converted at the current rates: each currency's total
// once, rounded once (half-even). A spend counts as positive: money out for
// the lines of an expense, in for an income's, and the net loss of a transfer
// for its fee lines. It is zero when the two agree, and only ever reported, so
// it reports false instead of failing when an amount does not fit.
func (s shape) difference(base Currency) (money.Money, bool) {
	paid := decimal.Zero
	byCurrency := map[int64]decimal.Decimal{}
	currencies := map[int64]Currency{}
	for _, p := range s.Payments {
		if p.Balancing {
			continue
		}
		c := *p.Account.Currency
		currencies[c.ID] = c
		byCurrency[c.ID] = byCurrency[c.ID].Add(p.Amount.Decimal())
	}
	for id, total := range byCurrency {
		paid = paid.Add(inBase(total, currencies[id], base))
	}
	if s.Type != "income" {
		paid = paid.Neg() // an expense's payments are negative
	}
	byCurrency, currencies = map[int64]decimal.Decimal{}, map[int64]Currency{}
	for _, l := range s.Lines {
		currencies[l.Currency.ID] = l.Currency
		byCurrency[l.Currency.ID] = byCurrency[l.Currency.ID].Add(l.Amount.Decimal())
	}
	lines := decimal.Zero
	for id, total := range byCurrency {
		lines = lines.Add(inBase(total, currencies[id], base))
	}
	d, err := money.FromInput(paid.Sub(lines), base.Unit())
	return d, err == nil
}

// inBase converts an amount in from to the base currency, rounded to its scale.
func inBase(amount decimal.Decimal, from, base Currency) decimal.Decimal {
	return base.ConvertFromBase(from.ConvertToBase(amount)).RoundBank(int32(base.Decimals))
}

// balancing finds what the payment without an amount (a recurring template)
// takes at today's rates: whatever makes the payments move what the lines
// account for, so the transaction agrees with itself. It returns the index of
// that payment and its amount in its account's currency, and false when no
// payment lacks an amount.
func (s shape) balancing() (int, decimal.Decimal, bool) {
	at := slices.IndexFunc(s.Payments, func(p shapePayment) bool { return p.Balancing })
	if at < 0 {
		return 0, decimal.Zero, false
	}
	lines, others := decimal.Zero, decimal.Zero
	for _, l := range s.Lines {
		lines = lines.Add(l.Currency.ConvertToBase(l.Amount.Decimal()))
	}
	for _, p := range s.Payments {
		if !p.Balancing {
			others = others.Add(p.Account.Currency.ConvertToBase(p.Amount.Decimal()))
		}
	}
	target := lines.Neg() // an expense or a transfer's fee takes what its lines say
	if s.Type == "income" {
		target = lines
	}
	return at, s.Payments[at].Account.Currency.ConvertFromBase(target.Sub(others)), true
}

// lineCategoryType is the type of category the lines of a transaction use: an
// income's are income, everything else's (a transfer's fee included) expense.
func lineCategoryType(txType string) string {
	if txType == "income" {
		return "income"
	}
	return "expense"
}
