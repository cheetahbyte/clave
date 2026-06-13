package delta

type OpKind string

const (
	OpPatch  OpKind = "patch"
	OpAdd    OpKind = "add"
	OpDelete OpKind = "delete"
	OpMkdir  OpKind = "mkdir"
)

type Operation struct {
	Type      string `json:"type"`
	Op        OpKind `json:"op"`
	Path      string `json:"path"`
	OldSHA256 string `json:"oldSha256,omitempty"`
	NewSHA256 string `json:"newSha256,omitempty"`
	SHA256    string `json:"sha256,omitempty"`
	Size      int64  `json:"size,omitempty"`
	Mode      uint32 `json:"mode,omitempty"`
	Target    string `json:"target,omitempty"`
}

type ReleaseMeta struct {
	Version        string `json:"version"`
	Build          string `json:"build"`
	ManifestSHA256 string `json:"manifestSha256"`
	ArtifactSHA256 string `json:"artifactSha256"`
}

type DeltaManifest struct {
	Schema     string      `json:"schema"`
	From       ReleaseMeta `json:"from"`
	To         ReleaseMeta `json:"to"`
	Operations []Operation `json:"operations"`
}

const deltaSchema = "clave.delta/v1"

func Diff(from, to *Manifest, fromMeta, toMeta ReleaseMeta) *DeltaManifest {
	dm := &DeltaManifest{
		Schema: deltaSchema,
		From:   fromMeta,
		To:     toMeta,
	}

	fromMap := make(map[string]ManifestEntry)
	for _, f := range from.Files {
		fromMap[f.Path] = f
	}
	toMap := make(map[string]ManifestEntry)
	for _, t := range to.Files {
		toMap[t.Path] = t
	}

	for _, toEntry := range to.Files {
		fromEntry, existed := fromMap[toEntry.Path]
		if !existed {
			switch toEntry.Type {
			case "file":
				dm.Operations = append(dm.Operations, Operation{
					Type:   "file",
					Op:     OpAdd,
					Path:   toEntry.Path,
					SHA256: toEntry.SHA256,
					Size:   toEntry.Size,
					Mode:   toEntry.Mode,
				})
			case "dir":
				dm.Operations = append(dm.Operations, Operation{
					Type: "dir",
					Op:   OpMkdir,
					Path: toEntry.Path,
					Mode: toEntry.Mode,
				})
			case "symlink":
				dm.Operations = append(dm.Operations, Operation{
					Type:   "symlink",
					Op:     OpAdd,
					Path:   toEntry.Path,
					Target: toEntry.Target,
				})
			}
			continue
		}

		if fromEntry.Type != toEntry.Type {
			dm.Operations = append(dm.Operations, Operation{
				Type: toEntry.Type,
				Op:   OpAdd,
				Path: toEntry.Path,
			})
			n := len(dm.Operations) - 1
			if toEntry.Type == "file" {
				dm.Operations[n].SHA256 = toEntry.SHA256
				dm.Operations[n].Size = toEntry.Size
				dm.Operations[n].Mode = toEntry.Mode
			}
			if toEntry.Type == "symlink" {
				dm.Operations[n].Target = toEntry.Target
			}
			continue
		}

		switch toEntry.Type {
		case "file":
			if fromEntry.SHA256 != toEntry.SHA256 {
				dm.Operations = append(dm.Operations, Operation{
					Type:      "file",
					Op:        OpPatch,
					Path:      toEntry.Path,
					OldSHA256: fromEntry.SHA256,
					NewSHA256: toEntry.SHA256,
					Size:      toEntry.Size,
					Mode:      toEntry.Mode,
				})
			}
		case "dir":
		case "symlink":
			if fromEntry.Target != toEntry.Target {
				dm.Operations = append(dm.Operations, Operation{
					Type:   "symlink",
					Op:     OpAdd,
					Path:   toEntry.Path,
					Target: toEntry.Target,
				})
			}
		}
	}

	for _, fromEntry := range from.Files {
		if _, exists := toMap[fromEntry.Path]; exists {
			continue
		}
		switch fromEntry.Type {
		case "file":
			dm.Operations = append(dm.Operations, Operation{
				Type: "file",
				Op:   OpDelete,
				Path: fromEntry.Path,
			})
		case "dir":
		case "symlink":
			dm.Operations = append(dm.Operations, Operation{
				Type: "symlink",
				Op:   OpDelete,
				Path: fromEntry.Path,
			})
		}
	}

	return dm
}
