package audit

import (
	"context"
	"log/slog"
	"math"
	"net/netip"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/observability"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type AuditEntry struct {
	AdminID      uuid.UUID
	OrgID        uuid.UUID
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	IP           *string
	UserAgent    *string
	Metadata     map[string]string
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type AuditFilter struct {
	Q            *string
	Action       *string
	ResourceType *string
	AdminEmail   *string
	From         *time.Time
	To           *time.Time
}

func (s *Service) List(ctx context.Context, orgID uuid.UUID, page, pageSize int, f *AuditFilter) (*AuditLogListResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}
	maxPage := math.MaxInt32 / pageSize
	if page > maxPage {
		page = maxPage
	}
	offset := (page - 1) * pageSize

	fromTs := pgtype.Timestamptz{}
	if f != nil && f.From != nil {
		fromTs = pgtype.Timestamptz{Time: *f.From, Valid: true}
	}
	toTs := pgtype.Timestamptz{}
	if f != nil && f.To != nil {
		toTs = pgtype.Timestamptz{Time: *f.To, Valid: true}
	}

	countParams := db.CountAuditLogsByOrganizationParams{OrganizationID: orgID}
	listParams := db.ListAuditLogsByOrganizationParams{
		OrganizationID: orgID,
		Limit:          int32(pageSize),
		Offset:         int32(offset),
	}
	if f != nil {
		countParams.Q = f.Q
		countParams.Action = f.Action
		countParams.ResourceType = f.ResourceType
		countParams.AdminEmail = f.AdminEmail
		countParams.FromTs = fromTs
		countParams.ToTs = toTs

		listParams.Q = f.Q
		listParams.Action = f.Action
		listParams.ResourceType = f.ResourceType
		listParams.AdminEmail = f.AdminEmail
		listParams.FromTs = fromTs
		listParams.ToTs = toTs
	}

	total, err := s.repo.CountByOrganization(ctx, countParams)
	if err != nil {
		return nil, err
	}

	rows, err := s.repo.ListByOrganization(ctx, listParams)
	if err != nil {
		return nil, err
	}

	items := make([]AuditLogItem, len(rows))
	for i, r := range rows {
		items[i] = AuditLogItem{
			ID:           r.ID.String(),
			Action:       r.Action,
			ResourceType: r.ResourceType,
			ResourceID:   uuidPtr(r.ResourceID),
			AdminEmail:   r.AdminEmail,
			IP:           ipPtr(r.Ip),
			UserAgent:    r.UserAgent,
			CreatedAt:    timePtr(r.CreatedAt),
		}
	}

	return &AuditLogListResponse{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *Service) Write(ctx context.Context, e AuditEntry) {
	var rid pgtype.UUID
	if e.ResourceID != nil {
		rid = pgtype.UUID{Bytes: *e.ResourceID, Valid: true}
	}

	var ip *netip.Addr
	if e.IP != nil {
		parsed, err := netip.ParseAddr(*e.IP)
		if err == nil {
			ip = &parsed
		}
	}

	auditCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	observability.CountAuditEvent(auditCtx, e.Action)
	if err := s.repo.InsertAuditLog(auditCtx, db.InsertAuditLogParams{
		AdminUserID:    pgtype.UUID{Bytes: e.AdminID, Valid: true},
		OrganizationID: pgtype.UUID{Bytes: e.OrgID, Valid: true},
		Action:         e.Action,
		ResourceType:   e.ResourceType,
		ResourceID:     rid,
		Ip:             ip,
		UserAgent:      e.UserAgent,
	}); err != nil {
		slog.Error("failed to write audit log", "action", e.Action, "err", err)
	}
}

func uuidPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuid.UUID(u.Bytes).String()
	return &s
}

func ipPtr(ip *netip.Addr) *string {
	if ip == nil {
		return nil
	}
	s := ip.String()
	return &s
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}
