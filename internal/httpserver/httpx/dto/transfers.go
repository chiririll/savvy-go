package dto

import (
	"github.com/shopspring/decimal"

	"savvy-go/internal/spaces"
	"savvy-go/internal/transfers"
)

func NewSpaceName(s spaces.Space) SpaceName { return SpaceName{ID: s.ID, Name: s.Name} }

// TransferSide is one end of a transfer. Amount is a string in major units;
// AccountID is given only to members of the space the side belongs to.
type TransferSide struct {
	Amount    string `json:"amount"`
	Currency  string `json:"currency"`
	AccountID *int64 `json:"accountId,omitempty"`
}

func newTransferSide(s transfers.Side, withAccount bool) TransferSide {
	out := TransferSide{Amount: decimal.New(s.Amount, -int32(s.Decimals)).String(), Currency: s.Currency}
	if withAccount {
		out.AccountID = &s.AccountID
	}
	return out
}

// RemoteTransfer is the other space's record of a transfer, shown when the two
// records differ.
type RemoteTransfer struct {
	From        TransferSide `json:"from"`
	To          TransferSide `json:"to"`
	Date        string       `json:"date"`
	Description *string      `json:"description"`
	Deleted     bool         `json:"deleted"`
}

// Transfer is a transfer between spaces as a member of one space sees it.
type Transfer struct {
	UUID        string          `json:"uuid"`
	From        TransferSide    `json:"from"`
	To          TransferSide    `json:"to"`
	Outgoing    bool            `json:"outgoing"`
	Date        string          `json:"date"`
	Description *string         `json:"description"`
	Version     int64           `json:"version"`
	Status      string          `json:"status"`
	Review      *string         `json:"review"`
	Frozen      bool            `json:"frozen"`
	Verified    bool            `json:"verified"`
	Deleted     bool            `json:"deleted"`
	OtherSpace  *SpaceName      `json:"otherSpace,omitempty"`
	Remote      *RemoteTransfer `json:"remote,omitempty"`
}

// NewTransfer converts v. seesOther says whether the viewer also belongs to the
// other space: only then is the account on the other side shown.
func NewTransfer(v transfers.View, seesOther bool) Transfer {
	out := Transfer{
		UUID: v.UUID, From: newTransferSide(v.From, v.Outgoing || seesOther), To: newTransferSide(v.To, !v.Outgoing || seesOther),
		Outgoing: v.Outgoing, Date: v.Date, Description: v.Description, Version: v.Version, Status: v.Status, Review: v.Review,
		Frozen: v.Frozen, Verified: v.Verified, Deleted: v.DeletedAt != nil,
		OtherSpace: ptr(v.OtherSpace, NewSpaceName),
	}
	if r := v.Remote; r != nil {
		out.Remote = &RemoteTransfer{
			From: newTransferSide(r.From, false), To: newTransferSide(r.To, false),
			Date: r.Date, Description: r.Description, Deleted: r.DeletedAt != nil,
		}
	}
	return out
}
