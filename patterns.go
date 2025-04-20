package router

import (
	"strings"
	"unicode"
)

type MatchFunc func(string) bool

var FunctionMatchers = map[string]MatchFunc{
	`isLowerAlpha`: isLowerAlpha,
	`isUpperAlpha`: isUpperAlpha,
	`isAlpha`:      isAlpha,
	`isDigits`:     isDigits,
	`isAlnum`:      isAlnum,
	`isWord`:       isWord,
	`isSlugSafe`:   isSlugSafe,
	`isSlug`:       isSlug,
	`isHex`:        isHex,
	`isUUID`:       isUUID,
	`isSafeText`:   isSafeText,
	`isUpperAlnum`: isUpperAlnum,
	`isBase64`:     isBase64,
	`isDateYMD`:    isDateYMD,
	`isSafePath`:   isSafePath,
	`any`:          isAny,
}

var PatternMatchers = map[string]MatchFunc{
	`[a-z]+`:            isLowerAlpha,
	`[A-Z]+`:            isUpperAlpha,
	`[a-zA-Z]+`:         isAlpha,
	`[0-9]+`:            isDigits,
	`\d+`:               isDigits,
	`[a-zA-Z0-9]+`:      isAlnum,
	`\w+`:               isWord,
	`[\w\-]+`:           isSlugSafe,
	`[a-z0-9\-]+`:       isSlug,
	`[a-fA-F0-9]+`:      isHex,
	`8-4-4-4-12`:        isUUID,
	`[a-zA-Z0-9 _.-]+`:  isSafeText,
	`[A-Z0-9]+`:         isUpperAlnum,
	`a-zA-Z0-9+/=`:      isBase64,
	`\d{4}-\d{2}-\d{2}`: isDateYMD,
	`[a-zA-Z0-9/._-]+`:  isSafePath,
	`.*`:                alwaysTrue,
}

func isAny(s string) bool {
	return true
}

func isLowerAlpha(s string) bool {
	if s == "" {
		return false
	}

	for i := 0; i < len(s); i++ {
		if s[i] < 'a' || s[i] > 'z' {
			return false
		}
	}

	return true
}

func isUpperAlpha(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 'A' || s[i] > 'Z' {
			return false
		}
	}
	return true
}

func isAlpha(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func isAlnum(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func isWord(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z') || (s[i] >= '0' && s[i] <= '9') || s[i] == '_' {
			continue
		}
		return false
	}
	return true
}

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_'
}

func isSlugSafe(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !isWordChar(r) && r != '-' {
			return false
		}
	}
	return true
}

func isSlug(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if (s[i] >= 'a' && s[i] <= 'z') || (s[i] >= '0' && s[i] <= '9') || s[i] == '-' {
			continue
		}
		return false
	}
	return true
}

func isSpaceOnly(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func alwaysTrue(string) bool {
	return true
}

func isHex(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !(s[i] >= '0' && s[i] <= '9' || s[i] >= 'a' && s[i] <= 'f' || s[i] >= 'A' && s[i] <= 'F') {
			return false
		}
	}
	return true
}

func isUUID(s string) bool {
	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		return false
	}
	lengths := []int{8, 4, 4, 4, 12}
	for i := 0; i < len(parts); i++ {
		if len(parts[i]) != lengths[i] || !isHex(parts[i]) {
			return false
		}
	}
	return true
}

func isSafeText(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !(s[i] >= 'a' && s[i] <= 'z' ||
			s[i] >= 'A' && s[i] <= 'Z' ||
			s[i] >= '0' && s[i] <= '9' ||
			s[i] == ' ' || s[i] == '_' || s[i] == '.' || s[i] == '-') {
			return false
		}
	}
	return true
}

func isUpperAlnum(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !(s[i] >= 'A' && s[i] <= 'Z' || s[i] >= '0' && s[i] <= '9') {
			return false
		}
	}
	return true
}

func isBase64(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !(s[i] >= 'a' && s[i] <= 'z' ||
			s[i] >= 'A' && s[i] <= 'Z' ||
			s[i] >= '0' && s[i] <= '9' ||
			s[i] == '+' || s[i] == '/' || s[i] == '=') {
			return false
		}
	}
	return true
}

func isDateYMD(s string) bool {
	if len(s) != 10 {
		return false
	}
	if s[4] != '-' || s[7] != '-' {
		return false
	}
	for i := 0; i < len(s); i++ {
		if i == 4 || i == 7 {
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func isSafePath(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !(s[i] >= 'a' && s[i] <= 'z' ||
			s[i] >= 'A' && s[i] <= 'Z' ||
			s[i] >= '0' && s[i] <= '9' ||
			s[i] == '/' || s[i] == '.' || s[i] == '_' || s[i] == '-') {
			return false
		}
	}
	return true
}

func isValidURLSegment(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] >= 'a' && s[i] <= 'z':
		case s[i] >= 'A' && s[i] <= 'Z':
		case s[i] >= '0' && s[i] <= '9':
		case s[i] == '-' || s[i] == '_' || s[i] == '.':
		default:
			return false
		}
	}
	return true
}
