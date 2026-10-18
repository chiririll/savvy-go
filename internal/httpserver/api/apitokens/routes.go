package apitokens

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/auth"
	"savvy-go/internal/httpserver/httpx/confirm"
	"savvy-go/internal/httpserver/httpx/routing"
	"savvy-go/internal/httpserver/ui"
	"savvy-go/internal/spaces"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	ui.AddSpec(spec)
	routing.Register(func(d *routing.Deps) []routing.Mount {
		h := Handler{
			Tokens:  routing.Get[auth.APITokens](d),
			Spaces:  routing.Get[spaces.Service](d),
			Confirm: confirm.Confirmer{TwoFactor: routing.Get[auth.TwoFactor](d), Guard: d.Guard},
		}
		return []routing.Mount{
			routing.Credential(h.Routes),
		}
	})
}

// Handler holds what the API token routes need.
type Handler struct {
	Tokens auth.APITokens
	Spaces spaces.Service
	// Confirm asks for the password or code before a token is issued or widened.
	Confirm confirm.Confirmer
}

func (h Handler) Routes(r chi.Router) {
	r.Get("/auth/api-token-capabilities", h.apiTokenCapabilities)
	r.Get("/auth/api-tokens", h.apiTokensIndex)
	r.Post("/auth/api-tokens", h.apiTokensStore)
	r.Patch("/auth/api-tokens/{id}", h.apiTokensUpdate)
	r.Delete("/auth/api-tokens/{id}", h.apiTokensDestroy)
}
