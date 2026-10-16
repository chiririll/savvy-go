package catalog

import (
	_ "embed"
	"net/http"
	"regexp"
	"strings"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
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
		httpx.WriteValidation(w, map[string][]string{"base": {"The base must be a currency code."}})
		return
	}
	list := h.Rates.Catalog(r.Context(), base)
	httpx.WriteData(w, http.StatusOK, dto.Map(list, dto.NewCatalogCurrency))
}

type presetCategory struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	Icon      string `json:"icon"`
	Color     string `json:"color"`
	IsDefault bool   `json:"isDefault"`
}

// presetsBody is the categories and tags a new space can start with.
type presetsBody struct {
	Categories []presetCategory `json:"categories"`
	Tags       []string         `json:"tags"`
}

// presets answers with what a new space can start with.
func (Handler) presets(w http.ResponseWriter, r *http.Request) {
	cats := seed.DefaultCategories()
	out := make([]presetCategory, len(cats))
	for i, c := range cats {
		out[i] = presetCategory(c)
	}
	httpx.WriteData(w, http.StatusOK, presetsBody{Categories: out, Tags: seed.DefaultTags()})
}

// Handler serves what a new space can start with, before the space exists.
type Handler struct{ Rates ledger.RateSource }
