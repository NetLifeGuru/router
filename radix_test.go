package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func handlerWithID(id string) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *Context) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(id))
	}
}

func TestLongestCommonPrefixStr(t *testing.T) {
	tests := []struct {
		a, b     string
		expected int
	}{
		{"", "", 0},
		{"abc", "", 0},
		{"", "xyz", 0},
		{"abc", "abc", 3},
		{"abcd", "abxy", 2},
		{"/user/", "/user/123", 6},
		{"/foo/bar", "/foo/baz", 7},
	}

	for _, tt := range tests {
		got := longestCommonPrefixStr(tt.a, tt.b)
		if got != tt.expected {
			t.Errorf("longestCommonPrefixStr(%q,%q) = %d, want %d", tt.a, tt.b, got, tt.expected)
		}
	}
}

func TestMatchPrefixWithStarStr_NoWildcard(t *testing.T) {
	consumed, ok := matchPrefixWithStarStr("/user/", "/user/123")
	if !ok {
		t.Fatalf("expected match")
	}
	if consumed != len("/user/") {
		t.Fatalf("expected consumed=%d, got %d", len("/user/"), consumed)
	}
}

func TestMatchPrefixWithStarStr_WithWildcardSingleSegment(t *testing.T) {
	prefix := "/user/*/profile"
	key := "/user/123/profile"

	consumed, ok := matchPrefixWithStarStr(prefix, key)
	if !ok {
		t.Fatalf("expected wildcard match")
	}

	if key[consumed:] != "" {
		t.Fatalf("expected full match, remaining %q", key[consumed:])
	}
}

func TestMatchPrefixWithStarStr_NoMatch(t *testing.T) {
	if _, ok := matchPrefixWithStarStr("/abc", "/xyz"); ok {
		t.Fatalf("expected no match for different prefixes")
	}
}

func newTestRouter() *Router {
	r := &Router{
		radixRoot:    &RadixNode{},
		staticRoutes: make(StaticRoutes),
	}
	return r
}

func TestRadixInsertAndSearch_SimpleExact(t *testing.T) {
	r := newTestRouter()

	entry := RouteEntry{
		Route:   "/users",
		Handler: handlerWithID("users"),
		Bitmask: GET,
	}

	r.insertNode("/users", entry)

	ctx := &Context{}
	found := r.searchAll("/users", ctx)
	if !found {
		t.Fatalf("expected to find /users route")
	}

	if len(ctx.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(ctx.Entries))
	}

	if ctx.Entries[0].Route != "/users" {
		t.Fatalf("expected route /users, got %q", ctx.Entries[0].Route)
	}
}

func TestRadixInsertAndSearch_PrefixSplit(t *testing.T) {
	r := newTestRouter()

	entryUsers := RouteEntry{
		Route:   "/users",
		Handler: handlerWithID("users"),
		Bitmask: GET,
	}
	entryUsersMe := RouteEntry{
		Route:   "/users/me",
		Handler: handlerWithID("users_me"),
		Bitmask: GET,
	}

	r.insertNode("/users", entryUsers)
	r.insertNode("/users/me", entryUsersMe)

	ctx := &Context{}
	if !r.searchAll("/users/me", ctx) {
		t.Fatalf("expected to find /users/me")
	}

	if len(ctx.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(ctx.Entries))
	}
	if ctx.Entries[0].Route != "/users/me" {
		t.Fatalf("expected route /users/me, got %q", ctx.Entries[0].Route)
	}
}

func TestRadixInsertSameKeyMultipleEntries(t *testing.T) {
	r := newTestRouter()

	entry1 := RouteEntry{
		Route:   "/same",
		Handler: handlerWithID("one"),
		Bitmask: GET,
	}
	entry2 := RouteEntry{
		Route:   "/same",
		Handler: handlerWithID("two"),
		Bitmask: POST,
	}

	r.insertNode("/same", entry1)
	r.insertNode("/same", entry2)

	ctx := &Context{}
	if !r.searchAll("/same", ctx) {
		t.Fatalf("expected to find /same")
	}

	if len(ctx.Entries) != 2 {
		t.Fatalf("expected 2 entries on same key, got %d", len(ctx.Entries))
	}
}

func TestRadixSearchAllWithWildcard(t *testing.T) {

	r := newTestRouter()

	entry := RouteEntry{
		Route:   "/users/<id:any>",
		Handler: handlerWithID("user-id"),
		Bitmask: GET,
	}

	r.insertNode("/users/*", entry)

	ctx := &Context{}
	if !r.searchAll("/users/123", ctx) {
		t.Fatalf("expected wildcard route to match")
	}

	if len(ctx.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(ctx.Entries))
	}
	if ctx.Entries[0].Route != "/users/<id:any>" {
		t.Fatalf("unexpected route stored: %q", ctx.Entries[0].Route)
	}
}

func TestRadixSearchAll_NoMatch(t *testing.T) {
	r := newTestRouter()

	entry := RouteEntry{
		Route:   "/foo",
		Handler: handlerWithID("foo"),
		Bitmask: GET,
	}
	r.insertNode("/foo", entry)

	ctx := &Context{}
	if r.searchAll("/bar", ctx) {
		t.Fatalf("expected no match for /bar")
	}
	if len(ctx.Entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(ctx.Entries))
	}
}

func TestRadixSearchIntegrationThroughHandleFunc(t *testing.T) {
	r := NewRouter().(*Router)

	r.HandleFunc("/users/<id:any>", "GET", handlerWithID("user-id"))

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if body := w.Body.String(); body != "user-id" {
		t.Fatalf("unexpected body: %q", body)
	}
}
