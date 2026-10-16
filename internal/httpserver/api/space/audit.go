package space

import (
	"net/http"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
	"savvy-go/internal/spaces"
)

func (h Handler) spaceAudit(w http.ResponseWriter, r *http.Request) {
	list, err := h.Spaces.AuditLog(r.Context(), httpx.Scope(r).ID)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, auditJSON))
}

func auditJSON(e spaces.AuditEntry) map[string]any {
	return map[string]any{"id": e.ID, "actor": e.Actor, "action": e.Action, "targetUserId": e.TargetUserID,
		"details": e.Details, "createdAt": httpx.TimeOrNil(e.CreatedAt)}
}
