package fts

import (
	"strings"
)

// BuildSearchTSVInput returns a single string from title + keywords + questions
// for use in SQL: to_tsvector('simple', $1).
// Keywords and questions are joined with space; config is 'simple' (MVP).
func BuildSearchTSVInput(title string, keywords, questions []string) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(title))
	b.WriteString(" ")
	b.WriteString(strings.Join(keywords, " "))
	b.WriteString(" ")
	b.WriteString(strings.Join(questions, " "))
	return strings.TrimSpace(b.String())
}
