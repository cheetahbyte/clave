package update

import (
	"context"
	"fmt"
	"strings"

	problem "github.com/cheetahbyte/problems"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/shared/clientchannels"
	"github.com/google/uuid"
)

func channelToDTO(ch db.UpdateChannel) ChannelDTO {
	features := ch.RequiredFeatures
	if features == nil {
		features = []string{}
	}
	var desc string
	if ch.Description != nil {
		desc = *ch.Description
	}
	return ChannelDTO{
		ID:               ch.ID.String(),
		ProductID:        ch.ProductID.String(),
		Name:             ch.Name,
		IsDefault:        ch.IsDefault,
		RequiredFeatures: features,
		Description:      desc,
	}
}

func (svc *Service) ListChannels(ctx context.Context, productID uuid.UUID) ([]ChannelDTO, error) {
	rows, err := svc.channelsForProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	out := make([]ChannelDTO, len(rows))
	for i, ch := range rows {
		out[i] = channelToDTO(ch)
	}
	return out, nil
}

func (svc *Service) AvailableChannels(ctx context.Context, productID uuid.UUID, features []string) ([]clientchannels.Channel, error) {
	rows, err := svc.channelsForProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	out := make([]clientchannels.Channel, 0, len(rows))
	for _, ch := range rows {
		if !hasAllFeatures(features, ch.RequiredFeatures) {
			continue
		}
		dto := clientchannels.Channel{
			Name:      ch.Name,
			IsDefault: ch.IsDefault,
		}
		if ch.Description != nil {
			dto.Description = *ch.Description
		}
		out = append(out, dto)
	}
	return out, nil
}

func (svc *Service) channelsForProduct(ctx context.Context, productID uuid.UUID) ([]db.UpdateChannel, error) {
	if channels, ok := svc.channelCache.get(productID); ok {
		return channels, nil
	}
	channels, err := svc.repo.GetChannelsForProduct(ctx, productID)
	if err != nil {
		return nil, err
	}
	svc.channelCache.set(productID, channels)
	return channels, nil
}

func (svc *Service) ClientChannels(ctx context.Context, data ChannelsRequest) (ChannelsResponse, error) {
	instance := "/updates/channels"

	_, lic, _, err := svc.parseActiveLicenseToken(ctx, data.Token, instance)
	if err != nil {
		return ChannelsResponse{}, err
	}

	channels, err := svc.AvailableChannels(ctx, lic.ProductID, lic.Features)
	if err != nil {
		return ChannelsResponse{}, problem.Of(500).
			Append(problem.Title("Failed to list update channels")).
			Append(problem.Instance(instance))
	}

	return ChannelsResponse{UpdateChannels: channels}, nil
}

func normalizeFeatures(in []string) []string {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, f := range in {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}

func (svc *Service) CreateChannel(ctx context.Context, orgID, productID uuid.UUID, req SaveChannelRequest) (*ChannelDTO, error) {
	name := strings.ToLower(strings.TrimSpace(req.Name))
	if name == "" {
		return nil, fmt.Errorf("channel name is required")
	}
	var desc *string
	if strings.TrimSpace(req.Description) != "" {
		d := req.Description
		desc = &d
	}
	normalizedFeatures := normalizeFeatures(req.RequiredFeatures)
	ch, err := svc.repo.CreateUpdateChannel(ctx, db.CreateUpdateChannelParams{
		OrganizationID:   orgID,
		ProductID:        productID,
		Name:             name,
		IsDefault:        req.IsDefault,
		RequiredFeatures: normalizedFeatures,
		Description:      desc,
	})
	if err != nil {
		return nil, err
	}

	// Sync update_channel_required_features join table.
	svc.syncChannelRequiredFeatures(ctx, ch.ID, orgID, productID, normalizedFeatures)
	svc.feedCache.invalidateProduct(productID)
	svc.channelCache.invalidate(productID)

	dto := channelToDTO(ch)
	return &dto, nil
}

func (svc *Service) UpdateChannel(ctx context.Context, orgID, channelID uuid.UUID, req SaveChannelRequest) (*ChannelDTO, error) {
	name := strings.ToLower(strings.TrimSpace(req.Name))
	if name == "" {
		return nil, fmt.Errorf("channel name is required")
	}
	var desc *string
	if strings.TrimSpace(req.Description) != "" {
		d := req.Description
		desc = &d
	}
	normalizedFeatures := normalizeFeatures(req.RequiredFeatures)
	ch, err := svc.repo.UpdateUpdateChannel(ctx, db.UpdateUpdateChannelParams{
		ID:               channelID,
		OrganizationID:   orgID,
		Name:             name,
		IsDefault:        req.IsDefault,
		RequiredFeatures: normalizedFeatures,
		Description:      desc,
	})
	if err != nil {
		return nil, err
	}

	// Sync update_channel_required_features join table.
	svc.syncChannelRequiredFeatures(ctx, ch.ID, orgID, ch.ProductID, normalizedFeatures)
	svc.feedCache.invalidateProduct(ch.ProductID)
	svc.channelCache.invalidate(ch.ProductID)

	dto := channelToDTO(ch)
	return &dto, nil
}

// syncChannelRequiredFeatures keeps the update_channel_required_features join
// table in sync with the TEXT[] required_features column on update_channels.
func (svc *Service) syncChannelRequiredFeatures(ctx context.Context, channelID, orgID, productID uuid.UUID, featureKeys []string) {
	_ = svc.repo.q.SetChannelRequiredFeatures(ctx, channelID)
	for _, key := range featureKeys {
		pf, err := svc.repo.q.GetProductFeatureByKey(ctx, db.GetProductFeatureByKeyParams{
			OrganizationID: orgID,
			ProductID:      productID,
			Key:            key,
		})
		if err != nil {
			// Feature not in catalog — create it automatically.
			pf, err = svc.repo.q.CreateProductFeature(ctx, db.CreateProductFeatureParams{
				OrganizationID: orgID,
				ProductID:      productID,
				Key:            key,
			})
			if err != nil {
				continue
			}
		}
		_ = svc.repo.q.AddChannelRequiredFeature(ctx, db.AddChannelRequiredFeatureParams{
			ChannelID: channelID,
			FeatureID: pf.ID,
		})
	}
}

func (svc *Service) DeleteChannel(ctx context.Context, orgID, channelID uuid.UUID) error {
	releases, err := svc.repo.CountReleasesForChannel(ctx, channelID)
	if err != nil {
		return err
	}
	configs, err := svc.repo.CountConfigsForChannel(ctx, channelID)
	if err != nil {
		return err
	}
	if releases > 0 || configs > 0 {
		return fmt.Errorf("channel in use: %d release(s) and %d source(s) reference it; remove them first", releases, configs)
	}

	ch, err := svc.repo.DeleteUpdateChannel(ctx, channelID, orgID)
	if err == nil {
		svc.feedCache.invalidateProduct(ch.ProductID)
		svc.channelCache.invalidate(ch.ProductID)
	}
	return err
}
