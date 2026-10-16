package apitokens

import (
	_ "embed"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/routing"
)

//go:embed openapi.yaml
var spec []byte

func init() {
	routing.Register(spec, func(d *routing.Deps) []routing.Mount {
		h := Handler{Tokens: d.APITokens, Spaces: d.Spaces}
		return []routing.Mount{
			routing.Credential(h.Routes),
		}
	})
}

func (h Handler) Routes(r chi.Router) {
	r.Get("/auth/api-token-capabilities", h.apiTokenCapabilities)
	r.Get("/auth/api-tokens", h.apiTokensIndex)
	r.Post("/auth/api-tokens", h.apiTokensStore)
	r.Patch("/auth/api-tokens/{id}", h.apiTokensUpdate)
	r.Delete("/auth/api-tokens/{id}", h.apiTokensDestroy)
}
