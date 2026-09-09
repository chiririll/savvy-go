package httpserver

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/chiririll/savvy-plus/internal/domain"
	"github.com/go-chi/chi/v5"
)

func (s *Server) automationIndex(w http.ResponseWriter, r *http.Request) {
	list, err := s.automation.All(r.Context())
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, mapSlice(list, domain.AutomationRule.JSON))
}

func (s *Server) automationTriggers(w http.ResponseWriter, r *http.Request) {
	createL, createD := domainTriggerMeta("on_transaction_create")
	updateL, updateD := domainTriggerMeta("on_transaction_update")
	writeJSON(w, http.StatusOK, []map[string]string{
		{"value": "on_transaction_create", "label": createL, "description": createD},
		{"value": "on_transaction_update", "label": updateL, "description": updateD},
	})
}

func domainTriggerMeta(v string) (string, string) {
	switch v {
	case "on_transaction_create":
		return "On Transaction Create", "Triggers when a new transaction is created"
	case "on_transaction_update":
		return "On Transaction Update", "Triggers when a transaction is updated"
	default:
		return v, ""
	}
}

func (s *Server) automationStore(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeAutomation(w, r)
	if !ok {
		return
	}
	rule, err := s.automation.Create(r.Context(), in)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusCreated, rule.JSON())
}

func (s *Server) automationShow(w http.ResponseWriter, r *http.Request) {
	rule := s.automationParam(w, r)
	if rule == nil {
		return
	}
	writeData(w, http.StatusOK, rule.JSON())
}

func (s *Server) automationUpdate(w http.ResponseWriter, r *http.Request) {
	cur := s.automationParam(w, r)
	if cur == nil {
		return
	}
	in, ok := decodeAutomation(w, r)
	if !ok {
		return
	}
	rule, err := s.automation.Update(r.Context(), cur.ID, in)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusOK, rule.JSON())
}

func (s *Server) automationDestroy(w http.ResponseWriter, r *http.Request) {
	rule := s.automationParam(w, r)
	if rule == nil {
		return
	}
	if err := s.automation.Delete(r.Context(), rule.ID); err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) automationToggle(w http.ResponseWriter, r *http.Request) {
	rule := s.automationParam(w, r)
	if rule == nil {
		return
	}
	out, err := s.automation.Toggle(r.Context(), rule.ID)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeData(w, http.StatusOK, out.JSON())
}

func (s *Server) automationReorder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Rules []struct {
			ID       int64 `json:"id"`
			Priority int   `json:"priority"`
		} `json:"rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Rules) == 0 {
		writeValidation(w, map[string][]string{"rules": {"The rules field is required."}})
		return
	}
	if err := s.automation.Reorder(r.Context(), body.Rules); err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) automationTest(w http.ResponseWriter, r *http.Request) {
	rule := s.automationParam(w, r)
	if rule == nil {
		return
	}
	var body struct {
		TransactionID int64 `json:"transaction_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.TransactionID == 0 {
		writeValidation(w, map[string][]string{"transaction_id": {"The transaction id field is required."}})
		return
	}
	result, err := s.automation.Test(r.Context(), rule.ID, body.TransactionID)
	if err != nil {
		writeMessage(w, 422, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) automationLogs(w http.ResponseWriter, r *http.Request) {
	rule := s.automationParam(w, r)
	if rule == nil {
		return
	}
	logs, err := s.automation.Logs(r.Context(), rule.ID)
	if err != nil {
		writeMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeData(w, http.StatusOK, mapSlice(logs, domain.AutomationLog.JSON))
}

func (s *Server) automationParam(w http.ResponseWriter, r *http.Request) *domain.AutomationRule {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	rule, _ := s.automation.ByID(r.Context(), id)
	if rule == nil {
		writeMessage(w, http.StatusNotFound, "Not found.")
	}
	return rule
}

func decodeAutomation(w http.ResponseWriter, r *http.Request) (domain.AutomationInput, bool) {
	var body struct {
		Name           string           `json:"name"`
		Description    *string          `json:"description"`
		TriggerType    string           `json:"trigger_type"`
		Priority       int              `json:"priority"`
		Conditions     map[string]any   `json:"conditions"`
		Actions        []map[string]any `json:"actions"`
		IsActive       *bool            `json:"is_active"`
		StopProcessing *bool            `json:"stop_processing"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.TriggerType == "" {
		writeValidation(w, map[string][]string{"name": {"The name field is required."}})
		return domain.AutomationInput{}, false
	}
	if body.Conditions == nil {
		writeValidation(w, map[string][]string{"conditions": {"At least one condition is required."}})
		return domain.AutomationInput{}, false
	}
	if len(body.Actions) == 0 {
		writeValidation(w, map[string][]string{"actions": {"At least one action is required."}})
		return domain.AutomationInput{}, false
	}
	return domain.AutomationInput{
		Name: body.Name, Description: body.Description, TriggerType: body.TriggerType,
		Priority: body.Priority, Conditions: body.Conditions, Actions: body.Actions,
		IsActive: body.IsActive, StopProcessing: body.StopProcessing,
	}, true
}
