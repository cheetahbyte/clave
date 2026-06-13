package update

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cheetahbyte/clave/internal/db"
	"github.com/cheetahbyte/clave/internal/shared/storage"
	"github.com/google/uuid"
)

// secretMask is returned in place of stored secrets so they are never sent to
// the client. On save, a config value equal to the mask (or empty) means
// "keep the existing secret".
const secretMask = "********"

// secretConfigKeys are config fields treated as secrets: masked on read,
// preserved from the existing config when not re-supplied on write.
var secretConfigKeys = []string{"secret_access_key"}

// GetProductStorageConfig returns the storage backend config for a product,
// with secrets masked. Products without an explicit config report the default
// local backend.
func (svc *Service) GetProductStorageConfig(ctx context.Context, productID uuid.UUID) (*StorageConfigDTO, error) {
	cfg, err := svc.repo.GetProductStorageConfig(ctx, productID)
	if err != nil {
		return &StorageConfigDTO{
			ProductID: productID.String(),
			Backend:   string(storage.KindLocal),
			Config:    map[string]any{},
		}, nil
	}

	config := MustParseProviderConfig(cfg.Config)
	if config == nil {
		config = map[string]any{}
	}
	maskSecrets(config)

	return &StorageConfigDTO{
		ProductID: cfg.ProductID.String(),
		Backend:   cfg.Backend,
		Config:    config,
	}, nil
}

// SaveProductStorageConfig validates and persists the storage backend config
// for a product. Secrets left blank (or set to the mask) are carried over from
// the existing config.
func (svc *Service) SaveProductStorageConfig(ctx context.Context, orgID, productID uuid.UUID, req SaveStorageConfigRequest) (*StorageConfigDTO, error) {
	kind := storage.Kind(req.Backend)
	if !knownStorageKind(kind) {
		return nil, fmt.Errorf("unsupported storage backend: %s", req.Backend)
	}

	config := req.Config
	if config == nil {
		config = map[string]any{}
	}

	// Preserve secrets that were not re-supplied.
	if existing, err := svc.repo.GetProductStorageConfig(ctx, productID); err == nil {
		prev := MustParseProviderConfig(existing.Config)
		preserveSecrets(config, prev)
	}

	raw, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshal storage config: %w", err)
	}

	// Validate the config actually builds a working backend (local without an
	// explicit base_path is allowed; it falls back to the server default).
	if !(kind == storage.KindLocal && len(raw) <= 2) {
		if _, err := storage.New(kind, raw); err != nil {
			return nil, err
		}
	}

	saved, err := svc.repo.UpsertProductStorageConfig(ctx, db.UpsertProductStorageConfigParams{
		OrganizationID: orgID,
		ProductID:      productID,
		Backend:        req.Backend,
		Config:         raw,
	})
	if err != nil {
		return nil, fmt.Errorf("save storage config: %w", err)
	}

	out := MustParseProviderConfig(saved.Config)
	if out == nil {
		out = map[string]any{}
	}
	maskSecrets(out)
	return &StorageConfigDTO{
		ProductID: saved.ProductID.String(),
		Backend:   saved.Backend,
		Config:    out,
	}, nil
}

// TestProductStorageConfig builds a backend from the supplied config (merging
// in any preserved secrets) and runs its connectivity check without persisting
// anything.
func (svc *Service) TestProductStorageConfig(ctx context.Context, productID uuid.UUID, req SaveStorageConfigRequest) error {
	kind := storage.Kind(req.Backend)
	if !knownStorageKind(kind) {
		return fmt.Errorf("unsupported storage backend: %s", req.Backend)
	}

	config := req.Config
	if config == nil {
		config = map[string]any{}
	}
	if existing, err := svc.repo.GetProductStorageConfig(ctx, productID); err == nil {
		preserveSecrets(config, MustParseProviderConfig(existing.Config))
	}

	raw, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal storage config: %w", err)
	}

	// Local without an explicit base_path uses the server default backend.
	if kind == storage.KindLocal && len(raw) <= 2 {
		return storage.Verify(ctx, svc.defaultStorage)
	}

	backend, err := storage.New(kind, raw)
	if err != nil {
		return err
	}
	return storage.Verify(ctx, backend)
}

func knownStorageKind(kind storage.Kind) bool {
	for _, k := range storage.Kinds() {
		if k == kind {
			return true
		}
	}
	return false
}

func maskSecrets(config map[string]any) {
	for _, k := range secretConfigKeys {
		if v, ok := config[k]; ok {
			if s, _ := v.(string); s != "" {
				config[k] = secretMask
			}
		}
	}
}

func preserveSecrets(next, prev map[string]any) {
	if prev == nil {
		return
	}
	for _, k := range secretConfigKeys {
		v, ok := next[k]
		s, _ := v.(string)
		if !ok || s == "" || s == secretMask {
			if pv, ok := prev[k]; ok {
				next[k] = pv
			}
		}
	}
}
