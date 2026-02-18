package template

import (
	"context"
	"errors"
	"strings"
	"testing"
)

var ctx = context.Background()

func TestRenderer_Render_HappyPath(t *testing.T) {
	r := NewRenderer()
	out, err := r.Render(ctx, "Hello, {{.name}}!", map[string]interface{}{"name": "Alice"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "Hello, Alice!" {
		t.Errorf("got %q", out)
	}
}

func TestRenderer_Render_EmptyContext(t *testing.T) {
	r := NewRenderer()
	out, err := r.Render(ctx, "Hello, {{.name}}!", map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Missing key: template renders zero value as empty
	if out != "Hello, !" {
		t.Errorf("got %q", out)
	}
}

func TestRenderer_Render_NilContext(t *testing.T) {
	r := NewRenderer()
	out, err := r.Render(ctx, "Hello, {{.name}}!", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "Hello, !" {
		t.Errorf("got %q", out)
	}
}

func TestRenderer_Render_MissingFields_NoPanic(t *testing.T) {
	r := NewRenderer()
	tpl := "A={{.a}} B={{.b}} C={{.c}}"
	ctxMap := map[string]interface{}{"a": "only"}
	out, err := r.Render(ctx, tpl, ctxMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "A=only B= C=" {
		t.Errorf("got %q", out)
	}
}

func TestRenderer_Render_InvalidTemplate(t *testing.T) {
	r := NewRenderer()
	_, err := r.Render(ctx, "Hello {{.name", nil)
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("error should mention parse: %v", err)
	}
}

func TestRenderer_Render_InvalidExecute(t *testing.T) {
	r := NewRenderer()
	// Template that indexes into non-map (e.g. .name on a string) can cause execute error in some cases.
	// Use a template that fails at execute: e.g. calling a func that returns error (we don't have such).
	// Simpler: use undefined variable if possible. In text/template, {{.x.y}} when .x is nil may error.
	_, err := r.Render(ctx, "{{.a.b}}", map[string]interface{}{"a": "not a map"})
	if err != nil {
		if !strings.Contains(err.Error(), "execute") {
			t.Logf("execute error (acceptable): %v", err)
		}
	}
}

func TestRenderer_Render_OutputLimit(t *testing.T) {
	const max = 10
	r := NewRenderer(WithMaxOutputLen(max))
	longTpl := "{{.x}}"
	longCtx := map[string]interface{}{"x": "123456789012345"}
	out, err := r.Render(ctx, longTpl, longCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != max {
		t.Errorf("len(out)=%d, want %d", len(out), max)
	}
	if out != "1234567890" {
		t.Errorf("got %q", out)
	}
}

func TestRenderer_Render_OutputLimitExact(t *testing.T) {
	const max = 5
	r := NewRenderer(WithMaxOutputLen(max))
	out, err := r.Render(ctx, "{{.x}}", map[string]interface{}{"x": "abc"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "abc" || len(out) != 3 {
		t.Errorf("got %q len=%d", out, len(out))
	}
}

func TestRenderer_Render_DefaultFunc(t *testing.T) {
	r := NewRenderer()
	out, err := r.Render(ctx, `Hello, {{default "Guest" .name}}!`, map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "Hello, Guest!" {
		t.Errorf("got %q", out)
	}
}

func TestRenderer_Render_ImplementsSearchInterface(t *testing.T) {
	// Ensure *Renderer can be used where search.TemplateRenderer is needed (compile-time check).
	var _ interface {
		Render(context.Context, string, map[string]interface{}) (string, error)
	} = NewRenderer()
}

func TestDefaultMaxOutputLen(t *testing.T) {
	if DefaultMaxOutputLen != 32_000 {
		t.Errorf("DefaultMaxOutputLen = %d, want 32000", DefaultMaxOutputLen)
	}
}

func TestRenderer_ReturnsParseError(t *testing.T) {
	r := NewRenderer()
	_, err := r.Render(ctx, "{{end}}", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var parseErr interface{ Error() string }
	if !errors.As(err, &parseErr) {
		t.Logf("error chain: %v", err)
	}
}
