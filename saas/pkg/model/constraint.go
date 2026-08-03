package model

import (
	"encoding/json"
	"strings"
)

// ParseConstraint 解析约束条件 JSON
func ParseConstraint(constraintStr string) (map[string]interface{}, error) {
	if constraintStr == "" {
		return nil, nil
	}

	var constraint map[string]interface{}
	err := json.Unmarshal([]byte(constraintStr), &constraint)
	return constraint, err
}

// ValidateConstraint 验证数据是否满足约束条件
func ValidateConstraint(data, constraint map[string]interface{}) bool {
	for key, expected := range constraint {
		actual, ok := data[key]
		if !ok {
			return false
		}

		if !compareValues(actual, expected) {
			return false
		}
	}

	return true
}

// compareValues 比较值
// 支持操作符语法：{"__eq": v} | {"__ne": v} | {"__gt": v} | {"__gte": v} |
// {"__lt": v} | {"__lte": v} | {"__in": [v1, v2, ...]} | {"__like": "prefix"}
// 无操作符时使用等值比较
func compareValues(actual, expected interface{}) bool {
	if opMap, ok := expected.(map[string]interface{}); ok {
		if v, has := opMap["__eq"]; has {
			return actual == v
		}
		if v, has := opMap["__ne"]; has {
			return actual != v
		}
		if v, has := opMap["__in"]; has {
			return valueIn(actual, v)
		}
		if v, has := opMap["__like"]; has {
			str, ok := toString(actual)
			if !ok {
				return false
			}
			return strings.Contains(str, v.(string))
		}
		if v, has := opMap["__gt"]; has {
			return compareNumeric(actual, v) > 0
		}
		if v, has := opMap["__gte"]; has {
			return compareNumeric(actual, v) >= 0
		}
		if v, has := opMap["__lt"]; has {
			return compareNumeric(actual, v) < 0
		}
		if v, has := opMap["__lte"]; has {
			return compareNumeric(actual, v) <= 0
		}
		// 未知操作符：保守拒绝
		return false
	}

	return actual == expected
}

// valueIn 判断 actual 是否在列表中
func valueIn(actual, list interface{}) bool {
	switch l := list.(type) {
	case []interface{}:
		for _, item := range l {
			if actual == item {
				return true
			}
		}
	case []string:
		for _, item := range l {
			if actual == item {
				return true
			}
		}
	case []int:
		for _, item := range l {
			if actual == item {
				return true
			}
		}
	case []float64:
		for _, item := range l {
			if actual == item {
				return true
			}
		}
	}
	return false
}

// compareNumeric 数值比较（-1, 0, 1）
func compareNumeric(actual, expected interface{}) int {
	a, ok := toFloat64(actual)
	if !ok {
		return -2 // 无法转换
	}
	b, ok := toFloat64(expected)
	if !ok {
		return 2
	}
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// toFloat64 将数字类型转为 float64
func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

// toString 将字符串类型取出
func toString(v interface{}) (string, bool) {
	s, ok := v.(string)
	return s, ok
}
