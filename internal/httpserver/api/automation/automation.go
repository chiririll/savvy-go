// Package automation serves /automation-rules of a space.
package automation

import (
	_ "embed"
	"encoding/json"
	"net/http"
	"savvy-go/internal/httpserver/httpx"
	"strconv"

	"github.com/go-chi/chi/v5"

	"savvy-go/internal/automation"
	"savvy-go/internal/httpserver/httpx/dto"
)

func automationIndex(w http.ResponseWriter, r *http.Request) {
	list, err := httpx.Scope(r).Automation.All(r.Context())
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, dto.NewAutomationRule))
}

// automationTriggers lists the trigger types; the SPA localizes them.
func automationTriggers(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, []string{"on_transaction_create", "on_transaction_update"})
}

func automationStore(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeAutomation(w, r, nil)
	if !ok {
		return
	}
	rule, err := httpx.Scope(r).Automation.Create(r.Context(), in)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusCreated, dto.NewAutomationRule(*rule))
}

func automationShow(w http.ResponseWriter, r *http.Request) {
	rule := automationParam(w, r)
	if rule == nil {
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewAutomationRule(*rule))
}

func automationUpdate(w http.ResponseWriter, r *http.Request) {
	cur := automationParam(w, r)
	if cur == nil {
		return
	}
	in, ok := decodeAutomation(w, r, cur)
	if !ok {
		return
	}
	rule, err := httpx.Scope(r).Automation.Update(r.Context(), cur.ID, in)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewAutomationRule(*rule))
}

func automationDestroy(w http.ResponseWriter, r *http.Request) {
	rule := automationParam(w, r)
	if rule == nil {
		return
	}
	if err := httpx.Scope(r).Automation.Delete(r.Context(), rule.ID); err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func automationToggle(w http.ResponseWriter, r *http.Request) {
	rule := automationParam(w, r)
	if rule == nil {
		return
	}
	out, err := httpx.Scope(r).Automation.Toggle(r.Context(), rule.ID)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.NewAutomationRule(*out))
}

func automationReorder(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Rules []struct {
			ID       int64 `json:"id"`
			Priority int   `json:"priority"`
		} `json:"rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Rules) == 0 {
		httpx.WriteValidation(w, map[string][]string{"rules": {"The rules field is required."}})
		return
	}
	if err := httpx.Scope(r).Automation.Reorder(r.Context(), body.Rules); err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteSuccess(w)
}

func automationTest(w http.ResponseWriter, r *http.Request) {
	rule := automationParam(w, r)
	if rule == nil {
		return
	}
	var body struct {
		TransactionID int64 `json:"transaction_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.TransactionID == 0 {
		httpx.WriteValidation(w, map[string][]string{"transaction_id": {"The transaction id field is required."}})
		return
	}
	result, err := httpx.Scope(r).Automation.Test(r.Context(), rule.ID, body.TransactionID)
	if err != nil {
		httpx.WriteMessage(w, 422, err.Error())
		return
	}
	httpx.WriteJSON(w, http.StatusOK, result)
}

func automationLogs(w http.ResponseWriter, r *http.Request) {
	rule := automationParam(w, r)
	if rule == nil {
		return
	}
	logs, err := httpx.Scope(r).Automation.Logs(r.Context(), rule.ID)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(logs, dto.NewAutomationLog))
}

func automationParam(w http.ResponseWriter, r *http.Request) *automation.Rule {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	rule, _ := httpx.Scope(r).Automation.ByID(r.Context(), id)
	if rule == nil {
		httpx.WriteMessage(w, http.StatusNotFound, "Not found.")
	}
	return rule
}

// decodeAutomation reads a rule from the body; on update, over base (see
// patch.go).
func decodeAutomation(w http.ResponseWriter, r *http.Request, base *automation.Rule) (automation.Input, bool) {
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
	if base != nil {
		body.Name, body.Description, body.TriggerType = base.Name, httpx.Clone(base.Description), base.TriggerType
		body.Priority, body.IsActive, body.StopProcessing = base.Priority, httpx.Clone(
			&base.IsActive,
		), httpx.Clone(
			&base.StopProcessing,
		)
	}
	err := json.NewDecoder(r.Body).Decode(&body)
	// Decoding into a filled map would merge keys, so these are taken whole.
	if base != nil && body.Conditions == nil {
		body.Conditions = base.Conditions
	}
	if base != nil && body.Actions == nil {
		body.Actions = base.Actions
	}
	if err != nil || body.Name == "" || body.TriggerType == "" {
		httpx.WriteValidation(w, map[string][]string{"name": {"The name field is required."}})
		return automation.Input{}, false
	}
	if body.Conditions == nil {
		httpx.WriteValidation(w, map[string][]string{"conditions": {"At least one condition is required."}})
		return automation.Input{}, false
	}
	if len(body.Actions) == 0 {
		httpx.WriteValidation(w, map[string][]string{"actions": {"At least one action is required."}})
		return automation.Input{}, false
	}
	return automation.Input{
		Name: body.Name, Description: body.Description, TriggerType: body.TriggerType,
		Priority: body.Priority, Conditions: body.Conditions, Actions: body.Actions,
		IsActive: body.IsActive, StopProcessing: body.StopProcessing,
	}, true
}
