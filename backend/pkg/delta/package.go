package delta

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func BuildDelta(dm *DeltaManifest, toManifest *Manifest, rootDir string) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	deltaJSON, err := json.MarshalIndent(dm, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal delta.json: %w", err)
	}
	if err := writeZipFile(w, "delta.json", deltaJSON); err != nil {
		return nil, err
	}

	targetJSON, err := json.MarshalIndent(toManifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal target-manifest.json: %w", err)
	}
	if err := writeZipFile(w, "target-manifest.json", targetJSON); err != nil {
		return nil, err
	}

	for _, op := range dm.Operations {
		switch op.Op {
		case OpPatch, OpAdd:
			if op.Type != "file" {
				continue
			}
			src := filepath.Join(rootDir, filepath.FromSlash(op.Path))
			dst := "added/" + strings.ReplaceAll(op.Path, "/", "_")
			if err := addFileToZip(w, dst, src); err != nil {
				return nil, fmt.Errorf("add %s: %w", op.Path, err)
			}
		}
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}
	return buf.Bytes(), nil
}

func writeZipFile(w *zip.Writer, name string, data []byte) error {
	f, err := w.Create(name)
	if err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write %s: %w", name, err)
	}
	return nil
}

func addFileToZip(w *zip.Writer, zipPath, diskPath string) error {
	src, err := os.Open(diskPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := w.Create(zipPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}
