// Package categories serves /categories of a space.
package categories

import (
	_ "embed"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/ledger"
	"savvy-go/internal/refsync"
	"savvy-go/internal/reports"
)

func (s Handler) categoriesIndex(w http.ResponseWriter, r *http.Request) {
	list, err := httpx.Scope(r).Categories.All(r.Context(), r.URL.Query().Get("type"))
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, dto.NewCategory))
}

func (s Handler) categoriesStore(w http.ResponseWriter, r *http.Request) {
	var body ledger.Category
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		httpx.WriteValidation(w, map[string][]string{"name": {"The name field is required."}})
		return
	}
	c, err := s.Refsync.CreateCategory(r.Context(), httpx.User(r), httpx.Scope(r).ID, body)
	if err != nil {
		if !httpx.WriteSyncError(w, err) {
			httpx.WriteMessage(w, 422, err.Error())
		}
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewCategory(*c))
}

func (s Handler) categoriesShow(w http.ResponseWriter, r *http.Request) {
	c := s.categoryParam(w, r)
	if c == nil {
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewCategory(*c))
}

func (s Handler) categoriesUpdate(w http.ResponseWriter, r *http.Request) {
	cur := s.categoryParam(w, r)
	if cur == nil {
		return
	}
	body := *cur
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpx.WriteValidation(w, map[string][]string{"name": {"The given data was invalid."}})
		return
	}
	c, err := s.Refsync.UpdateCategory(r.Context(), httpx.User(r), httpx.Scope(r).ID, cur.ID, body)
	if err != nil {
		if !httpx.WriteSyncError(w, err) {
			httpx.WriteMessage(w, 422, err.Error())
		}
		return
	}
	if c == nil {
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewCategory(*c))
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
			httpx.WriteValidation(w, map[string][]string{"successor_id": {"The successor id is invalid."}})
			return
		}
		successorID = &id
	}
	if err := s.Refsync.DeleteCategory(r.Context(), httpx.User(r), httpx.Scope(r).ID, c.ID, successorID); err != nil {
		var mirror *refsync.MirrorError
		switch {
		case errors.As(err, &mirror): // it failed in another space of the group
			httpx.WriteMessage(w, 422, "In "+mirror.Name+": "+categoryDeleteMessage(mirror.Err))
		case httpx.WriteSyncError(w, err):
		default:
			httpx.WriteMessage(w, 422, categoryDeleteMessage(err))
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
	updated, err := s.Refsync.SetDefaultCategory(r.Context(), httpx.User(r), httpx.Scope(r).ID, c.ID)
	if err != nil {
		if !httpx.WriteSyncError(w, err) {
			httpx.WriteMessage(w, 422, err.Error())
		}
		return
	}
	if updated == nil {
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewCategory(*updated))
}

func (s Handler) categoriesStatistics(w http.ResponseWriter, r *http.Request) {
	c := s.categoryParam(w, r)
	if c == nil {
		return
	}
	stats, err := httpx.Scope(
		r,
	).Categories.Statistics(
		r.Context(),
		c.ID,
		r.URL.Query().Get("start_date"),
		r.URL.Query().Get("end_date"),
	)
	if err != nil || stats == nil {
		httpx.WriteMessage(w, 422, "Could not compute category statistics.")
		return
	}
	httpx.WriteJSON(w, http.StatusOK, dto.NewCategoryStatistics(*stats))
}

func (s Handler) categoriesSummary(w http.ResponseWriter, r *http.Request) {
	typ := r.URL.Query().Get("type")
	if typ != "income" && typ != "expense" {
		httpx.WriteValidation(w, map[string][]string{"type": {"The type field is required."}})
		return
	}
	q := r.URL.Query()
	filter := reports.Filter{
		PeriodType: "custom",
		StartDate:  q.Get("start_date"),
		EndDate:    q.Get("end_date"),
	}
	if filter.Validate() != nil {
		httpx.WriteValidation(w, map[string][]string{"end_date": {"The period must be a valid date range."}})
		return
	}
	list, total := httpx.Scope(r).Reports.CategorySummary(r.Context(), filter, typ)
	base, _ := httpx.Scope(r).Currencies.Base(r.Context())
	code := ""
	if base != nil {
		code = base.Code
	}
	httpx.WriteJSON(
		w,
		http.StatusOK,
		dto.CategorySummary{Data: dto.Map(list, dto.NewCategory), Total: total, Currency: code},
	)
}

func (s Handler) categoryParam(w http.ResponseWriter, r *http.Request) *ledger.Category {
	return httpx.Lookup(w, r, httpx.Scope(r).Categories.ByID)
}
