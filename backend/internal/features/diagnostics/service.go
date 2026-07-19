package diagnostics

import (
	"context"
	"math"
	"sort"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"
)

type adoptionRepository interface {
	ListLatestCheckins(context.Context, uuid.UUID, pgtype.UUID, int) ([]LatestCheckin, error)
	ListDailyVersions(context.Context, uuid.UUID, pgtype.UUID, int) ([]DailyVersion, error)
}

type Service struct {
	repo adoptionRepository
}

func NewService(repo adoptionRepository) *Service {
	return &Service{repo: repo}
}

func (svc *Service) VersionAdoption(ctx context.Context, orgID uuid.UUID, productID pgtype.UUID, days int) (VersionAdoptionResponse, error) {
	var latest []LatestCheckin
	var daily []DailyVersion
	group, groupCtx := errgroup.WithContext(ctx)
	group.Go(func() error {
		var err error
		latest, err = svc.repo.ListLatestCheckins(groupCtx, orgID, productID, days)
		return err
	})
	group.Go(func() error {
		var err error
		daily, err = svc.repo.ListDailyVersions(groupCtx, orgID, productID, days)
		return err
	})
	if err := group.Wait(); err != nil {
		return VersionAdoptionResponse{}, err
	}

	counts := make(map[string]int64)
	devices := make([]VersionDevice, 0, len(latest))
	for _, row := range latest {
		counts[row.Version]++
		devices = append(devices, VersionDevice{
			ActivationID: row.ActivationID.String(), Hostname: row.Hostname,
			Version: row.Version, Build: row.Build, Platform: row.Platform,
			Arch: row.Arch, OSVersion: row.OSVersion, LastCheckin: row.CreatedAt,
		})
	}

	distribution := make([]VersionDistribution, 0, len(counts))
	activeDevices := int64(len(latest))
	for version, count := range counts {
		percentage := 0.0
		if activeDevices > 0 {
			percentage = math.Round((float64(count)/float64(activeDevices))*1000) / 10
		}
		distribution = append(distribution, VersionDistribution{
			Version: version, DeviceCount: count, Percentage: percentage,
		})
	}
	sort.Slice(distribution, func(i, j int) bool {
		if distribution[i].DeviceCount == distribution[j].DeviceCount {
			return distribution[i].Version < distribution[j].Version
		}
		return distribution[i].DeviceCount > distribution[j].DeviceCount
	})

	trend := make([]VersionTrendPoint, 0)
	for _, row := range daily {
		date := row.Date.Format("2006-01-02")
		if len(trend) == 0 || trend[len(trend)-1].Date != date {
			trend = append(trend, VersionTrendPoint{Date: date, Versions: []VersionTrendValue{}})
		}
		last := &trend[len(trend)-1]
		last.Versions = append(last.Versions, VersionTrendValue{Version: row.Version, DeviceCount: row.DeviceCount})
	}

	return VersionAdoptionResponse{
		ActiveDevices: activeDevices,
		VersionCount:  len(distribution),
		Distribution:  distribution,
		Trend:         trend,
		Devices:       devices,
	}, nil
}
