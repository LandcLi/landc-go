// Package tests 提供 landc-go/frame 框架的全链路集成测试
//
// 覆盖链路：
//
//	Config(YAML) → DB(sqlite) → DI → Controller(Meta Tag 路由)
//	→ 中间件(Trace/Auth/CORS/Recovery) → JWT 签发/校验 → 统一响应
package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/LandcLi/landc-go/frame/pkg/auth"
	"github.com/LandcLi/landc-go/frame/pkg/config"
	"github.com/LandcLi/landc-go/frame/pkg/db"
	"github.com/LandcLi/landc-go/frame/pkg/di"
	"github.com/LandcLi/landc-go/frame/pkg/meta"
	"github.com/LandcLi/landc-go/frame/pkg/middleware"
	"github.com/LandcLi/landc-go/frame/pkg/response"
	"github.com/LandcLi/landc-go/frame/pkg/web"
	"github.com/gin-gonic/gin"
	sqlite "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ============================================================
// 业务模型与分层组件（用于全链路测试）
// ============================================================

// Account 业务表模型（对应 DB 层）
type Account struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func (Account) TableName() string { return "accounts" }

// ==================== API 层请求/响应 ====================

type CreateAccountRequest struct {
	meta.Meta `path:"/accounts" method:"POST" description:"创建账户"`
	Username  string `json:"username" binding:"required,min=3"`
	Email     string `json:"email" binding:"required,email"`
}

type GetAccountRequest struct {
	meta.Meta `path:"/accounts/:id" method:"GET" description:"获取账户"`
	ID        uint `source:"path" name:"id"`
}

type ListAccountsRequest struct {
	meta.Meta `path:"/accounts" method:"GET" description:"账户列表"`
	Page      int `source:"query" name:"page" d:"1"`
	Size      int `source:"query" name:"size" d:"20"`
}

type AccountResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// ==================== Service 层接口与实现 ====================

type AccountService interface {
	CreateAccount(username, email string) (*AccountResponse, error)
	GetAccount(id uint) (*AccountResponse, error)
	ListAccounts() ([]*AccountResponse, error)
}

type accountServiceImpl struct {
	db *gorm.DB
}

func newAccountService(database *gorm.DB) AccountService {
	return &accountServiceImpl{db: database}
}

func (s *accountServiceImpl) CreateAccount(username, email string) (*AccountResponse, error) {
	acc := &Account{Username: username, Email: email}
	if err := s.db.Create(acc).Error; err != nil {
		return nil, err
	}
	return &AccountResponse{ID: acc.ID, Username: acc.Username, Email: acc.Email}, nil
}

func (s *accountServiceImpl) GetAccount(id uint) (*AccountResponse, error) {
	var acc Account
	if err := s.db.First(&acc, id).Error; err != nil {
		return nil, err
	}
	return &AccountResponse{ID: acc.ID, Username: acc.Username, Email: acc.Email}, nil
}

func (s *accountServiceImpl) ListAccounts() ([]*AccountResponse, error) {
	var accs []Account
	if err := s.db.Order("id ASC").Find(&accs).Error; err != nil {
		return nil, err
	}
	out := make([]*AccountResponse, 0, len(accs))
	for i := range accs {
		out = append(out, &AccountResponse{ID: accs[i].ID, Username: accs[i].Username, Email: accs[i].Email})
	}
	return out, nil
}

// ==================== Controller 层（Meta Tag 路由） ====================

type AccountController struct {
	service AccountService
}

func (c *AccountController) CreateAccount(req *CreateAccountRequest) (*AccountResponse, error) {
	return c.service.CreateAccount(req.Username, req.Email)
}

func (c *AccountController) GetAccount(req *GetAccountRequest) (*AccountResponse, error) {
	return c.service.GetAccount(req.ID)
}

func (c *AccountController) ListAccounts(req *ListAccountsRequest) ([]*AccountResponse, error) {
	return c.service.ListAccounts()
}

// ============================================================
// 测试辅助
// ============================================================

// yamlConfig 写入临时 YAML 配置并加载
func yamlConfig(t *testing.T) *config.Config {
	t.Helper()
	content := `
server:
  addr: "127.0.0.1"
  port: 8080
  read_timeout: 30
  write_timeout: 30
  use_default_routes: true
  health_check:
    enabled: true
    liveness_path: "/health"
    readiness_path: "/ready"
log:
  level: "warn"
  format: "json"
  output: "stdout"
database:
  driver: "sqlite"
  dsn: ""
jwt:
  secret: "integration-test-secret-key-with-32-chars"
  expire_time: "2h"
  issuer: "landc-integration-test"
`
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.LoadYAMLConfig(path)
	if err != nil {
		t.Fatalf("load yaml config: %v", err)
	}
	return cfg
}

// resetState 清理全局单例，避免测试间污染
func resetState(t *testing.T) {
	t.Helper()
	// DB 与 JWT/Config 的全局实例需重置
	_ = db.Close()
	gin.SetMode(gin.TestMode)
}

// newTestServer 组装完整链路并返回可测试的 engine
func newTestServer(t *testing.T, cfg *config.Config) *gin.Engine {
	t.Helper()

	// 1. Config：注入全局配置
	if err := config.InitGlobalConfigWithConfig(cfg); err != nil {
		t.Fatalf("init global config: %v", err)
	}

	// 2. DB：sqlite 文件库 + 迁移业务表
	sqliteDSN := filepath.Join(t.TempDir(), "test.db")
	database, err := gorm.Open(sqlite.Open(sqliteDSN), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := database.AutoMigrate(&Account{}); err != nil {
		t.Fatalf("migrate account: %v", err)
	}
	db.InitGlobalDBWithObject(database)

	// 3. DI：注册 Service 依赖
	_ = di.Register("account.service", newAccountService(database), true)

	// 4. JWT：初始化签名配置（HS256）
	auth.InitJWT(&auth.JWTConfig{
		Secret:     "integration-test-secret-key-with-32-chars",
		ExpireTime: time.Hour,
		Issuer:     "landc-integration-test",
	})

	// 5. Web Server + 中间件 + Controller 路由
	server := web.NewServer(&web.ServerConfig{Addr: ":0"})
	engine := server.Engine()
	engine.Use(middleware.Trace())
	engine.Use(middleware.Recovery())

	// 受保护分组：需要 JWT
	protected := server.Group("/api/v1", middleware.Auth(), middleware.CORS("https://example.com"))

	ctl := &AccountController{service: mustGetService(t)}
	if err := protected.RegisterHandler(ctl); err != nil {
		t.Fatalf("register handler: %v", err)
	}

	return engine
}

// mustGetService 从 DI 获取 Service
func mustGetService(t *testing.T) AccountService {
	t.Helper()
	srv, err := di.Get("account.service")
	if err != nil {
		t.Fatalf("di get: %v", err)
	}
	return srv.(AccountService)
}

// doRequest 发送 HTTP 请求并返回 recorder
func doRequest(t *testing.T, engine *gin.Engine, method, path, token string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, http.NoBody)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

// parseResponse 解析统一响应结构
func parseResponse(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response %q: %v", w.Body.String(), err)
	}
	return resp
}

// ============================================================
// 链路测试
// ============================================================

// TestFullChainConfigYAML 验证 YAML 配置加载链路
func TestFullChainConfigYAML(t *testing.T) {
	cfg := yamlConfig(t)
	if cfg.Server.Addr != "127.0.0.1" || cfg.Server.Port != 8080 {
		t.Errorf("server config not loaded: %+v", cfg.Server)
	}
	if cfg.Log.Level != "warn" {
		t.Errorf("log level = %q, want warn", cfg.Log.Level)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Errorf("db driver = %q, want sqlite", cfg.Database.Driver)
	}
	if cfg.JWT.Secret != "integration-test-secret-key-with-32-chars" {
		t.Error("jwt secret not loaded from yaml")
	}
}

// TestFullChainJWT 验证 JWT 签发/校验/篡改拒绝
func TestFullChainJWT(t *testing.T) {
	resetState(t)
	cfg := yamlConfig(t)
	auth.InitJWT(&auth.JWTConfig{
		Secret:     cfg.JWT.Secret,
		ExpireTime: time.Hour,
		Issuer:     cfg.JWT.Issuer,
	})

	// 弱密钥拒绝
	if err := auth.ValidateSecret("short"); err == nil {
		t.Fatal("weak secret should be rejected")
	}

	// 签发
	token, err := auth.GenerateToken(42, "tester", "admin")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	// 校验
	claims, err := auth.ParseToken(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.UserID != 42 || claims.Username != "tester" || claims.Role != "admin" {
		t.Errorf("claims mismatch: %+v", claims)
	}

	// 篡改 token 拒绝
	if _, err := auth.ParseToken(token + "x"); err == nil {
		t.Fatal("tampered token should be rejected")
	}
}

// TestFullChainProtectedAPI 验证完整 HTTP 链路：认证 → 路由 → 业务 → 统一响应
func TestFullChainProtectedAPI(t *testing.T) {
	resetState(t)
	cfg := yamlConfig(t)
	engine := newTestServer(t, cfg)

	// ---- 1. 未带 token 访问受保护接口 → 401 ----
	w := doRequest(t, engine, http.MethodGet, "/api/v1/accounts", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("no-token status = %d, want 401", w.Code)
	}

	// ---- 2. 签发 token ----
	token, err := auth.GenerateToken(1, "alice")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	// ---- 3. 创建账户（POST /api/v1/accounts）----
	createBody := []byte(`{"username":"alice","email":"alice@example.com"}`)
	w = doRequest(t, engine, http.MethodPost, "/api/v1/accounts", token, createBody)
	if w.Code != http.StatusOK {
		t.Fatalf("create status = %d, body=%s", w.Code, w.Body.String())
	}
	resp := parseResponse(t, w)
	if code, _ := resp["code"].(float64); int(code) != response.CodeSuccess {
		t.Errorf("create code = %v, want %d", resp["code"], response.CodeSuccess)
	}
	data, _ := resp["data"].(map[string]interface{})
	if data == nil || data["username"] != "alice" {
		t.Errorf("create data missing: %+v", resp["data"])
	}
	accountID := uint(data["id"].(float64))

	// ---- 4. 获取账户（GET /api/v1/accounts/:id）----
	w = doRequest(t, engine, http.MethodGet, "/api/v1/accounts/1", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get status = %d, body=%s", w.Code, w.Body.String())
	}
	resp = parseResponse(t, w)
	if code, _ := resp["code"].(float64); int(code) != response.CodeSuccess {
		t.Errorf("get code = %v", resp["code"])
	}

	// ---- 5. 列表（GET /api/v1/accounts，带默认分页参数）----
	w = doRequest(t, engine, http.MethodGet, "/api/v1/accounts?page=1&size=20", token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list status = %d, body=%s", w.Code, w.Body.String())
	}
	resp = parseResponse(t, w)
	if code, _ := resp["code"].(float64); int(code) != response.CodeSuccess {
		t.Errorf("list code = %v", resp["code"])
	}

	// ---- 6. 非法 token → 401 ----
	w = doRequest(t, engine, http.MethodGet, "/api/v1/accounts", "invalid-token", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("invalid-token status = %d, want 401", w.Code)
	}

	_ = accountID
}

// TestFullChainValidation 验证参数校验失败返回统一错误响应
func TestFullChainValidation(t *testing.T) {
	resetState(t)
	cfg := yamlConfig(t)
	engine := newTestServer(t, cfg)

	token, err := auth.GenerateToken(1, "bob")
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	// 用户名过短 + 邮箱非法 → 校验失败
	badBody := []byte(`{"username":"ab","email":"not-an-email"}`)
	w := doRequest(t, engine, http.MethodPost, "/api/v1/accounts", token, badBody)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("validation status = %d, body=%s", w.Code, w.Body.String())
	}
	resp := parseResponse(t, w)
	if code, _ := resp["code"].(float64); int(code) != response.CodeBadRequest {
		t.Errorf("validation code = %v, want %d", resp["code"], response.CodeBadRequest)
	}
}

// TestFullChainDI 验证 DI 容器服务解析
func TestFullChainDI(t *testing.T) {
	resetState(t)
	cfg := yamlConfig(t)
	newTestServer(t, cfg)

	srv, err := di.Get("account.service")
	if err != nil {
		t.Fatalf("di get: %v", err)
	}
	if _, ok := srv.(AccountService); !ok {
		t.Fatalf("di service wrong type: %T", srv)
	}

	// 未注册服务报错
	if _, err := di.Get("missing.service"); err == nil {
		t.Fatal("missing service should error")
	}
}

// TestFullChainResponseFormat 验证手写 handler 使用 response.Success 的完整统一响应格式
func TestFullChainResponseFormat(t *testing.T) {
	resetState(t)
	cfg := yamlConfig(t)
	engine := newTestServer(t, cfg)

	// 手写 handler 返回统一响应（验证 trace_id/latency/timestamp 完整格式）
	engine.GET("/handwritten", func(c *gin.Context) {
		response.Success(c, map[string]string{"status": "ok"})
	})

	// 客户端回传 trace_id，验证透传
	req := httptest.NewRequest(http.MethodGet, "/handwritten", http.NoBody)
	req.Header.Set("X-Trace-ID", "trace-abc-123")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", w.Code, w.Body.String())
	}
	resp := parseResponse(t, w)
	if code, _ := resp["code"].(float64); int(code) != response.CodeSuccess {
		t.Errorf("code = %v, want %d", resp["code"], response.CodeSuccess)
	}
	if traceID, _ := resp["trace_id"].(string); traceID != "trace-abc-123" {
		t.Errorf("trace_id = %q, want trace-abc-123 (client id passthrough)", traceID)
	}
	if latency, _ := resp["latency"].(float64); latency < 0 {
		t.Error("latency should be non-negative")
	}
	if ts, _ := resp["timestamp"].(string); ts == "" {
		t.Error("timestamp should be present")
	}
}
