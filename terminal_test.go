package router

import (
	"strings"
	"testing"
)

func TestGetTextColor_KnownColor(t *testing.T) {
	color := getTextColor("red")
	if color != "\033[31m" {
		t.Errorf("expected \\033[31m for red, got %s", color)
	}
}

func TestGetTextColor_UnknownColor(t *testing.T) {
	color := getTextColor("unknown")
	if color != "\033[39m" {
		t.Errorf("expected default \\033[39m for unknown color, got %s", color)
	}
}

func TestGetBgColor_KnownColor(t *testing.T) {
	bg := getBgColor("blue")
	if bg != "\033[44m" {
		t.Errorf("expected \\033[44m for blue, got %s", bg)
	}
}

func TestGetBgColor_UnknownColor(t *testing.T) {
	bg := getBgColor("invisible")
	if bg != "\033[49m" {
		t.Errorf("expected default \\033[49m for unknown color, got %s", bg)
	}
}

func TestColorsFunction(t *testing.T) {
	result := colors("green", "Hello")
	expectedStart := "\033[32mHello\033[0m"

	if result != expectedStart {
		t.Errorf("expected '%s', got '%s'", expectedStart, result)
	}
}

func TestFormatText(t *testing.T) {
	result := formatText(FormatText{
		color:      "yellow",
		background: "blue",
		text:       "Test",
	})

	if !strings.HasPrefix(result, "\033[33m\033[44m") || !strings.HasSuffix(result, "Test\033[0m") {
		t.Errorf("unexpected formatText output: %s", result)
	}
}
