package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMemoryStore_Basic(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	// Set
	data := map[string]interface{}{"user": "alice", "age": 25}
	if err := store.Set(ctx, "session1", data, 10*time.Minute); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get
	got, err := store.Get(ctx, "session1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got["user"] != "alice" {
		t.Errorf("expected user='alice', got '%v'", got["user"])
	}

	// Exists
	exists, _ := store.Exists(ctx, "session1")
	if !exists {
		t.Error("expected session1 to exist")
	}

	// Delete
	store.Delete(ctx, "session1")
	exists, _ = store.Exists(ctx, "session1")
	if exists {
		t.Error("expected session1 to be deleted")
	}
}

func TestMemoryStore_Expiration(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	store.Set(ctx, "expired", map[string]interface{}{"x": 1}, 50*time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	got, _ := store.Get(ctx, "expired")
	if got != nil {
		t.Error("expected nil for expired session")
	}

	exists, _ := store.Exists(ctx, "expired")
	if exists {
		t.Error("expected expired session to not exist")
	}
}

func TestSessionMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := NewMemoryStore()
	config := &Config{
		CookieName: "test_session",
		MaxAge:     1 * time.Hour,
	}

	r := gin.New()
	r.Use(Middleware(store, config))

	r.POST("/login", func(c *gin.Context) {
		sess := FromContext(c)
		sess.Set("user_id", "12345")
		sess.Set("username", "alice")
		c.String(200, "logged in")
	})

	r.GET("/profile", func(c *gin.Context) {
		sess := FromContext(c)
		username := sess.GetString("username")
		c.String(200, username)
	})

	r.POST("/logout", func(c *gin.Context) {
		sess := FromContext(c)
		sess.Clear()
		c.String(200, "logged out")
	})

	// 1. 登录
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/login", http.NoBody)
	r.ServeHTTP(w1, req1)

	if w1.Code != 200 {
		t.Fatalf("login: expected 200, got %d", w1.Code)
	}

	// 获取 cookie
	var sessionCookie *http.Cookie
	for _, c := range w1.Result().Cookies() {
		if c.Name == "test_session" {
			sessionCookie = c
			break
		}
	}
	// 也检查 Set-Cookie header
	if sessionCookie == nil {
		setCookie := w1.Header().Get("Set-Cookie")
		if setCookie == "" {
			t.Fatalf("expected session cookie to be set, headers: %v", w1.Header())
		}
		// 手动解析
		resp := &http.Response{Header: w1.Header()}
		for _, c := range resp.Cookies() {
			if c.Name == "test_session" {
				sessionCookie = c
				break
			}
		}
		if sessionCookie == nil {
			t.Fatalf("expected session cookie to be set in Set-Cookie header: %s", setCookie)
		}
	}

	// 2. 访问 profile（带 session cookie）
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/profile", http.NoBody)
	req2.AddCookie(sessionCookie)
	r.ServeHTTP(w2, req2)

	if w2.Body.String() != "alice" {
		t.Errorf("expected 'alice', got '%s'", w2.Body.String())
	}

	// 3. 登出
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("POST", "/logout", http.NoBody)
	req3.AddCookie(sessionCookie)
	r.ServeHTTP(w3, req3)

	// 4. 再次访问 profile（session 已清空）
	w4 := httptest.NewRecorder()
	req4, _ := http.NewRequest("GET", "/profile", http.NoBody)
	req4.AddCookie(sessionCookie)
	r.ServeHTTP(w4, req4)

	if w4.Body.String() != "" {
		t.Errorf("expected empty after logout, got '%s'", w4.Body.String())
	}
}

func TestSession_GetInt(t *testing.T) {
	sess := &Session{
		data: map[string]interface{}{
			"count":  float64(42), // JSON 反序列化后的数字是 float64
			"str":    "hello",
			"intVal": 10,
		},
	}

	if got := sess.GetInt("count"); got != 42 {
		t.Errorf("expected 42, got %d", got)
	}

	if got := sess.GetInt("intVal"); got != 10 {
		t.Errorf("expected 10, got %d", got)
	}

	if got := sess.GetInt("str"); got != 0 {
		t.Errorf("expected 0 for non-int, got %d", got)
	}

	if got := sess.GetInt("nonexist"); got != 0 {
		t.Errorf("expected 0 for missing key, got %d", got)
	}
}

func TestSession_All(t *testing.T) {
	sess := &Session{
		data: map[string]interface{}{
			"a": 1,
			"b": "two",
		},
	}

	all := sess.All()
	if len(all) != 2 {
		t.Errorf("expected 2 entries, got %d", len(all))
	}

	// 修改副本不应影响原始数据
	all["c"] = "three"
	if _, ok := sess.data["c"]; ok {
		t.Error("modifying All() result should not affect session data")
	}
}

func TestMemoryStore_Size(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	store.Set(ctx, "s1", map[string]interface{}{"a": 1}, 10*time.Minute)
	store.Set(ctx, "s2", map[string]interface{}{"b": 2}, 10*time.Minute)

	if store.Size() != 2 {
		t.Errorf("expected size 2, got %d", store.Size())
	}
}
