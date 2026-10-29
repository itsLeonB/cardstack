package mapper

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// SlugOf is a Slug (GLOSSARY.md, ADR-0025): text lower-cased and hyphenated,
// then the last 5 hex characters of the row's own id. Text with no a-z or 0-9
// becomes fallback. The profile Slug migration repeats these rules in SQL.
func SlugOf(text, fallback string, id uuid.UUID) string {
	stem := strings.Trim(nonSlugChars.ReplaceAllString(strings.ToLower(text), "-"), "-")
	if stem == "" {
		stem = fallback
	}
	return stem + "-" + id.String()[31:]
}
