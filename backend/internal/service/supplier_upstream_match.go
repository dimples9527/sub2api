package service

import (
	"strings"
	"unicode"
)

func normalizeSupplierUpstreamMatchName(name string, removeSeparators bool) string {
	normalized := strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	b.Grow(len(normalized))
	for _, r := range normalized {
		if isIgnorableSupplierUpstreamMatchRune(r) || unicode.IsSpace(r) {
			continue
		}
		if removeSeparators && (r == '_' || r == '-') {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func isIgnorableSupplierUpstreamMatchRune(r rune) bool {
	return (r >= '\uFE00' && r <= '\uFE0F') ||
		(r >= '\U000E0100' && r <= '\U000E01EF') ||
		unicode.Is(unicode.Cf, r)
}
