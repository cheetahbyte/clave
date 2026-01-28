package licensecrypto

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
)

func formatKey(prefix, raw string, groupSize int) string {
	raw = strings.ToUpper(raw)

	var parts []string
	for i := 0; i < len(raw); i += groupSize {
		end := i + groupSize
		if end > len(raw) {
			end = len(raw)
		}
		parts = append(parts, raw[i:end])
	}

	return prefix + "-" + strings.Join(parts, "-")
}

func GenerateLicenseKey() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	enc := base32.StdEncoding.
		WithPadding(base32.NoPadding)

	raw := enc.EncodeToString(b)

	return formatKey("LIC", raw, 4), nil
}
