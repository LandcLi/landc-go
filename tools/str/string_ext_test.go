package str

import (
	"testing"
)

func TestPadLeft(t *testing.T) {
	if PadLeft("hello", 10, "*") != "*****hello" {
		t.Errorf("PadLeft should return *****hello")
	}
	if PadLeft("hello", 3, "*") != "hello" {
		t.Errorf("PadLeft should return hello")
	}
}

func TestPadRight(t *testing.T) {
	if PadRight("hello", 10, "*") != "hello*****" {
		t.Errorf("PadRight should return hello*****")
	}
	if PadRight("hello", 3, "*") != "hello" {
		t.Errorf("PadRight should return hello")
	}
}

func TestPadCenter(t *testing.T) {
	if PadCenter("hello", 11, "*") != "***hello***" {
		t.Errorf("PadCenter should return ***hello***")
	}
	if PadCenter("hello", 3, "*") != "hello" {
		t.Errorf("PadCenter should return hello")
	}
}

func TestMatches(t *testing.T) {
	if !Matches(`^[a-z]+$`, "hello") {
		t.Errorf("Matches should return true")
	}
	if Matches(`^[a-z]+$`, "hello123") {
		t.Errorf("Matches should return false")
	}
}

func TestReplaceAllFunc(t *testing.T) {
	repl := func(s string) string {
		return "[" + s + "]"
	}
	result := ReplaceAllFunc("hello world", repl, `\w+`)
	if result != "[hello] [world]" {
		t.Errorf("ReplaceAllFunc should return [hello] [world]")
	}
}

func TestReplaceFirst(t *testing.T) {
	if ReplaceFirst("hello hello", "hello", "hi") != "hi hello" {
		t.Errorf("ReplaceFirst should return hi hello")
	}
}

func TestToCamelCase(t *testing.T) {
	if ToCamelCase("hello_world") != "helloWorld" {
		t.Errorf("ToCamelCase should return helloWorld")
	}
	if ToCamelCase("Hello World") != "helloWorld" {
		t.Errorf("ToCamelCase should return helloWorld")
	}
}

func TestToSnakeCase(t *testing.T) {
	if ToSnakeCase("helloWorld") != "hello_world" {
		t.Errorf("ToSnakeCase should return hello_world")
	}
	if ToSnakeCase("Hello World") != "hello_world" {
		t.Errorf("ToSnakeCase should return hello_world")
	}
}

func TestToKebabCase(t *testing.T) {
	if ToKebabCase("helloWorld") != "hello-world" {
		t.Errorf("ToKebabCase should return hello-world")
	}
	if ToKebabCase("Hello World") != "hello-world" {
		t.Errorf("ToKebabCase should return hello-world")
	}
}

func TestReverseWords(t *testing.T) {
	if ReverseWords("hello world") != "world hello" {
		t.Errorf("ReverseWords should return world hello")
	}
}

func TestSplitN(t *testing.T) {
	parts := SplitN("a,b,c,d", ",", 3)
	if len(parts) != 3 || parts[0] != "a" || parts[1] != "b" || parts[2] != "c,d" {
		t.Errorf("SplitN should return [a, b, c,d]")
	}
}

func TestSplitAfter(t *testing.T) {
	parts := SplitAfter("a,b,c", ",")
	if len(parts) != 3 || parts[0] != "a," || parts[1] != "b," || parts[2] != "c" {
		t.Errorf("SplitAfter should return [a,, b,, c]")
	}
}

func TestSplitAfterN(t *testing.T) {
	parts := SplitAfterN("a,b,c,d", ",", 2)
	if len(parts) != 2 || parts[0] != "a," || parts[1] != "b,c,d" {
		t.Errorf("SplitAfterN should return [a,, b,c,d]")
	}
}

func TestSubStringBetween(t *testing.T) {
	if SubStringBetween("hello [world]", "[", "]") != "world" {
		t.Errorf("SubStringBetween should return world")
	}
}

func TestSubStringBefore(t *testing.T) {
	if SubStringBefore("hello world", " ") != "hello" {
		t.Errorf("SubStringBefore should return hello")
	}
}

func TestSubStringAfter(t *testing.T) {
	if SubStringAfter("hello world", " ") != "world" {
		t.Errorf("SubStringAfter should return world")
	}
}

func TestCount(t *testing.T) {
	if Count("hello hello", "hello") != 2 {
		t.Errorf("Count should return 2")
	}
}

func TestCountChars(t *testing.T) {
	if CountChars("hello", 'l') != 2 {
		t.Errorf("CountChars should return 2")
	}
}

func TestCountWords(t *testing.T) {
	if CountWords("hello world") != 2 {
		t.Errorf("CountWords should return 2")
	}
}

func TestIsNumeric(t *testing.T) {
	if !IsNumeric("123") {
		t.Errorf("IsNumeric should return true")
	}
	if IsNumeric("123a") {
		t.Errorf("IsNumeric should return false")
	}
}

func TestIsAlphabetic(t *testing.T) {
	if !IsAlphabetic("hello") {
		t.Errorf("IsAlphabetic should return true")
	}
	if IsAlphabetic("hello123") {
		t.Errorf("IsAlphabetic should return false")
	}
}

func TestIsAlphanumeric(t *testing.T) {
	if !IsAlphanumeric("hello123") {
		t.Errorf("IsAlphanumeric should return true")
	}
	if IsAlphanumeric("hello 123") {
		t.Errorf("IsAlphanumeric should return false")
	}
}

func TestIsChinese(t *testing.T) {
	if !IsChinese("你好") {
		t.Errorf("IsChinese should return true")
	}
	if IsChinese("hello") {
		t.Errorf("IsChinese should return false")
	}
}

func TestIsEmail(t *testing.T) {
	if !IsEmail("test@example.com") {
		t.Errorf("IsEmail should return true")
	}
	if IsEmail("test") {
		t.Errorf("IsEmail should return false")
	}
}

func TestIsURL(t *testing.T) {
	if !IsURL("https://example.com") {
		t.Errorf("IsURL should return true")
	}
	if IsURL("example") {
		t.Errorf("IsURL should return false")
	}
}

func TestTruncate(t *testing.T) {
	if Truncate("hello world", 8, "...") != "hello..." {
		t.Errorf("Truncate should return hello...")
	}
}

func TestAbbreviate(t *testing.T) {
	if Abbreviate("hello world", 8) != "hello..." {
		t.Errorf("Abbreviate should return hello...")
	}
}

func TestCapitalize(t *testing.T) {
	if Capitalize("hello") != "Hello" {
		t.Errorf("Capitalize should return Hello")
	}
}

func TestUncapitalize(t *testing.T) {
	if Uncapitalize("Hello") != "hello" {
		t.Errorf("Uncapitalize should return hello")
	}
}

func TestRemovePrefix(t *testing.T) {
	if RemovePrefix("hello world", "hello ") != "world" {
		t.Errorf("RemovePrefix should return world")
	}
}

func TestRemoveSuffix(t *testing.T) {
	if RemoveSuffix("hello world", " world") != "hello" {
		t.Errorf("RemoveSuffix should return hello")
	}
}

func TestRemoveAll(t *testing.T) {
	if RemoveAll("hello hello", "hello") != " " {
		t.Errorf("RemoveAll should return  ")
	}
}

func TestJoin(t *testing.T) {
	if Join([]string{"a", "b", "c"}, ",") != "a,b,c" {
		t.Errorf("Join should return a,b,c")
	}
}

func TestEqualsIgnoreCase(t *testing.T) {
	if !EqualsIgnoreCase("Hello", "hello") {
		t.Errorf("EqualsIgnoreCase should return true")
	}
	if EqualsIgnoreCase("Hello", "world") {
		t.Errorf("EqualsIgnoreCase should return false")
	}
}

func TestTrimAll(t *testing.T) {
	if TrimAll("   hello   world   ") != "helloworld" {
		t.Errorf("TrimAll should return helloworld")
	}
}

func TestNormalizeSpace(t *testing.T) {
	if NormalizeSpace("   hello   world   ") != "hello world" {
		t.Errorf("NormalizeSpace should return hello world")
	}
}

func TestWrap(t *testing.T) {
	result := Wrap("hello world", 5)
	expected := []string{"hello", " worl", "d"}
	if len(result) != len(expected) {
		t.Errorf("Wrap should return %v", expected)
	}
	for i, s := range expected {
		if result[i] != s {
			t.Errorf("Wrap should return %v", expected)
		}
	}
}
