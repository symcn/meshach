package serviceconfig

import (
	"strings"
)

// To match label charactor limit
func truncated(s string) string {
	if len(s) > 62 {
		return strings.Trim(s[len(s)-62:], ".")
	}
	return strings.Trim(s, ".")
}

func mapContains(std, obj, renameMap map[string]string) bool {
	for sk, sv := range std {
		rk, ok := renameMap[sk]
		if !ok {
			rk = sk
		}

		if ov, ok := obj[rk]; !ok || ov != sv {
			return false
		}
	}
	return true
}
