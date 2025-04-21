package router

import (
	"fmt"
	"log"
	"net/http"
	"net/http/pprof"
	"os"
	"regexp"
	"regexp/syntax"
	"strconv"
	"strings"
	"sync"
	"time"
)

type HandlerFunc func(http.ResponseWriter, *http.Request, *Context)

type IRouter interface {
	MultiListenAndServe(listeners Listeners)
	ListenAndServe(port int)
	HandleFunc(url string, methods string, fn HandlerFunc)
	Proxy(segment string)
	Before(fn HandlerFunc)
	After(fn HandlerFunc)
	Recovery(fn HandlerFunc)
	Static(dir string, replace string)
	EnableProfiling(EnableProfiling string)
	TerminalOutput(terminalOutput bool)
	NotFound(fn HandlerFunc)
}

const serverName = `NetLifeGuru`
const serverVersion = `v1.0.3`
const MethodAny = "ANY"

type Listener struct {
	Listen string
	Domain string
	Encode string
}

type Listeners []Listener

type Pattern struct {
	Name          string
	Slug          string
	Type          int
	RegexCompiled *regexp.Regexp
	Fn            MatchFunc
}

type StaticMap map[string]http.Handler

type RouteEntry struct {
	Route    string
	Patterns []Pattern
	Handler  HandlerFunc
	Bitmask  int
}

type RouteIndex map[int]map[int]RouteEntry
type Routes map[string]RouteIndex
type StaticRoutes map[string]RouteEntry

type Router struct {
	routes         Routes
	staticRoutes   StaticRoutes
	mux            *http.ServeMux
	beforeHandlers []HandlerFunc
	afterHandlers  []HandlerFunc
	recovery       HandlerFunc
	notFound       HandlerFunc
	terminalOutput bool
	proxySegment   string
	staticFiles    StaticMap
	regex          *regexp.Regexp
}

func NewRouter() IRouter {
	return &Router{
		routes:         make(Routes),
		staticRoutes:   make(StaticRoutes),
		mux:            http.NewServeMux(),
		beforeHandlers: []HandlerFunc{},
		afterHandlers:  []HandlerFunc{},
		recovery:       nil,
		notFound:       nil,
		terminalOutput: false,
		proxySegment:   "",
		staticFiles:    make(StaticMap),
		regex:          regexp.MustCompile(`(([\p{L}0-9\-._~%]+)/)|(<(.*?)>/)|({(.*?)}/)`),
	}
}

type contextKey string

const routeParamsKey contextKey = "routeParams"

func (r *Router) validateMethod(allowedMethods string, requestMethod string) bool {
	methods := strings.Split(allowedMethods, ",")
	for i := 0; i < len(methods); i++ {
		if methods[i] == MethodAny || strings.TrimSpace(methods[i]) == requestMethod {
			return true
		}
	}
	return false
}

const (
	_STRING = iota + 1
	_PATTERN
	_MATCH
	_SUBMATCH
)

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

func (r *Router) parseSlug(isStatic bool, s string, url string) (Pattern, bool) {

	var str string
	var slugPattern Pattern

	if (len(s) >= 2 && s[0] == '<' && s[len(s)-1] == '>') || (len(s) >= 2 && s[0] == '{' && s[len(s)-1] == '}') {
		isStatic = false
		str = s[1 : len(s)-1]

		pt := ""
		for i := 0; i < len(s); i++ {
			if s[i] == ':' {
				slugPattern.Slug = str[0 : i-1]
				pt = str[i:]
				break
			} else {
				pt = "any"
			}
		}

		if c := countCaptureGroups(pt); c > 0 {
			//FindAllStringSubmatch
			if _, err := syntax.Parse(pt, syntax.PerlX); err != nil {
				fmt.Printf("Error: Wrong regular expression %s in URL pattern %s\n", pt, url)
				os.Exit(3)
			}
			slugPattern.RegexCompiled = regexp.MustCompile("^" + pt + "$")
			slugPattern.Type = _SUBMATCH
		} else {
			//Match
			slugPattern.Fn = r.findPatterns(pt)

			if slugPattern.Fn == nil {
				if _, err := syntax.Parse(pt, syntax.PerlX); err != nil {
					fmt.Printf("Error: Wrong regular expression %s in URL pattern %s\n", pt, url)
					os.Exit(3)
				}
				slugPattern.RegexCompiled = regexp.MustCompile("^" + pt + "$")
				slugPattern.Type = _MATCH
			} else {
				slugPattern.Type = _PATTERN
			}
		}

	} else {
		slugPattern.Slug = s
		slugPattern.Type = _STRING
	}

	return slugPattern, isStatic
}

func (r *Router) preparePattern(url string) (string, []Pattern, bool) {
	parts := make([]string, 0, 5)

	if url != "/" {
		matches := r.regex.FindAllStringSubmatch(url+`/`, -1)
		for i := 0; i < len(matches); i++ {
			part := matches[i][0][:len(matches[i][0])-1]
			parts = append(parts, part)
		}
	} else {
		return "", nil, true
	}

	first := parts[0]

	var patterns []Pattern

	var isStatic = true
	for i := 0; i < len(parts); i++ {
		p, st := r.parseSlug(isStatic, parts[i], url)
		isStatic = st
		patterns = append(patterns, p)
	}

	return first, patterns, isStatic
}

func (r *Router) HandleFunc(url string, methods string, fn HandlerFunc) {
	route, patterns, isStatic := r.preparePattern(url)

	entry := RouteEntry{
		Route:    url,
		Patterns: patterns,
		Handler:  fn,
		Bitmask:  r.bitmask(url, methods),
	}

	if isStatic {

		r.staticRoutes[url] = entry

	} else {

		if _, exists := r.routes[route]; !exists {
			r.routes[route] = make(RouteIndex)
		}

		depth := len(patterns)

		if _, exists := r.routes[route][depth]; !exists {
			r.routes[route][depth] = make(map[int]RouteEntry)
		}

		l := len(r.routes[route][depth])

		r.routes[route][depth][l] = entry
	}
}

func (r *Router) Static(dir string, replace string) {
	if !strings.HasSuffix(replace, "/") {
		replace += "/"
	}

	err := directoryExists(fmt.Sprintf("./%s", dir))
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

func (r *Router) Proxy(segment string) {
	r.proxySegment = segment
}

func (r *Router) Before(fn HandlerFunc) {
	r.beforeHandlers = append(r.beforeHandlers, fn)
}

func (r *Router) After(fn HandlerFunc) {
	r.afterHandlers = append(r.afterHandlers, fn)
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

	for i := 0; i < len(r.beforeHandlers); i++ {
		r.beforeHandlers[i](w, req, ctx)
	}

	if r.terminalOutput {
		start := time.Now()
		handler(w, req, ctx)
		logRequest(req, start)
	} else {
		handler(w, req, ctx)
	}

	for i := 0; i < len(r.afterHandlers); i++ {
		r.afterHandlers[i](w, req, ctx)
	}
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

	p := req.URL.Path
	t := r.staticRoutes[p]

	if t.Route != "" {
		r.Run(w, req, t.Handler, ctx)
	} else {
		var found bool
		var segment string

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

		if start != -1 {

			ctx.Segments = append(ctx.Segments, Seg{p[start:]})

			if handlers, ok := r.routes[ctx.Segments[0].Value][len(ctx.Segments)]; ok {

				method := r.getBitmaskIndex(req.Method)

				for i := 0; i < len(handlers); i++ {
					entry := handlers[i]
					if entry.Bitmask&method == 0 {
						continue
					}

					handler := entry.Handler
					patterns := entry.Patterns
					valid := true
					for z := 1; z < len(ctx.Segments); z++ {
						segment = ctx.Segments[z].Value
						p := patterns[z]

						switch p.Type {
						case _STRING:
							if segment != p.Slug {
								valid = false
							}
						case _MATCH:
							if !p.RegexCompiled.MatchString(segment) {
								valid = false
							}
						case _PATTERN:
							if !p.Fn(segment) {
								valid = false
							}
						case _SUBMATCH:
							if len(p.RegexCompiled.FindStringSubmatch(segment)) == 0 {
								valid = false
							}
						}
						if valid {
							ctx.Params = append(ctx.Params, Par{p.Slug, segment})
						}
					}

					if valid {
						found = true
						r.Run(w, req, handler, ctx)
					}
				}
			}
		}

		if !found {
			if r.notFound != nil {
				r.notFound(w, req, ctx)
			} else {
				http.NotFound(w, req)
			}
		}
	}
}

func (r *Router) Handler() http.HandlerFunc {
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		for prefix, staticHandler := range r.staticFiles {
			path := req.URL.Path
			pl := len(prefix)
			if len(path) >= pl && path[:pl] == prefix {
				staticHandler.ServeHTTP(w, req)
				return
			}
		}

		if r.proxySegment != "" {
			path := req.URL.Path
			seg := r.proxySegment
			segLen := len(seg)
			if len(path) >= segLen && path[:segLen] == seg {
				req.URL.Path = path[segLen:]
				if req.URL.Path == "" {
					req.URL.Path = "/"
				}
			}
		}

		r.ServeHTTP(w, req)
	})

	return handler
}

func (r *Router) MultiListenAndServe(listeners Listeners) {

	var wg sync.WaitGroup

	for i := 0; i < len(listeners); i++ {
		proxy := listeners[i]
		var (
			listen  = proxy.Listen
			portStr = strings.Split(listen, ":")[1]
			domain  = proxy.Domain
		)

		port, err := strconv.Atoi(portStr)
		if err != nil {
			log.Fatal(err)
		}

		wg.Add(1)
		go func(p int, t string) {
			defer wg.Done()

			if r.terminalOutput {
				printServerInfo(serverName, serverVersion, port)
			}

			err := http.ListenAndServe(listen, r.Handler())

			if err != nil {
				log.Fatal(err)
			}

		}(port, domain)
	}

	wg.Wait()
}

func (r *Router) ListenAndServe(port int) {

	if r.terminalOutput {
		printServerInfo(serverName, serverVersion, port)
	}

	err := http.ListenAndServe(fmt.Sprintf("localhost:%d", port), r.Handler())

	if err != nil {
		log.Fatal(err)
	}
}
