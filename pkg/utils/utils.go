package utils

import (
	"strings"
)

// FormatClusterName converts the cluster name to lower case and replaces the underscore
func FormatClusterName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, "_", "-"))
}
