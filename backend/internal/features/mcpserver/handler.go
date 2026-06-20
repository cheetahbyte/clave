package mcpserver

import (
	"log/slog"
	"net/http"

	"github.com/cheetahbyte/clave/internal/features/audit"
	"github.com/cheetahbyte/clave/internal/features/license"
	"github.com/cheetahbyte/clave/internal/shared/helpers"
	"github.com/cheetahbyte/clave/internal/shared/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	mcpauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Handler struct {
	svc        *Service
	licenseSvc *license.Service
	auditSvc   *audit.Service
	http       http.Handler
}

func NewHandler(svc *Service, licenseSvc *license.Service, auditSvc *audit.Service) *Handler {
	h := &Handler{svc: svc, licenseSvc: licenseSvc, auditSvc: auditSvc}
	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return newMCPServer(licenseSvc)
	}, nil)
	h.http = mcpauth.RequireBearerToken(h.tokenVerifier, nil)(mcpHandler)
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.http.ServeHTTP(w, r)
}

func (h *Handler) RegisterAdminRoutes(r chi.Router) {
	r.Get("/mcp-token", h.GetToken)
	r.Post("/mcp-token/regenerate", h.RegenerateToken)
}

func (h *Handler) GetToken(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	resp, err := h.svc.GetToken(r.Context(), orgID)
	if err != nil {
		slog.Error("get mcp token failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	helpers.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) RegenerateToken(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.AdminOrganizationIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	adminID, ok := middleware.AdminIDFromContext(r.Context())
	if !ok {
		helpers.WriteJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	tok, err := h.svc.RegenerateToken(r.Context(), orgID, adminID)
	if err != nil {
		slog.Error("regenerate mcp token failed", "err", err)
		helpers.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	h.audit(r, "mcp_token.regenerated", "mcp_token", adminID, orgID)
	helpers.WriteJSON(w, http.StatusCreated, RegenerateTokenResponse{
		Token:         tok.Token,
		Prefix:        tok.Prefix,
		RegeneratedAt: tok.RegeneratedAt,
	})
}

func (h *Handler) audit(r *http.Request, action, resourceType string, adminID, orgID uuid.UUID) {
	if h.auditSvc == nil {
		return
	}
	ip := helpers.ClientIP(r)
	var ipStr *string
	if ip != nil {
		s := ip.String()
		ipStr = &s
	}
	ua := r.UserAgent()
	h.auditSvc.Write(r.Context(), audit.AuditEntry{
		AdminID:      adminID,
		OrgID:        orgID,
		Action:       action,
		ResourceType: resourceType,
		IP:           ipStr,
		UserAgent:    &ua,
	})
}
