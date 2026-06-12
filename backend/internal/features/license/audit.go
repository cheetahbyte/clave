package license

import (
	"context"
	"log/slog"

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
	if err := svc.repo.InsertAuditLog(ctx, e.AdminID, e.OrgID, e.Action, e.ResourceType, e.ResourceID, ua); err != nil {
		slog.Error("failed to write audit log", "action", e.Action, "err", err)
	}
}
