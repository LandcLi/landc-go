package meta_test

import (
	"testing"

	"github.com/LandcLi/landc-go/mvc/pkg/meta"
)

type User struct {
	meta.Meta `meta:"user" db:"mysql" table:"users"`
	ID        int
	Name      string
	Email     string
}

type Product struct {
	meta.Meta `meta:"product" db:"postgres" table:"products" json:"product"`
	ID        int
	Name      string
	Price     float64
}

func TestData(t *testing.T) {
	user := User{}
	data := meta.Data(user)

	if data == nil {
		t.Fatal("Data returned nil")
	}

	if len(data) == 0 {
		t.Fatal("Data returned empty map")
	}

	if data["meta"] != "user" {
		t.Errorf("Expected 'user', got '%s'", data["meta"])
	}

	if data["db"] != "mysql" {
		t.Errorf("Expected 'mysql', got '%s'", data["db"])
	}

	if data["table"] != "users" {
		t.Errorf("Expected 'users', got '%s'", data["table"])
	}
}

func TestGet(t *testing.T) {
	user := User{}

	metaValue := meta.Get(user, "meta")
	if metaValue != "user" {
		t.Errorf("Expected 'user', got '%v'", metaValue)
	}

	dbValue := meta.Get(user, "db")
	if dbValue != "mysql" {
		t.Errorf("Expected 'mysql', got '%v'", dbValue)
	}

	tableValue := meta.Get(user, "table")
	if tableValue != "users" {
		t.Errorf("Expected 'users', got '%v'", tableValue)
	}

	nonExistentValue := meta.Get(user, "nonexistent")
	if nonExistentValue != nil {
		t.Errorf("Expected nil, got '%v'", nonExistentValue)
	}
}

func TestGetWithNil(t *testing.T) {
	result := meta.Get(nil, "meta")
	if result != nil {
		t.Errorf("Expected nil, got '%v'", result)
	}
}

func TestGetWithEmptyKey(t *testing.T) {
	user := User{}
	result := meta.Get(user, "")
	if result != nil {
		t.Errorf("Expected nil, got '%v'", result)
	}
}

func TestDataWithPointer(t *testing.T) {
	user := &User{}
	data := meta.Data(user)

	if data == nil {
		t.Fatal("Data returned nil for pointer")
	}

	if data["meta"] != "user" {
		t.Errorf("Expected 'user', got '%s'", data["meta"])
	}
}

func TestDataWithMultipleStructs(t *testing.T) {
	user := User{}
	product := Product{}

	userData := meta.Data(user)
	productData := meta.Data(product)

	if userData["meta"] != "user" {
		t.Errorf("Expected 'user', got '%s'", userData["meta"])
	}

	if productData["meta"] != "product" {
		t.Errorf("Expected 'product', got '%s'", productData["meta"])
	}

	if userData["db"] != "mysql" {
		t.Errorf("Expected 'mysql', got '%s'", userData["db"])
	}

	if productData["db"] != "postgres" {
		t.Errorf("Expected 'postgres', got '%s'", productData["db"])
	}
}

func TestDataWithNilInput(t *testing.T) {
	data := meta.Data(nil)

	if data == nil {
		t.Fatal("Data returned nil for nil input")
	}

	if len(data) != 0 {
		t.Errorf("Expected empty map, got %d elements", len(data))
	}
}

func TestGetWithNonStruct(t *testing.T) {
	result := meta.Get("string", "meta")
	if result != nil {
		t.Errorf("Expected nil for non-struct input, got '%v'", result)
	}
}
