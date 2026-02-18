package template

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
)

const DefaultMaxOutputLen = 32_000

// Renderer renders text/template with user context and output limits.
// Safe for missing keys (returns empty string), no cache, minimal func map.
type Renderer struct {
	maxOutputLen int
	funcMap      template.FuncMap
}

// Option configures a Renderer.
type Option func(*Renderer)

// WithMaxOutputLen sets the maximum allowed length of the rendered output (default: DefaultMaxOutputLen).
func WithMaxOutputLen(n int) Option {
	return func(r *Renderer) {
		r.maxOutputLen = n
	}
}

// NewRenderer creates a Renderer with optional configuration.
func NewRenderer(opts ...Option) *Renderer {
	r := &Renderer{
		maxOutputLen: DefaultMaxOutputLen,
		funcMap:      safeFuncMap(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// safeFuncMap returns a minimal func map: no exec, eval, file/network access.
func safeFuncMap() template.FuncMap {
	return template.FuncMap{
		"default": func(def string, val interface{}) string {
			if val == nil {
				return def
			}
			s := fmt.Sprint(val)
			if s == "" {
				return def
			}
			return s
		},
		"eq": func(a, b interface{}) bool { return eq(a, b) },
		"and": func(x ...interface{}) bool {
			for _, v := range x {
				if !isTrue(v) {
					return false
				}
			}
			return true
		},
		"or": func(x ...interface{}) bool {
			for _, v := range x {
				if isTrue(v) {
					return true
				}
			}
			return false
		},
		"not": func(x interface{}) bool { return !isTrue(x) },
		"len": func(x interface{}) int {
			switch v := x.(type) {
			case string:
				return len(v)
			case []interface{}:
				return len(v)
			case map[string]interface{}:
				return len(v)
			default:
				return 0
			}
		},
		"printf": fmt.Sprintf,
	}
}

func eq(a, b interface{}) bool {
	return fmt.Sprint(a) == fmt.Sprint(b)
}

func isTrue(v interface{}) bool {
	if v == nil {
		return false
	}
	s := strings.TrimSpace(fmt.Sprint(v))
	return s != "" && s != "0" && strings.ToLower(s) != "false"
}

// safeData wraps userContext so missing keys don't panic and render as empty.
// text/template already renders nil as "" for map keys; we ensure we pass a non-nil map.
func safeData(userContext map[string]interface{}) map[string]interface{} {
	if userContext == nil {
		return map[string]interface{}{}
	}
	return userContext
}

// Render implements the TemplateRenderer contract: compiles tpl, executes with userContext, returns result up to max length.
func (r *Renderer) Render(ctx context.Context, tpl string, userContext map[string]interface{}) (string, error) {
	_ = ctx // reserved for future use (e.g. timeout, cache key)

	t, err := template.New("").Funcs(r.funcMap).Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("template parse: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, safeData(userContext)); err != nil {
		return "", fmt.Errorf("template execute: %w", err)
	}

	out := buf.String()
	// text/template prints "<no value>" for missing map keys; normalize to empty string per spec.
	out = strings.ReplaceAll(out, "<no value>", "")
	if len(out) > r.maxOutputLen {
		out = out[:r.maxOutputLen]
	}
	return out, nil
}
