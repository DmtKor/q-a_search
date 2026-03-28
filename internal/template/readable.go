package template

import (
	"regexp"
	"strings"
)

// ReadableSegment represents one part of human-readable template view.
type ReadableSegment struct {
	Type        string `json:"type"`         // "literal" | "readable" | "raw"
	Text        string `json:"text,omitempty"`
	Code        string `json:"code,omitempty"`
	Description string `json:"description,omitempty"`
}

// ToReadableSegments splits template into segments and assigns Russian descriptions to parseable actions.
// Literal text is unchanged; actions are either "readable" (with description) or "raw" (code only, shown bold).
func ToReadableSegments(tpl string) []ReadableSegment {
	var out []ReadableSegment
	s := tpl
	for {
		idx := strings.Index(s, "{{")
		if idx < 0 {
			if len(s) > 0 {
				out = append(out, ReadableSegment{Type: "literal", Text: s})
			}
			break
		}
		if idx > 0 {
			out = append(out, ReadableSegment{Type: "literal", Text: s[:idx]})
		}
		end := findActionEnd(s[idx:])
		if end < 0 {
			out = append(out, ReadableSegment{Type: "literal", Text: s[idx:]})
			break
		}
		action := s[idx : idx+end]
		inner := strings.TrimSpace(action[2 : len(action)-2])
		code := action
		desc := describeAction(inner)
		if desc != "" {
			out = append(out, ReadableSegment{Type: "readable", Code: code, Description: desc})
		} else {
			out = append(out, ReadableSegment{Type: "raw", Code: code})
		}
		s = s[idx+end:]
	}
	return out
}

// findActionEnd returns length of {{ ... }} including content; -1 if not found.
// Handles quoted strings inside the action.
func findActionEnd(s string) int {
	if len(s) < 4 || s[0] != '{' || s[1] != '{' {
		return -1
	}
	i := 2
	for i < len(s)-1 {
		switch s[i] {
		case '}':
			if s[i+1] == '}' {
				return i + 2
			}
			i++
		case '"':
			i++
			for i < len(s) && s[i] != '"' {
				if s[i] == '\\' {
					i++
				}
				i++
			}
			if i < len(s) {
				i++
			}
		case '`':
			i++
			for i < len(s) && s[i] != '`' {
				i++
			}
			if i < len(s) {
				i++
			}
		default:
			i++
		}
	}
	return -1
}

// arg in comparisons: .field, "str", or number
const argPat = `(?:\.([a-zA-Z0-9_.]+)|"([^"]*)"|(-?\d+))`

var (
	reDotField   = regexp.MustCompile(`^\.([a-zA-Z0-9_.]+)\s*$`)
	reIf         = regexp.MustCompile(`^if\s+\.([a-zA-Z0-9_.]+)\s*$`)
	reRange      = regexp.MustCompile(`^range\s+\.([a-zA-Z0-9_.]+)\s*$`)
	reElse       = regexp.MustCompile(`^else\s*$`)
	reEnd        = regexp.MustCompile(`^end\s*$`)
	reDefaultDq  = regexp.MustCompile(`^default\s+"([^"]*)"\s+\.([a-zA-Z0-9_.]+)\s*$`)
	reDefaultBq  = regexp.MustCompile("^default\\s+`([^`]*)`\\s+\\.([a-zA-Z0-9_.]+)\\s*$")
	reDefaultSq  = regexp.MustCompile(`^default\s+'([^']*)'\s+\.([a-zA-Z0-9_.]+)\s*$`)
	reIfEq       = regexp.MustCompile(`^if\s+eq\s+` + argPat + `\s+` + argPat + `\s*$`)
	reIfNe       = regexp.MustCompile(`^if\s+ne\s+` + argPat + `\s+` + argPat + `\s*$`)
	reIfLt       = regexp.MustCompile(`^if\s+lt\s+` + argPat + `\s+` + argPat + `\s*$`)
	reIfLe       = regexp.MustCompile(`^if\s+le\s+` + argPat + `\s+` + argPat + `\s*$`)
	reIfGt       = regexp.MustCompile(`^if\s+gt\s+` + argPat + `\s+` + argPat + `\s*$`)
	reIfGe       = regexp.MustCompile(`^if\s+ge\s+` + argPat + `\s+` + argPat + `\s*$`)
	reIfNot      = regexp.MustCompile(`^if\s+not\s+\.([a-zA-Z0-9_.]+)\s*$`)
)

func formatArg(field, quoted, num string) string {
	if field != "" {
		return "поле «" + field + "»"
	}
	if quoted != "" {
		return "«" + quoted + "»"
	}
	return num
}

func describeAction(inner string) string {
	inner = strings.TrimSpace(inner)
	if inner == "" {
		return ""
	}
	if m := reDotField.FindStringSubmatch(inner); len(m) > 0 {
		return "[подставить значение поля «" + m[1] + "»]"
	}
	// comparisons (before simple "if .x")
	if m := reIfEq.FindStringSubmatch(inner); len(m) >= 7 {
		return "[если " + formatArg(m[1], m[2], m[3]) + " равно " + formatArg(m[4], m[5], m[6]) + ":]"
	}
	if m := reIfNe.FindStringSubmatch(inner); len(m) >= 7 {
		return "[если " + formatArg(m[1], m[2], m[3]) + " не равно " + formatArg(m[4], m[5], m[6]) + ":]"
	}
	if m := reIfLt.FindStringSubmatch(inner); len(m) >= 7 {
		return "[если " + formatArg(m[1], m[2], m[3]) + " меньше " + formatArg(m[4], m[5], m[6]) + ":]"
	}
	if m := reIfLe.FindStringSubmatch(inner); len(m) >= 7 {
		return "[если " + formatArg(m[1], m[2], m[3]) + " меньше или равно " + formatArg(m[4], m[5], m[6]) + ":]"
	}
	if m := reIfGt.FindStringSubmatch(inner); len(m) >= 7 {
		return "[если " + formatArg(m[1], m[2], m[3]) + " больше " + formatArg(m[4], m[5], m[6]) + ":]"
	}
	if m := reIfGe.FindStringSubmatch(inner); len(m) >= 7 {
		return "[если " + formatArg(m[1], m[2], m[3]) + " больше или равно " + formatArg(m[4], m[5], m[6]) + ":]"
	}
	if m := reIfNot.FindStringSubmatch(inner); len(m) > 0 {
		return "[если поле «" + m[1] + "» ложно (пусто или false):]"
	}
	if m := reIf.FindStringSubmatch(inner); len(m) > 0 {
		return "[если поле «" + m[1] + "» истинно/существует:]"
	}
	if reElse.MatchString(inner) {
		return "[иначе]"
	}
	if reEnd.MatchString(inner) {
		return "[конец блока]"
	}
	if m := reRange.FindStringSubmatch(inner); len(m) > 0 {
		return "[для каждого элемента в «" + m[1] + "»:]"
	}
	if m := reDefaultDq.FindStringSubmatch(inner); len(m) > 0 {
		return "[если «" + m[2] + "» пусто — «" + m[1] + "», иначе значение «" + m[2] + "»]"
	}
	if m := reDefaultBq.FindStringSubmatch(inner); len(m) > 0 {
		return "[если «" + m[2] + "» пусто — «" + m[1] + "», иначе значение «" + m[2] + "»]"
	}
	if m := reDefaultSq.FindStringSubmatch(inner); len(m) > 0 {
		return "[если «" + m[2] + "» пусто — «" + m[1] + "», иначе значение «" + m[2] + "»]"
	}
	return ""
}
