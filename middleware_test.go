package router

import (
	"bytes"
	"compress/gzip"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestContext() *Context {
	return &Context{}
}

func makeTrackingHandler(called *bool) HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *Context) {
		*called = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}

func TestRouterMiddlewareOrder(t *testing.T) {
	r := NewRouter().(*Router)

	var order []string

	m1 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) {
			order = append(order, "m1-before")
			next(w, r, c)
			order = append(order, "m1-after")
		}
	}

	m2 := func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) {
			order = append(order, "m2-before")
			next(w, r, c)
			order = append(order, "m2-after")
		}
	}

	r.Use(m1)
	r.Use(m2)

	finalCalled := false
	final := makeTrackingHandler(&finalCalled)

	wrapped := r.wrap("/", final)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := newTestContext()

	wrapped(rr, req, ctx)

	if !finalCalled {
		t.Fatalf("final handler not called")
	}

	expected := []string{
		"m1-before",
		"m2-before",
		"m2-after",
		"m1-after",
	}

	if len(order) != len(expected) {
		t.Fatalf("unexpected order length: got %d, want %d (%v)", len(order), len(expected), order)
	}
	for i := range expected {
		if order[i] != expected[i] {
			t.Fatalf("order[%d] = %q, want %q (full=%v)", i, order[i], expected[i], order)
		}
	}
}

func TestAllowContentTypeAllowed(t *testing.T) {
	m := AllowContentType("application/json")

	called := false
	h := m(makeTrackingHandler(&called))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"a":1}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler should have been called for allowed content type")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d, want %d", rr.Code, http.StatusOK)
	}
	if ctx.Aborted() {
		t.Fatalf("ctx should not be aborted")
	}
}

func TestAllowContentTypeNotAllowed(t *testing.T) {
	m := AllowContentType("application/json")

	called := false
	h := m(makeTrackingHandler(&called))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`<xml></xml>`))
	req.Header.Set("Content-Type", "text/xml")

	ctx := newTestContext()

	h(rr, req, ctx)

	if called {
		t.Fatalf("handler should NOT have been called for disallowed content type")
	}
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("unexpected status: got %d, want %d", rr.Code, http.StatusUnsupportedMediaType)
	}
	if !ctx.Aborted() {
		t.Fatalf("ctx should be aborted")
	}
}

func TestCleanPath(t *testing.T) {
	m := CleanPath()

	called := false
	h := m(makeTrackingHandler(&called))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "//foo//bar///baz", nil)
	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler should have been called")
	}
	if got, want := req.URL.Path, "/foo/bar/baz"; got != want {
		t.Fatalf("path not cleaned: got %q, want %q", got, want)
	}
}

func TestGetHead(t *testing.T) {
	m := GetHead()

	var methodSeen string
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		methodSeen = r.Method
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/x", nil)
	ctx := newTestContext()

	h(rr, req, ctx)

	if methodSeen != http.MethodGet {
		t.Fatalf("expected method to be rewritten to GET, got %q", methodSeen)
	}
}

func TestContentCharsetAllowed(t *testing.T) {
	m := ContentCharset("utf-8", "")

	called := false
	h := m(makeTrackingHandler(&called))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler should have been called for allowed charset")
	}
	if ctx.Aborted() {
		t.Fatalf("ctx should not be aborted")
	}
}

func TestContentCharsetNotAllowed(t *testing.T) {
	m := ContentCharset("utf-8")

	called := false
	h := m(makeTrackingHandler(&called))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "application/json; charset=latin1")

	ctx := newTestContext()

	h(rr, req, ctx)

	if called {
		t.Fatalf("handler should NOT have been called for disallowed charset")
	}
	if rr.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("unexpected status: got %d, want %d", rr.Code, http.StatusUnsupportedMediaType)
	}
	if !ctx.Aborted() {
		t.Fatalf("ctx should be aborted")
	}
}

func TestCompressSkipsWhenNoGzipAccepted(t *testing.T) {
	m := Compress(gzip.DefaultCompression, "text/plain")

	called := false
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		called = true
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("hello"))
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler should have been called")
	}
	if enc := rr.Header().Get("Content-Encoding"); enc != "" {
		t.Fatalf("expected no Content-Encoding, got %q", enc)
	}
	if body := rr.Body.String(); body != "hello" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestCompressGzipForAllowedType(t *testing.T) {
	m := Compress(gzip.DefaultCompression, "text/plain")

	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("hello gzip"))
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	ctx := newTestContext()

	h(rr, req, ctx)

	if enc := rr.Header().Get("Content-Encoding"); enc != "gzip" {
		t.Fatalf("expected Content-Encoding gzip, got %q", enc)
	}

	gr, err := gzip.NewReader(bytes.NewReader(rr.Body.Bytes()))
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}

	defer func(gr *gzip.Reader) {
		err := gr.Close()
		if err != nil {
			t.Fatalf("failed to read gzipped body: %v", err)
		}
	}(gr)

	b, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("failed to read gzipped body: %v", err)
	}

	if got, want := string(b), "hello gzip"; got != want {
		t.Fatalf("unexpected decompressed body: got %q, want %q", got, want)
	}
}

func TestMatchOrigin(t *testing.T) {
	tests := []struct {
		origin   string
		patterns []string
		want     bool
	}{
		{"https://example.com", []string{"https://*"}, true},
		{"https://example.com", []string{"*"}, true},
		{"http://foo.bar", []string{"https://*"}, false},
		{"http://foo.bar", []string{"http://*"}, true},
		{"https://api.test.com", []string{"https://api.test.com"}, true},
		{"https://api.test.com", []string{"https://other.com"}, false},
		{"", []string{"*"}, false},
	}

	for i, tt := range tests {
		got := matchOrigin(tt.origin, tt.patterns)
		if got != tt.want {
			t.Fatalf("case %d: matchOrigin(%q, %v) = %v, want %v", i, tt.origin, tt.patterns, got, tt.want)
		}
	}
}

func TestCORSNonCORSRequest(t *testing.T) {
	opts := CORSOptions{
		AllowedOrigins: []string{"https://*"},
	}
	m := CORS(opts)

	called := false
	h := m(makeTrackingHandler(&called))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler should have been called for non-CORS request")
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("expected no CORS headers")
	}
}

func TestCORSPreflight(t *testing.T) {
	opts := CORSOptions{
		AllowedOrigins: []string{"https://*"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Accept", "Authorization"},
		MaxAge:         300,
	}
	m := CORS(opts)

	called := false
	h := m(makeTrackingHandler(&called))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	ctx := newTestContext()

	h(rr, req, ctx)

	if called {
		t.Fatalf("handler should NOT be called on preflight")
	}
	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d, want %d", rr.Code, http.StatusNoContent)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("unexpected ACAO: %q", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatalf("expected Access-Control-Allow-Methods to be set")
	}
	if got := rr.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatalf("expected Access-Control-Allow-Headers to be set")
	}
	if got := rr.Header().Get("Access-Control-Max-Age"); got != "300" {
		t.Fatalf("unexpected Access-Control-Max-Age: %q", got)
	}
}

func TestCORSNormalCORSRequest(t *testing.T) {
	opts := CORSOptions{
		AllowedOrigins: []string{"https://*"},
		ExposedHeaders: []string{"Link"},
	}
	m := CORS(opts)

	called := false
	h := m(makeTrackingHandler(&called))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("Origin", "https://example.com")

	ctx := newTestContext()

	h(rr, req, ctx)

	if !called {
		t.Fatalf("handler should be called for normal CORS request")
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("unexpected ACAO: %q", got)
	}
	if got := rr.Header().Get("Access-Control-Expose-Headers"); got != "Link" {
		t.Fatalf("unexpected Expose-Headers: %q", got)
	}
}

func TestRequestIDGeneratedWhenMissing(t *testing.T) {
	m := RequestID()

	var capturedID string
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		capturedID = GetRequestID(r)
		if capturedID == "" {
			t.Errorf("GetRequestID returned empty")
		}
		if ctxVal := c.Get("request_id"); ctxVal != capturedID {
			t.Errorf("ctx request_id = %v, want %v", ctxVal, capturedID)
		}
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := newTestContext()

	h(rr, req, ctx)

	if capturedID == "" {
		t.Fatalf("no request id captured")
	}
	if headerID := rr.Header().Get("X-Request-ID"); headerID == "" || headerID != capturedID {
		t.Fatalf("X-Request-ID header = %q, want %q", headerID, capturedID)
	}
}

func TestRequestIDUsesExistingHeader(t *testing.T) {
	m := RequestID()

	const given = "abc-123"

	var capturedID string
	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		capturedID = GetRequestID(r)
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-ID", given)
	ctx := newTestContext()

	h(rr, req, ctx)

	if capturedID != given {
		t.Fatalf("expected request id %q, got %q", given, capturedID)
	}
	if headerID := rr.Header().Get("X-Request-ID"); headerID != given {
		t.Fatalf("X-Request-ID header = %q, want %q", headerID, given)
	}
}

func TestRealIPFromXRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "1.2.3.4")

	if got := realIPFromRequest(req); got != "1.2.3.4" {
		t.Fatalf("realIPFromRequest = %q, want %q", got, "1.2.3.4")
	}
}

func TestRealIPFromXForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "5.6.7.8, 9.10.11.12")

	if got := realIPFromRequest(req); got != "5.6.7.8" {
		t.Fatalf("realIPFromRequest = %q, want %q", got, "5.6.7.8")
	}
}

func TestRealIPFromRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = net.JoinHostPort("10.0.0.1", "12345")

	if got := realIPFromRequest(req); got != "10.0.0.1" {
		t.Fatalf("realIPFromRequest = %q, want %q", got, "10.0.0.1")
	}
}

func TestRealIPMiddleware(t *testing.T) {
	m := RealIP()

	var capturedIP string

	h := m(func(w http.ResponseWriter, r *http.Request, c *Context) {
		capturedIP = GetRealIP(r)
		if ctxVal := c.Get("real_ip"); ctxVal != capturedIP {
			t.Errorf("ctx real_ip = %v, want %v", ctxVal, capturedIP)
		}
		w.WriteHeader(http.StatusOK)
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "123.123.123.123")
	ctx := newTestContext()

	h(rr, req, ctx)

	if capturedIP != "123.123.123.123" {
		t.Fatalf("capturedIP = %q, want %q", capturedIP, "123.123.123.123")
	}
	if req.RemoteAddr != "123.123.123.123" {
		t.Fatalf("expected RemoteAddr to be updated, got %q", req.RemoteAddr)
	}
}

func TestNoCache(t *testing.T) {
	m := NoCache()

	h := m(makeTrackingHandler(new(bool)))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := newTestContext()

	h(rr, req, ctx)

	hdr := rr.Header()
	if got := hdr.Get("Cache-Control"); got == "" {
		t.Fatalf("Cache-Control not set")
	}
	if got := hdr.Get("Pragma"); got == "" {
		t.Fatalf("Pragma not set")
	}
	if got := hdr.Get("Expires"); got == "" {
		t.Fatalf("Expires not set")
	}
}
