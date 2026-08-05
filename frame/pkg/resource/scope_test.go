package resource

import (
	"context"
	"testing"
)

func TestScopeEmpty(t *testing.T) {
	if !(Scope{}).Empty() {
		t.Error("empty scope should report Empty()=true")
	}
	s := Scope{DB: "x"}
	if s.Empty() {
		t.Error("scope with DB should not be empty")
	}
}

func TestWithFromContext(t *testing.T) {
	ctx := WithScope(context.Background(), Scope{Name: "lum", DB: "lum"})
	s, ok := FromContext(ctx)
	if !ok {
		t.Fatal("FromContext on scoped ctx should return ok=true")
	}
	if s.Name != "lum" || s.DB != "lum" || s.Config != "" {
		t.Fatalf("FromContext = %+v, want Name=lum DB=lum", s)
	}

	if _, ok := FromContext(context.Background()); ok {
		t.Error("plain context should have no scope")
	}
}
