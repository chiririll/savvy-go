package httpserver

import (
	"net/http"
	"regexp"
	"strings"

	"savvy-go/internal/ledger"
	"savvy-go/internal/seed"
)

var currencyCode = regexp.MustCompile(`^[A-Za-z0-9]{2,10}$`)

// serverCurrencyCatalog lists every currency the data service knows, priced in
// ?base= (USD by default). Unlike a space's catalog it hides nothing: the
// create-space modal needs it before the space exists.
func (s *Server) serverCurrencyCatalog(w http.ResponseWriter, r *http.Request) {
	base := strings.TrimSpace(r.URL.Query().Get("base"))
	if base == "" {
		base = "USD"
	}
	if !currencyCode.MatchString(base) {
		writeValidation(w, map[string][]string{"base": {"The base must be a currency code."}})
		return
	}
	list := s.rates.Catalog(r.Context(), base)
	out := make([]map[string]any, 0, len(list))
	for _, c := range list {
		out = append(out, ledger.CatalogEntry(c))
	}
	writeData(w, http.StatusOK, out)
}

// presets are the categories and tags a new space can start with.
func (s *Server) presets(w http.ResponseWriter, r *http.Request) {
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
	writeData(w, http.StatusOK, map[string]any{"categories": out, "tags": seed.DefaultTags()})
}
