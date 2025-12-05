[![Go Version](https://img.shields.io/badge/go-%3E=1.19-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-brightgreen)](LICENSE)

# 🚀 NetLifeGuru Router v1.0.8

A clean, performant and idiomatic HTTP router & microframework for Go – built for modern backend APIs, apps, and
full-stack setups.

Includes built-in support for middleware, context, parameterized routing, rate limiting, profiling, static assets, and
multi-port servers.

---

## ✨ Features

- 🩺 Healthcheck endpoint `/ready` (enable with `r.Ready()`)
- 🌐 Custom routing with regex parameters
- 🧩 Recovery, Middleware, Route Grouping
- 🗂 Request context with per-request storage (pooled)
- 🧾 Accessing Query & Form Data
- 🛡 Simple RateLimit guard (per client IP + method + path)
- 📊 Built-in pprof profiling
- 📁 Static file serving (with favicon.ico support)
- 🎨 Terminal logging with color output
- 🔥 Panic recovery with log-to-file support
- 💥 Custom Recovery Handler
- 🧱 Custom 404 Page (NotFound Handler)
- 🕸️ Multi-server support (perfect for microservices)
- 🧾 Panic logging with daily rotation
- 🖥️ Terminal Logging

---

## 📦 Installation

```bash
go get github.com/NetLifeGuru/router@main
```

---

## 💡 Basic Usage

### 🩺 Healthcheck
Enable a simple readiness endpoint:

Call `r.Ready()` to register `GET /ready`, which returns **200 "ok"** while running and **503 "shutting down"** during graceful shutdown.

```go
    r.Ready()
    // GET /ready → 200 "ok" (while running), 503 "shutting down" during graceful shutdown
```

### 🔄 Single-Server Setup

```go
package main

import (
	"fmt"
	"net/http"
	"github.com/NetLifeGuru/router"
)

func main() {
	r := router.NewRouter()

	r.HandleFunc("/", "GET", func(w http.ResponseWriter, req *http.Request, ctx *router.Context) {
		fmt.Fprint(w, "Welcome to NetLifeGuru Router!")
	})

	r.ListenAndServe(8080)
}
```

---

### 🕸️ Multi-Server Setup (Microservices Ready)

For more advanced setups (e.g. microservices), you can run multiple servers using `MultiListenAndServe`.

```go
r := router.NewRouter()

r.HandleFunc("/", "GET", func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    w.WriteHeader(http.StatusOK)
})

listeners := router.Listeners{
    {Listen: "localhost:8000", Domain: "localhost:8000"},
    {Listen: "localhost:8001", Domain: "localhost:8001"},
}

r.MultiListenAndServe(listeners)
```

Each listener listens on its own port and serves the same routes. Ideal for testing subdomains or simulating
multi-service environments.

One of the key advantages of multi-server support is the ability to run a single application instance on multiple
domains or ports simultaneously — ideal for multi-tenant architectures, localized services, or parallel dev/staging
environments.


---

## Route Grouping
Route groups allow you to organize related routes under a shared URL prefix.
This helps keep your API structure clean and scalable while avoiding repetitive path definitions.
They also support group-level middleware, meaning middleware declared on a group runs only for routes inside that group.

### Groups help you:
 - keep a clean and predictable API structure,
 - avoid repeating common prefixes like /api/v1/...,
 - attach middleware only to a specific section of your API,
 - build modular and isolated API modules.

### Basic Usage
```go
api := r.Group("/api")

api.HandleFunc("/users", "GET", getUsersHandler)
api.HandleFunc("/posts", "POST", createPostHandler)
```

This registers the following endpoints:
 - `/api/users`
 - `/api/posts`

### Group-Level Middleware
Route groups support scoped middleware via:

```go
api.Use(middleware)
```
Middleware added this way runs only for routes inside this group, and it executes **after global middleware (closer to the handler)**.

### Example

```go
v1 := r.Group("/v1/api")

// Group-level middleware
v1.Use(func(next router.HandlerFunc) router.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
		fmt.Println("before v1")
		next(w, r, ctx)
		fmt.Println("after v1")
	}
})

// Group route
v1.HandleFunc("/users/{id}", "GET", func(w http.ResponseWriter, req *http.Request, ctx *router.Context) {
	router.JSON(w, 200, map[string]any{
		"users": []string{"A", "B"},
	})
})
```

Output when calling `/v1/api/users/123`:
```go
before v1
(after handler output)
after v1
```

### Why Use Route Groups?
 - Cleaner and more maintainable API structure
 - Avoid repeating common prefixes (e.g., /api/v1/...)
 - Easier to organize large projects
 - Allows future extension such as group-level middleware


## 🔐 Middleware

- Introduced unified `Use(middleware)` function.
- Middleware signature: `func(HandlerFunc) HandlerFunc`.
- Built-in middleware:
    - `AllowContentType`
    - `ContentCharset`
    - `CleanPath`
    - `Compress`
    - `CORS`
    - `RequestID`
    - `RealIP`
    - `NoCache`
    - `DefaultCompress`
- Removed: `Before` and `After` middleware.

```go
r := router.NewRouter()

r.Static("files/public", "/assets")

r.Use(func(next router.HandlerFunc) router.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
        next(w, r, ctx)
    }
})

r.Recovery(func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    http.Error(w, "Unexpected error occurred", http.StatusInternalServerError)
})
```

### Custom Middleware with r.Use

This router uses a classic middleware chain similar to Express, Chi, and many other frameworks.
A middleware is simply a function that takes a handler and returns a new handler:

**What this achieves**

 - You can run logic before the handler (logging, auth, rate limits…)
 - And logic after the handler (cleanup, metrics…)
 - Middleware runs in order of registration
 - Each middleware wraps the next one, forming a chain
 - Example behavior:

```
Use(M1) → Use(M2) → Handler

Call stack:
M1(before)
  M2(before)
    Handler
  M2(after)
M1(after)
```

Global middleware wrap everything, group middleware run closer to the handler.

*This gives full control over request flow without needing dedicated Before or After hooks.*

### UseDefaults()

UseDefaults() registers a sensible default middleware chain:

 - auto-map `HEAD` → `GET`
 - injects `X-Request-ID`
 - resolves real client IP
 - disables caching via `Cache-Control` headers

```go
r.UseDefaults()
```

### AllowContentType

```go
r.Use(router.AllowContentType("application/json", "text/xml"))
```

Ensures the request ``Content-Type`` header matches one of the allowed types.
If the request has a different content type, the router returns:

 - 415 Unsupported Media Type
 - and stops the middleware chain (ctx.Abort())
 
Useful when an endpoint accepts only JSON, XML, form data, etc.

### CleanPath

```go
r.Use(router.CleanPath())
```

Normalizes the request URL path using `path.Clean()`.

Examples:
 - `/api//users//123/` → `/api/users/123`

### ContentCharset
```go
allowedCharsets := []string{"UTF-8", "Latin-1", ""}
r.Use(router.ContentCharset(allowedCharsets...))
```

Validates the `charset` parameter of the `Content-Type` header.

```yaml
Content-Type: application/json; charset=UTF-8
```

If the charset is not in the allowed list → router responds with
`415 Unsupported Media Type` and stops processing.

Passing an empty string in the allowed list lets requests without a charset pass.



### DefaultCompress
```go
r.Use(router.DefaultCompress())
```

Enables `gzip compression` for responses when the client sends:

```yaml
Accept-Encoding: gzip
```

Compresses common MIME types:
 - text/html
 - text/plain
 - text/css
 - application/javascript
 - text/javascript
Automatically handles:
 - Content-Length removal
 - gzip writer lifecycle
 - skip for non-2xx responses
 - skip for HEAD method


### RequestID
```go
r.Use(router.RequestID())
```

Injects a unique request ID into:

 - `X-Request-ID` response header
 - request context (`ctx.Set("request_id", ...)`)
 - `req.Context()` under key `ContextKeyRequestID`

If the client sends its own `X-Request-ID`, the middleware preserves it.


### RealIP
```go
r.Use(router.RealIP())
```

Extracts the real client IP from:
 - X-Real-IP
 - X-Forwarded-For
If none is trusted, falls back to `req.RemoteAddr`.

The resolved IP is stored in:
 - r.RemoteAddr
 - req.Context()
 - ctx.Set("real_ip", ...)
Works together with trusted proxies defined via:

```go
router.SetTrustedProxies([]string{"10.0.0.0/8"})
```

### NoCache
```go
r.Use(router.NoCache())
```

Disables browser caching by adding:
```yaml
Cache-Control: no-store, no-cache, must-revalidate, max-age=0
Pragma: no-cache
Expires: 0
```

Useful for:
 - APIs
 - admin panels
 - sensitive data
 - development mode

### CORS

Full CORS middleware with:
 - dynamic origin matching (supports wildcards and prefix matches)
 - automatic OPTIONS preflight handling
 - credential support (cookies, auth headers)
 - custom exposed headers
```go
r.Use(router.CORS(router.CORSOptions{
	AllowedOrigins:   []string{"https://*", "http://*"},
	AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
	ExposedHeaders:   []string{"Link"},
	AllowCredentials: false,
	MaxAge:           300,
}))
```
If the origin does not match → middleware bypasses CORS handling and continues normally.

### GetHead

```go
r.Use(router.GetHead())
```
Automatically converts `HEAD` requests to `GET` before routing.

This avoids having to define duplicate routes:

```go
GET /users
HEAD /users
```

The router returns empty body but correct headers, as required.


## 🌐 Path Prefix Stripping for Frontend Apps (no reverse proxy)

Use `r.Prefix("/app")` to transparently strip the `/app` prefix before your handlers see the request path.
Note: This does **not** forward requests to a different process/host. For reverse proxying to a Next.js dev server or a separate service, use a real reverse proxy (nginx/Caddy) or implement an `httputil.ReverseProxy` handler.

This enables routing like:

- `http://localhost:8000/app/test`


```go
r := router.NewRouter()

r.Prefix("/app")

r.Static("files/public", "/assets")

```

### Use Case:

If you're running a modern frontend (like Next.js, Vite, React, or SvelteKit) with a development or production reverse
proxy setup, this allows all `/app` routes and assets (e.g., client JS, API calls, static files) to be served under the rewritten path.

Great for full-stack setups where both backend and frontend are served from the same origin:

- `/app/_next/static/...`
- `/app/api/...`
- `/test` (if defined client-side)

---

## 🗂 Working with ctx

Each request handler receives a `*router.Context` instance, which provides access to:

- **Route parameters** (`ctx.Param(key)`)
- **Custom data storage** (ctx.Set(key, value) and ctx.Get(key))

### 🔍 Accessing route parameters

If your route uses parameters, you can access them like this:

```go
r.HandleFunc("/user/<id:(\\d+)>", "GET", func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    id, ok := ctx.Param("id")
    if !ok {
        // handle missing param
        http.Error(w, "missing id", http.StatusBadRequest)
    return
    }
    fmt.Fprintf(w, "User ID: %s", id)
})
```

Note:
Since v1.0.6 ``ctx.Param(key)`` returns (string, bool) instead of only string to make it explicit whether the parameter exists.

### 🗂 Access all parameters at once

You can also retrieve all parameters as a map:
```go
r.HandleFunc("/user/<id:(\\d+)>/<slug>", "GET", func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    params := ctx.ParamMap()
    fmt.Println(params["id"])
    fmt.Println(params["slug"])
})
```

This is useful when you need to iterate or inspect multiple parameters.


These values are stored in a thread-safe per-request context and reset automatically after the request completes.

## 🚨 Handling errors in handlers

Use `router.Error` or `router.JSONError` to log errors and respond to the client, while keeping your handlers clean and
idiomatic.

### Plain text error response

```go
err := errors.New("something went wrong")

if router.Error(w, r, "Failed to do something", err) {
    return // important: stop handler after writing response
}
```

- Logs the error internally (with stack trace and request info).
- Sends a 500 Internal Server Error with a plain text message.
- Returns true if an error occurred, so you can exit early from the handler.

### JSON error response

```go
err := errors.New("something went wrong")

if router.JSONError(w, r, "Failed to do something", err) {
    return
}
```

- Logs the error internally.
- Sends a `500 Internal Server Error` with a JSON payload:

Using `JSONResponse`:
```go
router.JSONResponse(w, http.StatusOK, yourData, nil)
router.JSONResponse(w, http.StatusInternalServerError, nil, "Something went wrong")
```
Example JSON payloads:
```json
{"success":true,"data":{"...": "..."},"status":200}
```
```json
{"success":false,"error":"Something went wrong","status":500}
```

### ⚠️ Important

Both `router.Error` and `router.JSONError` **do not automatically stop** the handler execution.

👉 **Always return immediately after calling them** to prevent writing to the response multiple times.

## 📊 Profiling

Enable pprof support by calling:

```go
r.EnableProfiling("localhost:10000")
```

Access via:

```
http://localhost:10000/debug/pprof/
```

---

## 📁 Static Files

Serve static files by specifying a directory and a URL prefix:

```go
r.Static("files/public", "/assets")
```

This serves files like:

- `./files/public/style.css` → `http://yourdomain.com/assets/style.css`
- `./files/public/images/logo.png` → `http://yourdomain.com/assets/images/logo.png`

### 📌 Note on favicon.ico

If `favicon.ico` is found in your static directory, it will be automatically served at:

```source
http://yourdomain.com/favicon.ico
```

No need to define this route manually.

---

## 💥 Custom Recovery Handler

In addition to the built-in panic recovery, you can define your own `custom recovery handler` to fully control what
happens when a panic occurs. This is useful for logging, sending custom error responses (e.g., JSON, plain text, or
HTML), or gracefully notifying users.

Example:

```go
r.Recovery(func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusInternalServerError)
    fmt.Fprint(w, `{"error":true,"message":"Something went terribly wrong"}`)
})
```

If no recovery handler is defined, a default 500 Internal Server Error is returned.

## 🚧 Custom 404 Page

You can register a `custom 404 handler` to serve your own response when a route is not found. This allows you to return
HTML, JSON, plain text, or even render templates – whatever fits your use case.

Example – HTML Page:

```go
r.NotFound(func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    fmt.Fprint(w, `<html><body><h1>Error Page</h1></body></html>`)
})
```

### Alternative formats:

JSON response:

```go
r.NotFound(func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    router.JSON(w, http.StatusNotFound, map[string]any{
        "error":   true,
        "message": "Resource not found",
    })
})
```

Plain text:

```go
r.NotFound(func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    router.Text(w, http.StatusNotFound, "404 - Page Not Found")
})
```

The NotFound handler ensures your application responds consistently across environments — whether for APIs, web apps, or
full-stack apps.

## 🖥️ Terminal Logging

Want real-time request logging and a startup banner?
Enable terminal output mode like this:

```go
r.TerminalOutput(true)
```

Example output:

```go
› NetLifeGuru 1.0.7
› Web servers is running on: http: //localhost:8000

› NetLifeGuru 1.0.7
› Web servers is running on: http: //localhost:8001

2025-04-11 19:34:42:  Method[GET]  localhost:8000/landing in 8µs
2025-04-11 19:35:42:  Method[GET]  localhost:8000/landing in 5µs
2025-04-11 19:35:42:  Method[GET]  localhost:8000/landing in 6µs
2025-04-11 19:35:42:  Method[GET]  localhost:8001/landing in 5µs
2025-04-11 19:35:42:  Method[POST]  localhost:8001/sign-in in 6µs
2025-04-11 19:35:42:  Method[GET]  localhost:8001/account in 8µs
2025-04-11 19:35:42:  Method[GET]  localhost:8000/landing in 6µs

2025-04-11 19:37:41
Panic occurred on URL: [/err]
Method: [GET]
Error message: Failed to do something
/app/test.go:351

2025-04-13 19:37:41:  Method[GET]  localhost:8000/err in 59µs
```

This is especially helpful during development or performance testing.

## 🛡 RateLimit Guard

Limit requests per client by a time threshold (`time.Duration`):

```go
r := router.NewRouter()

r.Static("views", "/assets")

r.HandleFunc("/", "GET", func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    w.WriteHeader(http.StatusOK)
})

r.Use(func(next router.HandlerFunc) router.HandlerFunc {

    return func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {

        if router.RateLimit(w, r, 10*time.Millisecond) {
            router.Abort(ctx)
            return
    }

    next(w, r, ctx)

    }
})
```

---

## 💬 JSON & Text Helpers

Send raw JSON:

```go
router.JSON(w, http.StatusOK, map[string]string{"message": "OK"})
```

Send text:

```go
router.Text(w, http.StatusNotFound, "Not found")
```

Structured response:

```go
router.JSONResponse(w, http.StatusOK, yourData, nil)
router.JSONResponse(w, http.StatusInternalServerError, nil, "Something went wrong")
```

---

## 📐 Routing Rules & Patterns

### 🔒 Strict Routing

This router enforces strict route matching:

- `/test` ✅
- `/test/` ❌ (will not match `/test`)

Always define routes without trailing slashes unless explicitly needed.

### Accessing Query

Retrieve a query parameter from the URL

```go
email := router.Query(r, "email")
```

If the URL is:

```go
/search?email=test@example.com
```

Then email will be:

```go
"test@example.com"
```

If the parameter is missing, an empty string is returned.

Use this for simple GET query lookups without manually accessing `r.URL.Query()`.

You can retrieve GET query parameters or POST form data with the built-in helpers:

```go
query := router.Get(r)
form, err := router.Post(r)
```

```go
r.HandleFunc("/signup", "POST", func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    query := router.Get(r)
    fmt.Println("Email from query:", query.Get("email"))

    form, err := router.Post(r)
    if err != nil {
        router.JSONError(w, r, "Invalid form", err)
        return
    }
    
	fmt.Println("Name from form:", form.Get("name"))

    router.JSON(w, http.StatusOK, map[string]string{"message": "Form received!"})
})
```

### Quick Access Helpers

### 🧩 Parameterized Routes (Slugs - Regex Supported)

### Parameterized Routes Without Pattern (Simple Slugs)

You can define parameterized routes without specifying any pattern.
This is a **shorthand form** that matches **any non-slash segment** and extracts it as a named parameter.

```go
r.HandleFunc("/article/{id}", "GET", func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    id := ctx.Param("id")
    fmt.Fprintf(w, "Article ID: %s", id)
})
```

This will match:

- /article/123
- /article/abc
- /article/something-else

And extract "123", "abc", etc. into the parameter id.

**Syntax Options**
You can use either **{}** or **<>** syntax — they are equivalent:

```go
r.HandleFunc("/article/<id>", "GET", handler)
r.HandleFunc("/article/{id}", "GET", handler)
```

Both will behave the same and capture the segment into `ctx.Param("id")`.

### Behavior

- Matches exactly `one segment` (i.e., a part of the URL between slashes).
- No pattern is enforced — any string (except `/`) will be accepted.
- Ideal for IDs, slugs, or simple dynamic paths.
- If you want to restrict matching (e.g., digits only), add a pattern like `<id:\\d+>` or `<id:isDigits>`.

Use dynamic segments with named regex:

```go
r.HandleFunc("/article/<article:([\\S]+)>", "GET", func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    article := ctx.Param("article")
    fmt.Fprintf(w, "Test ID: %s", article)
})
```

```go
/article/<article:([\S]+)>
```

Examples:

- `/article/abc123`
- `/article/xyz-456`

Access via:

```go
ctx.Param("article")
```

### 📦 Fast Pattern Matchers (Regexp-less, for Performance)

To accelerate matching and reduce the overhead of full regexp evaluation, NetLifeGuru Router includes a set
of `built-in pattern matchers` that work without regular expressions.

Syntax: `<name:regex>`

Examples:

| Route                                       |              Description               |
|:--------------------------------------------|:--------------------------------------:|
| `/user/<id:(\\d+)>`                         |            Only numeric IDs            |
| `/post/<slug:([a-zA-Z0-9\-_]+)>`            | Slug-friendly with hyphens/underscores |
| `/file/<filename:([\S]+)>/<token:([0-9]+)>` |          More slugs in a row           |

### Supported Pattern Functions

You can use them in your routes by either their `regex-like` pattern or `function name`:

| Function     | Pattern          |           Description           |     Example Input      | 
|:-------------|------------------|:-------------------------------:|:----------------------:|
| isLowerAlpha | [a-z]+           |     Lowercase letters only      |         "abc"          |
| isUpperAlpha | [A-Z]+           |     Uppercase letters only      |         "ABC"          |
| isAlpha      | [a-zA-Z]+        |    Letters only (mixed case)    |         "Test"         |
| isDigits     | [0-9]+, \d+      |           Digits only           |        "123456"        |
| isAlnum      | [a-zA-Z0-9]+     |          Alphanumeric           |        "user42"        |
| isWord       | \w+              | Letters, digits, underscore (_) |     "hello_world"      |
| isSlugSafe   | [\w\-]+          | Like isWord, but also allows -  |      "post-title"      |
| isSlug       | [a-z0-9\-]+      |   Lowercase + digits + - only   |      "my-article"      |
| isHex        | [a-fA-F0-9]+     |       Hexadecimal string        |         "3fA9"         |
| isUUID       | 8-4-4-4-12       |           UUID format           |    "a1b2-c3d4-e5f6"    |
| isSafeText   | [a-zA-Z0-9 _.-]+ | Letters, digits, space, _, ., - |     "File name-1"      |
| isUpperAlnum | [A-Z0-9]+        |     Uppercase + digits only     |       "ADMIN99"        |
| isBase64     | a-zA-Z0-9+/=     |       Base64 safe string        |       "SGVsbG8="       |
| isDateYMD    | \d{4}-\d{2}-\d{2} |     Date format YYYY-MM-DD      |      "2025-04-20"      |
| isSafePath   | [a-zA-Z0-9/._-]+ |       Safe for URL paths        | "img/uploads/logo.png" |
| any          | .* / alwaysTrue  |         Always matches          |       Any input        |

### Example – Using Named Pattern Matchers

Instead of using complex regex in routes, use friendly pattern names:

```go
r.HandleFunc("/user/<id:isDigits>", "GET", func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    id := ctx.Param("id")
    fmt.Fprintf(w, "User ID: %s", id)
})
```

or

```go
r.HandleFunc("/user/{id:isDigits}", "GET", func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    id := ctx.Param("id")
    fmt.Fprintf(w, "User ID: %s", id)
})
```

Equivalent regex version:

```go
r.HandleFunc("/user/<id:(\\d+)>", "GET", func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    id := ctx.Param("id")
    fmt.Fprintf(w, "User ID: %s", id)
})
```

or

```go
r.HandleFunc("/user/{id:(\\d+)}", "GET", func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    id := ctx.Param("id")
    fmt.Fprintf(w, "User ID: %s", id)
})
```

Both versions behave the same, but the function matcher is **faster** and easier to read.

### How It Works Internally

Instead of running `regexp.MatchString`, the router will:

- Check if the pattern exists in the internal `PatternMatchers` or `FunctionMatchers`.
- If so, call the corresponding Go function (`isDigits`, `isSlug`, `isUUID`, etc.)
- These functions are pure Go code `(no regex)`, designed for ultra-fast string evaluation.

### When Not to Use Pattern Matchers

If you need more advanced regex (e.g., lookaheads, backreferences), fallback to traditional regex
with (`<name:regexp>`).

```go
r.HandleFunc("/match/<slug:([a-z]+_[0-9]{2})>", "GET", handler)
```

But if performance and simplicity are important, always prefer named pattern matchers.

### 🔁 HTTP Method Support

Each route must explicitly define allowed HTTP methods:

```go
r.HandleFunc("/users", "GET", handler)
r.HandleFunc("/users", "POST", handler)
r.HandleFunc("/users/<id:(\\d+)>", "PUT", handler)
```

Wildcard:

```go
r.HandleFunc("/ping", "ANY", handler)
```

Supported methods:

- GET
- POST
- PUT
- DELETE
- PATCH
- OPTIONS
- HEAD
- ANY (wildcard)

---

## 🔥 Panic Recovery

NetLifeGuru Router includes built-in `panic` recovery. If a panic occurs during request processing, the server will not
crash. Instead, the router automatically catches the panic, logs the error (including stack trace) to a file in
the `logs/` directory, and executes the `Recovery` middleware if defined. If no recovery handler is provided, a
default `500 Internal Server Error` is returned.

Example:

```go
r.Recovery(func (w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    http.Error(w, "Unexpected error occurred", http.StatusInternalServerError)
})
```

### 🧾 Panic Logging Example

On each panic, the router writes a detailed error log to a daily rotating log file in the `logs/` directory. The log
includes a timestamp, request path, method, error message, and the file/line where the panic occurred.

**Log filename format:**

```go
YYYY-MM-DD.error.log
```

**Example** – `2025-04-13.error.log`:

```log
2025/04/13 16:56:44 Panic occurred on URL /err | method [GET]
Error message: struct error
/project/app/handlers.go:18

_______________________________________________________________________________________________
```

This helps you quickly trace and debug issues without crashing your server.

## 📊 Benchmark Results

NetLifeGuru Router is designed with **performance in mind**, especially in the core routing logic. Below are the results
of micro-benchmarks targeting different route types and depths.

### Benchmark Summary

| Benchmark                | Ops/sec (↑)    | Time per op (↓) | Allocations | Bytes |
|--------------------------|----------------|-----------------|-------------|-------|
| Static route (/)         | ~35M ops/sec   | ~33 ns/op       | 1           | 9 B   |
| Route with 4 parameters  | ~16.7M ops/sec | ~71 ns/op       | 1           | 6 B   |
| Route with 7 parameters  | ~12.6M ops/sec | ~94 ns/op       | 1           | 7 B   |
| Route with 51 parameters | ~3.5M ops/sec  | ~347 ns/op      | 1           | 6 B   |

### 🚀 Benchmark Code Snippet

```go
func Benchmark_NetLifeGuruRouter_Static(b *testing.B) {

	r := router.NewRouter()
	concrete := r.(*router.Router)

	url := "/"

	concrete.HandleFunc(url, "GET", func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
		testHandler(w, r)
	})

	req := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		concrete.ServeHTTP(w, req)
	}
}
```

```go
func Benchmark_NetLifeGuruRouter_Param4(b *testing.B) {
	r := router.NewRouter()

	concrete := r.(*router.Router)
	r.HandleFunc("/shop/{a}/{b}/{c}/{d}", "GET", func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
		testHandler(w, r)
	})

	req := httptest.NewRequest("GET", "/shop/a/b/c/d", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		concrete.ServeHTTP(w, req)
	}
}
```

```go
func Benchmark_NetLifeGuruRouter_Param7(b *testing.B) {
	r := router.NewRouter()

	concrete := r.(*router.Router)

	r.HandleFunc("/article/{aaa}/{bbb}/{ccc}/{ddd}/{eee}/{fff}/{ggg}", "GET", func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
		testHandler(w, r)
	})

	req := httptest.NewRequest("GET", "/article/aaa/bbb/ccc/ddd/eee/fff/ggg", nil)
	w := httptest.NewRecorder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		concrete.ServeHTTP(w, req)
	}
}
```

```go
func Benchmark_NetLifeGuruRouter_Param50(b *testing.B) {
r := router.NewRouter()

concrete := r.(*router.Router)

r.HandleFunc("/test/<a1>/<a2>/<a3>/<a4>/<a5>/<a6>/<a7>/<a8>/<a9>/<a10>/<a11>/<a12>/<a13>/<a14>/<a15>/<a16>/<a17>/<a18>/<a19>/<a20>/<a21>/<a22>/<a23>/<a24>/<a25>/<a26>/<a27>/<a28>/<a29>/<a30>/<a31>/<a32>/<a33>/<a34>/<a35>/<a36>/<a37>/<a38>/<a39>/<a40>/<a41>/<a42>/<a43>/<a44>/<a45>/<a46>/<a47>/<a48>/<a49>/<a50>/<a51>", "GET", func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
testHandler(w, r)
})

req := httptest.NewRequest("GET", "/test/1/2/3/4/5/6/7/8/9/10/11/12/13/14/15/16/17/18/19/20/21/22/23/24/25/26/27/28/29/30/31/32/33/34/35/36/37/38/39/40/41/42/43/44/45/46/47/48/49/50/51", nil)
w := httptest.NewRecorder()

b.ResetTimer()
for i := 0; i < b.N; i++ {
concrete.ServeHTTP(w, req)
}
}
```

### Observations

- **Static routes** are blazing fast with sub-35ns latency and a single allocation.
- **Parameterized routes** (e.g., <id>, {slug}) without regex scale linearly with depth, but remain highly efficient.
- The use of non-regex slugs like <id> or <name:any> avoids full regexp parsing, making routing faster and lighter.
- Even at 50+ parameters, the router performs admirably under 350ns per op.

## 💼 Project Structure

- `router.go` – main router logic `(routing, dispatch, middleware, static/proxy handling)`
- `context.go` – request context pool with per-request storage and reset
- `helpers.go` – JSON, text, file utils, response helpers, file utilities, query/form parsing
- `error.go` – panic recovery, error logging with stack trace and file output
- `terminal.go` – colored terminal logging and startup banners
- `rate_limiter.go` – request throttling (RateLimit guard)
- `method_bitmask.go` – efficient method mapping using bitmasks (GET, POST, etc.)
- `patterns.go` – fast path parameter matchers (regex-free), includes named pattern functions like `isSlug`, `isUUID`, etc.

---

## 🤝 Contributing

This project is open to community contributions and feedback!

---

## 📢 Author

Created by NetLife Guru s.r.o.
Framework: **NetLifeGuru Router**  
Version: **v1.0.8**

---

## 🧬 Inspired by:

- Go’s net/http
- Gin, Chi, Fiber
- UNIX minimalism & purpose-driven tools
