package di

import (
	"errors"
	"testing"
)

type (
	TestService interface {
		Hello(name string) string
	}

	testServiceImpl struct{}
)

func (s *testServiceImpl) Hello(name string) string {
	return "Hello, " + name
}

func TestContainer_Register(t *testing.T) {
	container := NewContainer()

	service := &testServiceImpl{}
	err := container.Register("test", service, false)
	if err != nil {
		t.Fatalf("Failed to register service: %v", err)
	}

	if !container.Has("test") {
		t.Error("Service should be registered")
	}

	err = container.Register("test", service, false)
	if err == nil {
		t.Error("Should not allow duplicate registration without overwrite")
	}

	err = container.Register("test", service, true)
	if err != nil {
		t.Errorf("Should allow overwrite when overwrite is true: %v", err)
	}
}

func TestContainer_Get(t *testing.T) {
	container := NewContainer()

	service := &testServiceImpl{}
	container.Register("test", service, false)

	retrieved, err := container.Get("test")
	if err != nil {
		t.Fatalf("Failed to get service: %v", err)
	}

	if retrieved != service {
		t.Error("Retrieved service should be the same instance")
	}

	_, err = container.Get("nonexistent")
	if err == nil {
		t.Error("Should return error for nonexistent service")
	}
}

func TestContainer_MustGet(t *testing.T) {
	container := NewContainer()

	service := &testServiceImpl{}
	container.Register("test", service, false)

	retrieved := container.MustGet("test")
	if retrieved != service {
		t.Error("MustGet should return the same instance")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustGet should panic for nonexistent service")
		}
	}()
	container.MustGet("nonexistent")
}

func TestGlobalContainer(t *testing.T) {
	service := &testServiceImpl{}
	err := Register("global_test", service, false)
	if err != nil {
		t.Fatalf("Failed to register global service: %v", err)
	}

	retrieved, err := Get("global_test")
	if err != nil {
		t.Fatalf("Failed to get global service: %v", err)
	}

	if retrieved != service {
		t.Error("Retrieved global service should be the same instance")
	}
}

func TestRegisterInterface(t *testing.T) {
	service := &testServiceImpl{}
	err := RegisterInterface[TestService]("interface_test", service, false)
	if err != nil {
		t.Fatalf("Failed to register interface: %v", err)
	}

	retrieved, err := GetInterface[TestService]("interface_test")
	if err != nil {
		t.Fatalf("Failed to get interface: %v", err)
	}

	result := retrieved.Hello("World")
	if result != "Hello, World" {
		t.Errorf("Expected 'Hello, World', got '%s'", result)
	}
}

func TestRegisterSingleton(t *testing.T) {
	callCount := 0
	factory := func() *testServiceImpl {
		callCount++
		return &testServiceImpl{}
	}

	err := RegisterSingleton("singleton_test", factory, false)
	if err != nil {
		t.Fatalf("Failed to register singleton: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Factory should be called once, called %d times", callCount)
	}

	retrieved, err := GetInterface[*testServiceImpl]("singleton_test")
	if err != nil {
		t.Fatalf("Failed to get singleton: %v", err)
	}

	if retrieved == nil {
		t.Error("Singleton should not be nil")
	}
}

func TestCallMethod(t *testing.T) {
	service := &testServiceImpl{}

	results, err := CallMethod(service, "Hello", "Test")
	if err != nil {
		t.Fatalf("Failed to call method: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(results))
	}

	result, ok := results[0].(string)
	if !ok {
		t.Error("Result should be a string")
	}

	if result != "Hello, Test" {
		t.Errorf("Expected 'Hello, Test', got '%s'", result)
	}

	_, err = CallMethod(service, "NonExistentMethod")
	if err == nil {
		t.Error("Should return error for nonexistent method")
	}
}

type (
	ErrorService interface {
		Fail() error
	}

	errorServiceImpl struct{}
)

func (s *errorServiceImpl) Fail() error {
	return errors.New("test error")
}

func TestErrorHandling(t *testing.T) {
	service := &errorServiceImpl{}
	err := RegisterInterface[ErrorService]("error_test", service, false)
	if err != nil {
		t.Fatalf("Failed to register error service: %v", err)
	}

	retrieved, err := GetInterface[ErrorService]("error_test")
	if err != nil {
		t.Fatalf("Failed to get error service: %v", err)
	}

	err = retrieved.Fail()
	if err == nil {
		t.Error("Service should return an error")
	}
}
