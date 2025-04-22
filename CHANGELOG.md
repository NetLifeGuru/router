# 📄 Changelog

All notable changes to this project will be documented in this file.

This project adheres to [Semantic Versioning](https://semver.org/)  
and follows the [Keep a Changelog](https://keepachangelog.com/) format.

---

[1.0.5] – 2025-04-22

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
