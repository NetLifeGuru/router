# 📄 Changelog

All notable changes to this project will be documented in this file.

This project adheres to [Semantic Versioning](https://semver.org/)  
and follows the [Keep a Changelog](https://keepachangelog.com/) format.

---

## [1.0.2] – 2025-04-20


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
