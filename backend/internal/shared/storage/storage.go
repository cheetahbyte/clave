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
