package converter

import (
	"testing"
)

type User struct {
	Name   string `validate:"required,min=1,max=50"`
	Age    int    `validate:"required,min=0,max=150"`
	Email  string `validate:"required,email"`
	Active bool   `validate:"required"`
	Score  float64
}

type UserDTO struct {
	Name   string `validate:"required"`
	Age    int    `validate:"min=0,max=150"`
	Email  string `validate:"email"`
	Status string
}

type Product struct {
	ID       int     `validate:"required,min=1"`
	Name     string  `validate:"required,min=1,max=100"`
	Price    float64 `validate:"required,min=0"`
	Tags     []string
	Quantity int
}

type ProductDTO struct {
	ID       int     `validate:"required"`
	Title    string  `validate:"required"`
	Cost     float64 `validate:"min=0"`
	Category string
}

func TestNewConverter(t *testing.T) {
	converter := NewConverter()
	if converter == nil {
		t.Error("NewConverter should not return nil")
	}

	if converter.options == nil {
		t.Error("converter options should not be nil")
	}
}

func TestSetOptions(t *testing.T) {
	converter := NewConverter()
	options := &ConvertOptions{
		StrictMode:       true,
		IgnoreEmpty:      true,
		CaseInsensitive:  true,
		Validate:         false,
		FieldNameMapping: map[string]string{"old": "new"},
	}

	converter.SetOptions(options)

	if converter.options.StrictMode != true {
		t.Error("StrictMode should be true")
	}
	if converter.options.IgnoreEmpty != true {
		t.Error("IgnoreEmpty should be true")
	}
	if converter.options.CaseInsensitive != true {
		t.Error("CaseInsensitive should be true")
	}
	if converter.options.Validate != false {
		t.Error("Validate should be false")
	}
}

func TestWithStrictMode(t *testing.T) {
	converter := NewConverter().WithStrictMode(true)
	if converter.options.StrictMode != true {
		t.Error("StrictMode should be true")
	}
}

func TestWithIgnoreEmpty(t *testing.T) {
	converter := NewConverter().WithIgnoreEmpty(true)
	if converter.options.IgnoreEmpty != true {
		t.Error("IgnoreEmpty should be true")
	}
}

func TestWithCaseInsensitive(t *testing.T) {
	converter := NewConverter().WithCaseInsensitive(true)
	if converter.options.CaseInsensitive != true {
		t.Error("CaseInsensitive should be true")
	}
}

func TestWithValidate(t *testing.T) {
	converter := NewConverter().WithValidate(false)
	if converter.options.Validate != false {
		t.Error("Validate should be false")
	}
}

func TestWithFieldMapping(t *testing.T) {
	mapping := map[string]string{"old": "new"}
	converter := NewConverter().WithFieldMapping(mapping)
	if converter.options.FieldNameMapping["old"] != "new" {
		t.Error("FieldMapping should be set")
	}
}

func TestMapToStruct(t *testing.T) {
	converter := NewConverter()

	source := map[string]interface{}{
		"name":   "John Doe",
		"age":    30,
		"email":  "john@example.com",
		"active": true,
		"score":  95.5,
	}

	var target User
	err := converter.MapToStruct(source, &target)
	if err != nil {
		t.Errorf("MapToStruct failed: %v", err)
	}

	if target.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got '%s'", target.Name)
	}
	if target.Age != 30 {
		t.Errorf("Expected Age 30, got %d", target.Age)
	}
	if target.Email != "john@example.com" {
		t.Errorf("Expected Email 'john@example.com', got '%s'", target.Email)
	}
	if target.Active != true {
		t.Errorf("Expected Active true, got %v", target.Active)
	}
	if target.Score != 95.5 {
		t.Errorf("Expected Score 95.5, got %f", target.Score)
	}
}

func TestMapToStructNew(t *testing.T) {
	converter := NewConverter()

	source := map[string]interface{}{
		"name":   "John Doe",
		"age":    30,
		"email":  "john@example.com",
		"active": true,
		"score":  95.5,
	}

	result, err := converter.MapToStructNew(source, User{})
	if err != nil {
		t.Errorf("MapToStructNew failed: %v", err)
	}

	target, ok := result.(*User)
	if !ok {
		t.Error("Result should be *User")
	}

	if target.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got '%s'", target.Name)
	}
	if target.Age != 30 {
		t.Errorf("Expected Age 30, got %d", target.Age)
	}
}

func TestMapToStructWithValidation(t *testing.T) {
	converter := NewConverter()

	source := map[string]interface{}{
		"name":   "",
		"age":    30,
		"email":  "john@example.com",
		"active": true,
	}

	var target User
	err := converter.MapToStruct(source, &target)
	if err == nil {
		t.Error("MapToStruct should fail validation for empty name")
	}
}

func TestMapToStructWithValidationDisabled(t *testing.T) {
	converter := NewConverter().WithValidate(false)

	source := map[string]interface{}{
		"name":   "",
		"age":    30,
		"email":  "john@example.com",
		"active": true,
	}

	var target User
	err := converter.MapToStruct(source, &target)
	if err != nil {
		t.Errorf("MapToStruct should not fail when validation is disabled: %v", err)
	}
}

func TestMapToStructWithInvalidEmail(t *testing.T) {
	converter := NewConverter()

	source := map[string]interface{}{
		"name":   "John Doe",
		"age":    30,
		"email":  "invalid-email",
		"active": true,
	}

	var target User
	err := converter.MapToStruct(source, &target)
	if err == nil {
		t.Error("MapToStruct should fail validation for invalid email")
	}
}

func TestMapToStructStrictMode(t *testing.T) {
	converter := NewConverter().WithStrictMode(true)

	source := map[string]interface{}{
		"name":    "John Doe",
		"age":     30,
		"email":   "john@example.com",
		"active":  true,
		"invalid": "value",
	}

	var target User
	err := converter.MapToStruct(source, &target)
	if err == nil {
		t.Error("MapToStruct should fail in strict mode for unknown field")
	}
}

func TestMapToStructIgnoreEmpty(t *testing.T) {
	converter := NewConverter().WithIgnoreEmpty(true)

	source := map[string]interface{}{
		"name":   "John Doe",
		"age":    30,
		"email":  "",
		"active": true,
	}

	var target User
	err := converter.MapToStruct(source, &target)
	if err != nil {
		t.Errorf("MapToStruct should not fail with empty email when IgnoreEmpty is true: %v", err)
	}

	if target.Email != "" {
		t.Errorf("Expected Email to be empty, got '%s'", target.Email)
	}
}

func TestStructToStruct(t *testing.T) {
	converter := NewConverter()

	source := User{
		Name:   "John Doe",
		Age:    30,
		Email:  "john@example.com",
		Active: true,
		Score:  95.5,
	}

	var target UserDTO
	err := converter.StructToStruct(source, &target)
	if err != nil {
		t.Errorf("StructToStruct failed: %v", err)
	}

	if target.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got '%s'", target.Name)
	}
	if target.Age != 30 {
		t.Errorf("Expected Age 30, got %d", target.Age)
	}
	if target.Email != "john@example.com" {
		t.Errorf("Expected Email 'john@example.com', got '%s'", target.Email)
	}
}

func TestStructToStructNew(t *testing.T) {
	converter := NewConverter()

	source := User{
		Name:   "John Doe",
		Age:    30,
		Email:  "john@example.com",
		Active: true,
		Score:  95.5,
	}

	result, err := converter.StructToStructNew(source, UserDTO{})
	if err != nil {
		t.Errorf("StructToStructNew failed: %v", err)
	}

	target, ok := result.(*UserDTO)
	if !ok {
		t.Error("Result should be *UserDTO")
	}

	if target.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got '%s'", target.Name)
	}
}

func TestMapArrayToStructArray(t *testing.T) {
	converter := NewConverter()

	source := []map[string]interface{}{
		{
			"id":       1,
			"name":     "Product 1",
			"price":    10.99,
			"tags":     []string{"tag1", "tag2"},
			"quantity": 5,
		},
		{
			"id":       2,
			"name":     "Product 2",
			"price":    20.99,
			"tags":     []string{"tag3"},
			"quantity": 10,
		},
	}

	target := make([]Product, 2)
	err := converter.MapArrayToStructArray(source, &target)
	if err != nil {
		t.Errorf("MapArrayToStructArray failed: %v", err)
	}

	if len(target) != 2 {
		t.Errorf("Expected 2 products, got %d", len(target))
	}

	if target[0].ID != 1 {
		t.Errorf("Expected ID 1, got %d", target[0].ID)
	}
	if target[0].Name != "Product 1" {
		t.Errorf("Expected Name 'Product 1', got '%s'", target[0].Name)
	}
	if target[1].ID != 2 {
		t.Errorf("Expected ID 2, got %d", target[1].ID)
	}
}

func TestMapArrayToStructArrayNew(t *testing.T) {
	converter := NewConverter()

	source := []map[string]interface{}{
		{
			"id":       1,
			"name":     "Product 1",
			"price":    10.99,
			"tags":     []string{"tag1", "tag2"},
			"quantity": 5,
		},
		{
			"id":       2,
			"name":     "Product 2",
			"price":    20.99,
			"tags":     []string{"tag3"},
			"quantity": 10,
		},
	}

	result, err := converter.MapArrayToStructArrayNew(source, []Product{})
	if err != nil {
		t.Errorf("MapArrayToStructArrayNew failed: %v", err)
	}

	target, ok := result.([]Product)
	if !ok {
		t.Error("Result should be []Product")
	}

	if len(target) != 2 {
		t.Errorf("Expected 2 products, got %d", len(target))
	}
}

func TestStructArrayToStructArray(t *testing.T) {
	converter := NewConverter().WithFieldMapping(map[string]string{
		"Name":  "Title",
		"Price": "Cost",
	})

	source := []Product{
		{
			ID:       1,
			Name:     "Product 1",
			Price:    10.99,
			Tags:     []string{"tag1", "tag2"},
			Quantity: 5,
		},
		{
			ID:       2,
			Name:     "Product 2",
			Price:    20.99,
			Tags:     []string{"tag3"},
			Quantity: 10,
		},
	}

	target := make([]ProductDTO, 2)
	err := converter.StructArrayToStructArray(source, &target)
	if err != nil {
		t.Errorf("StructArrayToStructArray failed: %v", err)
	}

	if len(target) != 2 {
		t.Errorf("Expected 2 products, got %d", len(target))
	}

	if target[0].ID != 1 {
		t.Errorf("Expected ID 1, got %d", target[0].ID)
	}
	if target[0].Title != "Product 1" {
		t.Errorf("Expected Title 'Product 1', got '%s'", target[0].Title)
	}
}

func TestStructArrayToStructArrayNew(t *testing.T) {
	converter := NewConverter().WithFieldMapping(map[string]string{
		"Name":  "Title",
		"Price": "Cost",
	})

	source := []Product{
		{
			ID:       1,
			Name:     "Product 1",
			Price:    10.99,
			Tags:     []string{"tag1", "tag2"},
			Quantity: 5,
		},
		{
			ID:       2,
			Name:     "Product 2",
			Price:    20.99,
			Tags:     []string{"tag3"},
			Quantity: 10,
		},
	}

	result, err := converter.StructArrayToStructArrayNew(source, []ProductDTO{})
	if err != nil {
		t.Errorf("StructArrayToStructArrayNew failed: %v", err)
	}

	target, ok := result.([]ProductDTO)
	if !ok {
		t.Error("Result should be []ProductDTO")
	}

	if len(target) != 2 {
		t.Errorf("Expected 2 products, got %d", len(target))
	}
}

func TestConvertTo(t *testing.T) {
	converter := NewConverter()

	source := map[string]interface{}{
		"name":   "John Doe",
		"age":    30,
		"email":  "john@example.com",
		"active": true,
	}

	var target User
	err := converter.ConvertTo(source, &target)
	if err != nil {
		t.Errorf("ConvertTo failed: %v", err)
	}

	if target.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got '%s'", target.Name)
	}
}

func TestConvert(t *testing.T) {
	converter := NewConverter()

	source := map[string]interface{}{
		"name":   "John Doe",
		"age":    30,
		"email":  "john@example.com",
		"active": true,
	}

	result, err := converter.Convert(source, User{})
	if err != nil {
		t.Errorf("Convert failed: %v", err)
	}

	target, ok := result.(*User)
	if !ok {
		t.Error("Result should be *User")
	}

	if target.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got '%s'", target.Name)
	}
}

func TestConvertBasicTypes(t *testing.T) {
	converter := NewConverter()

	source := map[string]interface{}{
		"intValue":    "123",
		"floatValue":  "45.67",
		"boolValue":   "true",
		"stringValue": 456,
	}

	type TestStruct struct {
		IntValue    int
		FloatValue  float64
		BoolValue   bool
		StringValue string
	}

	var target TestStruct
	err := converter.MapToStruct(source, &target)
	if err != nil {
		t.Errorf("MapToStruct failed: %v", err)
	}

	if target.IntValue != 123 {
		t.Errorf("Expected IntValue 123, got %d", target.IntValue)
	}
	if target.FloatValue != 45.67 {
		t.Errorf("Expected FloatValue 45.67, got %f", target.FloatValue)
	}
	if target.BoolValue != true {
		t.Errorf("Expected BoolValue true, got %v", target.BoolValue)
	}
	if target.StringValue != "456" {
		t.Errorf("Expected StringValue '456', got '%s'", target.StringValue)
	}
}

func TestConvertWithFieldMapping(t *testing.T) {
	converter := NewConverter().WithFieldMapping(map[string]string{
		"user_name": "Name",
		"user_age":  "Age",
	})

	source := map[string]interface{}{
		"user_name": "John Doe",
		"user_age":  30,
		"email":     "john@example.com",
		"active":    true,
	}

	var target User
	err := converter.MapToStruct(source, &target)
	if err != nil {
		t.Errorf("MapToStruct failed: %v", err)
	}

	if target.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got '%s'", target.Name)
	}
	if target.Age != 30 {
		t.Errorf("Expected Age 30, got %d", target.Age)
	}
}

func TestConvertError(t *testing.T) {
	err := NewConvertError("name", "field is required", nil)
	if err == nil {
		t.Error("NewConvertError should not return nil")
	}

	if err.Field != "name" {
		t.Errorf("Expected Field 'name', got '%s'", err.Field)
	}
	if err.Message != "field is required" {
		t.Errorf("Expected Message 'field is required', got '%s'", err.Message)
	}
}

func TestConvertError_Error(t *testing.T) {
	err := NewConvertError("name", "field is required", "test")
	errorStr := err.Error()

	expected := "field 'name': field is required (value: test)"
	if errorStr != expected {
		t.Errorf("Expected error string '%s', got '%s'", expected, errorStr)
	}
}

func TestConvertWithNilSource(t *testing.T) {
	converter := NewConverter()

	var target User
	err := converter.MapToStruct(nil, &target)
	if err != nil {
		t.Errorf("MapToStruct with nil source should not fail: %v", err)
	}
}

func TestConvertWithEmptyMap(t *testing.T) {
	converter := NewConverter()

	source := map[string]interface{}{}
	var target User
	err := converter.MapToStruct(source, &target)
	if err != nil {
		t.Errorf("MapToStruct with empty map should not fail: %v", err)
	}
}

func TestConvertWithNestedStruct(t *testing.T) {
	converter := NewConverter()

	type Address struct {
		City    string `validate:"required"`
		Country string `validate:"required"`
	}

	type Person struct {
		Name    string `validate:"required"`
		Age     int    `validate:"min=0,max=150"`
		Address Address
	}

	source := map[string]interface{}{
		"name": "John Doe",
		"age":  30,
		"address": map[string]interface{}{
			"city":    "New York",
			"country": "USA",
		},
	}

	var target Person
	err := converter.MapToStruct(source, &target)
	if err != nil {
		t.Errorf("MapToStruct with nested struct failed: %v", err)
	}

	if target.Name != "John Doe" {
		t.Errorf("Expected Name 'John Doe', got '%s'", target.Name)
	}
	if target.Address.City != "New York" {
		t.Errorf("Expected City 'New York', got '%s'", target.Address.City)
	}
	if target.Address.Country != "USA" {
		t.Errorf("Expected Country 'USA', got '%s'", target.Address.Country)
	}
}

func TestConvertWithSlice(t *testing.T) {
	converter := NewConverter()

	type TestStruct struct {
		Tags []string
	}

	source := map[string]interface{}{
		"tags": []string{"tag1", "tag2", "tag3"},
	}

	var target TestStruct
	err := converter.MapToStruct(source, &target)
	if err != nil {
		t.Errorf("MapToStruct with slice failed: %v", err)
	}

	if len(target.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(target.Tags))
	}

	if target.Tags[0] != "tag1" {
		t.Errorf("Expected Tags[0] 'tag1', got '%s'", target.Tags[0])
	}
}

func TestConvertWithInvalidTarget(t *testing.T) {
	converter := NewConverter()

	source := map[string]interface{}{
		"name": "John Doe",
	}

	err := converter.MapToStruct(source, "invalid")
	if err == nil {
		t.Error("MapToStruct with invalid target should fail")
	}
}

func TestConvertWithInvalidTargetType(t *testing.T) {
	converter := NewConverter()

	source := map[string]interface{}{
		"name": "John Doe",
	}

	var target string
	err := converter.MapToStruct(source, &target)
	if err == nil {
		t.Error("MapToStruct with non-struct target should fail")
	}
}
