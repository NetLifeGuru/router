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
	req.RemoteAddr = "127.0.0.1:1234"
	w := httptest.NewRecorder()

	RateLimit(w, req, time.Millisecond*100)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", resp.StatusCode)
	}
}

func TestRateLimit_BlockSecondFastRequest(t *testing.T) {
	resetRequestCounter()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	w1 := httptest.NewRecorder()
	RateLimit(w1, req, time.Millisecond*100)

	w2 := httptest.NewRecorder()
	RateLimit(w2, req, time.Millisecond*100)

	resp := w2.Result()
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("expected status 429 TooManyRequests, got %d", resp.StatusCode)
	}
}

func TestRateLimit_AllowAfterThreshold(t *testing.T) {
	resetRequestCounter()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"

	w1 := httptest.NewRecorder()
	RateLimit(w1, req, time.Millisecond*100)

	time.Sleep(time.Millisecond * 110)

	w2 := httptest.NewRecorder()
	RateLimit(w2, req, time.Millisecond*100)

	resp := w2.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 OK, got %d", resp.StatusCode)
	}
}

func TestCleanupOldRequests(t *testing.T) {
	resetRequestCounter()

	key := "GET|127.0.0.1|/"

	oldTime := time.Now().Add(-2 * time.Second)
	requestCounter.lastRequest.Store(key, oldTime)

	threshold := time.Millisecond * 100

	cleanupOldRequests(time.Now(), threshold*2)

	_, exists := requestCounter.lastRequest.Load(key)
	if exists {
		t.Errorf("expected old key to be cleaned")
	}
}
