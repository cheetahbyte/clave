package update

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/google/uuid"
)

func changelogToDTO(c db.Changelog) ChangelogDTO {
	dto := ChangelogDTO{
		ID:        c.ID.String(),
		ProductID: c.ProductID.String(),
		Title:     c.Title,
		Body:      c.Body,
	}
	if c.CreatedAt.Valid {
		s := c.CreatedAt.Time.Format(time.RFC3339)
		dto.CreatedAt = &s
	}
	if c.UpdatedAt.Valid {
		s := c.UpdatedAt.Time.Format(time.RFC3339)
		dto.UpdatedAt = &s
	}
	return dto
}

func (svc *Service) ListChangelogs(ctx context.Context, productID uuid.UUID) ([]ChangelogDTO, error) {
	rows, err := svc.repo.ListChangelogsForProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	out := make([]ChangelogDTO, len(rows))
	for i, c := range rows {
		out[i] = changelogToDTO(c)
	}
	return out, nil
}

func (svc *Service) CreateChangelog(ctx context.Context, orgID, productID uuid.UUID, req SaveChangelogRequest) (*ChangelogDTO, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	c, err := svc.repo.CreateChangelog(ctx, db.CreateChangelogParams{
		OrganizationID: orgID,
		ProductID:      productID,
		Title:          title,
		Body:           req.Body,
	})
	if err != nil {
		return nil, err
	}
	svc.feedCache.invalidateProduct(c.ProductID)
	dto := changelogToDTO(c)
	return &dto, nil
}

func (svc *Service) UpdateChangelog(ctx context.Context, orgID, changelogID uuid.UUID, req SaveChangelogRequest) (*ChangelogDTO, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, fmt.Errorf("title is required")
	}
	c, err := svc.repo.UpdateChangelog(ctx, db.UpdateChangelogParams{
		ID:             changelogID,
		OrganizationID: orgID,
		Title:          title,
		Body:           req.Body,
	})
	if err != nil {
		return nil, err
	}
	svc.feedCache.invalidateProduct(c.ProductID)
	dto := changelogToDTO(c)
	return &dto, nil
}

func (svc *Service) DeleteChangelog(ctx context.Context, orgID, changelogID uuid.UUID) error {
	count, err := svc.repo.CountReleasesForChangelog(ctx, changelogID)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("changelog in use: attached to %d release(s); detach them first", count)
	}
	_, err = svc.repo.DeleteChangelog(ctx, changelogID, orgID)
	return err
}

// AttachReleaseChangelog sets or clears the changelog attached to a release.
func (svc *Service) AttachReleaseChangelog(ctx context.Context, releaseID uuid.UUID, changelogID *uuid.UUID) error {
	release, err := svc.repo.SetReleaseChangelog(ctx, releaseID, changelogID)
	if err == nil {
		svc.feedCache.invalidateProduct(release.ProductID)
	}
	return err
}

// releaseChangelog resolves the attached changelog's body and public URL for a
// release. Returns empty strings when no changelog is attached.
func (svc *Service) releaseChangelog(ctx context.Context, release db.UpdateRelease) (body, url string) {
	if !release.ChangelogID.Valid {
		return "", ""
	}
	cl, err := svc.repo.GetChangelog(ctx, uuid.UUID(release.ChangelogID.Bytes))
	if err != nil {
		return "", ""
	}
	return cl.Body, svc.ChangelogURL(release.ID)
}
