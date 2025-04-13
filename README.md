# 🚀 NetLifeGuru Router

A clean, performant and idiomatic HTTP router & microframework for Go – built for modern backend APIs, apps, and full-stack setups.

Includes built-in support for middleware, context, parameterized routing, rate limiting, profiling, static assets, and multi-port servers.

---

## ✨ Features

 - 🌐 Custom routing with regex parameters
 - 🧩 Lifecycle middleware: Init, Before, After, Recovery
 - 🗂 Request context with thread-safe storage (pooled)
 - 🛡 Simple RateLimit guard (per host)
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
go get github.com/NetLifeGuru/router
```

---

## 💡 Basic Usage


### 🔄 Single-Server Setup
```go
r := router.NewRouter()

r.HandleFunc("/", "GET", func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    w.WriteHeader(http.StatusOK)
})

r.ListenAndServe(8000)
```

---

### 🕸️ Multi-Server Setup (Microservices Ready)

For more advanced setups (e.g. microservices), you can run multiple servers using `MultiListenAndServe`.

```go
r := router.NewRouter()

r.HandleFunc("/", "GET", func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    w.WriteHeader(http.StatusOK)
})

listeners := r.Listeners{
	{Listen: "localhost:8000", Domain: "localhost:8000"},
	{Listen: "localhost:8001", Domain: "localhost:8001"},
}

r.MultiListenAndServe(listeners)
```

Each listener listens on its own port and serves the same routes. Ideal for testing subdomains or simulating multi-service environments.

One of the key advantages of multi-server support is the ability to run a single application instance on multiple domains or ports simultaneously — ideal for multi-tenant architectures, localized services, or parallel dev/staging environments.


---

## 🔐 Middleware Example

```go
r := router.NewRouter()

r.Static("/files/public", "/assets")

r.Init(func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
  // Runs once on app startup
})

r.Before(func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
  // Runs before every request
  r.Host = strings.Replace(r.Host, "127.0.0.1", "localhost", 1)
  r.RateLimit(w, r, 10000) // Drop if too frequent
})

r.After(func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
  // Runs after every request
})

r.Recovery(func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
  // Handles panics
  http.Error(w, "Unexpected error occurred", http.StatusInternalServerError)
})
```

## 🌐 Proxying Frontend Apps (e.g. Next.js)

Use `r.Proxy("/app")` to seamlessly forward requests to a frontend app like Next.js running behind a reverse proxy.

This enables routing like:

 - `http://localhost:8000/app/test`
 - `http://localhost:8000/test` (if handled by the frontend app)

```go
r := router.NewRouter()

r.Proxy("/app")

r.Static("/files/public", "/assets")

```

### Use Case:

If you're running a modern frontend (like Next.js, Vite, React, or SvelteKit) with a development or production reverse proxy setup, this allows all `/app` routes and assets (e.g., client JS, API calls, static files) to be transparently forwarded.

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
r.HandleFunc("/user/<id:(\\d+)>", "GET", func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
	id := ctx.Param("id")
	fmt.Fprintf(w, "User ID: %s", id)
})
```

### 🗂 Using `Set` and `Get`

You can attach and retrieve custom values during the request lifecycle using `Set` and `Get`. This is useful for passing data between middleware and handlers.

```go
r.Before(func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
	ctx.Set("startTime", time.Now())
})

r.After(func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
	start := ctx.Get("startTime").(time.Time)
	log.Printf("Request took %s", time.Since(start))
})
```

These values are stored in a thread-safe per-request context and reset automatically after the request completes.

## 🚨 Handling errors in handlers

Use `router.Error` or `router.JSONError` to log errors and respond to the client, while keeping your handlers clean and idiomatic.

### Plain text error response

```go
err := errors.New("something went wrong")

if router.Error("Failed to do something", err, w, r) {
	return
}
```

 - Logs the error internally (with stack trace and request info).
 - Sends a 500 Internal Server Error with a plain text message.
 - Returns true if an error occurred, so you can exit early from the handler.

### JSON error response

```go
err := errors.New("something went wrong")

if router.JSONError("Failed to do something", err, w, r) {
	return
}
```

 - Logs the error internally.
 - Sends a `500 Internal Server Error` with a JSON payload:

```json
{
  "error": true,
  "message": "Failed to do something"
}
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

In addition to the built-in panic recovery, you can define your own `custom recovery handler` to fully control what happens when a panic occurs. This is useful for logging, sending custom error responses (e.g., JSON, plain text, or HTML), or gracefully notifying users.

Example:
```go
r.Recovery(func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprint(w, `{"error":true,"message":"Something went terribly wrong"}`)
})
```

If no recovery handler is defined, a default 500 Internal Server Error is returned.

## 🚧 Custom 404 Page

You can register a `custom 404 handler` to serve your own response when a route is not found. This allows you to return HTML, JSON, plain text, or even render templates – whatever fits your use case.

Example – HTML Page:
```go
r.NotFound(func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, `<html><body><h1>Error Page</h1></body></html>`)
})
```

### Alternative formats:

JSON response:
```go
r.NotFound(func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
	router.JSON(w, 404, map[string]any{
		"error":   true,
		"message": "Resource not found",
	})
})
```

Plain text:
```go
r.NotFound(func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
	router.Text(w, 404, "404 - Page Not Found")
})
```
The NotFound handler ensures your application responds consistently across environments — whether for APIs, web apps, or full-stack apps.


## 🖥️ Terminal Logging

Want real-time request logging and a startup banner?
Enable terminal output mode like this:

```go
r.TerminalOutput(true)
```

Example output:
```go
› NetLifeGuru 1.0.0
› Web servers is running on: http://localhost:8000


› NetLifeGuru 1.0.0
› Web servers is running on: http://localhost:8001

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

Limit requests per host by time threshold (in nanoseconds):

```go
r := router.NewRouter()

r.Static("/views", "/assets")

r.HandleFunc("/", "GET", func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    w.WriteHeader(http.StatusOK)
})

r.Before(func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
    router.RateLimit(w, r, 10000) // Deny if requests are < 10ms apart
})
```

---

## 💬 JSON & Text Helpers

Send raw JSON:

```go
router.JSON(w, 200, map[string]string{"message": "OK"})
```

Send text:

```go
router.Text(w, 404, "Not found")
```

Structured response:

```go
router.JSONResponse(w, 200, yourData, nil)
router.JSONResponse(w, 500, nil, "Something went wrong")
```

---

## 📐 Routing Rules & Patterns

### 🔒 Strict Routing

This router enforces strict route matching:

- `/test` ✅
- `/test/` ❌ (will not match `/test`)

Always define routes without trailing slashes unless explicitly needed.

### 🧩 Parameterized Routes (Slugs - Regex Supported)

Use dynamic segments with named regex:

```go
r.HandleFunc("/article/<article:([\S]+)>", "GET", func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
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

### 📦 Parameter Format

Syntax: `<name:regex>`

Examples:

| Route                                       |              Description               |
|:--------------------------------------------|:--------------------------------------:|
| `/user/<id:(\d+)>`                          |            Only numeric IDs            |
| `/post/<slug:([a-zA-Z0-9\-_]+)>`            | Slug-friendly with hyphens/underscores |
| `/file/<filename:([\S]+)>/<token:([0-9]+)>` |          More slugs in a row           |

### 🔁 HTTP Method Support

Each route must explicitly define allowed HTTP methods:

```go
r.HandleFunc("/users", "GET", handler)
r.HandleFunc("/users", "POST", handler)
r.HandleFunc("/users/<id:(\d+)>", "PUT", handler)
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

NetLifeGuru Router includes built-in `panic` recovery. If a panic occurs during request processing, the server will not crash. Instead, the router automatically catches the panic, logs the error (including stack trace) to a file in the `logs/` directory, and executes the `Recovery` middleware if defined. If no recovery handler is provided, a default `500 Internal Server Error` is returned.

Example:

```go
r.Recovery(func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {
	http.Error(w, "Unexpected error occurred", http.StatusInternalServerError)
})
```
### 🧾 Panic Logging Example

On each panic, the router writes a detailed error log to a daily rotating log file in the `logs/` directory. The log includes a timestamp, request path, method, error message, and the file/line where the panic occurred.

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



## 💼 Project Structure

- `router.go` – main router logic
- `context.go` – request context pool
- `helpers.go` – JSON, text, file utils
- `logError.go` – panic handling & logging
- `terminal.go` – pretty terminal formatting
- `rate_limiter.go` – request throttling (RateLimit guard)

---

## ✅ Roadmap Ideas

- Per-route middleware (WithBefore, WithAfter)
- Websocket support
- Route grouping (/api, /admin)
- CLI generator / project scaffolder

---

## 🤝 Contributing

This project is open to community contributions and feedback!

---

## 📢 Author

Created by Martin Benadik  
Framework: **NetLifeGuru Router**  
Version: **v1.0.0**

---

## 🧬 Inspired by:

- Go’s net/http
- Chi, Echo, Fiber
- UNIX minimalism & purpose-driven tools
