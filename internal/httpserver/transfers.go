package httpserver

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"

	"savvy-go/internal/domain"
	"savvy-go/internal/httpserver/dto"
	"savvy-go/internal/money"
	"savvy-go/internal/store"
)

func writeTransferError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrTransferGone):
		writeMessage(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrTransferRights), errors.Is(err, domain.ErrLinkRights):
		writeMessage(w, http.StatusForbidden, err.Error())
	case errors.Is(err, store.ErrQuotaExceeded):
		writeMessage(w, http.StatusUnprocessableEntity, "A space involved has reached its storage limit.")
	case errors.Is(err, store.ErrUnavailable):
		writeMessage(w, http.StatusUnprocessableEntity, "A space involved is temporarily unavailable.")
	case errors.Is(err, money.ErrOutOfRange):
		writeValidation(w, map[string][]string{"amount": {"The amount is out of range."}})
	default:
		writeMessage(w, http.StatusUnprocessableEntity, err.Error())
	}
}

// --- links -----------------------------------------------------------------

func (s *Server) linksIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.transfers.Links(r.Context(), sp(r).id)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, dto.Map(list, func(x domain.Space) map[string]any {
		return map[string]any{"id": x.ID, "name": x.Name}
	}))
}

// linksStore links the space with another the caller administers. A target
// the caller does not belong to is a 404, so spaces cannot be probed (P52).
func (s *Server) linksStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TargetSpaceID int64 `json:"target_space_id"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if role, _ := s.spaces.Role(r.Context(), body.TargetSpaceID, userFrom(r).ID); role == "" {
		writeMessage(w, http.StatusNotFound, "Space not found.")
		return
	}
	if err := s.transfers.Link(r.Context(), userFrom(r), sp(r).id, body.TargetSpaceID); err != nil {
		writeTransferError(w, err)
		return
	}
	s.linksIndex(w, r)
}

func (s *Server) linksDestroy(w http.ResponseWriter, r *http.Request) {
	other, _ := strconv.ParseInt(chi.URLParam(r, "other"), 10, 64)
	if err := s.transfers.Unlink(r.Context(), sp(r).id, other); err != nil {
		writeTransferError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- transfers -------------------------------------------------------------

func sideJSON(side domain.TransferSide, withAccount bool) map[string]any {
	out := map[string]any{
		"amount":   decimal.New(side.Amount, -int32(side.Decimals)).String(),
		"currency": side.Currency,
	}
	if withAccount {
		out["accountId"] = side.AccountID
	}
	return out
}

// transferJSON shows a transfer to a member of the request's space: the
// other side's account only to who belongs to that space too (P30).
func (s *Server) transferJSON(r *http.Request, v domain.TransferView) map[string]any {
	seesOther := false
	if v.OtherSpace != nil {
		role, _ := s.spaces.Role(r.Context(), v.OtherSpace.ID, userFrom(r).ID)
		seesOther = role != ""
	}
	from, to := sideJSON(v.From, v.Outgoing || seesOther), sideJSON(v.To, !v.Outgoing || seesOther)
	out := map[string]any{
		"uuid": v.UUID, "from": from, "to": to, "outgoing": v.Outgoing, "date": v.Date, "description": v.Description,
		"version": v.Version, "status": v.Status, "review": v.Review, "frozen": v.Frozen, "verified": v.Verified,
		"deleted": v.DeletedAt != nil,
	}
	if v.OtherSpace != nil {
		out["otherSpace"] = map[string]any{"id": v.OtherSpace.ID, "name": v.OtherSpace.Name}
	}
	if v.Remote != nil {
		out["remote"] = map[string]any{
			"from": sideJSON(v.Remote.From, false), "to": sideJSON(v.Remote.To, false),
			"date": v.Remote.Date, "description": v.Remote.Description, "deleted": v.Remote.DeletedAt != nil,
		}
	}
	return out
}

func (s *Server) transfersIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.transfers.List(r.Context(), sp(r).id)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, dto.Map(list, func(v domain.TransferView) map[string]any { return s.transferJSON(r, v) }))
}

// findTransfer is a transfer of the request's space as its members see it.
func (s *Server) findTransfer(r *http.Request, id string) (*domain.TransferView, error) {
	list, err := s.transfers.List(r.Context(), sp(r).id)
	if err != nil {
		return nil, err
	}
	for _, v := range list {
		if v.UUID == id {
			return &v, nil
		}
	}
	return nil, domain.ErrTransferGone
}

func (s *Server) transferShow(w http.ResponseWriter, r *http.Request, id string, status int) {
	v, err := s.findTransfer(r, id)
	if err != nil {
		writeTransferError(w, err)
		return
	}
	writeData(w, status, s.transferJSON(r, *v))
}

// validDate reports whether d is a YYYY-MM-DD date.
func validDate(d string) bool {
	_, err := time.Parse("2006-01-02", d)
	return err == nil
}

// transfersStore sends money from the request's space to a linked one.
func (s *Server) transfersStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ToSpaceID     int64           `json:"to_space_id"`
		FromAccountID int64           `json:"from_account_id"`
		ToAccountID   int64           `json:"to_account_id"`
		FromAmount    decimal.Decimal `json:"from_amount"`
		ToAmount      decimal.Decimal `json:"to_amount"`
		Date          string          `json:"date"`
		Description   *string         `json:"description"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if !validDate(body.Date) {
		writeValidation(w, map[string][]string{"date": {"The date must be YYYY-MM-DD."}})
		return
	}
	t, err := s.transfers.Create(r.Context(), userFrom(r), domain.TransferInput{
		FromSpace: sp(r).id, FromAccount: body.FromAccountID, FromAmount: body.FromAmount,
		ToSpace: body.ToSpaceID, ToAccount: body.ToAccountID, ToAmount: body.ToAmount,
		Date: body.Date, Description: body.Description,
	})
	if err != nil {
		writeTransferError(w, err)
		return
	}
	s.transferShow(w, r, t.UUID, http.StatusCreated)
}

func (s *Server) transfersUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		FromAccountID *int64           `json:"from_account_id"`
		ToAccountID   *int64           `json:"to_account_id"`
		FromAmount    *decimal.Decimal `json:"from_amount"`
		ToAmount      *decimal.Decimal `json:"to_amount"`
		Date          *string          `json:"date"`
		Description   *string          `json:"description"`
	}
	var raw map[string]json.RawMessage
	if !decodeBody(w, r, &raw) {
		return
	}
	buf, _ := json.Marshal(raw)
	if err := json.Unmarshal(buf, &body); err != nil {
		writeValidation(w, map[string][]string{"body": {"The given data was invalid."}})
		return
	}
	_, hasDescription := raw["description"]
	if body.Date != nil && !validDate(*body.Date) {
		writeValidation(w, map[string][]string{"date": {"The date must be YYYY-MM-DD."}})
		return
	}
	id := chi.URLParam(r, "uuid")
	_, err := s.transfers.Update(r.Context(), userFrom(r), sp(r).id, id, domain.TransferUpdate{
		FromAccount: body.FromAccountID, ToAccount: body.ToAccountID, FromAmount: body.FromAmount, ToAmount: body.ToAmount,
		Date: body.Date, Description: body.Description, HasDescription: hasDescription,
	})
	if err != nil {
		writeTransferError(w, err)
		return
	}
	s.transferShow(w, r, id, http.StatusOK)
}

func (s *Server) transfersDestroy(w http.ResponseWriter, r *http.Request) {
	if err := s.transfers.Delete(r.Context(), userFrom(r), sp(r).id, chi.URLParam(r, "uuid")); err != nil {
		writeTransferError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) transfersResolve(accept bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "uuid")
		if err := s.transfers.Resolve(r.Context(), userFrom(r), sp(r).id, id, accept); err != nil {
			writeTransferError(w, err)
			return
		}
		if _, err := s.findTransfer(r, id); errors.Is(err, domain.ErrTransferGone) {
			w.WriteHeader(http.StatusNoContent) // the decision removed it here
			return
		}
		s.transferShow(w, r, id, http.StatusOK)
	}
}

func (s *Server) transfersTrust(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "uuid")
	if err := s.transfers.Trust(r.Context(), userFrom(r), sp(r).id, id); err != nil {
		writeTransferError(w, err)
		return
	}
	s.transferShow(w, r, id, http.StatusOK)
}

// --- signing keys (server admins) -------------------------------------------

func (s *Server) keysIndex(w http.ResponseWriter, r *http.Request) {
	key := s.keys.Key()
	trusted, err := s.transfers.TrustedKeysList(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, map[string]any{
		"own": map[string]any{"kid": key.KID(), "publicKey": base64.StdEncoding.EncodeToString(key.Public())},
		"trusted": dto.Map(trusted, func(k domain.TrustedKey) map[string]any {
			return map[string]any{"kid": k.KID, "publicKey": k.PublicKey, "name": k.Name, "createdAt": timeOrNil(k.CreatedAt)}
		}),
	})
}

func (s *Server) keysStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      string `json:"name"`
		PublicKey string `json:"public_key"`
	}
	if !decodeBody(w, r, &body) {
		return
	}
	if _, err := s.transfers.TrustKey(r.Context(), userFrom(r), body.Name, body.PublicKey); err != nil {
		writeValidation(w, map[string][]string{"public_key": {err.Error()}})
		return
	}
	s.keysIndex(w, r)
}

func (s *Server) keysDestroy(w http.ResponseWriter, r *http.Request) {
	if err := s.transfers.UntrustKey(r.Context(), userFrom(r), chi.URLParam(r, "kid")); err != nil {
		writeTransferError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) keysRotate(w http.ResponseWriter, r *http.Request) {
	if _, err := s.transfers.RotateKey(r.Context(), userFrom(r), s.cfg.DataDir); err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.keysIndex(w, r)
}
