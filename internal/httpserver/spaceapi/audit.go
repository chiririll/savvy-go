package spaceapi

import (
	"net/http"

	"savvy-go/internal/httpserver/api"
	"savvy-go/internal/httpserver/dto"
	"savvy-go/internal/spaces"
)

func (h Handler) spaceAudit(w http.ResponseWriter, r *http.Request) {
	list, err := h.Spaces.AuditLog(r.Context(), api.Scope(r).ID)
	if err != nil {
		api.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	api.WriteData(w, http.StatusOK, dto.Map(list, auditJSON))
}

func auditJSON(e spaces.AuditEntry) map[string]any {
	return map[string]any{"id": e.ID, "actor": e.Actor, "action": e.Action, "targetUserId": e.TargetUserID,
		"details": e.Details, "createdAt": api.TimeOrNil(e.CreatedAt)}
}
