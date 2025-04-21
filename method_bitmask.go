package router

import (
	"errors"
	"fmt"
	"log"
)

type HTTPMethod int

const (
	GET     = 1 << 0
	POST    = 1 << 1
	PUT     = 1 << 2
	DELETE  = 1 << 3
	PATCH   = 1 << 4
	HEAD    = 1 << 5
	OPTIONS = 1 << 6
	ANY     = 1 << 7
)

var methodMap = map[string]HTTPMethod{
	"GET":     GET,
	"POST":    POST,
	"PUT":     PUT,
	"DELETE":  DELETE,
	"PATCH":   PATCH,
	"HEAD":    HEAD,
	"OPTIONS": OPTIONS,
	"ANY":     ANY,
}

func (r *Router) getMethodIndex(method string) int {

	if val, ok := methodMap[method]; ok {
		return int(val)
	}
	return -128
}

func removeDuplicates(input []string) []string {
	seen := map[string]struct{}{}
	var result []string

	for _, val := range input {
		if _, ok := seen[val]; !ok {
			seen[val] = struct{}{}
			result = append(result, val)
		}
	}
	return result
}

func (r *Router) indexToBit(i int) int {
	switch i {
	case GET:
		return 0
	case POST:
		return 1
	case PUT:
		return 2
	case DELETE:
		return 3
	case PATCH:
		return 4
	case HEAD:
		return 5
	case OPTIONS:
		return 6
	case ANY:
		return 7
	}
	return 7
}

func (r *Router) getBitmaskIndex(m string) int {
	var method int

	switch m {
	case "GET":
		method = 1
	case "POST":
		method = 2
	case "PUT":
		method = 4
	case "DELETE":
		method = 8
	case "PATCH":
		method = 16
	case "HEAD":
		method = 32
	case "OPTIONS":
		method = 64
	}

	return method
}

func (r *Router) MethodsToBitmask(methods string) int {
	var bitmask int
	var seen [8]bool

	start := -1
	for i := 0; i < len(methods); i++ {
		if methods[i] != ' ' && start == -1 {
			start = i
		} else if methods[i] == ' ' && start != -1 {
			method := methods[start:i]
			start = -1
			index := r.getMethodIndex(method)
			if index == 128 {
				return 127
			}
			if index < 0 {
				return -1
			}
			if !seen[r.indexToBit(index)] {
				seen[r.indexToBit(index)] = true
				bitmask |= index
			}
		}
	}

	if start != -1 {
		method := methods[start:]
		index := r.getMethodIndex(method)
		if index == 128 {
			return 127
		}
		if index < 0 {
			return -1
		}
		if !seen[r.indexToBit(index)] {
			bitmask |= index
		}
	}

	return bitmask
}

func (r *Router) handleInvalidMethod(query, methods string) error {
	err := errors.New(fmt.Sprintf(
		"%s %s %s\n",
		colors("red", "✘ Fatal error:"),
		colors("yellow", "Invalid HTTP method in:"),
		colors("cyan", fmt.Sprintf(`r.HandleFunc("%s", "%s", func(w http.ResponseWriter, r *http.Request, ctx *router.Context) {}`, query, methods)),
	))

	return err
}

func (r *Router) bitmask(query, methods string) int {
	bitmask := r.MethodsToBitmask(methods)
	if bitmask < 0 {
		log.Fatal(r.handleInvalidMethod(query, methods))
	}
	return bitmask
}

func (r *Router) handleRoute(method string, methodsBitmask int) bool {
	if methodsBitmask == ANY {
		return true
	}

	index := r.getMethodIndex(method)

	if index < 0 {
		return false
	}

	return methodsBitmask&index != 0
}
