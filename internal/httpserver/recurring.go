package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"savvy-go/internal/domain"
	"savvy-go/internal/httpserver/dto"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

func (s *Server) recurringIndex(w http.ResponseWriter, r *http.Request) {
	list, err := sp(r).recurring.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, dto.Map(list, dto.NewRecurring))
}

func (s *Server) recurringUpcoming(w http.ResponseWriter, r *http.Request) {
	list, err := sp(r).recurring.Upcoming(r.Context(), 5)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, dto.Map(list, dto.NewRecurring))
}

func (s *Server) recurringStore(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeRecurring(w, r, nil)
	if !ok {
		return
	}
	rec, err := sp(r).recurring.Create(r.Context(), in)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusCreated, dto.NewRecurring(*rec))
}

func (s *Server) recurringShow(w http.ResponseWriter, r *http.Request) {
	rec := s.recurringParam(w, r)
	if rec == nil {
		return
	}
	writeData(w, http.StatusOK, dto.NewRecurring(*rec))
}

func (s *Server) recurringUpdate(w http.ResponseWriter, r *http.Request) {
	cur := s.recurringParam(w, r)
	if cur == nil {
		return
	}
	in, ok := decodeRecurring(w, r, cur)
	if !ok {
		return
	}
	rec, err := sp(r).recurring.Update(r.Context(), cur.ID, in)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusOK, dto.NewRecurring(*rec))
}

func (s *Server) recurringDestroy(w http.ResponseWriter, r *http.Request) {
	rec := s.recurringParam(w, r)
	if rec == nil {
		return
	}
	if err := sp(r).recurring.Delete(r.Context(), rec.ID); err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) recurringParam(w http.ResponseWriter, r *http.Request) *domain.Recurring {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	rec, _ := sp(r).recurring.ByID(r.Context(), id)
	if rec == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
	}
	return rec
}

// decodeRecurring reads a recurring template from the body. On update, base
// is the stored template: fields the body leaves out keep its values, and an
// explicit null clears a nullable one.
func decodeRecurring(w http.ResponseWriter, r *http.Request, base *domain.Recurring) (domain.RecurringInput, bool) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeValidation(w, map[string][]string{"type": {"The type field is required."}})
		return domain.RecurringInput{}, false
	}
	var body struct {
		Type        string           `json:"type"`
		AccountID   int64            `json:"account_id"`
		ToAccountID *int64           `json:"to_account_id"`
		CategoryID  *int64           `json:"category_id"`
		Amount      decimal.Decimal  `json:"amount"`
		ToAmount    *decimal.Decimal `json:"to_amount"`
		IsEstimated *bool            `json:"is_estimated"`
		Description *string          `json:"description"`
		Frequency   string           `json:"frequency"`
		Interval    int              `json:"interval"`
		DayOfWeek   *int             `json:"day_of_week"`
		DayOfMonth  *int             `json:"day_of_month"`
		StartDate   string           `json:"start_date"`
		EndDate     *string          `json:"end_date"`
		IsActive    *bool            `json:"is_active"`
		TagIDs      []int64          `json:"tag_ids"`
	}
	if base != nil {
		body.Type, body.AccountID, body.ToAccountID = base.Type, base.AccountID, clone(base.ToAccountID)
		body.CategoryID, body.Amount = clone(base.CategoryID), base.Amount.Decimal()
		if base.ToAmount != nil {
			toAmount := base.ToAmount.Decimal()
			body.ToAmount = &toAmount
		}
		body.IsEstimated, body.Description = clone(&base.IsEstimated), clone(base.Description)
		body.Frequency, body.Interval = base.Frequency, base.Interval
		body.DayOfWeek, body.DayOfMonth = clone(base.DayOfWeek), clone(base.DayOfMonth)
		body.StartDate, body.EndDate, body.IsActive = base.StartDate, clone(base.EndDate), clone(&base.IsActive)
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
		IsEstimated: body.IsEstimated, Description: body.Description, Frequency: body.Frequency, Interval: body.Interval,
		DayOfWeek: body.DayOfWeek, DayOfMonth: body.DayOfMonth, StartDate: body.StartDate,
		EndDate: body.EndDate, IsActive: body.IsActive, TagIDs: body.TagIDs, HasTagIDs: hasTags,
	}, true
}
