package num

import (
	"testing"
)

// ==================== 数值比较测试 ====================

func TestMin(t *testing.T) {
	if Min(3, 1, 4, 1, 5) != 1 {
		t.Errorf("Min should return 1")
	}
}

func TestMinFloat(t *testing.T) {
	if MinFloat(3.14, 1.41, 2.71) != 1.41 {
		t.Errorf("MinFloat should return 1.41")
	}
}

func TestMax(t *testing.T) {
	if Max(3, 1, 4, 1, 5) != 5 {
		t.Errorf("Max should return 5")
	}
}

func TestMaxFloat(t *testing.T) {
	if MaxFloat(3.14, 1.41, 2.71) != 3.14 {
		t.Errorf("MaxFloat should return 3.14")
	}
}

func TestClamp(t *testing.T) {
	if Clamp(5, 1, 10) != 5 {
		t.Errorf("Clamp should return 5")
	}
	if Clamp(0, 1, 10) != 1 {
		t.Errorf("Clamp should return 1")
	}
	if Clamp(15, 1, 10) != 10 {
		t.Errorf("Clamp should return 10")
	}
}

func TestClampFloat(t *testing.T) {
	if ClampFloat(5.5, 1.0, 10.0) != 5.5 {
		t.Errorf("ClampFloat should return 5.5")
	}
	if ClampFloat(0.5, 1.0, 10.0) != 1.0 {
		t.Errorf("ClampFloat should return 1.0")
	}
	if ClampFloat(15.5, 1.0, 10.0) != 10.0 {
		t.Errorf("ClampFloat should return 10.0")
	}
}

// ==================== 数值转换测试 ====================

func TestToInt(t *testing.T) {
	val, err := ToInt("123")
	if err != nil || val != 123 {
		t.Errorf("ToInt should return 123")
	}
}

func TestToIntDefault(t *testing.T) {
	if ToIntDefault("123", 0) != 123 {
		t.Errorf("ToIntDefault should return 123")
	}
	if ToIntDefault("abc", 0) != 0 {
		t.Errorf("ToIntDefault should return 0")
	}
}

func TestToFloat(t *testing.T) {
	val, err := ToFloat("123.45")
	if err != nil || val != 123.45 {
		t.Errorf("ToFloat should return 123.45")
	}
}

func TestToFloatDefault(t *testing.T) {
	if ToFloatDefault("123.45", 0.0) != 123.45 {
		t.Errorf("ToFloatDefault should return 123.45")
	}
	if ToFloatDefault("abc", 0.0) != 0.0 {
		t.Errorf("ToFloatDefault should return 0.0")
	}
}

func TestToString(t *testing.T) {
	if ToString(123) != "123" {
		t.Errorf("ToString should return \"123\"")
	}
	if ToString(123.45) != "123.45" {
		t.Errorf("ToString should return \"123.45\"")
	}
}

// ==================== 数值格式化测试 ====================

func TestFormatNumber(t *testing.T) {
	if FormatNumber(123.456, 2) != "123.46" {
		t.Errorf("FormatNumber should return \"123.46\"")
	}
}

func TestFormatNumberWithCommas(t *testing.T) {
	result := FormatNumberWithCommas(1234567.89)
	if result != "1,234,567.89" {
		t.Errorf("FormatNumberWithCommas should return \"1,234,567.89\"")
	}
}

func TestFormatCurrency(t *testing.T) {
	result := FormatCurrency(1234.56, "$")
	if result != "$1,234.56" {
		t.Errorf("FormatCurrency should return \"$1,234.56\"")
	}
}

// ==================== 数学计算测试 ====================

func TestAbs(t *testing.T) {
	if Abs(-5) != 5 {
		t.Errorf("Abs should return 5")
	}
}

func TestAbsFloat(t *testing.T) {
	if AbsFloat(-5.5) != 5.5 {
		t.Errorf("AbsFloat should return 5.5")
	}
}

func TestRound(t *testing.T) {
	if Round(3.6) != 4 {
		t.Errorf("Round should return 4")
	}
	if Round(3.4) != 3 {
		t.Errorf("Round should return 3")
	}
}

func TestRoundTo(t *testing.T) {
	if RoundTo(3.14159, 2) != 3.14 {
		t.Errorf("RoundTo should return 3.14")
	}
}

func TestFloor(t *testing.T) {
	if Floor(3.9) != 3 {
		t.Errorf("Floor should return 3")
	}
}

func TestCeil(t *testing.T) {
	if Ceil(3.1) != 4 {
		t.Errorf("Ceil should return 4")
	}
}

func TestMod(t *testing.T) {
	if Mod(10, 3) != 1 {
		t.Errorf("Mod should return 1")
	}
}

func TestModFloat(t *testing.T) {
	if ModFloat(10.5, 3.0) != 1.5 {
		t.Errorf("ModFloat should return 1.5")
	}
}

func TestSum(t *testing.T) {
	if Sum(1, 2, 3, 4, 5) != 15 {
		t.Errorf("Sum should return 15")
	}
}

func TestSumFloat(t *testing.T) {
	if SumFloat(1.1, 2.2, 3.3) != 6.6 {
		t.Errorf("SumFloat should return 6.6")
	}
}

func TestAverage(t *testing.T) {
	if Average(1, 2, 3, 4, 5) != 3.0 {
		t.Errorf("Average should return 3.0")
	}
}

func TestAverageFloat(t *testing.T) {
	if AverageFloat(1.0, 2.0, 3.0) != 2.0 {
		t.Errorf("AverageFloat should return 2.0")
	}
}

// ==================== 随机数生成测试 ====================

func TestRandomInt(t *testing.T) {
	val := RandomInt(1, 10)
	if val < 1 || val > 10 {
		t.Errorf("RandomInt should return value between 1 and 10")
	}
}

func TestRandomFloat(t *testing.T) {
	val := RandomFloat(1.0, 10.0)
	if val < 1.0 || val > 10.0 {
		t.Errorf("RandomFloat should return value between 1.0 and 10.0")
	}
}

// ==================== 进制转换测试 ====================

func TestToBinary(t *testing.T) {
	if ToBinary(10) != "1010" {
		t.Errorf("ToBinary should return \"1010\"")
	}
}

func TestToOctal(t *testing.T) {
	if ToOctal(10) != "12" {
		t.Errorf("ToOctal should return \"12\"")
	}
}

func TestToHex(t *testing.T) {
	if ToHex(10) != "a" {
		t.Errorf("ToHex should return \"a\"")
	}
}

func TestFromBinary(t *testing.T) {
	val, err := FromBinary("1010")
	if err != nil || val != 10 {
		t.Errorf("FromBinary should return 10")
	}
}

func TestFromOctal(t *testing.T) {
	val, err := FromOctal("12")
	if err != nil || val != 10 {
		t.Errorf("FromOctal should return 10")
	}
}

func TestFromHex(t *testing.T) {
	val, err := FromHex("a")
	if err != nil || val != 10 {
		t.Errorf("FromHex should return 10")
	}
}

// ==================== 数值验证测试 ====================

func TestIsEven(t *testing.T) {
	if !IsEven(4) {
		t.Errorf("IsEven should return true")
	}
	if IsEven(5) {
		t.Errorf("IsEven should return false")
	}
}

func TestIsOdd(t *testing.T) {
	if !IsOdd(5) {
		t.Errorf("IsOdd should return true")
	}
	if IsOdd(4) {
		t.Errorf("IsOdd should return false")
	}
}

func TestIsPrime(t *testing.T) {
	if !IsPrime(5) {
		t.Errorf("IsPrime should return true")
	}
	if IsPrime(4) {
		t.Errorf("IsPrime should return false")
	}
}

func TestIsPerfectSquare(t *testing.T) {
	if !IsPerfectSquare(9) {
		t.Errorf("IsPerfectSquare should return true")
	}
	if IsPerfectSquare(8) {
		t.Errorf("IsPerfectSquare should return false")
	}
}

// ==================== 其他实用功能测试 ====================

func TestGCD(t *testing.T) {
	if GCD(12, 18) != 6 {
		t.Errorf("GCD should return 6")
	}
}

func TestLCM(t *testing.T) {
	if LCM(4, 6) != 12 {
		t.Errorf("LCM should return 12")
	}
}

func TestFactorial(t *testing.T) {
	if Factorial(5) != 120 {
		t.Errorf("Factorial should return 120")
	}
}

func TestFibonacci(t *testing.T) {
	if Fibonacci(5) != 5 {
		t.Errorf("Fibonacci should return 5")
	}
}

func TestDigitCount(t *testing.T) {
	if DigitCount(12345) != 5 {
		t.Errorf("DigitCount should return 5")
	}
}

func TestSumOfDigits(t *testing.T) {
	if SumOfDigits(123) != 6 {
		t.Errorf("SumOfDigits should return 6")
	}
}

func TestReverseNumber(t *testing.T) {
	if ReverseNumber(12345) != 54321 {
		t.Errorf("ReverseNumber should return 54321")
	}
}
