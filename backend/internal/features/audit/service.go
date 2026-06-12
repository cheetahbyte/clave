package audit

import (
	"context"
	"math"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	q *db.Queries
}

func NewService(q *db.Queries) *Service {
	return &Service{q: q}
}

func (s *Service) List(ctx context.Context, orgID uuid.UUID, page, pageSize int) (*AuditLogListResponse, error) {
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

	total, err := s.q.CountAuditLogsByOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}

	rows, err := s.q.ListAuditLogsByOrganization(ctx, db.ListAuditLogsByOrganizationParams{
		OrganizationID: orgID,
		Limit:          int32(pageSize),
		Offset:         int32(offset),
	})
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
			IP:           ipPtr(r),
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

func uuidPtr(u pgtype.UUID) *string {
	if !u.Valid {
		return nil
	}
	s := uuid.UUID(u.Bytes).String()
	return &s
}

func ipPtr(r db.ListAuditLogsByOrganizationRow) *string {
	if r.Ip == nil {
		return nil
	}
	s := r.Ip.String()
	return &s
}

func timePtr(t pgtype.Timestamptz) *time.Time {
	if t.Valid {
		return &t.Time
	}
	return nil
}
