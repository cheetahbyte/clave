package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// LocalConfig configures the local filesystem backend.
type LocalConfig struct {
	BasePath string `json:"base_path"`
}

type localBackend struct {
	base string
}

// NewLocal returns a filesystem-backed Backend rooted at basePath.
func NewLocal(basePath string) Backend {
	return &localBackend{base: basePath}
}

func newLocalFromConfig(raw json.RawMessage) (Backend, error) {
	cfg := LocalConfig{}
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return nil, fmt.Errorf("parse local storage config: %w", err)
		}
	}
	if cfg.BasePath == "" {
		return nil, fmt.Errorf("local storage: base_path is required")
	}
	return NewLocal(cfg.BasePath), nil
}

func (l *localBackend) path(key string) string {
	return filepath.Join(l.base, filepath.FromSlash(key))
}

func (l *localBackend) Put(ctx context.Context, key string, r io.Reader) (int64, error) {
	p := l.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return 0, fmt.Errorf("create storage dir: %w", err)
	}
	f, err := os.Create(p)
	if err != nil {
		return 0, fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		os.Remove(p)
		return 0, fmt.Errorf("write file: %w", err)
	}
	return n, nil
}

func (l *localBackend) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	return os.Open(l.path(key))
}

func (l *localBackend) Delete(ctx context.Context, key string) error {
	if err := os.Remove(l.path(key)); err != nil && !os.IsNotExist(err) {
		return err
	}
	// Best-effort cleanup of the now-empty per-artifact directory.
	_ = os.Remove(filepath.Dir(l.path(key)))
	return nil
}
