package router

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type testableRouter interface {
	http.Handler
	IRouter
}

func newTestableRouter() testableRouter {
	return NewRouter().(*Router)
}

func TestHandleFuncAndServeHTTP(t *testing.T) {
	r := newTestableRouter()

	r.HandleFunc("/hello", "GET", func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("world"))
	})

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if string(body) != "world" {
		t.Errorf("expected body 'world', got %s", string(body))
	}
}

func TestBeforeAfterHandlers(t *testing.T) {
	r := newTestableRouter()

	calls := []string{}

	r.Before(func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		calls = append(calls, "before")
	})

	r.After(func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		calls = append(calls, "after")
	})

	r.HandleFunc("/test", "GET", func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		calls = append(calls, "handler")
		w.Write([]byte("done"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	expectedOrder := []string{"before", "handler", "after"}
	for i, call := range expectedOrder {
		if calls[i] != call {
			t.Errorf("expected %s at position %d, got %s", call, i, calls[i])
		}
	}
}

func TestRecoveryFromPanic(t *testing.T) {

	defer os.RemoveAll("./logs")

	r := newTestableRouter()

	r.Recovery(func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("Recovered"))
	})

	r.HandleFunc("/panic", "GET", func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		panic("fail")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusTeapot {
		t.Errorf("expected status 418, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Recovered") {
		t.Errorf("expected recovery message, got '%s'", w.Body.String())
	}
}

func TestProxySegmentMiddleware(t *testing.T) {
	r := newTestableRouter()
	r.Proxy("/api")

	r.HandleFunc("/hello", "GET", func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		w.Write([]byte("proxied"))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/hello", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK || w.Body.String() != "proxied" {
		t.Errorf("proxy segment failed: got %d '%s'", w.Code, w.Body.String())
	}
}

func TestRegexSegmentRoute(t *testing.T) {
	r := newTestableRouter()
	r.Proxy("/api")

	r.HandleFunc("/article/<article:([a-z]+)>", "GET", func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		w.Write([]byte(ctx.Params["article"].(string)))
	})

	req := httptest.NewRequest(http.MethodGet, "/api/article/hello", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK || w.Body.String() != "hello" {
		t.Errorf("proxy segment failed: got %d '%s'", w.Code, w.Body.String())
	}
}

func TestStaticFileServing(t *testing.T) {

	defer os.RemoveAll("files")

	dir := "files/public"
	_ = os.MkdirAll(dir, 0755)
	filePath := filepath.Join(dir, "test.txt")

	expectedContent := "Hello from static"
	err := os.WriteFile(filePath, []byte(expectedContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	r := newTestableRouter()
	r.Static("files/public", "/assets")

	req := httptest.NewRequest(http.MethodGet, "/assets/test.txt", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	if string(body) != expectedContent {
		t.Errorf("unexpected body: got '%s', expected '%s'", string(body), expectedContent)
	}
}

func TestListenAndServe(t *testing.T) {
	r := newTestableRouter()

	r.HandleFunc("/ping", "GET", func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		w.Write([]byte("pong"))
	})

	go func() {
		time.Sleep(100 * time.Millisecond)
		http.Get("http://localhost:8089/ping")
	}()

	go func() {
		r.ListenAndServe(8089)
	}()
	time.Sleep(300 * time.Millisecond)
}

func TestEnableProfiling(t *testing.T) {
	r := newTestableRouter()
	r.EnableProfiling("localhost:6060")

	time.Sleep(100 * time.Millisecond)
	resp, err := http.Get("http://localhost:6060/debug/pprof/")
	if err != nil || resp.StatusCode != 200 {
		t.Errorf("pprof endpoint not available: %v, status: %d", err, resp.StatusCode)
	}
}
