package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cheetahbyte/clave/pkg/delta"
)

type DeltaGenerateEvent struct {
	Type  string `json:"type"`
	JobID string `json:"jobId"`
}

type Outcome struct {
	Requeue bool
}

type Processor struct {
	client      *APIClient
	engine      delta.Engine
	maxBytes    int64
	makeTempDir func() (string, error)
}

func NewProcessor(client *APIClient, maxBytes int64) *Processor {
	return &Processor{
		client: client, engine: delta.BSDiffEngine{}, maxBytes: maxBytes,
		makeTempDir: func() (string, error) { return os.MkdirTemp("", "clave-delta-job-*") },
	}
}

func (p *Processor) Process(ctx context.Context, event DeltaGenerateEvent) Outcome {
	details, err := p.client.Claim(ctx, event.JobID)
	if err != nil {
		if isTransient(err) {
			_ = p.client.Terminal(ctx, event.JobID, "requeue", "", "", -1)
			return Outcome{Requeue: true}
		}
		return Outcome{}
	}
	if details.SourceSize > p.maxBytes || details.TargetSize > p.maxBytes {
		_ = p.client.Terminal(ctx, event.JobID, "skip", "artifact_too_large", "", -1)
		return Outcome{}
	}
	dir, err := p.makeTempDir()
	if err != nil {
		return p.handleError(ctx, event.JobID, err)
	}
	defer os.RemoveAll(dir)
	oldPath := filepath.Join(dir, "source")
	newPath := filepath.Join(dir, "target")
	patchPath := filepath.Join(dir, "patch")
	outputPath := filepath.Join(dir, "output")
	sourceSize, sourceSHA, err := p.client.Download(ctx, event.JobID, "source", oldPath, p.maxBytes)
	if err != nil {
		return p.handleError(ctx, event.JobID, err)
	}
	targetSize, targetSHA, err := p.client.Download(ctx, event.JobID, "target", newPath, p.maxBytes)
	if err != nil {
		return p.handleError(ctx, event.JobID, err)
	}
	if sourceSize != details.SourceSize || sourceSHA != details.SourceSHA256 || targetSize != details.TargetSize || targetSHA != details.TargetSHA256 {
		return p.handleError(ctx, event.JobID, errors.New("artifact size or SHA-256 mismatch"))
	}
	if err := createPatch(p.engine, oldPath, newPath, patchPath); err != nil {
		return p.handleError(ctx, event.JobID, fmt.Errorf("create patch: %w", err))
	}
	if err := applyPatch(p.engine, oldPath, patchPath, outputPath); err != nil {
		return p.handleError(ctx, event.JobID, fmt.Errorf("apply patch: %w", err))
	}
	outputSize, outputSHA, err := hashFile(outputPath)
	if err != nil || outputSize != targetSize || outputSHA != targetSHA {
		return p.handleError(ctx, event.JobID, errors.New("round-trip target verification failed"))
	}
	patchSize, patchSHA, err := hashFile(patchPath)
	if err != nil {
		return p.handleError(ctx, event.JobID, err)
	}
	if patchSize*100 >= targetSize*70 {
		if err := p.client.Terminal(ctx, event.JobID, "skip", "patch_not_worthwhile", patchSHA, patchSize); err != nil {
			return p.handleError(ctx, event.JobID, err)
		}
		return Outcome{}
	}
	if err := p.client.Complete(ctx, event.JobID, patchPath, patchSHA, patchSize); err != nil {
		return p.handleError(ctx, event.JobID, err)
	}
	return Outcome{}
}

func (p *Processor) handleError(ctx context.Context, jobID string, err error) Outcome {
	if isTransient(err) {
		if requeueErr := p.client.Terminal(ctx, jobID, "requeue", "", "", -1); requeueErr == nil {
			return Outcome{Requeue: true}
		}
	}
	_ = p.client.Terminal(ctx, jobID, "fail", err.Error(), "", -1)
	return Outcome{}
}

func isTransient(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Transient()
}

func createPatch(engine delta.Engine, oldPath, newPath, patchPath string) error {
	oldFile, err := os.Open(oldPath)
	if err != nil {
		return err
	}
	defer oldFile.Close()
	newFile, err := os.Open(newPath)
	if err != nil {
		return err
	}
	defer newFile.Close()
	patchFile, err := os.OpenFile(patchPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer patchFile.Close()
	return engine.Create(oldFile, newFile, patchFile)
}

func applyPatch(engine delta.Engine, oldPath, patchPath, outputPath string) error {
	oldFile, err := os.Open(oldPath)
	if err != nil {
		return err
	}
	defer oldFile.Close()
	patchFile, err := os.Open(patchPath)
	if err != nil {
		return err
	}
	defer patchFile.Close()
	outputFile, err := os.OpenFile(outputPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	return engine.Apply(oldFile, patchFile, outputFile)
}

func hashFile(path string) (int64, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, "", err
	}
	defer file.Close()
	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return 0, "", err
	}
	return size, fmt.Sprintf("%x", hasher.Sum(nil)), nil
}
