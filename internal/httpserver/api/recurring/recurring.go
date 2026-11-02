// Package recurring serves /recurring of a space.
package recurring

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"savvy-go/internal/httpserver/httpx"

	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/recurring"
)

func index(w http.ResponseWriter, r *http.Request) {
	list, err := httpx.Scope(r).Recurring.All(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, dto.NewRecurring))
}

func upcoming(w http.ResponseWriter, r *http.Request) {
	list, err := httpx.Scope(r).Recurring.Upcoming(r.Context(), 5)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, dto.NewRecurring))
}

func store(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeRecurring(w, r, nil)
	if !ok {
		return
	}
	rec, err := httpx.Scope(r).Recurring.Create(r.Context(), in)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewRecurring(*rec))
}

func show(w http.ResponseWriter, r *http.Request) {
	rec := param(w, r)
	if rec == nil {
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewRecurring(*rec))
}

func update(w http.ResponseWriter, r *http.Request) {
	cur := param(w, r)
	if cur == nil {
		return
	}
	in, ok := decodeRecurring(w, r, cur)
	if !ok {
		return
	}
	rec, err := httpx.Scope(r).Recurring.Update(r.Context(), cur.ID, in)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewRecurring(*rec))
}

func destroy(w http.ResponseWriter, r *http.Request) {
	rec := param(w, r)
	if rec == nil {
		return
	}
	if err := httpx.Scope(r).Recurring.Delete(r.Context(), rec.ID); err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func param(w http.ResponseWriter, r *http.Request) *recurring.Rule {
	return httpx.Lookup(w, r, httpx.Scope(r).Recurring.ByID)
}

// decodeRecurring reads a rule from the body: template (what to create, in the
// shape of "create transaction"), schedule (when) and is_active. On update, what
// the body leaves out is kept.
func decodeRecurring(w http.ResponseWriter, r *http.Request, base *recurring.Rule) (recurring.Input, bool) {
	var body struct {
		Template json.RawMessage     `json:"template"`
		Schedule *recurring.Schedule `json:"schedule"`
		IsActive *bool               `json:"is_active"`
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		httpx.WriteMessage(w, 422, "The given data was invalid: "+err.Error())
		return recurring.Input{}, false
	}
	in := recurring.Input{Template: body.Template, IsActive: body.IsActive}
	if body.Schedule != nil {
		in.Schedule = *body.Schedule
	}
	if base != nil {
		if in.Template == nil {
			in.Template = base.Template
		}
		if body.Schedule == nil {
			in.Schedule = base.Schedule
		}
	}
	if in.Template == nil {
		httpx.WriteValidation(w, map[string][]string{"template": {"The template field is required."}})
		return recurring.Input{}, false
	}
	return in, true
}
