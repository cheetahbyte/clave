package license

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type AuditEntry struct {
	AdminID      uuid.UUID
	OrgID        uuid.UUID
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	UserAgent    string
}

func (svc *Service) WriteAudit(ctx context.Context, e AuditEntry) {
	var ua *string
	if e.UserAgent != "" {
		ua = &e.UserAgent
	}
	auditCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := svc.repo.InsertAuditLog(auditCtx, e.AdminID, e.OrgID, e.Action, e.ResourceType, e.ResourceID, ua); err != nil {
		slog.Error("failed to write audit log", "action", e.Action, "err", err)
	}
}
