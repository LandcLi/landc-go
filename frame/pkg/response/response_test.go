package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/LandcLi/landc-go/api/core"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestSuccess(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Success(c, gin.H{"hello": "world"})

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp core.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != CodeSuccess {
		t.Errorf("Expected code %d, got %d", CodeSuccess, resp.Code)
	}
	if resp.Message != "success" {
		t.Errorf("Expected message 'success', got '%s'", resp.Message)
	}
	if resp.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestSuccessPage(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	list := []string{"a", "b", "c"}
	SuccessPage(c, list, 100, 1, 10)

	var resp core.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != CodeSuccess {
		t.Errorf("Expected code %d, got %d", CodeSuccess, resp.Code)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected Data to be a map")
	}
	if data["total"].(float64) != 100 {
		t.Errorf("Expected total 100, got %v", data["total"])
	}
}

func TestBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	BadRequest(c, "invalid params")

	var resp core.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != CodeBadRequest {
		t.Errorf("Expected code %d, got %d", CodeBadRequest, resp.Code)
	}
	if resp.Message != "invalid params" {
		t.Errorf("Expected message 'invalid params', got '%s'", resp.Message)
	}
}

func TestUnauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Unauthorized(c, "token expired")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected HTTP 401, got %d", w.Code)
	}

	var resp core.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != CodeUnauthorized {
		t.Errorf("Expected code %d, got %d", CodeUnauthorized, resp.Code)
	}
}

func TestInternalServerError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	InternalServerError(c, "something went wrong")

	var resp core.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != CodeInternalServerError {
		t.Errorf("Expected code %d, got %d", CodeInternalServerError, resp.Code)
	}
}

func TestSuccessWithTraceID(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("trace_id", "test-trace-123")

	Success(c, nil)

	var resp core.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.TraceID != "test-trace-123" {
		t.Errorf("Expected trace_id 'test-trace-123', got '%s'", resp.TraceID)
	}
}

func TestErrorFromCoreError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	apiErr := core.NewError(core.ErrorCodeBusinessError, "business rule violated")
	ErrorFromCoreError(c, apiErr)

	var resp core.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != int(core.ErrorCodeBusinessError) {
		t.Errorf("Expected code %d, got %d", core.ErrorCodeBusinessError, resp.Code)
	}
	if resp.Message != "business rule violated" {
		t.Errorf("Expected message 'business rule violated', got '%s'", resp.Message)
	}
}

func TestDatabaseError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	DatabaseError(c, "connection refused")

	var resp core.Response
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Code != CodeDatabaseError {
		t.Errorf("Expected code %d, got %d", CodeDatabaseError, resp.Code)
	}
}
