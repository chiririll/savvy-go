package catalog

import (
	_ "embed"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/apidocs"
	"savvy-go/internal/ledger"
	"savvy-go/internal/seed"
)

var currencyCode = regexp.MustCompile(`^[A-Za-z0-9]{2,10}$`)

// serverCurrencyCatalog lists every currency the data service knows, priced in
// ?base= (USD by default). Unlike a space's catalog it hides nothing: the
// create-space modal needs it before the space exists.
func (h Handler) currencies(w http.ResponseWriter, r *http.Request) {
	base := strings.TrimSpace(r.URL.Query().Get("base"))
	if base == "" {
		base = "USD"
	}
	if !currencyCode.MatchString(base) {
		api.WriteValidation(w, map[string][]string{"base": {"The base must be a currency code."}})
		return
	}
	list := h.Rates.Catalog(r.Context(), base)
	out := make([]map[string]any, 0, len(list))
	for _, c := range list {
		out = append(out, ledger.CatalogEntry(c))
	}
	api.WriteData(w, http.StatusOK, out)
}

// presets are the categories and tags a new space can start with.
func (Handler) presets(w http.ResponseWriter, r *http.Request) {
	type category struct {
		Name      string `json:"name"`
		Type      string `json:"type"`
		Icon      string `json:"icon"`
		Color     string `json:"color"`
		IsDefault bool   `json:"isDefault"`
	}
	cats := seed.DefaultCategories()
	out := make([]category, len(cats))
	for i, c := range cats {
		out[i] = category(c)
	}
	api.WriteData(w, http.StatusOK, map[string]any{"categories": out, "tags": seed.DefaultTags()})
}

// Handler serves what a new space can start with, before the space exists.
type Handler struct{ Rates ledger.RateSource }

//go:embed openapi.yaml
var spec []byte

func (h Handler) Routes(r chi.Router) {
	apidocs.Add(spec)
	r.Get("/currencies/catalog", h.currencies)
	r.Get("/presets", h.presets)
}
