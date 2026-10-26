package transactions

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/ledger"
)

// decodeLegs reads the legs of a payment from the body.
func decodeLegs(w http.ResponseWriter, r *http.Request) ([]ledger.TxInput, bool) {
	var body struct {
		Legs []txBody `json:"legs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteValidation(w, map[string][]string{"legs": {"The legs field is required."}})
		return nil, false
	}
	legs := make([]ledger.TxInput, 0, len(body.Legs))
	for _, leg := range body.Legs {
		if leg.AccountID == 0 {
			httpx.WriteValidation(w, map[string][]string{"legs": {"Every leg needs an account."}})
			return nil, false
		}
		legs = append(legs, leg.input())
	}
	return legs, true
}

// groupFailed answers an error of a payment: a missing one is 404, a refused
// one 422.
func groupFailed(w http.ResponseWriter, err error) {
	if errors.Is(err, ledger.ErrGroupNotFound) {
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	httpx.WriteMessage(w, http.StatusUnprocessableEntity, err.Error())
}

// groupSettled runs the rules for the legs just written, as for a transaction
// created on its own, and returns the payment as it is after them.
func groupSettled(r *http.Request, g *ledger.Group, trigger string) *ledger.Group {
	scope := httpx.Scope(r)
	for i := range g.Legs {
		scope.Automation.Process(r.Context(), trigger, &g.Legs[i])
	}
	if fresh, err := scope.Groups.ByUUID(r.Context(), g.UUID); err == nil && fresh != nil {
		return fresh
	}
	return g
}

func groupsStore(w http.ResponseWriter, r *http.Request) {
	legs, ok := decodeLegs(w, r)
	if !ok {
		return
	}
	g, err := httpx.Scope(r).Groups.Create(r.Context(), legs)
	if err != nil {
		groupFailed(w, err)
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewTransactionGroup(*groupSettled(r, g, "on_transaction_create")))
}

func groupsShow(w http.ResponseWriter, r *http.Request) {
	g, err := httpx.Scope(r).Groups.ByUUID(r.Context(), chi.URLParam(r, "uuid"))
	if err != nil || g == nil {
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewTransactionGroup(*g))
}

func groupsUpdate(w http.ResponseWriter, r *http.Request) {
	legs, ok := decodeLegs(w, r)
	if !ok {
		return
	}
	g, err := httpx.Scope(r).Groups.Update(r.Context(), chi.URLParam(r, "uuid"), legs)
	if err != nil {
		groupFailed(w, err)
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewTransactionGroup(*groupSettled(r, g, "on_transaction_update")))
}

func groupsDestroy(w http.ResponseWriter, r *http.Request) {
	if err := httpx.Scope(r).Groups.Delete(r.Context(), chi.URLParam(r, "uuid")); err != nil {
		groupFailed(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
