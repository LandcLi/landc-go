package model

import (
	"testing"
)

func TestParseConstraintModel(t *testing.T) {
	if c, err := ParseConstraint(""); err != nil || c != nil {
		t.Errorf("empty constraint should return nil, got %v err=%v", c, err)
	}
	c, err := ParseConstraint(`{"status":"active","amount":100}`)
	if err != nil || c["status"] != "active" {
		t.Errorf("parse failed: %v err=%v", c, err)
	}
	if _, err := ParseConstraint(`{bad`); err == nil {
		t.Error("invalid json should error")
	}
}

func TestCompareValuesOperatorsModel(t *testing.T) {
	cases := []struct {
		name     string
		actual   interface{}
		expected interface{}
		want     bool
	}{
		{"__eq match", 100, map[string]interface{}{"__eq": 100}, true},
		{"__eq mismatch", 101, map[string]interface{}{"__eq": 100}, false},
		{"__ne", 101, map[string]interface{}{"__ne": 100}, true},
		{"__gt", 101, map[string]interface{}{"__gt": 100}, true},
		{"__gt reject", 99, map[string]interface{}{"__gt": 100}, false},
		{"__gte boundary", 100, map[string]interface{}{"__gte": 100}, true},
		{"__lt", 99, map[string]interface{}{"__lt": 100}, true},
		{"__lte boundary", 100, map[string]interface{}{"__lte": 100}, true},
		{"__in match", "b", map[string]interface{}{"__in": []interface{}{"a", "b"}}, true},
		{"__in miss", "z", map[string]interface{}{"__in": []interface{}{"a", "b"}}, false},
		{"__like", "hello world", map[string]interface{}{"__like": "world"}, true},
		{"__like miss", "hello", map[string]interface{}{"__like": "xyz"}, false},
		{"unknown op", 100, map[string]interface{}{"__bogus": 100}, false},
		{"plain eq", 100, 100, true},
		{"plain ne", 100, 101, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := compareValues(tc.actual, tc.expected); got != tc.want {
				t.Errorf("compareValues(%v, %v) = %v, want %v", tc.actual, tc.expected, got, tc.want)
			}
		})
	}
}

func TestDataAccessCheckConstraint(t *testing.T) {
	t.Run("no constraint always passes", func(t *testing.T) {
		d := &DataAccess{Constraints: ""}
		if !d.CheckConstraint(map[string]interface{}{"x": 1}) {
			t.Error("empty constraint should pass")
		}
	})

	t.Run("satisfied constraint passes", func(t *testing.T) {
		d := &DataAccess{Constraints: `{"status":"active"}`}
		if !d.CheckConstraint(map[string]interface{}{"status": "active"}) {
			t.Error("matching constraint should pass")
		}
	})

	t.Run("unsatisfied constraint fails", func(t *testing.T) {
		d := &DataAccess{Constraints: `{"status":"active"}`}
		if d.CheckConstraint(map[string]interface{}{"status": "disabled"}) {
			t.Error("mismatched constraint should fail")
		}
	})

	t.Run("operator constraint", func(t *testing.T) {
		d := &DataAccess{Constraints: `{"amount":{"__gt":100}}`}
		if !d.CheckConstraint(map[string]interface{}{"amount": 200}) {
			t.Error("__gt constraint should pass")
		}
		if d.CheckConstraint(map[string]interface{}{"amount": 50}) {
			t.Error("__gt constraint should fail")
		}
	})

	t.Run("invalid json rejected", func(t *testing.T) {
		d := &DataAccess{Constraints: `{bad json`}
		if d.CheckConstraint(map[string]interface{}{"x": 1}) {
			t.Error("invalid constraint json must be rejected (fail-closed)")
		}
	})

	t.Run("missing field fails", func(t *testing.T) {
		d := &DataAccess{Constraints: `{"missing":"x"}`}
		if d.CheckConstraint(map[string]interface{}{"other": 1}) {
			t.Error("missing field should fail")
		}
	})
}
