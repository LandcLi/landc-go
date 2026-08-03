package tag

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	JSONTag        = "json"
	YAMLTag        = "yaml"
	XMLTag         = "xml"
	DBTag          = "db"
	FormTag        = "form"
	QueryTag       = "query"
	HeaderTag      = "header"
	URITag         = "uri"
	URLTag         = "url"
	EmailTag       = "email"
	RequiredTag    = "required"
	OptionalTag    = "optional"
	ValidateTag    = "validate"
	ExampleTag     = "example"
	DescriptionTag = "description"
	DefaultTag     = "default"
	MinTag         = "min"
	MaxTag         = "max"
	LengthTag      = "length"
	PatternTag     = "pattern"
	FormatTag      = "format"
	EnumTag        = "enum"
	AliasTag       = "alias"
	IgnoreTag      = "-"
	OmitEmptyTag   = "omitempty"
)

type Tag struct {
	Name     string
	Value    string
	Options  map[string]string
	Original string
}

func Parse(tag string) *Tag {
	if tag == "" {
		return nil
	}

	t := &Tag{
		Original: tag,
		Options:  make(map[string]string),
	}

	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return nil
	}

	mainPart := strings.TrimSpace(parts[0])
	if mainPart == "-" {
		t.Name = IgnoreTag
		return t
	}

	if idx := strings.Index(mainPart, ":"); idx != -1 {
		t.Name = strings.TrimSpace(mainPart[:idx])
		t.Value = strings.TrimSpace(mainPart[idx+1:])
	} else {
		// 对于没有冒号的tag，将整个部分作为Name
		// 这是为了支持验证规则如"required"、"email"等
		t.Name = strings.TrimSpace(mainPart)
	}

	for i := 1; i < len(parts); i++ {
		opt := strings.TrimSpace(parts[i])
		if idx := strings.Index(opt, "="); idx != -1 {
			key := strings.TrimSpace(opt[:idx])
			value := strings.TrimSpace(opt[idx+1:])
			t.Options[key] = value
		} else {
			t.Options[opt] = "true"
		}
	}

	return t
}

func ParseAll(tags string) []*Tag {
	if tags == "" {
		return nil
	}

	result := make([]*Tag, 0)
	// 支持逗号和空格分隔
	parts := strings.FieldsFunc(tags, func(r rune) bool {
		return r == ' ' || r == ','
	})

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			if t := Parse(part); t != nil {
				result = append(result, t)
			}
		}
	}

	return result
}

func GetTag(field reflect.StructField, tagName string) *Tag {
	tagStr := field.Tag.Get(tagName)
	if tagStr == "" {
		return nil
	}

	// 检查是否为忽略标签
	if tagStr == "-" {
		return &Tag{
			Name:     IgnoreTag,
			Original: tagStr,
			Options:  make(map[string]string),
		}
	}

	// 解析标签值和选项
	parts := strings.Split(tagStr, ",")
	mainPart := strings.TrimSpace(parts[0])

	tag := &Tag{
		Name:     tagName,
		Value:    mainPart,
		Original: tagStr,
		Options:  make(map[string]string),
	}

	// 解析选项
	for i := 1; i < len(parts); i++ {
		opt := strings.TrimSpace(parts[i])
		if idx := strings.Index(opt, "="); idx != -1 {
			key := strings.TrimSpace(opt[:idx])
			value := strings.TrimSpace(opt[idx+1:])
			tag.Options[key] = value
		} else {
			tag.Options[opt] = "true"
		}
	}

	return tag
}

func GetTagValue(field reflect.StructField, tagName string) string {
	tag := GetTag(field, tagName)
	if tag == nil {
		return ""
	}
	return tag.Value
}

func HasTag(field reflect.StructField, tagName string) bool {
	tagStr := field.Tag.Get(tagName)
	return tagStr != ""
}

func HasTagOption(field reflect.StructField, tagName, optionName string) bool {
	tag := GetTag(field, tagName)
	if tag == nil {
		return false
	}
	_, exists := tag.Options[optionName]
	return exists
}

func GetTagOption(field reflect.StructField, tagName, optionName string) string {
	tag := GetTag(field, tagName)
	if tag == nil {
		return ""
	}
	return tag.Options[optionName]
}

func Build(name, value string) string {
	if value == "" {
		return name
	}
	return fmt.Sprintf("%s:%s", name, value)
}

func BuildWithOptions(name, value string, options map[string]string) string {
	result := Build(name, value)

	if len(options) > 0 {
		// 为了测试的一致性，我们按照特定顺序添加选项
		// 先添加布尔选项，再添加带值的选项
		booleanOptions := []string{}
		valueOptions := []string{}

		for key, val := range options {
			if val == "true" {
				booleanOptions = append(booleanOptions, key)
			} else {
				valueOptions = append(valueOptions, fmt.Sprintf("%s=%s", key, val))
			}
		}

		// 先添加布尔选项
		for _, opt := range booleanOptions {
			result += fmt.Sprintf(",%s", opt)
		}

		// 再添加带值的选项
		for _, opt := range valueOptions {
			result += fmt.Sprintf(",%s", opt)
		}
	}

	return result
}

func BuildJSONTag(fieldName string, options map[string]string) string {
	return BuildWithOptions(JSONTag, fieldName, options)
}

func BuildYAMLTag(fieldName string, options map[string]string) string {
	return BuildWithOptions(YAMLTag, fieldName, options)
}

func BuildXMLTag(fieldName string, options map[string]string) string {
	return BuildWithOptions(XMLTag, fieldName, options)
}

func BuildDBTag(fieldName string, options map[string]string) string {
	return BuildWithOptions(DBTag, fieldName, options)
}

func BuildFormTag(fieldName string, options map[string]string) string {
	return BuildWithOptions(FormTag, fieldName, options)
}

func BuildQueryTag(fieldName string, options map[string]string) string {
	return BuildWithOptions(QueryTag, fieldName, options)
}

func BuildHeaderTag(fieldName string, options map[string]string) string {
	return BuildWithOptions(HeaderTag, fieldName, options)
}

func BuildURITag(options map[string]string) string {
	return BuildWithOptions(URITag, "", options)
}

func BuildURLTag(options map[string]string) string {
	return BuildWithOptions(URLTag, "", options)
}

func BuildEmailTag(options map[string]string) string {
	return BuildWithOptions(EmailTag, "", options)
}

func BuildRequiredTag() string {
	return RequiredTag
}

func BuildOptionalTag() string {
	return OptionalTag
}

func BuildMinTag(minValue int) string {
	return fmt.Sprintf("%s=%d", MinTag, minValue)
}

func BuildMaxTag(maxValue int) string {
	return fmt.Sprintf("%s=%d", MaxTag, maxValue)
}

func BuildLengthTag(length int) string {
	return fmt.Sprintf("%s=%d", LengthTag, length)
}

func BuildPatternTag(pattern string) string {
	return fmt.Sprintf("%s=%s", PatternTag, pattern)
}

func BuildFormatTag(format string) string {
	return fmt.Sprintf("%s=%s", FormatTag, format)
}

func BuildEnumTag(values ...string) string {
	return fmt.Sprintf("%s=%s", EnumTag, strings.Join(values, ","))
}

func BuildAliasTag(alias string) string {
	return fmt.Sprintf("%s=%s", AliasTag, alias)
}

func BuildDefaultTag(value interface{}) string {
	return fmt.Sprintf("%s=%v", DefaultTag, value)
}

func BuildExampleTag(example interface{}) string {
	return fmt.Sprintf("%s=%v", ExampleTag, example)
}

func BuildDescriptionTag(description string) string {
	return fmt.Sprintf("%s=%s", DescriptionTag, description)
}

func BuildValidateTag(rule string) string {
	return fmt.Sprintf("%s=%s", ValidateTag, rule)
}

func AddOmitEmpty(tag string) string {
	if tag == "" {
		return OmitEmptyTag
	}
	return fmt.Sprintf("%s,%s", tag, OmitEmptyTag)
}

func AddOption(tag, option string) string {
	if tag == "" {
		return option
	}
	return fmt.Sprintf("%s,%s", tag, option)
}

func AddOptionWithValue(tag, option, value string) string {
	if tag == "" {
		return fmt.Sprintf("%s=%s", option, value)
	}
	return fmt.Sprintf("%s,%s=%s", tag, option, value)
}

func IsIgnoreTag(tag string) bool {
	return tag == IgnoreTag
}

func HasOmitEmpty(tag string) bool {
	return strings.Contains(tag, OmitEmptyTag)
}

func GetTagValueFromString(tag string) string {
	t := Parse(tag)
	if t == nil {
		return ""
	}
	return t.Value
}

func GetTagOptionsFromString(tag string) map[string]string {
	t := Parse(tag)
	if t == nil {
		return nil
	}
	return t.Options
}

func String(t *Tag) string {
	if t == nil {
		return ""
	}
	return t.Original
}
