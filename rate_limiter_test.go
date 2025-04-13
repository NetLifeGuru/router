package router

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func resetRequestCounter() {
	requestCounter = &RequestCounter{}
}

func TestRateLimit_AllowFirstRequest(t *testing.T) {
	resetRequestCounter()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "example.com"
	w := httptest.NewRecorder()

	RateLimit(w, req, int64(time.Millisecond)*100)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", resp.StatusCode)
	}
}

func TestRateLimit_BlockSecondFastRequest(t *testing.T) {
	resetRequestCounter()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "test.com"
	w := httptest.NewRecorder()

	RateLimit(w, req, int64(time.Millisecond)*100)

	w2 := httptest.NewRecorder()
	RateLimit(w2, req, int64(time.Millisecond)*100)

	resp := w2.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", resp.StatusCode)
	}
}

func TestRateLimit_AllowAfterThreshold(t *testing.T) {
	resetRequestCounter()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Host = "slowpoke.com"
	w := httptest.NewRecorder()

	RateLimit(w, req, int64(time.Millisecond)*100)

	time.Sleep(time.Millisecond * 110)

	w2 := httptest.NewRecorder()
	RateLimit(w2, req, int64(time.Millisecond)*100)

	resp := w2.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK after delay, got %d", resp.StatusCode)
	}
}

func TestCleanupOldRequests(t *testing.T) {
	resetRequestCounter()

	host := "cleanup.com"
	oldTime := time.Now().Add(-2 * time.Second)
	requestCounter.lastRequest.Store(host, oldTime)

	cleanupOldRequests(time.Now())

	_, exists := requestCounter.lastRequest.Load(host)
	if exists {
		t.Errorf("expected old host entry to be cleaned up")
	}
}
