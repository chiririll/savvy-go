package space

import (
	"net/http"

	"savvy-go/internal/httpserver/httpx"
	"savvy-go/internal/httpserver/httpx/dto"
)

func (h Handler) spaceAudit(w http.ResponseWriter, r *http.Request) {
	list, err := h.Spaces.AuditLog(r.Context(), httpx.Scope(r).ID)
	if err != nil {
		httpx.WriteMessage(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteData(w, http.StatusOK, dto.Map(list, dto.NewAuditEntry))
}
