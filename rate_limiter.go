package router

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type RequestCounter struct {
	lastRequest sync.Map
}

var trustedCIDRs []*net.IPNet
var requestCounter = &RequestCounter{}

func SetTrustedProxies(cidrs []string) {
	trustedCIDRs = trustedCIDRs[:0]
	for _, c := range cidrs {
		if _, ipn, err := net.ParseCIDR(c); err == nil {
			trustedCIDRs = append(trustedCIDRs, ipn)
		}
	}
}

func isTrustedRemote(remoteAddr string) bool {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		host = remoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	for _, n := range trustedCIDRs {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func fastTrimSpace(s string) string {
	start := 0
	end := len(s)

	for start < end {
		c := s[start]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			start++
		} else {
			break
		}
	}
	for end > start {
		c := s[end-1]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' {
			end--
		} else {
			break
		}
	}
	return s[start:end]
}

func firstCommaPart(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			return s[:i]
		}
	}
	return s
}

func clientIP(r *http.Request) string {
	if isTrustedRemote(r.RemoteAddr) {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return fastTrimSpace(firstCommaPart(xff))
		}
		if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
			return fastTrimSpace(xrip)
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}

func makeKey(r *http.Request) string {
	var b strings.Builder
	ip := clientIP(r)
	path := r.URL.Path

	b.Grow(len(r.Method) + 1 + len(ip) + 1 + len(path))
	b.WriteString(r.Method)
	b.WriteByte('|')
	b.WriteString(ip)
	b.WriteByte('|')
	b.WriteString(path)
	return b.String()
}

func Abort(ctx *Context) {
	ctx.Abort()
}

func RateLimit(w http.ResponseWriter, r *http.Request, threshold time.Duration) bool {
	now := time.Now()
	key := makeKey(r)

	if v, ok := requestCounter.lastRequest.Load(key); ok {
		if last, ok := v.(time.Time); ok && now.Sub(last) < threshold {
			w.Header().Set("Retry-After", "1")
			JSON(w, http.StatusTooManyRequests, Msg{
				Title: "too_many_requests", Message: "Please slow down.", StatusCode: http.StatusTooManyRequests,
			})

			return true
		}
	}

	requestCounter.lastRequest.Store(key, now)
	cleanupOldRequests(now, threshold*2)

	return false
}

func cleanupOldRequests(now time.Time, ttl time.Duration) {
	cutoff := now.Add(-ttl)
	requestCounter.lastRequest.Range(func(k, v any) bool {
		if t, ok := v.(time.Time); ok && t.Before(cutoff) {
			requestCounter.lastRequest.Delete(k)
		}
		return true
	})
}
