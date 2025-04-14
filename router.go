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
	Init(fn HandlerFunc)
	Recovery(fn HandlerFunc)
	Static(dir string, replace string)
	EnableProfiling(EnableProfiling string)
	TerminalOutput(terminalOutput bool)
	NotFound(fn HandlerFunc)
}

const serverName = `NetLifeGuru`
const serverVersion = `1.0.1`
const MethodAny = "ANY"

type Listener struct {
	Listen string
	Domain string
	Encode string
}

type Listeners []Listener

type Params map[string]interface{}

type Pattern struct {
	Name          string
	RegexCompiled *regexp.Regexp
}

type RouteEntry struct {
	Methods  string
	Patterns []Pattern
	Handler  HandlerFunc
}

type StaticMap map[string]http.Handler

type Routes map[string][]RouteEntry

type Router struct {
	routes         Routes
	mux            *http.ServeMux
	beforeHandlers []HandlerFunc
	afterHandlers  []HandlerFunc
	init           []HandlerFunc
	recovery       HandlerFunc
	notFound       HandlerFunc
	terminalOutput bool
	proxySegment   string
	staticFiles    StaticMap
}

func NewRouter() IRouter {
	return &Router{
		routes:         make(Routes),
		mux:            http.NewServeMux(),
		beforeHandlers: []HandlerFunc{},
		afterHandlers:  []HandlerFunc{},
		init:           []HandlerFunc{},
		recovery:       nil,
		notFound:       nil,
		terminalOutput: false,
		proxySegment:   "",
		staticFiles:    make(StaticMap),
	}
}

type contextKey string

const routeParamsKey contextKey = "routeParams"

func (r *Router) validateUrlParameters(req *http.Request, rx []Pattern, parameters []string) (bool, Params) {
	if len(parameters) != len(rx) {
		return false, nil
	}

	queryParams := req.URL.Query()
	values := make(Params)

	for i, regExp := range rx {
		matches := regExp.RegexCompiled.FindStringSubmatch(parameters[i])
		if len(matches) == 0 {
			continue
		}

		values[regExp.Name] = matches
		for j, match := range matches {
			if j == 0 {
				queryParams.Set(regExp.Name, match)
			} else {
				queryParams.Add(regExp.Name, match)
			}
		}

		if len(matches) == 2 {
			values[regExp.Name] = matches[1]
		}
	}

	req.URL.RawQuery = queryParams.Encode()

	if len(values) == len(rx) {
		return true, values
	}

	return false, nil
}

func (r *Router) validateMethod(allowedMethods string, requestMethod string) bool {
	methods := strings.Split(allowedMethods, ",")
	for _, method := range methods {
		if method == MethodAny || strings.TrimSpace(method) == requestMethod {
			return true
		}
	}
	return false
}

func (r *Router) getUrlParams(url string) []string {

	regex := regexp.MustCompile("/$")
	match := regex.MatchString(url)
	if match == true {
		url = url + "·"
	}

	u := strings.FieldsFunc(strings.Trim(url, "/"), func(r rune) bool {
		return r == '/'
	})

	if len(u) > 0 {
		return u[1:]
	}

	return nil
}

func (r *Router) createRegexPatterns(url string) ([]Pattern, string) {
	var (
		regexPattern   []Pattern
		parameterName  string
		parameterRegex string
	)

	str := strings.Trim(url, "/")
	parts := strings.Split(str, "/")

	pattern := regexp.MustCompile("<(\\w+):(.*?)>")
	u := parts[:1]
	parts = parts[1:]

	if u[0] == "" {
		u[0] = "/"
	}

	if len(parts) > 0 {
		for _, p := range parts {

			arr := pattern.FindStringSubmatch(p)

			if len(arr) > 2 {
				parameterName = arr[1]
				parameterRegex = arr[2]
			} else {
				parameterName = p
				parameterRegex = p
			}

			_, err := syntax.Parse(parameterRegex, syntax.PerlX)
			if err != nil {
				fmt.Printf("Error: Wrong regular expression %s in URL pattern %s\n", parameterRegex, url)
				os.Exit(3)
			}

			item := Pattern{
				Name:          parameterName,
				RegexCompiled: regexp.MustCompile("^" + parameterRegex + "$"),
			}

			regexPattern = append(regexPattern, item)
		}

		return regexPattern, u[0]
	} else {
		return nil, u[0]
	}
}

func (r *Router) HandleFunc(url string, methods string, fn HandlerFunc) {
	patterns, route := r.createRegexPatterns(url)

	entry := RouteEntry{
		Methods:  methods,
		Patterns: patterns,
		Handler:  fn,
	}

	if r.routes[route] != nil {
		r.routes[route] = append(r.routes[route], entry)
	} else {
		r.routes[route] = []RouteEntry{entry}
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

func (r *Router) Init(fn HandlerFunc) {
	r.init = append(r.init, fn)
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

func (r *Router) runInitHandlers() {
	for _, fn := range r.init {
		fn(nil, nil, &Context{})
	}
}

func (r *Router) getRoute(fullPath string) string {
	path := strings.Trim(fullPath, "/")
	if path == "" {
		return "/"
	}
	route := strings.SplitN(path, "/", 2)[0]

	return route
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
	if message := recover(); message != nil {
		logError(req, message, r.getErrorMessage(message), r.terminalOutput)
		http.Error(w, msg, http.StatusInternalServerError)
	}

	if ctx != nil {
		contextPool.Put(ctx)
	}
}

func (r *Router) handler(w http.ResponseWriter, req *http.Request, route RouteEntry, params Params) {

	ctx := contextPool.Get().(*Context)
	ctx.reset()
	ctx.Params = params

	defer func() {
		if message := recover(); message != nil {
			err := r.getErrorMessage(message)

			if err != nil {
				logError(req, message, err, r.terminalOutput)

				if r.recovery != nil {
					defer r.secondaryRecover(w, req, ctx, "Recovery middleware failed: an error occurred while executing the recovery handler.")
					r.recovery(w, req, ctx)
				} else {
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}
		}

		if ctx != nil {
			contextPool.Put(ctx)
		}
	}()

	for _, before := range r.beforeHandlers {
		before(w, req, ctx)
	}

	if r.terminalOutput {
		start := time.Now()
		route.Handler(w, req, ctx)
		logRequest(req, start)
	} else {
		route.Handler(w, req, ctx)
	}

	for _, after := range r.afterHandlers {
		after(w, req, ctx)
	}
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {

	for prefix, handler := range r.staticFiles {
		if strings.HasPrefix(req.URL.Path, prefix) {
			handler.ServeHTTP(w, req)
			return
		}
	}

	path := req.URL.Path

	if r.proxySegment != "" {
		path = strings.TrimPrefix(path, r.proxySegment)
	}

	notFound := true

	if routes, ok := r.routes[r.getRoute(path)]; ok {

		parameters := r.getUrlParams(path)
		requestMethod := req.Method

		for _, route := range routes {
			if r.validateMethod(route.Methods, requestMethod) {
				if ok, params := r.validateUrlParameters(req, route.Patterns, parameters); ok {

					notFound = false

					r.handler(w, req, route, params)

					break
				}
			}
		}
	}

	if notFound {
		if r.notFound != nil {
			ctx := contextPool.Get().(*Context)
			ctx.reset()

			defer r.secondaryRecover(w, req, ctx, "Recovery middleware failed. An error occurred while handling the custom error page!")

			r.notFound(w, req, ctx)

		} else {
			http.NotFound(w, req)
		}
	}
}

func (r *Router) MultiListenAndServe(listeners Listeners) {

	r.runInitHandlers()

	var wg sync.WaitGroup

	for _, proxy := range listeners {
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

			err := http.ListenAndServe(listen, r)

			if err != nil {
				log.Fatal(err)
			}

		}(port, domain)
	}

	wg.Wait()
}

func (r *Router) ListenAndServe(port int) {

	r.runInitHandlers()

	if r.terminalOutput {
		printServerInfo(serverName, serverVersion, port)
	}

	err := http.ListenAndServe(fmt.Sprintf("localhost:%d", port), r)

	if err != nil {
		log.Fatal(err)
	}
}
