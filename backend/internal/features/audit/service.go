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
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
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

	total, err := s.repo.CountByOrganization(ctx, orgID)
	if err != nil {
		return nil, err
	}

	rows, err := s.repo.ListByOrganization(ctx, orgID, int32(pageSize), int32(offset))
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
