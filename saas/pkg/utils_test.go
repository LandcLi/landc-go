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

// TestValidateConstraintOperators 验证带操作符的约束（委托 model 实现）
func TestValidateConstraintOperators(t *testing.T) {
	data := map[string]interface{}{
		"status": "active",
		"amount": 100,
		"tags":   "a,b,c",
	}

	t.Run("__gt", func(t *testing.T) {
		if !ValidateConstraint(data, map[string]interface{}{"amount": map[string]interface{}{"__gt": 50}}) {
			t.Error("__gt should match")
		}
		if ValidateConstraint(data, map[string]interface{}{"amount": map[string]interface{}{"__gt": 200}}) {
			t.Error("__gt should not match")
		}
	})

	t.Run("__in", func(t *testing.T) {
		if !ValidateConstraint(data, map[string]interface{}{"status": map[string]interface{}{"__in": []interface{}{"active", "disabled"}}}) {
			t.Error("__in should match")
		}
	})

	t.Run("__like", func(t *testing.T) {
		if !ValidateConstraint(data, map[string]interface{}{"tags": map[string]interface{}{"__like": "b"}}) {
			t.Error("__like should match substring")
		}
	})

	t.Run("unknown operator rejected", func(t *testing.T) {
		if ValidateConstraint(data, map[string]interface{}{"amount": map[string]interface{}{"__bogus": 100}}) {
			t.Error("unknown operator should be rejected")
		}
	})
}
