package num

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strconv"
	"strings"
)

// ==================== 数值比较 ====================

// Min 求最小值
func Min(nums ...int) int {
	if len(nums) == 0 {
		return 0
	}
	min := nums[0]
	for _, num := range nums[1:] {
		if num < min {
			min = num
		}
	}
	return min
}

// MinFloat 求浮点数最小值
func MinFloat(nums ...float64) float64 {
	if len(nums) == 0 {
		return 0
	}
	min := nums[0]
	for _, num := range nums[1:] {
		if num < min {
			min = num
		}
	}
	return min
}

// Max 求最大值
func Max(nums ...int) int {
	if len(nums) == 0 {
		return 0
	}
	max := nums[0]
	for _, num := range nums[1:] {
		if num > max {
			max = num
		}
	}
	return max
}

// MaxFloat 求浮点数最大值
func MaxFloat(nums ...float64) float64 {
	if len(nums) == 0 {
		return 0
	}
	max := nums[0]
	for _, num := range nums[1:] {
		if num > max {
			max = num
		}
	}
	return max
}

// Clamp 限制数值在指定范围内
func Clamp(num, min, max int) int {
	if num < min {
		return min
	}
	if num > max {
		return max
	}
	return num
}

// ClampFloat 限制浮点数在指定范围内
func ClampFloat(num, min, max float64) float64 {
	if num < min {
		return min
	}
	if num > max {
		return max
	}
	return num
}

// ==================== 数值转换 ====================

// ToInt 转换为整数
func ToInt(v interface{}) (int, error) {
	switch val := v.(type) {
	case int:
		return val, nil
	case int8:
		return int(val), nil
	case int16:
		return int(val), nil
	case int32:
		return int(val), nil
	case int64:
		return int(val), nil
	case uint:
		return int(val), nil
	case uint8:
		return int(val), nil
	case uint16:
		return int(val), nil
	case uint32:
		return int(val), nil
	case uint64:
		return int(val), nil
	case float32:
		return int(val), nil
	case float64:
		return int(val), nil
	case string:
		return strconv.Atoi(val)
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", v)
	}
}

// ToIntDefault 转换为整数，失败返回默认值
func ToIntDefault(v interface{}, defaultValue int) int {
	val, err := ToInt(v)
	if err != nil {
		return defaultValue
	}
	return val
}

// ToFloat 转换为浮点数
func ToFloat(v interface{}) (float64, error) {
	switch val := v.(type) {
	case int:
		return float64(val), nil
	case int8:
		return float64(val), nil
	case int16:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case uint:
		return float64(val), nil
	case uint8:
		return float64(val), nil
	case uint16:
		return float64(val), nil
	case uint32:
		return float64(val), nil
	case uint64:
		return float64(val), nil
	case float32:
		return float64(val), nil
	case float64:
		return val, nil
	case string:
		return strconv.ParseFloat(val, 64)
	case bool:
		if val {
			return 1.0, nil
		}
		return 0.0, nil
	default:
		return 0.0, fmt.Errorf("unsupported type: %T", v)
	}
}

// ToFloatDefault 转换为浮点数，失败返回默认值
func ToFloatDefault(v interface{}, defaultValue float64) float64 {
	val, err := ToFloat(v)
	if err != nil {
		return defaultValue
	}
	return val
}

// ToString 转换为字符串
func ToString(v interface{}) string {
	switch val := v.(type) {
	case int:
		return strconv.Itoa(val)
	case int8:
		return strconv.FormatInt(int64(val), 10)
	case int16:
		return strconv.FormatInt(int64(val), 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint8:
		return strconv.FormatUint(uint64(val), 10)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float32:
		return strconv.FormatFloat(float64(val), 'g', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'g', -1, 64)
	case string:
		return val
	case bool:
		return strconv.FormatBool(val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ==================== 数值格式化 ====================

// FormatNumber 格式化数字
func FormatNumber(num float64, precision int) string {
	return strconv.FormatFloat(num, 'f', precision, 64)
}

// FormatNumberWithCommas 格式化数字（添加千位分隔符）
func FormatNumberWithCommas(num float64) string {
	s := strconv.FormatFloat(num, 'f', -1, 64)
	parts := strings.Split(s, ".")
	integerPart := parts[0]

	var result []byte
	count := 0
	for i := len(integerPart) - 1; i >= 0; i-- {
		if count > 0 && count%3 == 0 {
			result = append([]byte{','}, result...)
		}
		result = append([]byte{integerPart[i]}, result...)
		count++
	}

	if len(parts) > 1 {
		return string(result) + "." + parts[1]
	}
	return string(result)
}

// FormatCurrency 格式化货币
func FormatCurrency(num float64, symbol string) string {
	return symbol + FormatNumberWithCommas(num)
}

// ==================== 数学计算 ====================

// Abs 绝对值
func Abs(num int) int {
	if num < 0 {
		return -num
	}
	return num
}

// AbsFloat 浮点数绝对值
func AbsFloat(num float64) float64 {
	return math.Abs(num)
}

// Round 四舍五入
func Round(num float64) int {
	return int(math.Round(num))
}

// RoundTo 四舍五入到指定小数位
func RoundTo(num float64, precision int) float64 {
	multiplier := math.Pow(10, float64(precision))
	return math.Round(num*multiplier) / multiplier
}

// Floor 向下取整
func Floor(num float64) int {
	return int(math.Floor(num))
}

// Ceil 向上取整
func Ceil(num float64) int {
	return int(math.Ceil(num))
}

// Mod 取模
func Mod(a, b int) int {
	return a % b
}

// ModFloat 浮点数取模
func ModFloat(a, b float64) float64 {
	return math.Mod(a, b)
}

// Pow 幂运算
func Pow(base, exponent float64) float64 {
	return math.Pow(base, exponent)
}

// Sqrt 平方根
func Sqrt(num float64) float64 {
	return math.Sqrt(num)
}

// Cbrt 立方根
func Cbrt(num float64) float64 {
	return math.Cbrt(num)
}

// Sin 正弦
func Sin(angle float64) float64 {
	return math.Sin(angle)
}

// Cos 余弦
func Cos(angle float64) float64 {
	return math.Cos(angle)
}

// Tan 正切
func Tan(angle float64) float64 {
	return math.Tan(angle)
}

// Asin 反正弦
func Asin(value float64) float64 {
	return math.Asin(value)
}

// Acos 反余弦
func Acos(value float64) float64 {
	return math.Acos(value)
}

// Atan 反正切
func Atan(value float64) float64 {
	return math.Atan(value)
}

// Sum 求和
func Sum(nums ...int) int {
	sum := 0
	for _, num := range nums {
		sum += num
	}
	return sum
}

// SumFloat 浮点数求和
func SumFloat(nums ...float64) float64 {
	sum := 0.0
	for _, num := range nums {
		sum += num
	}
	return sum
}

// Average 平均值
func Average(nums ...int) float64 {
	if len(nums) == 0 {
		return 0
	}
	return float64(Sum(nums...)) / float64(len(nums))
}

// AverageFloat 浮点数平均值
func AverageFloat(nums ...float64) float64 {
	if len(nums) == 0 {
		return 0
	}
	return SumFloat(nums...) / float64(len(nums))
}

// ==================== 随机数生成 ====================

// RandomInt 生成指定范围内的随机整数 [min, max]
func RandomInt(min, max int) int {
	return rand.IntN(max-min+1) + min
}

// RandomFloat 生成指定范围内的随机浮点数 [min, max)
func RandomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

// RandomInts 生成指定数量的随机整数
func RandomInts(count, min, max int) []int {
	ints := make([]int, count)
	for i := 0; i < count; i++ {
		ints[i] = RandomInt(min, max)
	}
	return ints
}

// RandomFloats 生成指定数量的随机浮点数
func RandomFloats(count int, min, max float64) []float64 {
	floats := make([]float64, count)
	for i := 0; i < count; i++ {
		floats[i] = RandomFloat(min, max)
	}
	return floats
}

// ==================== 进制转换 ====================

// ToBinary 转换为二进制
func ToBinary(num int) string {
	return strconv.FormatInt(int64(num), 2)
}

// ToOctal 转换为八进制
func ToOctal(num int) string {
	return strconv.FormatInt(int64(num), 8)
}

// ToHex 转换为十六进制
func ToHex(num int) string {
	return strconv.FormatInt(int64(num), 16)
}

// FromBinary 从二进制转换为整数
func FromBinary(s string) (int, error) {
	val, err := strconv.ParseInt(s, 2, 64)
	if err != nil {
		return 0, err
	}
	return int(val), nil
}

// FromOctal 从八进制转换为整数
func FromOctal(s string) (int, error) {
	val, err := strconv.ParseInt(s, 8, 64)
	if err != nil {
		return 0, err
	}
	return int(val), nil
}

// FromHex 从十六进制转换为整数
func FromHex(s string) (int, error) {
	val, err := strconv.ParseInt(s, 16, 64)
	if err != nil {
		return 0, err
	}
	return int(val), nil
}

// ToBase 转换为指定进制
func ToBase(num int, base int) string {
	if base < 2 || base > 36 {
		return ""
	}
	return strconv.FormatInt(int64(num), base)
}

// FromBase 从指定进制转换为整数
func FromBase(s string, base int) (int, error) {
	val, err := strconv.ParseInt(s, base, 64)
	if err != nil {
		return 0, err
	}
	return int(val), nil
}

// ==================== 数值验证 ====================

// IsEven 是否为偶数
func IsEven(num int) bool {
	return num%2 == 0
}

// IsOdd 是否为奇数
func IsOdd(num int) bool {
	return num%2 != 0
}

// IsPositive 是否为正数
func IsPositive(num float64) bool {
	return num > 0
}

// IsNegative 是否为负数
func IsNegative(num float64) bool {
	return num < 0
}

// IsZero 是否为零
func IsZero(num float64) bool {
	return num == 0
}

// IsInteger 是否为整数
func IsInteger(num float64) bool {
	return math.Floor(num) == num
}

// IsPrime 是否为质数
func IsPrime(num int) bool {
	if num <= 1 {
		return false
	}
	if num <= 3 {
		return true
	}
	if num%2 == 0 || num%3 == 0 {
		return false
	}
	for i := 5; i*i <= num; i += 6 {
		if num%i == 0 || num%(i+2) == 0 {
			return false
		}
	}
	return true
}

// IsPerfectSquare 是否为完全平方数
func IsPerfectSquare(num int) bool {
	if num < 0 {
		return false
	}
	sqrt := int(math.Sqrt(float64(num)))
	return sqrt*sqrt == num
}

// ==================== 其他实用功能 ====================

// GCD 最大公约数
func GCD(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// LCM 最小公倍数
func LCM(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	return a * b / GCD(a, b)
}

// Factorial 阶乘
func Factorial(num int) int {
	if num < 0 {
		return 0
	}
	if num == 0 || num == 1 {
		return 1
	}
	result := 1
	for i := 2; i <= num; i++ {
		result *= i
	}
	return result
}

// Fibonacci 斐波那契数列
func Fibonacci(n int) int {
	if n <= 0 {
		return 0
	}
	if n == 1 || n == 2 {
		return 1
	}
	a, b := 1, 1
	for i := 3; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}

// FibonacciSequence 生成斐波那契数列
func FibonacciSequence(n int) []int {
	sequence := make([]int, n)
	if n >= 1 {
		sequence[0] = 1
	}
	if n >= 2 {
		sequence[1] = 1
	}
	for i := 2; i < n; i++ {
		sequence[i] = sequence[i-1] + sequence[i-2]
	}
	return sequence
}

// DigitCount 数字位数
func DigitCount(num int) int {
	if num == 0 {
		return 1
	}
	count := 0
	for num != 0 {
		num /= 10
		count++
	}
	return count
}

// SumOfDigits 数字之和
func SumOfDigits(num int) int {
	sum := 0
	for num != 0 {
		sum += num % 10
		num /= 10
	}
	return sum
}

// ReverseNumber 反转数字
func ReverseNumber(num int) int {
	reversed := 0
	for num != 0 {
		digit := num % 10
		reversed = reversed*10 + digit
		num /= 10
	}
	return reversed
}
