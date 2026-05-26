package str

import (
	"testing"
)

func TestIsEmpty(t *testing.T) {
	if !IsEmpty("") {
		t.Errorf("IsEmpty should return true")
	}
	if IsEmpty("hello") {
		t.Errorf("IsEmpty should return false")
	}
}

func TestIsNotEmpty(t *testing.T) {
	if IsNotEmpty("") {
		t.Errorf("IsNotEmpty should return false")
	}
	if !IsNotEmpty("hello") {
		t.Errorf("IsNotEmpty should return true")
	}
}

func TestIsBlank(t *testing.T) {
	if !IsBlank("") {
		t.Errorf("IsBlank should return true")
	}
	if !IsBlank("   ") {
		t.Errorf("IsBlank should return true")
	}
	if IsBlank("hello") {
		t.Errorf("IsBlank should return false")
	}
}

func TestIsNotBlank(t *testing.T) {
	if IsNotBlank("") {
		t.Errorf("IsNotBlank should return false")
	}
	if IsNotBlank("   ") {
		t.Errorf("IsNotBlank should return false")
	}
	if !IsNotBlank("hello") {
		t.Errorf("IsNotBlank should return true")
	}
}

func TestLength(t *testing.T) {
	if Length("") != 0 {
		t.Errorf("Length should return 0")
	}
	if Length("hello") != 5 {
		t.Errorf("Length should return 5")
	}
	if Length("你好") != 2 {
		t.Errorf("Length should return 2")
	}
}

func TestSubString(t *testing.T) {
	if SubString("hello", 0, 3) != "hel" {
		t.Errorf("SubString should return hel")
	}
	if SubString("hello", 2, 10) != "llo" {
		t.Errorf("SubString should return llo")
	}
	if SubString("你好世界", 1, 3) != "好世" {
		t.Errorf("SubString should return 好世")
	}
}

func TestSubStringStart(t *testing.T) {
	if SubStringStart("hello", 2) != "llo" {
		t.Errorf("SubStringStart should return llo")
	}
	if SubStringStart("你好世界", 1) != "好世界" {
		t.Errorf("SubStringStart should return 好世界")
	}
}

func TestConcat(t *testing.T) {
	if Concat("hello", " ", "world") != "hello world" {
		t.Errorf("Concat should return hello world")
	}
}

func TestReplace(t *testing.T) {
	if Replace("hello world", "world", "go", 1) != "hello go" {
		t.Errorf("Replace should return hello go")
	}
}

func TestReplaceAll(t *testing.T) {
	if ReplaceAll("hello hello", "hello", "hi") != "hi hi" {
		t.Errorf("ReplaceAll should return hi hi")
	}
}

func TestSplit(t *testing.T) {
	parts := Split("a,b,c", ",")
	if len(parts) != 3 || parts[0] != "a" || parts[1] != "b" || parts[2] != "c" {
		t.Errorf("Split should return [a, b, c]")
	}
}

func TestSplitTrim(t *testing.T) {
	parts := SplitTrim(" a , b , c ", ",")
	if len(parts) != 3 || parts[0] != "a" || parts[1] != "b" || parts[2] != "c" {
		t.Errorf("SplitTrim should return [a, b, c]")
	}
}

func TestToUpper(t *testing.T) {
	if ToUpper("hello") != "HELLO" {
		t.Errorf("ToUpper should return HELLO")
	}
}

func TestToLower(t *testing.T) {
	if ToLower("HELLO") != "hello" {
		t.Errorf("ToLower should return hello")
	}
}

func TestToTitle(t *testing.T) {
	if ToTitle("hello world") != "Hello World" {
		t.Errorf("ToTitle should return Hello World")
	}
}

func TestTrim(t *testing.T) {
	if Trim("   hello   ") != "hello" {
		t.Errorf("Trim should return hello")
	}
}

func TestTrimLeft(t *testing.T) {
	if TrimLeft("   hello   ") != "hello   " {
		t.Errorf("TrimLeft should return hello   ")
	}
}

func TestTrimRight(t *testing.T) {
	if TrimRight("   hello   ") != "   hello" {
		t.Errorf("TrimRight should return   hello")
	}
}

func TestContains(t *testing.T) {
	if !Contains("hello world", "world") {
		t.Errorf("Contains should return true")
	}
	if Contains("hello world", "go") {
		t.Errorf("Contains should return false")
	}
}

func TestStartsWith(t *testing.T) {
	if !StartsWith("hello world", "hello") {
		t.Errorf("StartsWith should return true")
	}
	if StartsWith("hello world", "world") {
		t.Errorf("StartsWith should return false")
	}
}

func TestEndsWith(t *testing.T) {
	if !EndsWith("hello world", "world") {
		t.Errorf("EndsWith should return true")
	}
	if EndsWith("hello world", "hello") {
		t.Errorf("EndsWith should return false")
	}
}

func TestIndexOf(t *testing.T) {
	if IndexOf("hello world", "world") != 6 {
		t.Errorf("IndexOf should return 6")
	}
	if IndexOf("hello world", "go") != -1 {
		t.Errorf("IndexOf should return -1")
	}
}

func TestLastIndexOf(t *testing.T) {
	if LastIndexOf("hello hello", "hello") != 6 {
		t.Errorf("LastIndexOf should return 6")
	}
	if LastIndexOf("hello hello", "go") != -1 {
		t.Errorf("LastIndexOf should return -1")
	}
}

func TestRepeat(t *testing.T) {
	if Repeat("hello", 3) != "hellohellohello" {
		t.Errorf("Repeat should return hellohellohello")
	}
}

func TestReverse(t *testing.T) {
	if Reverse("hello") != "olleh" {
		t.Errorf("Reverse should return olleh")
	}
	if Reverse("你好") != "好你" {
		t.Errorf("Reverse should return 好你")
	}
}
