# 📄 Changelog

All notable changes to this project will be documented in this file.

This project adheres to [Semantic Versioning](https://semver.org/)  
and follows the [Keep a Changelog](https://keepachangelog.com/) format.

---

## [1.0.7] – 2025-11-25

### Added
- Unified middleware system (`Use`) replacing old lifecycle hooks.
- New built-in middleware: AllowContentType, ContentCharset, CleanPath, DefaultCompress, Compress, CORS, RequestID, RealIP, NoCache, GetHead.
- New `UseDefaults()` helper registering a standard middleware chain.
- Improved terminal logging with better panic formatting and colors.
- Extended built-in pattern matchers and optimized non-regex path evaluation.

### Changed
- Removed legacy `Before` and `After` middleware API.
- Context system now resets more efficiently; improved parameter handling.
- Enhanced RealIP middleware with trusted proxy list via SetTrustedProxies.
- CORS system rewritten with wildcard support, exposed headers & credential rules.
- Static path rewriting improved for frontend-prefix usage.
- Compression middleware improved for HEAD responses & automatic MIME detection.

### Removed
- Deprecated `Before` and `After` lifecycle middleware.
- All remaining internal references to old lifecycle hooks.

### Fixed
- Improved radix tree matching for star-prefixed and wildcard nodes.
- Better prefix selection logic inside routing tree.
- Compression now consistently removes Content-Length when required.
- Panic logs now always include method and normalized request path.

### Performance
- Faster radix prefix matching with fewer allocations.
- More efficient middleware chaining (lighter wrapper structure).
- Faster RealIP resolution and reduced header parsing overhead.
- Named pattern matchers now outperform equivalent regex.

### Migration
- Replace:
  - r.Before(...) and  r.After(...)
    With:
  - r.Use(func(next router.HandlerFunc) router.HandlerFunc { ... })

- Remove custom HEAD handling; use:
  r.Use(router.GetHead())

- Using old compression wrappers? Replace with:
  r.Use(router.DefaultCompress())

- Using proxy IP detection? Configure:
  router.SetTrustedProxies([]string{"10.0.0.0/8"})

---

## [1.0.6] – 2025-09-24

### Added
- **Rewritten `MultiListenAndServe`**
    - Uses all available CPU cores (`runtime.GOMAXPROCS`) automatically.
    - On Unix systems enables **`SO_REUSEPORT` + `SO_REUSEADDR`** for true multi-listener concurrency per port.
    - Built-in graceful shutdown with signal handling (`os.Interrupt`/`SIGTERM`) and per-server timeouts (`ReadTimeout`, `WriteTimeout`, `IdleTimeout`, `ReadHeaderTimeout`).
    - Logs CPU core usage when `TerminalOutput(true)` is enabled.
- **Healthcheck endpoint `/ready`** (via `r.Ready()`): returns `200 OK` while running and `503 Service Unavailable` during graceful shutdown.
- **Radix tree routing index** for faster parameter and wildcard matching.
- **405 Method Not Allowed** responses with correct `Allow` header when a route matches but method does not.
- **Internal readiness flag** (atomic) backing `/ready`.
- **Early-abort flow**: `router.Abort(ctx)` now respected even `Before` or `After` handlers.

### Changed
- **Routing**
    - Replaced regex-based URL splitting with manual segment parsing (fewer allocations, faster).
    - Parameter validation now only runs if the segment uses something other than `any`.
- **Rate limiting API**
    - Signature changed from `RateLimit(w, r, threshold int64 /*ns*/)` to `RateLimit(w, r, threshold time.Duration) bool`.
    - Limit is now **per client IP + method + path** (previously per host only).
    - When throttled, responds with **HTTP 429 Too Many Requests** + `Retry-After` instead of blocking with `sleep()`.
- **Context**
    - `Param(key)` now returns `(string, bool)` instead of only string.
    - Added `ParamMap()` and internal lazy parameter setting.
    - Context pooling aggressively resets per request.
- **Static files**
    - Uses `ensureDirectory()` and consistent relative paths; automatically serves `favicon.ico` if present.
- **Logging**
    - Cleaner request log output (ns/µs/ms) and startup banner.

### Removed
- **Legacy listener networking** in multi-server mode.  
  On Unix, multi-listener with `SO_REUSEPORT` replaces the old “one listener per port” model.
- **Global per-host rate-limiting with blocking sleep** — replaced by non-blocking 429 response.
- **Group API** removed from public interface.

### Fixed
- Static routes now consistently respect HTTP methods (follow-up to 1.0.5).
- More robust recovery handler with secondary `recover` wrapper to avoid panics during panic handling.

### Performance
- **Radix routing** significantly improves candidate selection for deep or complex paths.
- Fewer allocations when parsing paths and building cache keys.
- Higher throughput on Unix thanks to multiple listener workers via `SO_REUSEPORT`.

### Migration
- **RateLimit** usage:
  ```diff
  - router.RateLimit(w, r, 10000) // nanoseconds
  + if router.RateLimit(w, r, 10*time.Millisecond) {
  +   ctx.Abort()
  +   return
  + }
### Context params:
 ```diff
 - id := ctx.Param("id")
 + id, ok := ctx.Param("id")
 + if !ok { /* handle missing */ }
 ```

or use ctx.ParamMap().

- Always return immediately after calling ctx.Abort() inside middleware.
- For reverse proxying to a separate process, continue to use nginx/Caddy or ``httputil.ReverseProxy``; the router’s ``Proxy`` only strips prefixes.

## [1.0.5] – 2025-04-22

### Fixed
 - Static route handling now correctly checks HTTP method
 - Previously, static routes were matched based solely on path without validating the HTTP method (e.g. GET, POST). This could cause handlers to run for unintended methods. This fix ensures method validation is consistent across all route types.

## [1.0.4] – 2025-04-21

### Added
 - Introduced a Code of Conduct to help foster a welcoming and respectful community
 - Added a Contributing Guide outlining best practices for issues, pull requests, commits, and testing

### Changed

- Internal refactor: moved indexToBit and getBitmaskIndex into the Router as methods for better cohesion
`(no impact on public API or behavior)`

### Fixed
 - Cleaned up redundant type conversion in route handling
 - Improved how HandleFunc is stored and resolved during request routing
 - Resolved edge cases in ServeHTTP route matching logic

### Performance
 - Replaced `http.NotFound` with a lightweight custom 404 handler to reduce memory allocations

## [1.0.3] – 2025-04-21

### Fixed

 - Critical logic bug in parameter matching loop:
 - Routing logic failed to correctly iterate over registered handlers in HandleFunc due to improper nesting and index usage.
 - This caused certain valid routes (especially parameterized ones) to be skipped or ignored under specific conditions.

### Changed
 - **Routing performance improved** by correcting traversal logic of nested handler maps (`map[int][]RouteEntry`), ensuring precise matching based on the number of path parameters (`len(patterns)`).
 - **More accurate dispatching** of parameterized routes by leveraging the correct segment depth (`len(ctx.Segments)`) as routing key.

This patch ensures stable and deterministic matching of all registered routes, especially in deeply nested or heavily parameterized path structures.

### Added

 - Support for fast, regex-free pattern matchers via FunctionMatchers and PatternMatchers
 - New built-in slug types (e.g. `isSlug`, `isUUID`, `isDigits`, etc.)
 - Auto-slug matching using <id> or {id} without requiring regex
 - Documentation: extensive `README` update with new examples, behavior descriptions, and benchmark section
 - Detailed benchmark suite covering static and parameterized routes
 - Markdown badges and project metadata

### Changed

 - Unified parameter matching to prioritize named functions for performance
 - Simplified internal routing logic for patternless parameters
 - Minor documentation tweaks and formatting improvements


## [1.0.1] – 2025-04-14

### Changed
 - Renamed internal functions for more consistent and idiomatic naming
 - Standardized usage of w `http.ResponseWriter` and r `*http.Request` in all handlers to ensure cleaner and more maintainable code
 - Refined documentation with improved feature explanations, clearer usage examples, and better overall structure

---

## [1.0.0] – 2025-04-13

### Added
- Initial implementation of NetLifeGuru Router
- Routing with regex parameters
- Middleware lifecycle (Init, Before, After, Recovery)
- Static file support with auto favicon
- Terminal output with request logging
- Multi-server support (`MultiListenAndServe`)
- JSON/text response helpers
- Panic recovery and error logging
- Built-in pprof profiling
- Project structure and documentation
]()
