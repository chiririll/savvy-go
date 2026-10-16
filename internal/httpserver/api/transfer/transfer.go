// Package transfer serves cross-space transfers and space links.
package transfer

import (
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/signing"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/money"
	"savvy-go/internal/spaces"
	"savvy-go/internal/store"
	"savvy-go/internal/transfers"
)

func writeTransferError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, transfers.ErrGone):
		httpx.WriteMessage(w, http.StatusNotFound, err.Error())
	case errors.Is(err, transfers.ErrRights), errors.Is(err, transfers.ErrLinkRights):
		httpx.WriteMessage(w, http.StatusForbidden, err.Error())
	case errors.Is(err, store.ErrQuotaExceeded):
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, "A space involved has reached its storage limit.")
	case errors.Is(err, store.ErrUnavailable):
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, "A space involved is temporarily unavailable.")
	case errors.Is(err, money.ErrOutOfRange):
		httpx.WriteValidation(w, map[string][]string{"amount": {"The amount is out of range."}})
	default:
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, err.Error())
	}
}

// --- links -----------------------------------------------------------------

// Handler holds what the transfer routes need.
type Handler struct {
	Spaces    spaces.Service
	Transfers *transfers.Service
	Keys      *signing.Holder
}

func (h Handler) linksIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.Transfers.Links(r.Context(), httpx.Scope(r).ID)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, dto.NewSpaceName))
}

// linksStore links the space with another the caller administers. A target
// the caller does not belong to is a 404, so spaces cannot be probed (P52).
func (h Handler) linksStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TargetSpaceID int64 `json:"target_space_id"`
	}
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	if role, _ := h.Spaces.Role(r.Context(), body.TargetSpaceID, httpx.User(r).ID); role == "" {
		httpx.WriteMessage(w, http.StatusNotFound, "Space not found.")
		return
	}
	if err := h.Transfers.Link(r.Context(), httpx.User(r), httpx.Scope(r).ID, body.TargetSpaceID); err != nil {
		writeTransferError(w, err)
		return
	}
	h.linksIndex(w, r)
}

func (h Handler) linksDestroy(w http.ResponseWriter, r *http.Request) {
	other, _ := strconv.ParseInt(chi.URLParam(r, "other"), 10, 64)
	if err := h.Transfers.Unlink(r.Context(), httpx.Scope(r).ID, other); err != nil {
		writeTransferError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- transfers -------------------------------------------------------------

// transferJSON shows a transfer to a member of the request's space: the
// other side's account only to who belongs to that space too (P30).
func (h Handler) transferJSON(r *http.Request, v transfers.View) dto.Transfer {
	seesOther := false
	if v.OtherSpace != nil {
		role, _ := h.Spaces.Role(r.Context(), v.OtherSpace.ID, httpx.User(r).ID)
		seesOther = role != ""
	}
	return dto.NewTransfer(v, seesOther)
}

func (h Handler) transfersIndex(w http.ResponseWriter, r *http.Request) {
	list, err := h.Transfers.List(r.Context(), httpx.Scope(r).ID)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, func(v transfers.View) dto.Transfer { return h.transferJSON(r, v) }))
}

// findTransfer is a transfer of the request's space as its members see it.
func (h Handler) findTransfer(r *http.Request, id string) (*transfers.View, error) {
	list, err := h.Transfers.List(r.Context(), httpx.Scope(r).ID)
	if err != nil {
		return nil, err
	}
	for _, v := range list {
		if v.UUID == id {
			return &v, nil
		}
	}
	return nil, transfers.ErrGone
}

func (h Handler) transferShow(w http.ResponseWriter, r *http.Request, id string, status int) {
	v, err := h.findTransfer(r, id)
	if err != nil {
		writeTransferError(w, err)
		return
	}
	httpx.WriteData(w, status, h.transferJSON(r, *v))
}

// validDate reports whether d is a YYYY-MM-DD date.
func validDate(d string) bool {
	_, err := time.Parse("2006-01-02", d)
	return err == nil
}

// transfersStore sends money from the request's space to a linked one.
func (h Handler) transfersStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ToSpaceID     int64           `json:"to_space_id"`
		FromAccountID int64           `json:"from_account_id"`
		ToAccountID   int64           `json:"to_account_id"`
		FromAmount    decimal.Decimal `json:"from_amount"`
		ToAmount      decimal.Decimal `json:"to_amount"`
		Date          string          `json:"date"`
		Description   *string         `json:"description"`
	}
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	if !validDate(body.Date) {
		httpx.WriteValidation(w, map[string][]string{"date": {"The date must be YYYY-MM-DD."}})
		return
	}
	t, err := h.Transfers.Create(r.Context(), httpx.User(r), transfers.Input{
		FromSpace: httpx.Scope(r).ID, FromAccount: body.FromAccountID, FromAmount: body.FromAmount,
		ToSpace: body.ToSpaceID, ToAccount: body.ToAccountID, ToAmount: body.ToAmount,
		Date: body.Date, Description: body.Description,
	})
	if err != nil {
		writeTransferError(w, err)
		return
	}
	h.transferShow(w, r, t.UUID, http.StatusCreated)
}

func (h Handler) transfersUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FromAccountID *int64           `json:"from_account_id"`
		ToAccountID   *int64           `json:"to_account_id"`
		FromAmount    *decimal.Decimal `json:"from_amount"`
		ToAmount      *decimal.Decimal `json:"to_amount"`
		Date          *string          `json:"date"`
		Description   *string          `json:"description"`
	}
	var raw map[string]json.RawMessage
	if !httpx.DecodeBody(w, r, &raw) {
		return
	}
	buf, _ := json.Marshal(raw)
	if err := json.Unmarshal(buf, &body); err != nil {
		httpx.WriteValidation(w, map[string][]string{"body": {"The given data was invalid."}})
		return
	}
	_, hasDescription := raw["description"]
	if body.Date != nil && !validDate(*body.Date) {
		httpx.WriteValidation(w, map[string][]string{"date": {"The date must be YYYY-MM-DD."}})
		return
	}
	id := chi.URLParam(r, "uuid")
	_, err := h.Transfers.Update(r.Context(), httpx.User(r), httpx.Scope(r).ID, id, transfers.Update{
		FromAccount: body.FromAccountID, ToAccount: body.ToAccountID, FromAmount: body.FromAmount, ToAmount: body.ToAmount,
		Date: body.Date, Description: body.Description, HasDescription: hasDescription,
	})
	if err != nil {
		writeTransferError(w, err)
		return
	}
	h.transferShow(w, r, id, http.StatusOK)
}

func (h Handler) transfersDestroy(w http.ResponseWriter, r *http.Request) {
	if err := h.Transfers.Delete(r.Context(), httpx.User(r), httpx.Scope(r).ID, chi.URLParam(r, "uuid")); err != nil {
		writeTransferError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) transfersResolve(accept bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "uuid")
		if err := h.Transfers.Resolve(r.Context(), httpx.User(r), httpx.Scope(r).ID, id, accept); err != nil {
			writeTransferError(w, err)
			return
		}
		if _, err := h.findTransfer(r, id); errors.Is(err, transfers.ErrGone) {
			w.WriteHeader(http.StatusNoContent) // the decision removed it here
			return
		}
		h.transferShow(w, r, id, http.StatusOK)
	}
}

func (h Handler) transfersTrust(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "uuid")
	if err := h.Transfers.Trust(r.Context(), httpx.User(r), httpx.Scope(r).ID, id); err != nil {
		writeTransferError(w, err)
		return
	}
	h.transferShow(w, r, id, http.StatusOK)
}
