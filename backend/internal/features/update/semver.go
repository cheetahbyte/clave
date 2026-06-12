package update

import (
	"strconv"
	"strings"
)

func CompareVersions(a, b string) int {
	if a == b {
		return 0
	}

	partsA := splitVersion(a)
	partsB := splitVersion(b)

	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}

	for i := 0; i < maxLen; i++ {
		va := 0
		vb := 0
		var aIsNum, bIsNum bool

		if i < len(partsA) {
			va, aIsNum = parsePart(partsA[i])
		}
		if i < len(partsB) {
			vb, bIsNum = parsePart(partsB[i])
		}

		if aIsNum && bIsNum {
			if va < vb {
				return -1
			}
			if va > vb {
				return 1
			}
			continue
		}

		if !aIsNum && !bIsNum {
			cmp := strings.Compare(partsA[i], partsB[i])
			if cmp != 0 {
				return cmp
			}
			continue
		}

		if aIsNum && !bIsNum {
			return -1
		}
		return 1
	}

	return 0
}

func splitVersion(v string) []string {
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")

	parts := strings.Split(v, ".")
	for i, p := range parts {
		hyphenIdx := strings.IndexAny(p, "-+")
		if hyphenIdx >= 0 {
			parts[i] = p[:hyphenIdx]
			parts = append(parts, p[hyphenIdx+1:])
		}
	}
	return parts
}

func parsePart(s string) (int, bool) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}
