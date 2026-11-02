package ledger

// simple is an income or an expense of amount on an account: one payment (signed
// by the type) and one uncategorised line.
func simple(typ string, account int64, amount string) TxInput {
	a := dec(amount)
	paid := a
	if typ == "expense" {
		paid = a.Neg()
	}
	return TxInput{
		Type:     typ,
		Payments: []PaymentInput{{AccountID: account, Amount: &paid}},
		Lines:    []LineInput{{Amount: a}},
	}
}

// move is a transfer: out leaves from, in arrives on to, each in its account's
// currency. No lines, so no fee.
func move(from, to int64, out, in string) TxInput {
	o, i := dec(out).Neg(), dec(in)
	return TxInput{
		Type:     "transfer",
		Payments: []PaymentInput{{AccountID: from, Amount: &o}, {AccountID: to, Amount: &i}},
	}
}
