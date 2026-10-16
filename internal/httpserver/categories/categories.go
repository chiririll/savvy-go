// Package categories serves /categories of a space.
package categories

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/dto"
	"savvy-go/internal/ledger"
	"savvy-go/internal/refsync"
	"savvy-go/internal/reports"
)

// Handler serves the categories routes; changes go through the sync groups.
type Handler struct{ Refsync refsync.Service }

func (s Handler) categoriesIndex(w http.ResponseWriter, r *http.Request) {
	list, err := api.Scope(r).Categories.All(r.Context(), r.URL.Query().Get("type"))
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.WriteData(w, http.StatusOK, dto.Map(list, dto.NewCategory))
}

func (s Handler) categoriesStore(w http.ResponseWriter, r *http.Request) {
	var body ledger.Category
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		api.WriteValidation(w, map[string][]string{"name": {"The name field is required."}})
		return
	}
	c, err := s.Refsync.CreateCategory(r.Context(), api.User(r), api.Scope(r).ID, body)
	if err != nil {
		if !api.WriteSyncError(w, err) {
			api.WriteMessage(w, 422, err.Error())
		}
		return
	}
	api.WriteData(w, http.StatusCreated, dto.NewCategory(*c))
}

func (s Handler) categoriesShow(w http.ResponseWriter, r *http.Request) {
	c := s.categoryParam(w, r)
	if c == nil {
		return
	}
	api.WriteData(w, http.StatusOK, dto.NewCategory(*c))
}

func (s Handler) categoriesUpdate(w http.ResponseWriter, r *http.Request) {
	cur := s.categoryParam(w, r)
	if cur == nil {
		return
	}
	body := *cur
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return
	}
	c, err := s.Refsync.UpdateCategory(r.Context(), api.User(r), api.Scope(r).ID, cur.ID, body)
	if err != nil {
		if !api.WriteSyncError(w, err) {
			api.WriteMessage(w, 422, err.Error())
		}
		return
	}
	if c == nil {
		api.WriteMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	api.WriteData(w, http.StatusOK, dto.NewCategory(*c))
}

func (s Handler) categoriesDestroy(w http.ResponseWriter, r *http.Request) {
	c := s.categoryParam(w, r)
	if c == nil {
		return
	}
	var successorID *int64
	if raw := r.URL.Query().Get("successor_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			api.WriteValidation(w, map[string][]string{"successor_id": {"The successor id is invalid."}})
			return
		}
		successorID = &id
	}
	if err := s.Refsync.DeleteCategory(r.Context(), api.User(r), api.Scope(r).ID, c.ID, successorID); err != nil {
		var mirror *refsync.MirrorError
		switch {
		case errors.As(err, &mirror): // it failed in another space of the group
			api.WriteMessage(w, 422, "In "+mirror.Name+": "+categoryDeleteMessage(mirror.Err))
		case api.WriteSyncError(w, err):
		default:
			api.WriteMessage(w, 422, categoryDeleteMessage(err))
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func categoryDeleteMessage(err error) string {
	switch err.Error() {
	case "default":
		return "Cannot delete the default category. Set another category as default first."
	case "has transactions":
		return "Cannot delete category that has transactions."
	case "invalid successor":
		return "The selected replacement category is invalid."
	case "last":
		return "Cannot delete the last category of this type."
	}
	return err.Error()
}

func (s Handler) categoriesSetDefault(w http.ResponseWriter, r *http.Request) {
	c := s.categoryParam(w, r)
	if c == nil {
		return
	}
	updated, err := s.Refsync.SetDefaultCategory(r.Context(), api.User(r), api.Scope(r).ID, c.ID)
	if err != nil {
		if !api.WriteSyncError(w, err) {
			api.WriteMessage(w, 422, err.Error())
		}
		return
	}
	if updated == nil {
		api.WriteMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	api.WriteData(w, http.StatusOK, dto.NewCategory(*updated))
}

func (s Handler) categoriesStatistics(w http.ResponseWriter, r *http.Request) {
	c := s.categoryParam(w, r)
	if c == nil {
		return
	}
	stats, err := api.Scope(r).Categories.Statistics(r.Context(), c.ID, r.URL.Query().Get("start_date"), r.URL.Query().Get("end_date"))
	if err != nil || stats == nil {
		api.WriteMessage(w, 422, "Could not compute category statistics.")
		return
	}
	api.WriteJSON(w, http.StatusOK, dto.NewCategoryStatistics(*stats))
}

func (s Handler) categoriesSummary(w http.ResponseWriter, r *http.Request) {
	typ := r.URL.Query().Get("type")
	if typ != "income" && typ != "expense" {
		api.WriteValidation(w, map[string][]string{"type": {"The type field is required."}})
		return
	}
	q := r.URL.Query()
	list, total := api.Scope(r).Reports.CategorySummary(r.Context(), reports.Filter{
		PeriodType: "custom",
		StartDate:  q.Get("start_date"),
		EndDate:    q.Get("end_date"),
	}, typ)
	base, _ := api.Scope(r).Currencies.Base(r.Context())
	code := ""
	if base != nil {
		code = base.Code
	}
	api.WriteJSON(w, http.StatusOK, map[string]any{
		"data":     dto.Map(list, dto.NewCategory),
		"total":    total,
		"currency": code,
	})
}

func (s Handler) categoryParam(w http.ResponseWriter, r *http.Request) *ledger.Category {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	c, _ := api.Scope(r).Categories.ByID(r.Context(), id)
	if c == nil {
		api.WriteMessage(w, http.StatusNotFound, "Not found.")
	}
	return c
}

func (s Handler) Routes(r chi.Router) {
	r.Get("/categories", s.categoriesIndex)
	r.Post("/categories", s.categoriesStore)
	r.Get("/categories/{id}", s.categoriesShow)
	r.Put("/categories/{id}", s.categoriesUpdate)
	r.Patch("/categories/{id}", s.categoriesUpdate)
	r.Delete("/categories/{id}", s.categoriesDestroy)
	r.Post("/categories/{id}/set-default", s.categoriesSetDefault)
	r.Get("/categories/{id}/statistics", s.categoriesStatistics)
	r.Get("/categories-summary", s.categoriesSummary)
}
