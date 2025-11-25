package router

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"mime"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
)

type Middleware func(HandlerFunc) HandlerFunc

func (r *Router) Use(m Middleware) {
	r.middlewares = append(r.middlewares, m)
}

func (r *Router) wrap(h HandlerFunc) HandlerFunc {
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		h = r.middlewares[i](h)
	}
	return h
}

func AllowContentType(types ...string) Middleware {
	allowed := make(map[string]struct{}, len(types))
	for _, t := range types {
		allowed[t] = struct{}{}
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, ctx *Context) {
			ct := r.Header.Get("Content-Type")
			if ct != "" {
				if i := strings.Index(ct, ";"); i >= 0 {
					ct = ct[:i]
				}

				if _, ok := allowed[ct]; !ok {
					http.Error(w, "Unsupported Content-Type", http.StatusUnsupportedMediaType)
					ctx.Abort()
					return
				}
			}

			next(w, r, ctx)
		}
	}
}

func CleanPath() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, ctx *Context) {
			normalized := path.Clean(r.URL.Path)
			if normalized != r.URL.Path {
				r.URL.Path = normalized
			}
			next(w, r, ctx)
		}
	}
}

func GetHead() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) {
			if r.Method == http.MethodHead {
				r.Method = http.MethodGet
			}
			next(w, r, c)
		}
	}
}

func ContentCharset(charsets ...string) Middleware {
	allowed := make(map[string]struct{}, len(charsets))
	for _, ch := range charsets {
		allowed[strings.ToLower(ch)] = struct{}{}
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, ctx *Context) {
			ct := r.Header.Get("Content-Type")
			if ct == "" {
				next(w, r, ctx)
				return
			}

			_, params, err := mime.ParseMediaType(ct)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
				ctx.Abort()
				return
			}

			cs := strings.ToLower(params["charset"])
			if _, ok := allowed[cs]; !ok {
				http.Error(w, http.StatusText(http.StatusUnsupportedMediaType), http.StatusUnsupportedMediaType)
				ctx.Abort()
				return
			}

			next(w, r, ctx)
		}
	}
}

type compressResponseWriter struct {
	http.ResponseWriter
	types    map[string]struct{}
	level    int
	gz       *gzip.Writer
	status   int
	wroteHdr bool
}

func (cw *compressResponseWriter) WriteHeader(status int) {
	if cw.wroteHdr {
		return
	}
	cw.wroteHdr = true
	cw.status = status
	cw.ResponseWriter.WriteHeader(status)
}

func (cw *compressResponseWriter) Header() http.Header {
	return cw.ResponseWriter.Header()
}

func (cw *compressResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := cw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, fmt.Errorf("hijacker not supported")
}

func (cw *compressResponseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := cw.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (cw *compressResponseWriter) enableGzip() {
	if cw.gz != nil {
		return
	}

	cw.Header().Del("Content-Length")
	cw.Header().Set("Content-Encoding", "gzip")

	gz, err := gzip.NewWriterLevel(cw.ResponseWriter, cw.level)
	if err != nil {
		return
	}
	cw.gz = gz
}

func (cw *compressResponseWriter) Write(b []byte) (int, error) {

	if !cw.wroteHdr {
		cw.status = http.StatusOK
		cw.WriteHeader(cw.status)
	}

	if cw.status < 200 || cw.status >= 300 || cw.status == 204 {
		return cw.ResponseWriter.Write(b)
	}

	if cw.gz == nil {
		ct := cw.Header().Get("Content-Type")
		if i := strings.Index(ct, ";"); i >= 0 {
			ct = ct[:i]
		}
		ct = strings.ToLower(strings.TrimSpace(ct))

		if _, ok := cw.types[ct]; !ok {
			return cw.ResponseWriter.Write(b)
		}

		cw.enableGzip()
	}

	if cw.gz != nil {
		return cw.gz.Write(b)
	}

	return cw.ResponseWriter.Write(b)
}

func (cw *compressResponseWriter) Flush() {
	if fl, ok := cw.ResponseWriter.(http.Flusher); ok {
		fl.Flush()
	}
}

func Compress(level int, types ...string) Middleware {

	allowed := make(map[string]struct{}, len(types))

	for _, t := range types {
		if t == "" {
			continue
		}
		allowed[strings.ToLower(t)] = struct{}{}
	}

	if level < gzip.HuffmanOnly {
		level = gzip.DefaultCompression
	}

	if level > gzip.BestCompression {
		level = gzip.BestCompression
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) {
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next(w, r, c)
				return
			}

			if r.Method == http.MethodHead {
				next(w, r, c)
				return
			}

			cw := &compressResponseWriter{
				ResponseWriter: w,
				types:          allowed,
				level:          level,
			}
			defer func() {
				if cw.gz != nil {
					_ = cw.gz.Close()
				}
			}()

			next(cw, r, c)

			if cw.gz != nil {
				if fl, ok := w.(http.Flusher); ok {
					fl.Flush()
				}
			}
		}
	}
}

type CORSOptions struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

func matchOrigin(origin string, patterns []string) bool {
	if origin == "" {
		return false
	}
	origin = strings.ToLower(origin)

	for _, p := range patterns {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "*" {
			return true
		}
		if strings.HasSuffix(p, "*") {
			prefix := strings.TrimSuffix(p, "*")
			if strings.HasPrefix(origin, prefix) {
				return true
			}
		} else if origin == p {
			return true
		}
	}
	return false
}

func CORS(opts CORSOptions) Middleware {
	allowedMethods := strings.Join(opts.AllowedMethods, ", ")
	allowedHeaders := strings.Join(opts.AllowedHeaders, ", ")
	exposedHeaders := strings.Join(opts.ExposedHeaders, ", ")
	maxAge := ""
	if opts.MaxAge > 0 {
		maxAge = strconv.Itoa(opts.MaxAge)
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next(w, r, c)
				return
			}

			if !matchOrigin(origin, opts.AllowedOrigins) {
				next(w, r, c)
				return
			}

			h := w.Header()
			h.Set("Vary", "Origin")
			h.Set("Access-Control-Allow-Origin", origin)

			if opts.AllowCredentials {
				h.Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions {
				reqMethod := r.Header.Get("Access-Control-Request-Method")
				if reqMethod == "" {
					w.WriteHeader(http.StatusNoContent)
					return
				}

				if allowedMethods != "" {
					h.Set("Access-Control-Allow-Methods", allowedMethods)
				}
				if allowedHeaders != "" {
					h.Set("Access-Control-Allow-Headers", allowedHeaders)
				}
				if maxAge != "" {
					h.Set("Access-Control-Max-Age", maxAge)
				}

				w.WriteHeader(http.StatusNoContent)
				return
			}

			if exposedHeaders != "" {
				h.Set("Access-Control-Expose-Headers", exposedHeaders)
			}

			next(w, r, c)
		}
	}
}

type ctxKey string

const (
	ContextKeyRequestID ctxKey = "request_id"
	ContextKeyRealIP    ctxKey = "real_ip"
)

var reqIDCounter uint64

func nextRequestID() string {
	id := atomic.AddUint64(&reqIDCounter, 1)
	return strconv.FormatUint(id, 10)
}

func RequestID() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) {
			id := r.Header.Get("X-Request-ID")
			if id == "" {
				id = nextRequestID()
			}

			ctx := context.WithValue(r.Context(), ContextKeyRequestID, id)
			r = r.WithContext(ctx)

			c.Set("request_id", id)

			w.Header().Set("X-Request-ID", id)

			next(w, r, c)
		}
	}
}

func GetRequestID(r *http.Request) string {
	v := r.Context().Value(ContextKeyRequestID)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func RealIP() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) {
			ip := realIPFromRequest(r)

			if ip != "" {
				r.RemoteAddr = ip
			}

			ctx := context.WithValue(r.Context(), ContextKeyRealIP, ip)
			r = r.WithContext(ctx)

			c.Set("real_ip", ip)

			next(w, r, c)
		}
	}
}

func realIPFromRequest(r *http.Request) string {

	if rip := strings.TrimSpace(r.Header.Get("X-Real-IP")); rip != "" {
		return rip
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}

	return r.RemoteAddr
}

func GetRealIP(r *http.Request) string {
	v := r.Context().Value(ContextKeyRealIP)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func NoCache() Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request, c *Context) {
			h := w.Header()
			h.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			h.Set("Pragma", "no-cache")
			h.Set("Expires", "0")

			next(w, r, c)
		}
	}
}

func DefaultCompress() Middleware {
	return Compress(
		gzip.DefaultCompression,
		"text/html",
		"text/plain",
		"text/css",
		"application/javascript",
		"text/javascript",
	)
}

func (r *Router) UseDefaults() {
	r.Use(GetHead())
	r.Use(RequestID())
	r.Use(RealIP())
	r.Use(NoCache())
}
