package web

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LandcLi/landc-go/api/core"
	"github.com/LandcLi/landc-go/frame/pkg/meta"
	"github.com/gin-gonic/gin"
)

type ErrorTestController struct{}

type ErrorTestRequest struct {
	meta.Meta `path:"/err/:kind" method:"GET"`
	Kind      string `source:"path" name:"kind"`
}

func (c *ErrorTestController) Test(req *ErrorTestRequest) (*TestResponse, error) {
	switch req.Kind {
	case "unauthorized":
		return nil, core.NewError(core.ErrorCodeUnauthorized, "login required")
	case "notfound":
		return nil, core.ErrNotFound()
	case "custom":
		e, err := core.NewCustomError(60001, "insufficient balance")
		if err != nil {
			return nil, err
		}
		return nil, e
	case "wrapped":
		return nil, fmt.Errorf("wrap: %w", core.NewError(core.ErrorCodeConflict, "conflict"))
	case "plain":
		return nil, errors.New("boom")
	case "panic":
		panic("boom")
	}
	return &TestResponse{ID: 1, Name: req.Kind}, nil
}

func TestHTTPStatusFromCode(t *testing.T) {
	tests := []struct {
		code core.ErrorCode
		want int
	}{
		{core.ErrorCodeBadRequest, 400},
		{core.ErrorCodeInvalidParams, 400},
		{core.ErrorCodeBusinessError, 400},
		{core.ErrorCodeUnauthorized, 401},
		{core.ErrorCodeForbidden, 403},
		{core.ErrorCodeNotFound, 404},
		{core.ErrorCodeMethodNotAllowed, 405},
		{core.ErrorCodeConflict, 409},
		{core.ErrorCodeUnprocessableEntity, 422},
		{core.ErrorCodeValidationFailed, 422},
		{core.ErrorCodeInternalServerError, 500},
		{core.ErrorCodeDatabaseError, 500},
		{core.ErrorCodeNotImplemented, 501},
		{core.ErrorCodeBadGateway, 502},
		{core.ErrorCodeServiceUnavailable, 503},
		{core.ErrorCodeGatewayTimeout, 504},
		{core.ErrorCodeCustomMin, 400},
		{core.ErrorCodeCustomMax, 400},
	}
	for _, tt := range tests {
		if got := httpStatusFromCode(tt.code); got != tt.want {
			t.Errorf("httpStatusFromCode(%d) = %d, want %d", tt.code, got, tt.want)
		}
	}
}

func TestErrorTransmission(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := NewServer(&ServerConfig{Addr: ":8080"})
	if err := server.RegisterHandler(&ErrorTestController{}); err != nil {
		t.Fatalf("register handler: %v", err)
	}
	router := server.Engine()

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "*core.Error 透传 40100 → HTTP 401",
			path:       "/err/unauthorized",
			wantStatus: 401,
			wantBody:   `{"code":40100,"message":"login required"}`,
		},
		{
			name:       "预定义 ErrNotFound 40400 → HTTP 404",
			path:       "/err/notfound",
			wantStatus: 404,
			wantBody:   `{"code":40400,"message":"Not Found"}`,
		},
		{
			name:       "自定义错误 60001 → HTTP 400",
			path:       "/err/custom",
			wantStatus: 400,
			wantBody:   `{"code":60001,"message":"insufficient balance"}`,
		},
		{
			name:       "%w 包装链透传 40900 → HTTP 409",
			path:       "/err/wrapped",
			wantStatus: 409,
			wantBody:   `{"code":40900,"message":"conflict"}`,
		},
		{
			name:       "普通 error → 500 + 50000（不透传内部细节）",
			path:       "/err/plain",
			wantStatus: 500,
			wantBody:   `{"code":50000,"message":"internal server error"}`,
		},
		{
			name:       "panic → 500 + 50000",
			path:       "/err/panic",
			wantStatus: 500,
			wantBody:   `{"code":50000,"message":"internal server error"}`,
		},
		{
			name:       "成功响应不受影响 → 200 + 10000",
			path:       "/err/ok",
			wantStatus: 200,
			wantBody:   `{"code":10000,"data":{"id":1,"name":"ok"},"message":"success"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", tt.path, http.NoBody)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if w.Body.String() != tt.wantBody {
				t.Errorf("body = %s, want %s", w.Body.String(), tt.wantBody)
			}
		})
	}
}
