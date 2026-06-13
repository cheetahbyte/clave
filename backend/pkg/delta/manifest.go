package delta

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

const manifestSchema = "clave.manifest/v1"

type ManifestEntry struct {
	Path   string `json:"path"`
	Type   string `json:"type"`
	SHA256 string `json:"sha256,omitempty"`
	Size   int64  `json:"size,omitempty"`
	Mode   uint32 `json:"mode,omitempty"`
	Target string `json:"target,omitempty"`
}

type Manifest struct {
	Schema string          `json:"schema"`
	Files  []ManifestEntry `json:"files"`
}

func BuildManifest(root string) (*Manifest, string, error) {
	m := &Manifest{Schema: manifestSchema}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		entry := ManifestEntry{
			Path: filepath.ToSlash(rel),
			Mode: uint32(info.Mode().Perm()),
		}

		if d.IsDir() {
			entry.Type = "dir"
		} else if info.Mode()&os.ModeSymlink != 0 {
			entry.Type = "symlink"
			target, rerr := os.Readlink(path)
			if rerr != nil {
				return rerr
			}
			entry.Target = target
		} else {
			entry.Type = "file"
			entry.Size = info.Size()
			hash, herr := fileSHA256(path)
			if herr != nil {
				return herr
			}
			entry.SHA256 = fmt.Sprintf("%x", hash)
		}

		m.Files = append(m.Files, entry)
		return nil
	})
	if err != nil {
		return nil, "", err
	}

	sort.Slice(m.Files, func(i, j int) bool {
		return m.Files[i].Path < m.Files[j].Path
	})

	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, "", err
	}
	hash := sha256.Sum256(raw)
	return m, fmt.Sprintf("%x", hash), nil
}

func fileSHA256(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}
