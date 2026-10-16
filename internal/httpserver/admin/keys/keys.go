// Package keys serves the server admin's signing keys: this server's own key and the
// keys of the servers whose cross-space transfers it trusts.
package keys

import (
	"encoding/base64"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/signing"
	"savvy-go/internal/transfers"
)

// Handler holds what the key routes need.
type Handler struct {
	DataDir   string
	Transfers *transfers.Service
	Keys      *signing.Holder
}

func (h Handler) keysIndex(w http.ResponseWriter, r *http.Request) {
	key := h.Keys.Key()
	trusted, err := h.Transfers.TrustedKeysList(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Keys{
		Own:     dto.SigningKey{KID: key.KID(), PublicKey: base64.StdEncoding.EncodeToString(key.Public())},
		Trusted: dto.Map(trusted, dto.NewTrustedKey),
	})
}

func (h Handler) keysStore(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      string `json:"name"`
		PublicKey string `json:"public_key"`
	}
	if !httpx.DecodeBody(w, r, &body) {
		return
	}
	if _, err := h.Transfers.TrustKey(r.Context(), httpx.User(r), body.Name, body.PublicKey); err != nil {
		httpx.WriteValidation(w, map[string][]string{"public_key": {err.Error()}})
		return
	}
	h.keysIndex(w, r)
}

func (h Handler) keysDestroy(w http.ResponseWriter, r *http.Request) {
	if err := h.Transfers.UntrustKey(r.Context(), httpx.User(r), chi.URLParam(r, "kid")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) keysRotate(w http.ResponseWriter, r *http.Request) {
	if _, err := h.Transfers.RotateKey(r.Context(), httpx.User(r), h.DataDir); err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	h.keysIndex(w, r)
}

func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, transfers.ErrGone):
		httpx.WriteMessage(w, http.StatusNotFound, err.Error())
	case errors.Is(err, transfers.ErrRights):
		httpx.WriteMessage(w, http.StatusForbidden, err.Error())
	default:
		httpx.WriteMessage(w, http.StatusUnprocessableEntity, err.Error())
	}
}
