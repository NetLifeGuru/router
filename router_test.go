package router

import (
	"fmt"
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
		_, err := w.Write([]byte("world"))
		if err != nil {
			return
		}
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

func TestRecoveryFromPanic(t *testing.T) {

	defer func() {
		err := os.RemoveAll("./logs")
		if err != nil {

		}
	}()

	r := newTestableRouter()

	r.Recovery(func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		w.WriteHeader(http.StatusTeapot)
		_, err := w.Write([]byte("Recovered"))
		if err != nil {
			return
		}
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

func TestPrefixSegmentMiddleware(t *testing.T) {
	r := NewRouter().(*Router)
	r.Prefix("/api")

	r.HandleFunc("/hello", "GET", func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		_, err := w.Write([]byte("ok"))
		if err != nil {
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/hello", nil)
	w := httptest.NewRecorder()

	r.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK || w.Body.String() != "ok" {
		t.Errorf("prefix segment failed: got %d '%s'", w.Code, w.Body.String())
	}
}

func TestRegexSegmentRoute(t *testing.T) {
	r := NewRouter().(*Router)
	r.Prefix("/api")

	r.HandleFunc("/article/<article:([a-z]+)>", "GET", func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		article, _ := ctx.Param("article")
		_, err := w.Write([]byte(article))
		if err != nil {
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/article/hello", nil)
	w := httptest.NewRecorder()
	r.Handler().ServeHTTP(w, req)

	fmt.Println(w.Body.String())
	if w.Code != http.StatusOK || w.Body.String() != "hello" {
		t.Errorf("Prefix segment failed: got %d '%s'", w.Code, w.Body.String())
	}
}

func TestStaticFileServing(t *testing.T) {
	defer func() {
		err := os.RemoveAll("files")
		if err != nil {

		}
	}()

	dir := "files/public"
	_ = os.MkdirAll(dir, 0755)
	filePath := filepath.Join(dir, "test.txt")

	expectedContent := "Hello from static"
	err := os.WriteFile(filePath, []byte(expectedContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	r := NewRouter().(*Router)

	r.Static("files/public", "/assets")

	req := httptest.NewRequest(http.MethodGet, "/assets/test.txt", nil)
	w := httptest.NewRecorder()

	r.Handler().ServeHTTP(w, req)

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
		_, err := w.Write([]byte("pong"))
		if err != nil {
			return
		}
	})

	go func() {
		time.Sleep(100 * time.Millisecond)
		_, err := http.Get("http://localhost:8089/ping")
		if err != nil {
			return
		}
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

func TestRouteGrouping(t *testing.T) {
	r := NewRouter().(*Router)

	v1 := r.Group("/api/v1")
	v1.HandleFunc("/status", "GET", func(w http.ResponseWriter, _ *http.Request, ctx *Context) {
		_, err := w.Write([]byte("ok"))
		if err != nil {
			return
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	w := httptest.NewRecorder()

	r.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got '%s'", w.Body.String())
	}
}
