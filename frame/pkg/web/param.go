package web

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/LandcLi/landc-go/tools/tag"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type landcToolsValidator struct {
	validate *validator.Validate
}

func newLandcToolsValidator() *landcToolsValidator {
	v, _ := binding.Validator.Engine().(*validator.Validate)
	return &landcToolsValidator{validate: v}
}

func (v *landcToolsValidator) validateStruct(obj interface{}) error {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		if err := v.validateField(field, fieldValue); err != nil {
			return err
		}
	}

	return nil
}

func (v *landcToolsValidator) validateField(field reflect.StructField, fieldValue reflect.Value) error {
	bindingTag := tag.GetTagValue(field, "binding")

	if bindingTag != "" {
		return nil
	}

	if err := v.validateWithLandcTools(field, fieldValue); err != nil {
		return err
	}

	return nil
}

func (v *landcToolsValidator) validateWithLandcTools(field reflect.StructField, fieldValue reflect.Value) error {
	tags := tag.ParseAll(field.Tag.Get("validate"))
	if tags == nil {
		return nil
	}

	value := fieldValue.Interface()
	for _, t := range tags {
		if err := v.validateWithTag(t, value); err != nil {
			return fmt.Errorf("%s: %w", field.Name, err)
		}
	}

	return nil
}

//nolint:gocyclo // 验证器分派为线性 switch，拆分收益低
func (v *landcToolsValidator) validateWithTag(t *tag.Tag, value interface{}) error {
	switch t.Name {
	case "required":
		vd := &tag.RequiredValidator{}
		return vd.Validate(value)
	case "email":
		vd := &tag.EmailValidator{}
		return vd.Validate(value)
	case "phone":
		vd := &tag.PhoneValidator{}
		return vd.Validate(value)
	case "phone-loose":
		vd := &tag.PhoneLooseValidator{}
		return vd.Validate(value)
	case "telephone":
		vd := &tag.TelephoneValidator{}
		return vd.Validate(value)
	case "password":
		vd := &tag.PasswordValidator{}
		return vd.Validate(value)
	case "password2":
		vd := &tag.Password2Validator{}
		return vd.Validate(value)
	case "min":
		if minVal, err := strconv.Atoi(t.Value); err == nil {
			vd := &tag.MinValidator{Min: minVal}
			return vd.Validate(value)
		}
	case "max":
		if maxVal, err := strconv.Atoi(t.Value); err == nil {
			vd := &tag.MaxValidator{Max: maxVal}
			return vd.Validate(value)
		}
	case "length":
		if length, err := strconv.Atoi(t.Value); err == nil {
			vd := &tag.LengthValidator{Length: length}
			return vd.Validate(value)
		}
	case "pattern":
		vd := &tag.PatternValidator{Pattern: t.Value}
		return vd.Validate(value)
	case "date":
		vd := &tag.DateValidator{}
		return vd.Validate(value)
	case "datetime":
		vd := &tag.DateTimeValidator{}
		return vd.Validate(value)
	case "date-format":
		vd := &tag.DateFormatValidator{Format: t.Value}
		return vd.Validate(value)
	case "array":
		vd := &tag.ArrayValidator{}
		return vd.Validate(value)
	case "enum":
		values := strings.Split(t.Value, ",")
		vd := &tag.EnumsValidator{Values: values}
		return vd.Validate(value)
	case "format":
		vd := &tag.FormatValidator{Format: t.Value}
		return vd.Validate(value)
	}

	return nil
}

type ParamSource string

const (
	ParamSourceQuery  ParamSource = "query"
	ParamSourcePath   ParamSource = "path"
	ParamSourceHeader ParamSource = "header"
	ParamSourceBody   ParamSource = "body"
	ParamSourceForm   ParamSource = "form"
)

type ParamMeta struct {
	Name    string
	Source  ParamSource
	Type    reflect.Type
	Default string // 默认值（来自 d tag）
}

func parseParamMeta(field reflect.StructField) (*ParamMeta, error) {
	meta := &ParamMeta{
		Type: field.Type,
	}

	sourceTag := tag.GetTagValue(field, "source")
	if sourceTag != "" {
		meta.Source = ParamSource(sourceTag)
	} else {
		meta.Source = ParamSourceBody
	}

	nameTag := tag.GetTagValue(field, "name")
	if nameTag != "" {
		meta.Name = nameTag
	} else {
		meta.Name = tag.GetTagValue(field, "json")
		if meta.Name == "" {
			meta.Name = field.Name
		}
	}

	// 解析默认值 d tag
	meta.Default = tag.GetTagValue(field, "d")

	return meta, nil
}

//nolint:gocyclo // 参数来源/类型分派分支多，拆分收益低
func parseParamsFromContext(c *gin.Context, paramType reflect.Type) ([]reflect.Value, error) {
	if paramType.Kind() == reflect.Pointer {
		paramType = paramType.Elem()
	}

	if paramType.Kind() != reflect.Struct {
		return nil, errors.New("parameter must be a struct or pointer to struct")
	}

	paramValue := reflect.New(paramType).Elem()

	// 第一步：检查是否有 body 字段，如果有则先整体 ShouldBind 一次（避免多次读取 body 导致 EOF）
	hasBodyFields := false
	for i := 0; i < paramType.NumField(); i++ {
		field := paramType.Field(i)
		if !field.IsExported() {
			continue
		}
		if field.Type.Name() == "Meta" {
			continue
		}
		sourceTag := tag.GetTagValue(field, "source")
		if sourceTag == "" || sourceTag == "body" {
			hasBodyFields = true
			break
		}
	}

	if hasBodyFields {
		// 对整个结构体做一次 ShouldBind，绑定所有 body/json 字段
		if err := c.ShouldBind(paramValue.Addr().Interface()); err != nil {
			// 忽略 EOF（GET 请求没有 body 是正常的）
			if err.Error() != "EOF" {
				return nil, err
			}
		}
	}

	// 第二步：处理非 body 来源的字段（query/path/header/form），覆盖 body 绑定的值
	for i := 0; i < paramType.NumField(); i++ {
		field := paramType.Field(i)
		fieldValue := paramValue.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		if field.Type.Name() == "Meta" {
			continue
		}

		meta, err := parseParamMeta(field)
		if err != nil {
			return nil, fmt.Errorf("failed to parse param meta for field %s: %w", field.Name, err)
		}

		switch meta.Source {
		case ParamSourceQuery:
			if err := setFromQuery(c, meta.Name, fieldValue); err != nil {
				return nil, err
			}
		case ParamSourcePath:
			if err := setFromPath(c, meta.Name, fieldValue); err != nil {
				return nil, err
			}
		case ParamSourceHeader:
			if err := setFromHeader(c, meta.Name, fieldValue); err != nil {
				return nil, err
			}
		case ParamSourceForm:
			if err := setFromForm(c, meta.Name, fieldValue); err != nil {
				return nil, err
			}
		case ParamSourceBody:
			// body 字段已在第一步整体绑定，跳过
		}

		// 应用默认值：当字段为零值且有 d tag 时设置默认值
		if meta.Default != "" && fieldValue.IsZero() {
			if err := setFieldValue(fieldValue, meta.Default); err != nil {
				return nil, fmt.Errorf("failed to apply default value for field %s: %w", field.Name, err)
			}
		}
	}

	if err := validateStruct(paramValue.Addr().Interface()); err != nil {
		return nil, err
	}

	landcToolsValidator := newLandcToolsValidator()
	if err := landcToolsValidator.validateStruct(paramValue.Addr().Interface()); err != nil {
		return nil, err
	}

	return []reflect.Value{paramValue.Addr()}, nil
}

func validateStruct(obj interface{}) error {
	validate, ok := binding.Validator.Engine().(*validator.Validate)
	if !ok {
		return nil
	}

	if err := validate.Struct(obj); err != nil {
		return formatValidationError(err)
	}

	return nil
}

func formatValidationError(err error) error {
	var validationErrors validator.ValidationErrors
	if errors.As(err, &validationErrors) {
		var errMsgs []string
		for _, e := range validationErrors {
			fieldName := e.Field()
			tagName := e.Tag()
			param := e.Param()

			switch tagName {
			case "required":
				errMsgs = append(errMsgs, fmt.Sprintf("%s is required", fieldName))
			case "email":
				errMsgs = append(errMsgs, fmt.Sprintf("%s must be a valid email", fieldName))
			case "min":
				errMsgs = append(errMsgs, fmt.Sprintf("%s must be at least %s", fieldName, param))
			case "max":
				errMsgs = append(errMsgs, fmt.Sprintf("%s must be at most %s", fieldName, param))
			case "len":
				errMsgs = append(errMsgs, fmt.Sprintf("%s must be %s characters", fieldName, param))
			case "gt":
				errMsgs = append(errMsgs, fmt.Sprintf("%s must be greater than %s", fieldName, param))
			case "gte":
				errMsgs = append(errMsgs, fmt.Sprintf("%s must be greater than or equal to %s", fieldName, param))
			case "lt":
				errMsgs = append(errMsgs, fmt.Sprintf("%s must be less than %s", fieldName, param))
			case "lte":
				errMsgs = append(errMsgs, fmt.Sprintf("%s must be less than or equal to %s", fieldName, param))
			default:
				errMsgs = append(errMsgs, fmt.Sprintf("%s validation failed on %s", fieldName, tagName))
			}
		}
		return fmt.Errorf("validation failed: %s", strings.Join(errMsgs, "; "))
	}
	return err
}

func setFromQuery(c *gin.Context, name string, field reflect.Value) error {
	value := c.Query(name)
	return setFieldValue(field, value)
}

func setFromPath(c *gin.Context, name string, field reflect.Value) error {
	value := c.Param(name)
	return setFieldValue(field, value)
}

func setFromHeader(c *gin.Context, name string, field reflect.Value) error {
	value := c.GetHeader(name)
	return setFieldValue(field, value)
}

func setFromForm(c *gin.Context, name string, field reflect.Value) error {
	value := c.PostForm(name)
	return setFieldValue(field, value)
}

func setFieldValue(field reflect.Value, value string) error {
	if value == "" {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolValue)
	case reflect.Slice:
		if field.Type().Elem().Kind() == reflect.String {
			values := strings.Split(value, ",")
			slice := reflect.MakeSlice(field.Type(), len(values), len(values))
			for i, v := range values {
				slice.Index(i).SetString(v)
			}
			field.Set(slice)
		}
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}

	return nil
}
