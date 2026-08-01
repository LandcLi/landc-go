package saas

import (
	"testing"
)

func TestParseConstraint(t *testing.T) {
	t.Run("empty string returns nil", func(t *testing.T) {
		c, err := ParseConstraint("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c != nil {
			t.Errorf("expected nil constraint, got %v", c)
		}
	})

	t.Run("valid json", func(t *testing.T) {
		c, err := ParseConstraint(`{"status":"active","amount":100}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c["status"] != "active" {
			t.Errorf("expected status=active, got %v", c["status"])
		}
	})

	t.Run("invalid json returns error", func(t *testing.T) {
		if _, err := ParseConstraint(`{invalid`); err == nil {
			t.Error("expected error for invalid json")
		}
	})
}

func TestValidateConstraint(t *testing.T) {
	data := map[string]interface{}{
		"status": "active",
		"amount": 100,
		"tags":   "a,b,c",
	}

	t.Run("simple equality pass", func(t *testing.T) {
		ok := ValidateConstraint(data, map[string]interface{}{"status": "active"})
		if !ok {
			t.Error("expected constraint to pass")
		}
	})

	t.Run("missing field fails", func(t *testing.T) {
		ok := ValidateConstraint(data, map[string]interface{}{"not_exist": "x"})
		if ok {
			t.Error("expected constraint to fail on missing field")
		}
	})

	t.Run("mismatch fails", func(t *testing.T) {
		ok := ValidateConstraint(data, map[string]interface{}{"status": "disabled"})
		if ok {
			t.Error("expected constraint to fail on mismatch")
		}
	})

	t.Run("multiple conditions all must pass", func(t *testing.T) {
		ok := ValidateConstraint(data, map[string]interface{}{
			"status": "active",
			"amount": 100,
		})
		if !ok {
			t.Error("expected all conditions to pass")
		}
	})
}

func TestCompareValuesOperators(t *testing.T) {
	t.Run("__eq", func(t *testing.T) {
		if !compareValues(100, map[string]interface{}{"__eq": 100}) {
			t.Error("__eq should match")
		}
		if compareValues(101, map[string]interface{}{"__eq": 100}) {
			t.Error("__eq should not match")
		}
	})

	t.Run("__ne", func(t *testing.T) {
		if !compareValues(101, map[string]interface{}{"__ne": 100}) {
			t.Error("__ne should match different value")
		}
	})

	t.Run("__gt __gte __lt __lte", func(t *testing.T) {
		if !compareValues(101, map[string]interface{}{"__gt": 100}) {
			t.Error("__gt should match")
		}
		if !compareValues(100, map[string]interface{}{"__gte": 100}) {
			t.Error("__gte should match boundary")
		}
		if !compareValues(99, map[string]interface{}{"__lt": 100}) {
			t.Error("__lt should match")
		}
		if !compareValues(100, map[string]interface{}{"__lte": 100}) {
			t.Error("__lte should match boundary")
		}
		if compareValues(99, map[string]interface{}{"__gt": 100}) {
			t.Error("__gt should not match smaller")
		}
	})

	t.Run("__in", func(t *testing.T) {
		if !compareValues("b", map[string]interface{}{"__in": []interface{}{"a", "b", "c"}}) {
			t.Error("__in should match")
		}
		if compareValues("d", map[string]interface{}{"__in": []interface{}{"a", "b", "c"}}) {
			t.Error("__in should not match")
		}
	})

	t.Run("__like", func(t *testing.T) {
		if !compareValues("hello world", map[string]interface{}{"__like": "world"}) {
			t.Error("__like should match substring")
		}
		if compareValues("hello", map[string]interface{}{"__like": "xyz"}) {
			t.Error("__like should not match")
		}
	})

	t.Run("unknown operator rejected", func(t *testing.T) {
		if compareValues(100, map[string]interface{}{"__bogus": 100}) {
			t.Error("unknown operator should be rejected")
		}
	})
}
