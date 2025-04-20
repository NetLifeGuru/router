package router

import (
	"testing"
)

type dummyRouter struct {
	Router
}

func TestGetMethodIndex(t *testing.T) {
	r := &dummyRouter{}

	tests := map[string]int{
		"GET":     GET,
		"POST":    POST,
		"PUT":     PUT,
		"DELETE":  DELETE,
		"PATCH":   PATCH,
		"HEAD":    HEAD,
		"OPTIONS": OPTIONS,
		"ANY":     ANY,
		"INVALID": -128,
	}

	for method, expected := range tests {
		got := r.getMethodIndex(method)
		if got != expected {
			t.Errorf("getMethodIndex(%q) = %d, want %d", method, got, expected)
		}
	}
}

func TestRemoveDuplicates(t *testing.T) {
	input := []string{"GET", "POST", "GET", "DELETE", "POST"}
	expected := []string{"GET", "POST", "DELETE"}
	result := removeDuplicates(input)

	if len(result) != len(expected) {
		t.Fatalf("expected length %d, got %d", len(expected), len(result))
	}
	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("expected %s at index %d, got %s", expected[i], i, result[i])
		}
	}
}

func TestIndexToBit(t *testing.T) {
	tests := map[int]int{
		GET:     0,
		POST:    1,
		PUT:     2,
		DELETE:  3,
		PATCH:   4,
		HEAD:    5,
		OPTIONS: 6,
		ANY:     7,
		99:      7,
	}

	for input, expected := range tests {
		got := indexToBit(input)
		if got != expected {
			t.Errorf("indexToBit(%d) = %d, want %d", input, got, expected)
		}
	}
}

func TestGetBitmaskIndex(t *testing.T) {
	tests := map[string]int{
		"GET":     1,
		"POST":    2,
		"PUT":     4,
		"DELETE":  8,
		"PATCH":   16,
		"HEAD":    32,
		"OPTIONS": 64,
		"UNKNOWN": 0,
	}

	for method, expected := range tests {
		got := getBitmaskIndex(method)
		if got != expected {
			t.Errorf("getBitmaskIndex(%q) = %d, want %d", method, got, expected)
		}
	}
}

func TestMethodsToBitmask(t *testing.T) {
	r := &dummyRouter{}

	tests := []struct {
		input    string
		expected int
	}{
		{"GET ", GET},
		{"GET POST ", GET | POST},
		{"PUT DELETE POST ", PUT | DELETE | POST},
		{"INVALID ", -1},
		{"", 0},
	}

	for _, tt := range tests {
		result := r.MethodsToBitmask(tt.input)
		if result != tt.expected {
			t.Errorf("MethodsToBitmask(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestHandleRoute(t *testing.T) {
	r := &dummyRouter{}
	bitmask := GET | POST | PUT

	tests := []struct {
		method   string
		expected bool
	}{
		{"GET", true},
		{"POST", true},
		{"DELETE", false},
		{"PUT", true},
		{"UNKNOWN", false},
	}

	for _, tt := range tests {
		result := r.handleRoute(tt.method, bitmask)
		if result != tt.expected {
			t.Errorf("handleRoute(%q, %b) = %v, want %v", tt.method, bitmask, result, tt.expected)
		}
	}

	if !r.handleRoute("GET", int(ANY)) {
		t.Error("Expected handleRoute to return true for ANY method")
	}
}
