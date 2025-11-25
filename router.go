package router

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"regexp"
	"regexp/syntax"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type headWriter struct{ http.ResponseWriter }

type HandlerFunc func(http.ResponseWriter, *http.Request, *Context)

type IRouter interface {
	MultiListenAndServe(listeners Listeners)
	ListenAndServe(port int)
	HandleFunc(url string, methods string, fn HandlerFunc)
	Prefix(segment string)
	Use(m Middleware)
	Recovery(fn HandlerFunc)
	Static(dir string, replace string)
	EnableProfiling(EnableProfiling string)
	TerminalOutput(terminalOutput bool)
	NotFound(fn HandlerFunc)
	Ready()
}

const serverName = `NetLifeGuru`
const serverVersion = `v1.0.7`

type Listener struct {
	Listen string
	Domain string
	Encode string
}

type Listeners []Listener

type Pattern struct {
	Slug          string
	Type          int
	RegexCompiled *regexp.Regexp
	Fn            MatchFunc
}

type Group struct {
	Slug string
}

type StaticSegments struct {
	Slug  string
	Order int
}

var notFound = []byte("404 page not found")

type StaticMap map[string]http.Handler

type RouteEntry struct {
	Route      string
	Patterns   []Pattern
	Handler    HandlerFunc
	Bitmask    int
	Validation bool
}

type StaticRoutes map[string]RouteEntry

type Router struct {
	radixRoot      *RadixNode
	staticRoutes   StaticRoutes
	mux            *http.ServeMux
	recovery       HandlerFunc
	notFound       HandlerFunc
	terminalOutput bool
	prefixSegment  string
	staticFiles    StaticMap
	ready          atomic.Bool
	middlewares    []Middleware
}

func NewRouter() IRouter {
	r := &Router{
		radixRoot:      &RadixNode{},
		staticRoutes:   make(StaticRoutes),
		mux:            http.NewServeMux(),
		middlewares:    []Middleware{},
		recovery:       nil,
		notFound:       nil,
		terminalOutput: false,
		prefixSegment:  "",
		staticFiles:    make(StaticMap),
	}

	r.ready.Store(true)

	return r
}

func (r *Router) SetReady(ready bool) {
	r.ready.Store(ready)
}

func (r *Router) IsReady() bool {
	return r.ready.Load()
}

type contextKey string

const (
	_STRING = iota + 1
	_PATTERN
	_MATCH
	_SUBMATCH
)

func (hw headWriter) Write(b []byte) (int, error) { return len(b), nil }

func (r *Router) removeWrapper(s string, start string, end string) string {
	var str string

	if len(s) >= 2 && s[0] == start[0] && s[len(s)-1] == end[0] {
		str = s[1 : len(s)-1]
	} else {
		str = s
	}

	return str
}

func (r *Router) findPatterns(str string) MatchFunc {
	possibleRegExpPattern := r.removeWrapper(str, "(", ")")

	if pattern, ok := PatternMatchers[possibleRegExpPattern]; ok {
		// function
		return pattern
	} else if pattern, ok := FunctionMatchers[possibleRegExpPattern]; ok {
		// function
		return pattern
	}

	return nil
}

func (r *Router) parseSlug(isStatic, reqValidation bool, s, url string) (Pattern, bool, bool) {
	var slugPattern Pattern

	if (len(s) >= 2 && (s[0] == '<' && s[len(s)-1] == '>')) || (len(s) >= 2 && (s[0] == '{' && s[len(s)-1] == '}')) {
		isStatic = false
		str := s[1 : len(s)-1]

		name := str
		pt := "any"

		if i := strings.IndexByte(str, ':'); i != -1 {
			name = str[:i]
			if i+1 < len(str) {
				pt = str[i+1:]
			} else {
				pt = ""
			}
		}

		slugPattern.Slug = name
		if pt == "" {
			fmt.Printf("Error: Empty pattern in URL segment %q (route %s)\n", s, url)
			os.Exit(3)
		}
		if pt != "any" {
			reqValidation = true
		}

		if c := countCaptureGroups(pt); c > 0 {
			//FindAllStringSubmatch
			if _, err := syntax.Parse(pt, syntax.PerlX); err != nil {
				fmt.Printf("Error: Wrong regular expression %q in URL pattern %s\n", pt, url)
				os.Exit(3)
			}
			slugPattern.RegexCompiled = regexp.MustCompile("^" + pt + "$")
			slugPattern.Type = _SUBMATCH
		} else {
			//Match
			if fn := r.findPatterns(pt); fn != nil {
				slugPattern.Fn = fn
				slugPattern.Type = _PATTERN
			} else {
				if _, err := syntax.Parse(pt, syntax.PerlX); err != nil {
					fmt.Printf("Error: Wrong regular expression %q in URL pattern %s\n", pt, url)
					os.Exit(3)
				}
				slugPattern.RegexCompiled = regexp.MustCompile("^" + pt + "$")
				slugPattern.Type = _MATCH
			}
		}
		return slugPattern, isStatic, reqValidation
	}

	slugPattern.Slug = s
	slugPattern.Type = _STRING
	return slugPattern, isStatic, reqValidation
}

func splitPath(path string) []string {
	var segments []string
	start := -1

	for i := 0; i < len(path); i++ {
		if path[i] != '/' {
			if start == -1 {
				start = i
			}
		} else if start != -1 {
			segments = append(segments, path[start:i])
			start = -1
		}
	}

	if start != -1 {
		segments = append(segments, path[start:])
	}

	return segments
}

func (r *Router) preparePattern(url string) (string, []Pattern, bool, bool, string) {
	var (
		first         string
		patterns      []Pattern
		isStatic      = true
		reqValidation = false
	)

	segments := splitPath(url)
	parts := make([]string, 0, len(segments))

	for _, seg := range segments {
		if seg == "" {
			continue
		}

		if first == "" {
			first = seg
		}

		p, st, rv := r.parseSlug(isStatic, reqValidation, seg, url)

		slugPart := p.Slug
		if p.Type != _STRING {
			slugPart = "*"
		}

		parts = append(parts, slugPart)

		isStatic = st
		reqValidation = rv
		patterns = append(patterns, p)
	}

	radixURL := "/" + strings.Join(parts, "/")

	return first, patterns, isStatic, reqValidation, radixURL
}

func (r *Router) HandleFunc(url string, methods string, fn HandlerFunc) {
	_, patterns, isStatic, reqValidation, radixURL := r.preparePattern(url)

	entry := RouteEntry{
		Route:      url,
		Patterns:   patterns,
		Handler:    fn,
		Bitmask:    r.MethodsToBitmask(methods),
		Validation: reqValidation,
	}

	if entry.Bitmask < 0 {
		log.Fatalf("Invalid HTTP method in route %q methods %q", url, methods)
	}

	if isStatic {
		r.staticRoutes[url] = entry
	}

	r.insertNode(radixURL, entry)

}

func (r *Router) Static(dir string, replace string) {
	if !strings.HasSuffix(replace, "/") {
		replace += "/"
	}

	err := ensureDirectory(fmt.Sprintf("./%s", dir))

	if err != nil {
		log.Printf("Failed to create directory %s", err)
	}

	if r.staticFiles == nil {
		r.staticFiles = make(map[string]http.Handler)
	}

	fs := http.FileServer(http.Dir("./" + dir))
	r.staticFiles[replace] = http.StripPrefix(replace, fs)

	faviconPath := fmt.Sprintf("./%s/favicon.ico", dir)
	if _, err := os.Stat(faviconPath); err == nil {
		r.HandleFunc("/favicon.ico", "GET", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
			http.ServeFile(w, req, faviconPath)
		})
	}
}

func (r *Router) EnableProfiling(profilingServer string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	go func() {
		log.Printf("[pprof] Profiling enabled at http://%s/debug/pprof/", profilingServer)
		if err := http.ListenAndServe(profilingServer, mux); err != nil {
			log.Printf("[pprof] Error: %v", err)
		}
	}()
}

func (r *Router) Prefix(segment string) {
	if segment == "" || segment == "/" {
		r.prefixSegment = ""
		return
	}

	if !strings.HasPrefix(segment, "/") {
		segment = "/" + segment
	}

	for len(segment) > 1 && strings.HasSuffix(segment, "/") {
		segment = segment[:len(segment)-1]
	}

	r.prefixSegment = segment
}

func (r *Router) Recovery(fn HandlerFunc) {
	r.recovery = fn
}

func (r *Router) NotFound(fn HandlerFunc) {
	r.notFound = fn
}

func (r *Router) TerminalOutput(terminal bool) {
	r.terminalOutput = terminal
}

func (r *Router) getErrorMessage(message any) error {
	var err error

	switch v := message.(type) {
	case error:
		err = v
	case string:
		err = fmt.Errorf("panic occurred: %s", v)
	default:
		err = fmt.Errorf("panic occurred with unknown type: %v", v)
	}

	return err
}

func (r *Router) secondaryRecover(w http.ResponseWriter, req *http.Request, ctx *Context, msg string) {
	func() {
		if message := recover(); message != nil {
			logError(req, message, r.getErrorMessage(message), r.terminalOutput)
			http.Error(w, msg, http.StatusInternalServerError)
		}

		if ctx != nil {
			contextPool.Put(ctx)
		}
	}()
}

func (r *Router) Run(w http.ResponseWriter, req *http.Request, handler HandlerFunc, ctx *Context) {
	if r.terminalOutput {
		start := time.Now()
		handler(w, req, ctx)
		logRequest(req, start)
	} else {
		handler(w, req, ctx)
	}
}

func (r *Router) write405(w http.ResponseWriter, mask int) {
	allow := r.maskToAllowHeader(mask)
	if allow != "" {
		w.Header().Set("Allow", allow)
	}
	w.WriteHeader(http.StatusMethodNotAllowed)
	_, _ = w.Write([]byte("405 method not allowed"))
}

func (r *Router) maskToAllowHeader(mask int) string {
	have := map[string]bool{}
	if mask&GET != 0 {
		have["GET"], have["HEAD"] = true, true
	}
	if mask&POST != 0 {
		have["POST"] = true
	}
	if mask&PUT != 0 {
		have["PUT"] = true
	}
	if mask&DELETE != 0 {
		have["DELETE"] = true
	}
	if mask&PATCH != 0 {
		have["PATCH"] = true
	}

	have["OPTIONS"] = true
	order := []string{"GET", "HEAD", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"}
	out := make([]string, 0, len(order))
	for _, m := range order {
		if have[m] {
			out = append(out, m)
		}
	}
	return strings.Join(out, ", ")
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	ctx := GetContext()

	defer func() {
		if m := recover(); m != nil {
			err := r.getErrorMessage(m)
			if err != nil {
				logError(req, m, err, r.terminalOutput)
				if r.recovery != nil {
					defer r.secondaryRecover(w, req, ctx, "Recovery middleware failed: an error occurred while executing the recovery handler.")
					r.recovery(w, req, ctx)
				} else {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}
		}

		PutContext(ctx)
	}()

	var foundPath bool
	var allowedMask int

	p := req.URL.Path
	t := r.staticRoutes[p]

	if t.Route != "" {

		method := r.getBitmaskIndex(req.Method)

		if t.Bitmask&method != 0 {
			ctx.Params = ctx.Params[:0]
			ctx.paramMap = nil
			ctx.Entries = ctx.Entries[:0]

			handler := r.wrap(t.Handler)

			r.Run(w, req, handler, ctx)
			return
		}

		r.write405(w, t.Bitmask)
		return

	} else if ok := r.searchAll(p, ctx); ok {

		start := -1

		for j := 0; j < len(p); j++ {
			if p[j] != '/' {
				if start == -1 {
					start = j
				}
			} else if start != -1 {
				ctx.Segments = append(ctx.Segments, Seg{p[start:j]})
				start = -1
			}
		}

		foundPath = false
		allowedMask = 0

		if start != -1 {

			ctx.Segments = append(ctx.Segments, Seg{p[start:]})

			bitmask := r.getBitmaskIndex(req.Method)

		outer:
			for _, entry := range ctx.Entries {
				foundPath = true
				allowedMask |= entry.Bitmask

				if entry.Bitmask&bitmask == 0 {
					continue
				}

				if entry.Validation {
					for depth, p := range entry.Patterns {
						segment := ctx.Segments[depth].Value
						if p.Type != _STRING {
							switch p.Type {
							case _MATCH:
								if !p.RegexCompiled.MatchString(segment) {
									continue outer
								}
							case _PATTERN:
								if !p.Fn(segment) {
									continue outer
								}
							case _SUBMATCH:
								if len(p.RegexCompiled.FindStringSubmatch(segment)) == 0 {
									continue outer
								}
							}
						}
					}
				}

				ctx.Params = ctx.Params[:0]
				ctx.paramMap = nil
				ctx.Entries = ctx.Entries[:0]
				ctx.Entries = append(ctx.Entries, entry)

				r.Run(w, req, entry.Handler, ctx)
				return
			}
		}
	}

	if foundPath {
		r.write405(w, allowedMask)
		return
	}

	if r.notFound != nil {
		r.notFound(w, req, ctx)
	} else {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(notFound)
	}
}

func (r *Router) Handler() http.HandlerFunc {
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		path := req.URL.Path
		best := ""
		var bestH http.Handler
		for prefix, h := range r.staticFiles {
			if strings.HasPrefix(path, prefix) && len(prefix) > len(best) {
				best, bestH = prefix, h
			}
		}
		if bestH != nil {
			bestH.ServeHTTP(w, req)
			return
		}

		if r.prefixSegment != "" {
			seg := r.prefixSegment
			segLen := len(seg)
			if path == seg {
				req.URL.Path = "/"
			} else if len(path) > segLen &&
				path[:segLen] == seg &&
				path[segLen] == '/' {

				req.URL.Path = path[segLen:]
			}
		}

		r.ServeHTTP(w, req)
	})

	return handler
}

func (r *Router) Ready() {
	r.HandleFunc("/ready", "GET", func(w http.ResponseWriter, req *http.Request, ctx *Context) {
		if r.IsReady() {
			w.WriteHeader(http.StatusOK)
			_, err := w.Write([]byte("ok"))
			if err != nil {
				return
			}
		} else {
			http.Error(w, "shutting down", http.StatusServiceUnavailable)
		}
	})
}

func (r *Router) MultiListenAndServe(listeners Listeners) {
	debug.SetGCPercent(300)

	workers := runtime.NumCPU()
	runtime.GOMAXPROCS(workers)

	if r.terminalOutput {
		Log("INFO", "Using %d CPU core%s", workers, map[bool]string{true: "s", false: ""}[workers != 1])
	}

	var (
		wg      sync.WaitGroup
		servers []*http.Server
		mu      sync.Mutex
	)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	for _, ln := range listeners {
		listenAddr := ln.Listen

		_, portStr, err := net.SplitHostPort(listenAddr)
		if err != nil {
			log.Fatalf("Invalid listen address %s: %v", listenAddr, err)
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			log.Fatalf("Invalid port format for %s: %v", listenAddr, err)
		}

		if r.terminalOutput {
			printServerInfo(serverName, serverVersion, port)
		}

		useReusePort := runtime.GOOS != "windows"
		var reuseErr error

		var probe net.Listener
		if useReusePort {
			lc := net.ListenConfig{
				Control: func(network, address string, c syscall.RawConn) error {
					var sockErr error
					if err := c.Control(func(fd uintptr) {
						_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
						if e := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); e != nil {
							sockErr = e
						}
					}); err != nil {
						return err
					}
					return sockErr
				},
			}
			probe, reuseErr = lc.Listen(context.Background(), "tcp", listenAddr)
			if reuseErr == nil {
				_ = probe.Close()
			}
		}

		if useReusePort && reuseErr == nil {
			for i := 0; i < workers; i++ {
				wg.Add(1)
				go func(addr string) {
					defer wg.Done()

					lc := net.ListenConfig{
						Control: func(network, address string, c syscall.RawConn) error {
							var sockErr error
							if err := c.Control(func(fd uintptr) {
								_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
								if e := unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1); e != nil {
									sockErr = e
								}
							}); err != nil {
								return err
							}
							return sockErr
						},
					}

					raw, err := lc.Listen(context.Background(), "tcp", addr)
					if err != nil {
						if r.terminalOutput {
							Log("ERROR", "REUSEPORT listen failed on %s: %v", addr, err)
						}
						return
					}

					listener := raw

					server := &http.Server{
						Handler:           r.Handler(),
						ReadTimeout:       5 * time.Second,
						WriteTimeout:      10 * time.Second,
						IdleTimeout:       120 * time.Second,
						ReadHeaderTimeout: 2 * time.Second,
					}

					mu.Lock()
					servers = append(servers, server)
					mu.Unlock()

					if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
						if r.terminalOutput {
							Log("ERROR", "Server error on %s: %v", addr, err)
						}
					}
				}(listenAddr)
			}
		} else {

			if r.terminalOutput && reuseErr != nil {
				Log("WARN", "REUSEPORT unavailable on %s: %v; falling back to single listener", listenAddr, reuseErr)
			}

			wg.Add(1)
			go func(addr string) {
				defer wg.Done()

				l, err := net.Listen("tcp", addr)
				if err != nil {
					log.Fatalf("Failed to listen on %s: %v", addr, err)
				}

				server := &http.Server{
					Handler:           r.Handler(),
					ReadTimeout:       5 * time.Second,
					WriteTimeout:      10 * time.Second,
					IdleTimeout:       120 * time.Second,
					ReadHeaderTimeout: 2 * time.Second,
				}

				mu.Lock()
				servers = append(servers, server)
				mu.Unlock()

				if err := server.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
					if r.terminalOutput {
						Log("ERROR", "Server error on %s: %v", addr, err)
					}
				}
			}(listenAddr)
		}
	}

	<-stop
	r.SetReady(false)
	if r.terminalOutput {
		Log("INFO", "Shutdown signal received. Shutting down servers...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	mu.Lock()
	for _, srv := range servers {
		go func(s *http.Server) {
			if err := s.Shutdown(shutdownCtx); err != nil && r.terminalOutput {
				Log("WARN", "Server shutdown error: %v", err)
			}
		}(srv)
	}
	mu.Unlock()

	wg.Wait()

	if r.terminalOutput {
		Log("INFO", "All servers shut down gracefully.")
	}
}

func (r *Router) ListenAndServe(port int) {

	listen := fmt.Sprintf("localhost:%d", port)
	r.MultiListenAndServe(Listeners{
		{Listen: listen, Domain: listen},
	})
}
