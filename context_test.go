package router

import (
	"testing"
)

func TestContext_SetAndGet(t *testing.T) {
	ctx := &Context{}

	key := "foo"
	value := "bar"

	ctx.Set(key, value)

	got := ctx.Get(key)
	if got != value {
		t.Errorf("expected %v, got %v", value, got)
	}
}

func TestContext_GetFromEmpty(t *testing.T) {
	ctx := &Context{}
	got := ctx.Get("nonexistent")

	if got != nil {
		t.Errorf("expected nil for nonexistent key, got %v", got)
	}
}

func TestContext_Param(t *testing.T) {
	ctx := &Context{
		Params: map[string]interface{}{
			"id": 123,
		},
	}

	got := ctx.Param("id")
	if got != "123" {
		t.Errorf("expected '123', got %s", got)
	}

	// non-existing key
	none := ctx.Param("name")
	if none != "" {
		t.Errorf("expected empty string for missing key, got %s", none)
	}
}

func TestContext_Reset(t *testing.T) {
	ctx := &Context{
		Params: map[string]interface{}{
			"id": "value",
		},
		data: map[string]interface{}{
			"key": "value",
		},
	}

	ctx.reset()

	if len(ctx.Params) != 0 {
		t.Errorf("expected Params to be empty after reset, got %v", ctx.Params)
	}
	if len(ctx.data) != 0 {
		t.Errorf("expected data to be empty after reset, got %v", ctx.data)
	}
}

func TestContextPool(t *testing.T) {
	ctx := contextPool.Get().(*Context)

	// simulate usage
	ctx.Set("test", "value")
	ctx.Params["id"] = 42

	ctx.reset()
	contextPool.Put(ctx)

	// Get context again
	newCtx := contextPool.Get().(*Context)

	if val := newCtx.Get("test"); val != nil {
		t.Errorf("expected nil after reset, got %v", val)
	}

	if param := newCtx.Param("id"); param != "" {
		t.Errorf("expected empty string after reset, got %s", param)
	}
}
