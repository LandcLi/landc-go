package tag

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected *Tag
	}{
		{
			name:     "simple tag",
			tag:      "json",
			expected: &Tag{Name: "json", Value: "", Options: map[string]string{}, Original: "json"},
		},
		{
			name:     "tag with value",
			tag:      "json:field_name",
			expected: &Tag{Name: "json", Value: "field_name", Options: map[string]string{}, Original: "json:field_name"},
		},
		{
			name:     "tag with options",
			tag:      "json,required",
			expected: &Tag{Name: "json", Value: "", Options: map[string]string{"required": "true"}, Original: "json,required"},
		},
		{
			name:     "tag with value and options",
			tag:      "json:field_name,required,min=1",
			expected: &Tag{Name: "json", Value: "field_name", Options: map[string]string{"required": "true", "min": "1"}, Original: "json:field_name,required,min=1"},
		},
		{
			name:     "ignore tag",
			tag:      "-",
			expected: &Tag{Name: "-", Value: "", Options: map[string]string{}, Original: "-"},
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Parse(tt.tag)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("Parse(%q) = %v, want nil", tt.tag, result)
				}
				return
			}
			if result == nil {
				t.Errorf("Parse(%q) = nil, want %v", tt.tag, tt.expected)
				return
			}
			if result.Name != tt.expected.Name {
				t.Errorf("Parse(%q).Name = %q, want %q", tt.tag, result.Name, tt.expected.Name)
			}
			if result.Value != tt.expected.Value {
				t.Errorf("Parse(%q).Value = %q, want %q", tt.tag, result.Value, tt.expected.Value)
			}
			if len(result.Options) != len(tt.expected.Options) {
				t.Errorf("Parse(%q).Options length = %d, want %d", tt.tag, len(result.Options), len(tt.expected.Options))
			}
		})
	}
}

func TestParseAll(t *testing.T) {
	tests := []struct {
		name     string
		tags     string
		expected int
	}{
		{
			name:     "single tag",
			tags:     "json",
			expected: 1,
		},
		{
			name:     "multiple tags",
			tags:     "json yaml xml",
			expected: 3,
		},
		{
			name:     "tags with options",
			tags:     "json,required yaml xml",
			expected: 4,
		},
		{
			name:     "empty tags",
			tags:     "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseAll(tt.tags)
			if len(result) != tt.expected {
				t.Errorf("ParseAll(%q) length = %d, want %d", tt.tags, len(result), tt.expected)
			}
		})
	}
}

func TestGetTag(t *testing.T) {
	type TestStruct struct {
		Field1 string `json:"field1"`
		Field2 string `json:"field2,omitempty"`
		Field3 string `json:"-"`
	}

	tests := []struct {
		name     string
		field    string
		tagName  string
		expected *Tag
	}{
		{
			name:     "existing tag",
			field:    "Field1",
			tagName:  "json",
			expected: &Tag{Name: "json", Value: "field1", Options: map[string]string{}, Original: "json:field1"},
		},
		{
			name:     "tag with omitempty",
			field:    "Field2",
			tagName:  "json",
			expected: &Tag{Name: "json", Value: "field2", Options: map[string]string{"omitempty": "true"}, Original: "json:field2,omitempty"},
		},
		{
			name:     "ignore tag",
			field:    "Field3",
			tagName:  "json",
			expected: &Tag{Name: "-", Value: "", Options: map[string]string{}, Original: "-"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val := reflect.ValueOf(TestStruct{})
			field, _ := val.Type().FieldByName(tt.field)
			result := GetTag(field, tt.tagName)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("GetTag(%s, %s) = %v, want nil", tt.field, tt.tagName, result)
				}
				return
			}
			if result == nil {
				t.Errorf("GetTag(%s, %s) = nil, want %v", tt.field, tt.tagName, tt.expected)
				return
			}
			if result.Name != tt.expected.Name {
				t.Errorf("GetTag(%s, %s).Name = %q, want %q", tt.field, tt.tagName, result.Name, tt.expected.Name)
			}
		})
	}
}

func TestGetTagValue(t *testing.T) {
	type TestStruct struct {
		Field1 string `json:"field1"`
		Field2 string `json:"field2,omitempty"`
	}

	val := reflect.ValueOf(TestStruct{})
	field1, _ := val.Type().FieldByName("Field1")
	field2, _ := val.Type().FieldByName("Field2")

	tests := []struct {
		name     string
		field    reflect.StructField
		tagName  string
		expected string
	}{
		{
			name:     "simple tag value",
			field:    field1,
			tagName:  "json",
			expected: "field1",
		},
		{
			name:     "tag with omitempty",
			field:    field2,
			tagName:  "json",
			expected: "field2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTagValue(tt.field, tt.tagName)
			if result != tt.expected {
				t.Errorf("GetTagValue(%s, %s) = %q, want %q", tt.field.Name, tt.tagName, result, tt.expected)
			}
		})
	}
}

func TestHasTag(t *testing.T) {
	type TestStruct struct {
		Field1 string `json:"field1"`
		Field2 string `yaml:"field2"`
		Field3 string
	}

	val := reflect.ValueOf(TestStruct{})
	field1, _ := val.Type().FieldByName("Field1")
	field2, _ := val.Type().FieldByName("Field2")
	field3, _ := val.Type().FieldByName("Field3")

	tests := []struct {
		name     string
		field    reflect.StructField
		tagName  string
		expected bool
	}{
		{
			name:     "existing json tag",
			field:    field1,
			tagName:  "json",
			expected: true,
		},
		{
			name:     "existing yaml tag",
			field:    field2,
			tagName:  "yaml",
			expected: true,
		},
		{
			name:     "non-existent tag",
			field:    field3,
			tagName:  "json",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasTag(tt.field, tt.tagName)
			if result != tt.expected {
				t.Errorf("HasTag(%s, %s) = %v, want %v", tt.field.Name, tt.tagName, result, tt.expected)
			}
		})
	}
}

func TestBuild(t *testing.T) {
	tests := []struct {
		name     string
		tagName  string
		value    string
		expected string
	}{
		{
			name:     "simple tag",
			tagName:  "json",
			value:    "field_name",
			expected: "json:field_name",
		},
		{
			name:     "tag without value",
			tagName:  "json",
			value:    "",
			expected: "json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Build(tt.tagName, tt.value)
			if result != tt.expected {
				t.Errorf("Build(%s, %s) = %s, want %s", tt.tagName, tt.value, result, tt.expected)
			}
		})
	}
}

func TestBuildWithOptions(t *testing.T) {
	tests := []struct {
		name     string
		tagName  string
		value    string
		options  map[string]string
		expected string
	}{
		{
			name:     "tag with options",
			tagName:  "json",
			value:    "field_name",
			options:  map[string]string{"required": "true", "min": "1"},
			expected: "json:field_name,required,min=1",
		},
		{
			name:     "tag without options",
			tagName:  "json",
			value:    "field_name",
			options:  nil,
			expected: "json:field_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildWithOptions(tt.tagName, tt.value, tt.options)
			if result != tt.expected {
				t.Errorf("BuildWithOptions(%s, %s, %v) = %s, want %s", tt.tagName, tt.value, tt.options, result, tt.expected)
			}
		})
	}
}

func TestBuildJSONTag(t *testing.T) {
	options := map[string]string{"required": "true", "min": "1"}
	result := BuildJSONTag("field_name", options)
	expected := "json:field_name,required,min=1"
	if result != expected {
		t.Errorf("BuildJSONTag() = %s, want %s", result, expected)
	}
}

func TestBuildYAMLTag(t *testing.T) {
	options := map[string]string{"required": "true"}
	result := BuildYAMLTag("field_name", options)
	expected := "yaml:field_name,required"
	if result != expected {
		t.Errorf("BuildYAMLTag() = %s, want %s", result, expected)
	}
}

func TestBuildXMLTag(t *testing.T) {
	options := map[string]string{"required": "true"}
	result := BuildXMLTag("field_name", options)
	expected := "xml:field_name,required"
	if result != expected {
		t.Errorf("BuildXMLTag() = %s, want %s", result, expected)
	}
}

func TestBuildDBTag(t *testing.T) {
	options := map[string]string{"required": "true"}
	result := BuildDBTag("field_name", options)
	expected := "db:field_name,required"
	if result != expected {
		t.Errorf("BuildDBTag() = %s, want %s", result, expected)
	}
}

func TestBuildFormTag(t *testing.T) {
	options := map[string]string{"required": "true"}
	result := BuildFormTag("field_name", options)
	expected := "form:field_name,required"
	if result != expected {
		t.Errorf("BuildFormTag() = %s, want %s", result, expected)
	}
}

func TestBuildQueryTag(t *testing.T) {
	options := map[string]string{"required": "true"}
	result := BuildQueryTag("field_name", options)
	expected := "query:field_name,required"
	if result != expected {
		t.Errorf("BuildQueryTag() = %s, want %s", result, expected)
	}
}

func TestBuildHeaderTag(t *testing.T) {
	options := map[string]string{"required": "true"}
	result := BuildHeaderTag("field_name", options)
	expected := "header:field_name,required"
	if result != expected {
		t.Errorf("BuildHeaderTag() = %s, want %s", result, expected)
	}
}

func TestBuildURITag(t *testing.T) {
	options := map[string]string{"required": "true"}
	result := BuildURITag(options)
	expected := "uri,required"
	if result != expected {
		t.Errorf("BuildURITag() = %s, want %s", result, expected)
	}
}

func TestBuildURLTag(t *testing.T) {
	options := map[string]string{"required": "true"}
	result := BuildURLTag(options)
	expected := "url,required"
	if result != expected {
		t.Errorf("BuildURLTag() = %s, want %s", result, expected)
	}
}

func TestBuildEmailTag(t *testing.T) {
	options := map[string]string{"required": "true"}
	result := BuildEmailTag(options)
	expected := "email,required"
	if result != expected {
		t.Errorf("BuildEmailTag() = %s, want %s", result, expected)
	}
}

func TestBuildRequiredTag(t *testing.T) {
	result := BuildRequiredTag()
	expected := "required"
	if result != expected {
		t.Errorf("BuildRequiredTag() = %s, want %s", result, expected)
	}
}

func TestBuildOptionalTag(t *testing.T) {
	result := BuildOptionalTag()
	expected := "optional"
	if result != expected {
		t.Errorf("BuildOptionalTag() = %s, want %s", result, expected)
	}
}

func TestBuildMinTag(t *testing.T) {
	result := BuildMinTag(10)
	expected := "min=10"
	if result != expected {
		t.Errorf("BuildMinTag() = %s, want %s", result, expected)
	}
}

func TestBuildMaxTag(t *testing.T) {
	result := BuildMaxTag(100)
	expected := "max=100"
	if result != expected {
		t.Errorf("BuildMaxTag() = %s, want %s", result, expected)
	}
}

func TestBuildLengthTag(t *testing.T) {
	result := BuildLengthTag(50)
	expected := "length=50"
	if result != expected {
		t.Errorf("BuildLengthTag() = %s, want %s", result, expected)
	}
}

func TestBuildPatternTag(t *testing.T) {
	result := BuildPatternTag("^[a-z]+$")
	expected := "pattern=^[a-z]+$"
	if result != expected {
		t.Errorf("BuildPatternTag() = %s, want %s", result, expected)
	}
}

func TestBuildFormatTag(t *testing.T) {
	result := BuildFormatTag("email")
	expected := "format=email"
	if result != expected {
		t.Errorf("BuildFormatTag() = %s, want %s", result, expected)
	}
}

func TestBuildEnumTag(t *testing.T) {
	result := BuildEnumTag("value1", "value2", "value3")
	expected := "enum=value1,value2,value3"
	if result != expected {
		t.Errorf("BuildEnumTag() = %s, want %s", result, expected)
	}
}

func TestBuildAliasTag(t *testing.T) {
	result := BuildAliasTag("field_alias")
	expected := "alias=field_alias"
	if result != expected {
		t.Errorf("BuildAliasTag() = %s, want %s", result, expected)
	}
}

func TestBuildDefaultTag(t *testing.T) {
	result := BuildDefaultTag("default_value")
	expected := "default=default_value"
	if result != expected {
		t.Errorf("BuildDefaultTag() = %s, want %s", result, expected)
	}
}

func TestBuildExampleTag(t *testing.T) {
	result := BuildExampleTag("example_value")
	expected := "example=example_value"
	if result != expected {
		t.Errorf("BuildExampleTag() = %s, want %s", result, expected)
	}
}

func TestBuildDescriptionTag(t *testing.T) {
	result := BuildDescriptionTag("field description")
	expected := "description=field description"
	if result != expected {
		t.Errorf("BuildDescriptionTag() = %s, want %s", result, expected)
	}
}

func TestBuildValidateTag(t *testing.T) {
	result := BuildValidateTag("email")
	expected := "validate=email"
	if result != expected {
		t.Errorf("BuildValidateTag() = %s, want %s", result, expected)
	}
}

func TestAddOmitEmpty(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "add to existing tag",
			tag:      "json:field_name",
			expected: "json:field_name,omitempty",
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: "omitempty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddOmitEmpty(tt.tag)
			if result != tt.expected {
				t.Errorf("AddOmitEmpty(%s) = %s, want %s", tt.tag, result, tt.expected)
			}
		})
	}
}

func TestAddOption(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		option   string
		expected string
	}{
		{
			name:     "add to existing tag",
			tag:      "json:field_name",
			option:   "required",
			expected: "json:field_name,required",
		},
		{
			name:     "empty tag",
			tag:      "",
			option:   "required",
			expected: "required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddOption(tt.tag, tt.option)
			if result != tt.expected {
				t.Errorf("AddOption(%s, %s) = %s, want %s", tt.tag, tt.option, result, tt.expected)
			}
		})
	}
}

func TestAddOptionWithValue(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		option   string
		value    string
		expected string
	}{
		{
			name:     "add to existing tag",
			tag:      "json:field_name",
			option:   "min",
			value:    "1",
			expected: "json:field_name,min=1",
		},
		{
			name:     "empty tag",
			tag:      "",
			option:   "min",
			value:    "1",
			expected: "min=1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddOptionWithValue(tt.tag, tt.option, tt.value)
			if result != tt.expected {
				t.Errorf("AddOptionWithValue(%s, %s, %s) = %s, want %s", tt.tag, tt.option, tt.value, result, tt.expected)
			}
		})
	}
}

func TestIsIgnoreTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected bool
	}{
		{
			name:     "ignore tag",
			tag:      "-",
			expected: true,
		},
		{
			name:     "normal tag",
			tag:      "json:field_name",
			expected: false,
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIgnoreTag(tt.tag)
			if result != tt.expected {
				t.Errorf("IsIgnoreTag(%s) = %v, want %v", tt.tag, result, tt.expected)
			}
		})
	}
}

func TestHasOmitEmpty(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected bool
	}{
		{
			name:     "tag with omitempty",
			tag:      "json:field_name,omitempty",
			expected: true,
		},
		{
			name:     "tag without omitempty",
			tag:      "json:field_name",
			expected: false,
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasOmitEmpty(tt.tag)
			if result != tt.expected {
				t.Errorf("HasOmitEmpty(%s) = %v, want %v", tt.tag, result, tt.expected)
			}
		})
	}
}

func TestGetTagValueFromString(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected string
	}{
		{
			name:     "tag with value",
			tag:      "json:field_name",
			expected: "field_name",
		},
		{
			name:     "tag without value",
			tag:      "json",
			expected: "",
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTagValueFromString(tt.tag)
			if result != tt.expected {
				t.Errorf("GetTagValueFromString(%s) = %s, want %s", tt.tag, result, tt.expected)
			}
		})
	}
}

func TestGetTagOptionsFromString(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected map[string]string
	}{
		{
			name:     "tag with options",
			tag:      "json:field_name,required,min=1",
			expected: map[string]string{"required": "true", "min": "1"},
		},
		{
			name:     "tag without options",
			tag:      "json:field_name",
			expected: map[string]string{},
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetTagOptionsFromString(tt.tag)
			if tt.expected == nil {
				if result != nil {
					t.Errorf("GetTagOptionsFromString(%s) = %v, want nil", tt.tag, result)
				}
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("GetTagOptionsFromString(%s) length = %d, want %d", tt.tag, len(result), len(tt.expected))
			}
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		tag      *Tag
		expected string
	}{
		{
			name:     "normal tag",
			tag:      &Tag{Name: "json", Value: "field_name", Options: map[string]string{"required": "true"}, Original: "json:field_name,required"},
			expected: "json:field_name,required",
		},
		{
			name:     "nil tag",
			tag:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := String(tt.tag)
			if result != tt.expected {
				t.Errorf("String(%v) = %s, want %s", tt.tag, result, tt.expected)
			}
		})
	}
}
