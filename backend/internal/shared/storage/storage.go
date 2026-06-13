// Package storage provides a generic blob storage abstraction for update
// artifacts. Backends (local filesystem, S3, ...) implement the Backend
// interface and are constructed from a JSON config via New. To add a new
// backend, implement Backend and register a factory with Register.
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Backend is generic blob storage addressed by opaque string keys.
type Backend interface {
	// Put stores everything read from r under key and returns bytes written.
	Put(ctx context.Context, key string, r io.Reader) (int64, error)
	// Open returns a reader for the object stored under key. Caller closes it.
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the object under key. A missing object is not an error.
	Delete(ctx context.Context, key string) error
}

// Verifier is an optional interface for backends that can check connectivity
// and credentials without storing an object.
type Verifier interface {
	Verify(ctx context.Context) error
}

// Verify runs a backend's connectivity check if it implements Verifier.
// Backends without a check are treated as reachable.
func Verify(ctx context.Context, b Backend) error {
	if v, ok := b.(Verifier); ok {
		return v.Verify(ctx)
	}
	return nil
}

// Presigner is an optional interface for backends that can hand out a
// time-limited URL the client fetches directly, offloading transfer from the
// app server. Backends that cannot presign (e.g. local disk) omit it.
type Presigner interface {
	// PresignGet returns a URL valid for ttl that grants read access to key.
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
}

// PresignGet returns (url, true, nil) if b can presign key, or ("", false, nil)
// if the backend does not support presigning (caller should stream instead).
func PresignGet(ctx context.Context, b Backend, key string, ttl time.Duration) (string, bool, error) {
	p, ok := b.(Presigner)
	if !ok {
		return "", false, nil
	}
	url, err := p.PresignGet(ctx, key, ttl)
	if err != nil {
		return "", false, err
	}
	return url, true, nil
}

// Kind identifies a storage backend implementation.
type Kind string

const (
	KindLocal Kind = "local"
	KindS3    Kind = "s3"
)

// Factory builds a Backend from its raw JSON config.
type Factory func(raw json.RawMessage) (Backend, error)

var factories = map[Kind]Factory{
	KindLocal: newLocalFromConfig,
	KindS3:    newS3FromConfig,
}

// Register adds or overrides a backend factory. Intended for future
// extension with custom backends.
func Register(kind Kind, f Factory) { factories[kind] = f }

// New builds a Backend of the given kind from raw JSON config.
func New(kind Kind, raw json.RawMessage) (Backend, error) {
	f, ok := factories[kind]
	if !ok {
		return nil, fmt.Errorf("unknown storage backend: %q", kind)
	}
	return f(raw)
}

// Kinds returns the registered backend kinds.
func Kinds() []Kind {
	out := make([]Kind, 0, len(factories))
	for k := range factories {
		out = append(out, k)
	}
	return out
}
