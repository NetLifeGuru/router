package router

import (
	"testing"
)

func TestContextSetAndGet(t *testing.T) {
	ctx := &Context{}
	key := "foo"
	val := "bar"

	ctx.Set(key, val)

	got := ctx.Get(key)
	if got != val {
		t.Errorf("expected Get(%q) = %q, got %q", key, val, got)
	}

	if ctx.Get("nonexistent") != nil {
		t.Error("expected Get(nonexistent) to return nil")
	}
}

func TestContextParam(t *testing.T) {
	ctx := &Context{
		Params: []Par{
			{"id", "123"},
			{"name", "john"},
		},
	}

	if v, ok := ctx.Param("id"); !ok || v != "123" {
		t.Errorf(`expected Param("id") = "123", got %q (ok=%v)`, v, ok)
	}

	if v, ok := ctx.Param("name"); !ok || v != "john" {
		t.Errorf(`expected Param("name") = "john", got %q (ok=%v)`, v, ok)
	}

	if v, ok := ctx.Param("missing"); ok || v != "" {
		t.Errorf(`expected Param("missing") = "" and ok=false, got %q (ok=%v)`, v, ok)
	}
}

func TestContextReset(t *testing.T) {
	ctx := &Context{
		Params: []Par{{"a", "1"}, {"b", "2"}},
		Segments: []Seg{
			{"segment"},
		},
		Data: map[string]interface{}{
			"key": "value",
		},
	}

	ctx.reset()

	if len(ctx.Params) != 0 {
		t.Errorf("expected Params to be empty after reset, got %d", len(ctx.Params))
	}
	if len(ctx.Segments) != 0 {
		t.Errorf("expected Segments to be empty after reset, got %d", len(ctx.Segments))
	}
	if ctx.Data != nil {
		t.Error("expected Data to be nil after reset")
	}
}

func TestGetContextAndPutContext(t *testing.T) {
	ctx1 := GetContext()
	ctx1.Set("x", "1")
	ctx1.Params = append(ctx1.Params, Par{"key", "value"})
	PutContext(ctx1)

	ctx2 := GetContext()

	if ctx2 == nil {
		t.Fatal("expected non-nil context")
	}
	if len(ctx2.Params) != 0 {
		t.Errorf("expected Params to be reset, got %v", ctx2.Params)
	}
	if len(ctx2.Segments) != 0 {
		t.Errorf("expected Segments to be reset, got %v", ctx2.Segments)
	}
	if ctx2.Get("x") != nil {
		t.Errorf("expected Data to be reset, got %v", ctx2.Get("x"))
	}
}
