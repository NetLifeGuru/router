package router

import "testing"

func TestIsLowerAlpha(t *testing.T) {
	ok := []string{"abc", "lowercase"}
	fail := []string{"ABC", "123", "a1", ""}

	runMatcherTest(t, isLowerAlpha, ok, fail)
}

func TestIsUpperAlpha(t *testing.T) {
	ok := []string{"ABC", "UPPER"}
	fail := []string{"abc", "123", "A1", ""}
	runMatcherTest(t, isUpperAlpha, ok, fail)
}

func TestIsAlpha(t *testing.T) {
	ok := []string{"abc", "XYZ", "Test"}
	fail := []string{"123", "a1", ""}
	runMatcherTest(t, isAlpha, ok, fail)
}

func TestIsDigits(t *testing.T) {
	ok := []string{"123", "456789"}
	fail := []string{"abc", "12a", ""}
	runMatcherTest(t, isDigits, ok, fail)
}

func TestIsAlNum(t *testing.T) {
	ok := []string{"abc123", "Test1"}
	fail := []string{"!", "-", "", "abc_"}
	runMatcherTest(t, isAlnum, ok, fail)
}

func TestIsWord(t *testing.T) {
	ok := []string{"abc_123", "A1_b"}
	fail := []string{"!", "-", "", "abc-"}
	runMatcherTest(t, isWord, ok, fail)
}

func TestIsSlugSafe(t *testing.T) {
	ok := []string{"abc-123", "A_b-C"}
	fail := []string{"!", "+", "", "abc+"}
	runMatcherTest(t, isSlugSafe, ok, fail)
}

func TestIsSlug(t *testing.T) {
	ok := []string{"abc-123", "a1-b2"}
	fail := []string{"ABC", "_", "", "aB"}
	runMatcherTest(t, isSlug, ok, fail)
}

func TestIsHex(t *testing.T) {
	ok := []string{"deadBEEF", "123abc"}
	fail := []string{"g", "xyz", ""}
	runMatcherTest(t, isHex, ok, fail)
}

func TestIsUUID(t *testing.T) {
	ok := []string{"550e8400-e29b-41d4-a716-446655440000"}
	fail := []string{"550e8400e29b41d4a716446655440000", "", "123"}
	runMatcherTest(t, isUUID, ok, fail)
}

func TestIsSafeText(t *testing.T) {
	ok := []string{"Hello-World_123.txt"}
	fail := []string{"#", "", "fail"}
	runMatcherTest(t, isSafeText, ok, fail)
}

func TestIsUpperAlNum(t *testing.T) {
	ok := []string{"ABC123", "Z9"}
	fail := []string{"abc", "a1", "", "_"}
	runMatcherTest(t, isUpperAlnum, ok, fail)
}

func TestIsBase64(t *testing.T) {
	ok := []string{"dGVzdA==", "QWxhZGRpbjpPcGVuU2VzYW1l"}
	fail := []string{"?", "#", "", "abc-"}
	runMatcherTest(t, isBase64, ok, fail)
}

func TestIsDateYMD(t *testing.T) {
	ok := []string{"2024-04-17", "1999-12-31"}
	fail := []string{"17-04-2024", "20240417", "", "abcd-ef-gh"}
	runMatcherTest(t, isDateYMD, ok, fail)
}

func TestIsSafePath(t *testing.T) {
	ok := []string{"folder/file_name-1.txt"}
	fail := []string{"|", "~", "", "\\"}
	runMatcherTest(t, isSafePath, ok, fail)
}

func TestAlwaysTrue(t *testing.T) {
	ok := []string{"", "anything", "!@#$%^&*()"}
	runMatcherTest(t, alwaysTrue, ok, nil)
}

func runMatcherTest(t *testing.T, fn MatchFunc, shouldMatch, shouldNotMatch []string) {
	for _, input := range shouldMatch {
		if !fn(input) {
			t.Errorf("Expected %q to match but it did not", input)
		}
	}
	for _, input := range shouldNotMatch {
		if fn(input) {
			t.Errorf("Expected %q to not match but it did", input)
		}
	}
}
