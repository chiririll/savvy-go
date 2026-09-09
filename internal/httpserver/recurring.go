package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/chiririll/savvy-plus/internal/domain"
	"github.com/go-chi/chi/v5"
)

func (s *Server) recurringIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.recurring.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, mapSlice(list, domain.Recurring.JSON))
}

func (s *Server) recurringUpcoming(w http.ResponseWriter, r *http.Request) {
	list, err := s.recurring.Upcoming(r.Context(), 5)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, mapSlice(list, domain.Recurring.JSON))
}

func (s *Server) recurringStore(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeRecurring(w, r)
	if !ok {
		return
	}
	rec, err := s.recurring.Create(r.Context(), in)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusCreated, rec.JSON())
}

func (s *Server) recurringShow(w http.ResponseWriter, r *http.Request) {
	rec := s.recurringParam(w, r)
	if rec == nil {
		return
	}
	writeData(w, http.StatusOK, rec.JSON())
}

func (s *Server) recurringUpdate(w http.ResponseWriter, r *http.Request) {
	cur := s.recurringParam(w, r)
	if cur == nil {
		return
	}
	in, ok := decodeRecurring(w, r)
	if !ok {
		return
	}
	rec, err := s.recurring.Update(r.Context(), cur.ID, in)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusOK, rec.JSON())
}

func (s *Server) recurringDestroy(w http.ResponseWriter, r *http.Request) {
	rec := s.recurringParam(w, r)
	if rec == nil {
		return
	}
	if err := s.recurring.Delete(r.Context(), rec.ID); err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) recurringParam(w http.ResponseWriter, r *http.Request) *domain.Recurring {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	rec, _ := s.recurring.ByID(r.Context(), id)
	if rec == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
	}
	return rec
}

func decodeRecurring(w http.ResponseWriter, r *http.Request) (domain.RecurringInput, bool) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeValidation(w, map[string][]string{"type": {"The type field is required."}})
		return domain.RecurringInput{}, false
	}
	var body struct {
		Type        string   `json:"type"`
		AccountID   int64    `json:"account_id"`
		ToAccountID *int64   `json:"to_account_id"`
		CategoryID  *int64   `json:"category_id"`
		Amount      float64  `json:"amount"`
		ToAmount    *float64 `json:"to_amount"`
		Description *string  `json:"description"`
		Frequency   string   `json:"frequency"`
		Interval    int      `json:"interval"`
		DayOfWeek   *int     `json:"day_of_week"`
		DayOfMonth  *int     `json:"day_of_month"`
		StartDate   string   `json:"start_date"`
		EndDate     *string  `json:"end_date"`
		IsActive    *bool    `json:"is_active"`
		TagIDs      []int64  `json:"tag_ids"`
	}
	buf, _ := json.Marshal(raw)
	if err := json.Unmarshal(buf, &body); err != nil {
		writeValidation(w, map[string][]string{"type": {"The given data was invalid."}})
		return domain.RecurringInput{}, false
	}
	_, hasTags := raw["tag_ids"]
	return domain.RecurringInput{
		Type: body.Type, AccountID: body.AccountID, ToAccountID: body.ToAccountID,
		CategoryID: body.CategoryID, Amount: body.Amount, ToAmount: body.ToAmount,
		Description: body.Description, Frequency: body.Frequency, Interval: body.Interval,
		DayOfWeek: body.DayOfWeek, DayOfMonth: body.DayOfMonth, StartDate: body.StartDate,
		EndDate: body.EndDate, IsActive: body.IsActive, TagIDs: body.TagIDs, HasTagIDs: hasTags,
	}, true
}
