package jetstream

import "strings"

// Takes a prefix path and replaces invalid elements for jetstream with their valid identifiers
func sanitizePrefix(prefix string) string {
	return strings.ReplaceAll(strings.ReplaceAll(prefix, "/", "-"), ".", "_")
}
