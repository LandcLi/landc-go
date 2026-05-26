package converter

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/LandcLi/landc-go/tools/tag"
)

// Converter 对象转换器
// 提供各种类型之间的转换功能，并支持验证规则
type Converter struct {
	options *ConvertOptions
}

// ConvertOptions 转换选项
type ConvertOptions struct {
	// StrictMode 严格模式，如果源数据中有目标结构体不存在的字段，返回错误
	StrictMode bool
	// IgnoreEmpty 忽略空值
	IgnoreEmpty bool
	// CaseInsensitive 字段名不区分大小写
	CaseInsensitive bool
	// Validate 验证数据
	Validate bool
	// FieldNameMapping 字段名映射
	FieldNameMapping map[string]string
}

// NewConverter 创建新的转换器
func NewConverter() *Converter {
	return &Converter{
		options: &ConvertOptions{
			StrictMode:       false,
			IgnoreEmpty:      false,
			CaseInsensitive:  false,
			Validate:         true,
			FieldNameMapping: make(map[string]string),
		},
	}
}

// SetOptions 设置转换选项
func (c *Converter) SetOptions(options *ConvertOptions) *Converter {
	if options != nil {
		c.options = options
	}
	return c
}

// WithStrictMode 设置严格模式
func (c *Converter) WithStrictMode(strict bool) *Converter {
	c.options.StrictMode = strict
	return c
}

// WithIgnoreEmpty 设置忽略空值
func (c *Converter) WithIgnoreEmpty(ignore bool) *Converter {
	c.options.IgnoreEmpty = ignore
	return c
}

// WithCaseInsensitive 设置不区分大小写
func (c *Converter) WithCaseInsensitive(insensitive bool) *Converter {
	c.options.CaseInsensitive = insensitive
	return c
}

// WithValidate 设置是否验证
func (c *Converter) WithValidate(validate bool) *Converter {
	c.options.Validate = validate
	return c
}

// WithFieldMapping 设置字段名映射
func (c *Converter) WithFieldMapping(mapping map[string]string) *Converter {
	c.options.FieldNameMapping = mapping
	return c
}

// ConvertError 转换错误
type ConvertError struct {
	Field   string
	Message string
	Value   interface{}
}

func (e *ConvertError) Error() string {
	return fmt.Sprintf("field '%s': %s (value: %v)", e.Field, e.Message, e.Value)
}

// NewConvertError 创建转换错误
func NewConvertError(field, message string, value interface{}) *ConvertError {
	return &ConvertError{
		Field:   field,
		Message: message,
		Value:   value,
	}
}

// ConvertTo 将源对象转换为目标对象（传入目标对象）
// target 必须是指向结构体的指针
func (c *Converter) ConvertTo(source, target interface{}) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr || targetValue.Elem().Kind() != reflect.Struct {
		return errors.New("target must be a pointer to struct")
	}

	targetValue = targetValue.Elem()
	sourceValue := reflect.ValueOf(source)

	return c.convertValue(sourceValue, targetValue, "")
}

// Convert 将源对象转换为目标对象（生成目标对象）
// target 必须是结构体类型（非指针）
func (c *Converter) Convert(source, target interface{}) (interface{}, error) {
	targetType := reflect.TypeOf(target)
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	if targetType.Kind() != reflect.Struct {
		return nil, errors.New("target must be a struct or pointer to struct")
	}

	targetValue := reflect.New(targetType).Elem()
	sourceValue := reflect.ValueOf(source)

	err := c.convertValue(sourceValue, targetValue, "")
	if err != nil {
		return nil, err
	}

	return targetValue.Addr().Interface(), nil
}

// MapToStruct 将map转换为结构体（传入目标对象）
func (c *Converter) MapToStruct(source map[string]interface{}, target interface{}) error {
	return c.ConvertTo(source, target)
}

// MapToStructNew 将map转换为结构体（生成目标对象）
func (c *Converter) MapToStructNew(source map[string]interface{}, target interface{}) (interface{}, error) {
	return c.Convert(source, target)
}

// StructToStruct 将结构体转换为结构体（同名属性）
func (c *Converter) StructToStruct(source, target interface{}) error {
	return c.ConvertTo(source, target)
}

// StructToStructNew 将结构体转换为结构体（生成目标对象）
func (c *Converter) StructToStructNew(source, target interface{}) (interface{}, error) {
	return c.Convert(source, target)
}

// MapArrayToStructArray 将map数组转换为结构体数组
func (c *Converter) MapArrayToStructArray(source []map[string]interface{}, target interface{}) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr {
		return errors.New("target must be a pointer to slice")
	}

	targetValue = targetValue.Elem()
	if targetValue.Kind() != reflect.Slice {
		return errors.New("target must be a pointer to slice")
	}

	if targetValue.IsNil() {
		targetValue.Set(reflect.MakeSlice(targetValue.Type(), len(source), len(source)))
	}

	if targetValue.Len() != len(source) {
		return fmt.Errorf("target slice length %d does not match source length %d", targetValue.Len(), len(source))
	}

	for i, srcMap := range source {
		elem := targetValue.Index(i)
		if elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				elem.Set(reflect.New(elem.Type().Elem()))
			}
			elem = elem.Elem()
		}

		err := c.convertValue(reflect.ValueOf(srcMap), elem, "")
		if err != nil {
			return err
		}
	}

	return nil
}

// MapArrayToStructArrayNew 将map数组转换为结构体数组（生成目标对象）
func (c *Converter) MapArrayToStructArrayNew(source []map[string]interface{}, target interface{}) (interface{}, error) {
	targetType := reflect.TypeOf(target)
	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	if targetType.Kind() != reflect.Slice {
		return nil, errors.New("target must be a slice or pointer to slice")
	}

	elemType := targetType.Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	if elemType.Kind() != reflect.Struct {
		return nil, errors.New("slice element type must be struct")
	}

	sliceValue := reflect.MakeSlice(targetType, len(source), len(source))

	for i, srcMap := range source {
		elem := reflect.New(elemType).Elem()
		err := c.convertValue(reflect.ValueOf(srcMap), elem, "")
		if err != nil {
			return nil, err
		}
		// 根据切片元素类型设置值
		if targetType.Elem().Kind() == reflect.Ptr {
			sliceValue.Index(i).Set(elem.Addr())
		} else {
			sliceValue.Index(i).Set(elem)
		}
	}

	return sliceValue.Interface(), nil
}

// StructArrayToStructArray 将结构体数组转换为结构体数组（同名属性）
func (c *Converter) StructArrayToStructArray(source, target interface{}) error {
	sourceValue := reflect.ValueOf(source)
	targetValue := reflect.ValueOf(target)

	if targetValue.Kind() != reflect.Ptr {
		return errors.New("target must be a pointer to slice")
	}

	targetValue = targetValue.Elem()
	if targetValue.Kind() != reflect.Slice {
		return errors.New("target must be a pointer to slice")
	}

	if sourceValue.Kind() != reflect.Slice && sourceValue.Kind() != reflect.Array {
		return errors.New("source must be a slice or array")
	}

	if targetValue.IsNil() {
		targetValue.Set(reflect.MakeSlice(targetValue.Type(), sourceValue.Len(), sourceValue.Len()))
	}

	if targetValue.Len() != sourceValue.Len() {
		return fmt.Errorf("target slice length %d does not match source length %d", targetValue.Len(), sourceValue.Len())
	}

	for i := 0; i < sourceValue.Len(); i++ {
		srcElem := sourceValue.Index(i)
		tgtElem := targetValue.Index(i)

		if tgtElem.Kind() == reflect.Ptr {
			if tgtElem.IsNil() {
				tgtElem.Set(reflect.New(tgtElem.Type().Elem()))
			}
			tgtElem = tgtElem.Elem()
		}

		if srcElem.Kind() == reflect.Ptr {
			if srcElem.IsNil() {
				continue
			}
			srcElem = srcElem.Elem()
		}

		err := c.convertValue(srcElem, tgtElem, "")
		if err != nil {
			return err
		}
	}

	return nil
}

// StructArrayToStructArrayNew 将结构体数组转换为结构体数组（生成目标对象）
func (c *Converter) StructArrayToStructArrayNew(source, target interface{}) (interface{}, error) {
	sourceValue := reflect.ValueOf(source)
	targetType := reflect.TypeOf(target)

	if targetType.Kind() == reflect.Ptr {
		targetType = targetType.Elem()
	}

	if targetType.Kind() != reflect.Slice {
		return nil, errors.New("target must be a slice or pointer to slice")
	}

	elemType := targetType.Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}

	if elemType.Kind() != reflect.Struct {
		return nil, errors.New("slice element type must be struct")
	}

	if sourceValue.Kind() != reflect.Slice && sourceValue.Kind() != reflect.Array {
		return nil, errors.New("source must be a slice or array")
	}

	sliceValue := reflect.MakeSlice(targetType, sourceValue.Len(), sourceValue.Len())

	for i := 0; i < sourceValue.Len(); i++ {
		srcElem := sourceValue.Index(i)
		if srcElem.Kind() == reflect.Ptr {
			if srcElem.IsNil() {
				continue
			}
			srcElem = srcElem.Elem()
		}

		elem := reflect.New(elemType).Elem()
		err := c.convertValue(srcElem, elem, "")
		if err != nil {
			return nil, err
		}
		// 根据切片元素类型设置值
		if targetType.Elem().Kind() == reflect.Ptr {
			sliceValue.Index(i).Set(elem.Addr())
		} else {
			sliceValue.Index(i).Set(elem)
		}
	}

	return sliceValue.Interface(), nil
}

// convertValue 转换值
func (c *Converter) convertValue(source, target reflect.Value, parentField string) error {
	sourceKind := source.Kind()
	targetKind := target.Kind()

	if sourceKind == reflect.Invalid || targetKind == reflect.Invalid {
		return nil
	}

	if !target.CanSet() {
		return nil
	}

	// 处理interface{}类型的源值
	if sourceKind == reflect.Interface {
		if source.IsNil() {
			return nil
		}
		source = source.Elem()
		sourceKind = source.Kind()
	}

	// 如果源是nil且目标不是指针，跳过转换
	if sourceKind == reflect.Ptr {
		if source.IsNil() {
			return nil
		}
	}

	switch targetKind {
	case reflect.Struct:
		return c.convertToStruct(source, target, parentField)
	case reflect.Slice, reflect.Array:
		return c.convertToArray(source, target, parentField)
	case reflect.Map:
		return c.convertToMap(source, target, parentField)
	case reflect.Ptr:
		return c.convertToPtr(source, target, parentField)
	default:
		return c.convertToBasic(source, target, parentField)
	}
}

// convertToStruct 转换为结构体
func (c *Converter) convertToStruct(source, target reflect.Value, parentField string) error {
	switch source.Kind() {
	case reflect.Map:
		if source.Type().Key().Kind() != reflect.String {
			return fmt.Errorf("map key must be string, got %v", source.Type().Key())
		}

		for _, key := range source.MapKeys() {
			fieldName := c.getFieldName(key.String())
			field := c.findFieldByName(target, fieldName)
			if !field.IsValid() {
				if c.options.StrictMode {
					return NewConvertError(parentField+fieldName, "field not found in target struct", source.MapIndex(key).Interface())
				}
				continue
			}

			value := source.MapIndex(key)
			if c.options.IgnoreEmpty && isEmpty(value) {
				continue
			}

			fieldPath := parentField + fieldName + "."
			if err := c.convertValue(value, field, fieldPath); err != nil {
				return err
			}
		}

	case reflect.Struct:
		sourceType := source.Type()
		for i := 0; i < source.NumField(); i++ {
			sourceField := sourceType.Field(i)
			if !isExported(sourceField.Name) {
				continue
			}

			fieldName := c.getFieldName(sourceField.Name)
			targetField := c.findFieldByName(target, fieldName)
			if !targetField.IsValid() {
				if c.options.StrictMode {
					return NewConvertError(parentField+fieldName, "field not found in target struct", source.Field(i).Interface())
				}
				continue
			}

			value := source.Field(i)
			if c.options.IgnoreEmpty && isEmpty(value) {
				continue
			}

			fieldPath := parentField + fieldName + "."
			if err := c.convertValue(value, targetField, fieldPath); err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("cannot convert %v to struct", source.Type())
	}

	return c.validateStruct(target, parentField)
}

// convertToArray 转换为数组
func (c *Converter) convertToArray(source, target reflect.Value, parentField string) error {
	if source.Kind() != reflect.Slice && source.Kind() != reflect.Array {
		return fmt.Errorf("cannot convert %v to array", source.Type())
	}

	if target.IsNil() {
		target.Set(reflect.MakeSlice(target.Type(), source.Len(), source.Len()))
	}

	if target.Len() != source.Len() {
		return fmt.Errorf("array length mismatch: source %d, target %d", source.Len(), target.Len())
	}

	for i := 0; i < source.Len(); i++ {
		srcElem := source.Index(i)
		tgtElem := target.Index(i)

		// 处理interface{}类型的源元素
		if srcElem.Kind() == reflect.Interface {
			if srcElem.IsNil() {
				continue
			}
			srcElem = srcElem.Elem()
		}

		if tgtElem.Kind() == reflect.Ptr {
			if tgtElem.IsNil() {
				tgtElem.Set(reflect.New(tgtElem.Type().Elem()))
			}
			tgtElem = tgtElem.Elem()
		}

		if srcElem.Kind() == reflect.Ptr {
			if srcElem.IsNil() {
				continue
			}
			srcElem = srcElem.Elem()
		}

		fieldPath := fmt.Sprintf("%s[%d].", parentField, i)
		if err := c.convertValue(srcElem, tgtElem, fieldPath); err != nil {
			return err
		}
	}

	return nil
}

// convertToMap 转换为map
func (c *Converter) convertToMap(source, target reflect.Value, parentField string) error {
	if source.Kind() != reflect.Map {
		return fmt.Errorf("cannot convert %v to map", source.Type())
	}

	if target.IsNil() {
		target.Set(reflect.MakeMap(target.Type()))
	}

	for _, key := range source.MapKeys() {
		value := source.MapIndex(key)
		if c.options.IgnoreEmpty && isEmpty(value) {
			continue
		}

		targetValue := reflect.New(target.Type().Elem()).Elem()
		fieldPath := parentField + key.String() + "."
		if err := c.convertValue(value, targetValue, fieldPath); err != nil {
			return err
		}

		target.SetMapIndex(key, targetValue)
	}

	return nil
}

// convertToPtr 转换为指针
func (c *Converter) convertToPtr(source, target reflect.Value, parentField string) error {
	if target.IsNil() {
		target.Set(reflect.New(target.Type().Elem()))
	}

	return c.convertValue(source, target.Elem(), parentField)
}

// convertToBasic 转换为基本类型
func (c *Converter) convertToBasic(source, target reflect.Value, parentField string) error {
	if !source.IsValid() {
		return nil
	}

	if c.options.IgnoreEmpty && isEmpty(source) {
		return nil
	}

	converted, err := convertBasicType(source, target.Type())
	if err != nil {
		return NewConvertError(parentField, err.Error(), source.Interface())
	}

	target.Set(converted)
	return nil
}

// getFieldName 获取字段名（考虑大小写和映射）
func (c *Converter) getFieldName(name string) string {
	if c.options.CaseInsensitive {
		// 不区分大小写时，尝试匹配字段名
		return name // 直接返回，让反射自动处理
	}

	if mapped, ok := c.options.FieldNameMapping[name]; ok {
		return mapped
	}

	// 将小写的字段名转换为大写开头的字段名
	if len(name) > 0 {
		return strings.ToUpper(name[:1]) + name[1:]
	}

	return name
}

// findFieldByName 查找字段（支持大小写不敏感）
func (c *Converter) findFieldByName(target reflect.Value, name string) reflect.Value {
	field := target.FieldByName(name)
	if field.IsValid() {
		return field
	}

	// 如果不区分大小写，尝试匹配字段名
	if c.options.CaseInsensitive {
		targetType := target.Type()
		for i := 0; i < target.NumField(); i++ {
			field := targetType.Field(i)
			if strings.EqualFold(field.Name, name) {
				return target.Field(i)
			}
		}
	}

	// 尝试将小写的字段名转换为大写开头的字段名
	if len(name) > 0 {
		upperName := strings.ToUpper(name[:1]) + name[1:]
		field = target.FieldByName(upperName)
		if field.IsValid() {
			return field
		}
	}

	// 尝试将整个字段名转为大写（处理ID等缩写）
	field = target.FieldByName(strings.ToUpper(name))
	if field.IsValid() {
		return field
	}

	return reflect.Value{}
}

// validateStruct 验证结构体
func (c *Converter) validateStruct(target reflect.Value, parentField string) error {
	if !c.options.Validate {
		return nil
	}

	// 检查是否所有字段都是零值（可能是nil源导致的）
	allZero := true
	targetType := target.Type()
	for i := 0; i < target.NumField(); i++ {
		field := target.Field(i)
		if !isEmpty(field) {
			allZero = false
			break
		}
	}

	// 如果所有字段都是零值，跳过验证
	if allZero {
		return nil
	}

	for i := 0; i < target.NumField(); i++ {
		field := targetType.Field(i)
		if !isExported(field.Name) {
			continue
		}

		fieldValue := target.Field(i)
		fieldPath := parentField + field.Name

		if err := c.validateField(field, fieldValue, fieldPath); err != nil {
			return err
		}
	}

	return nil
}

// validateField 验证字段
func (c *Converter) validateField(field reflect.StructField, value reflect.Value, fieldPath string) error {
	structTag := field.Tag.Get("validate")
	if structTag == "" {
		return nil
	}

	// 如果启用了IgnoreEmpty且值为空，跳过验证
	if c.options.IgnoreEmpty && isEmpty(value) {
		return nil
	}

	tags := tag.ParseAll(structTag)
	for _, t := range tags {
		if tag.IsIgnoreTag(t.Original) {
			continue
		}

		validator, err := c.createValidator(t)
		if err != nil {
			return NewConvertError(fieldPath, fmt.Sprintf("create validator failed: %v", err), value.Interface())
		}

		if validator != nil {
			if err := validator.Validate(value.Interface()); err != nil {
				return NewConvertError(fieldPath, err.Error(), value.Interface())
			}
		}
	}

	return nil
}

// createValidator 创建验证器
func (c *Converter) createValidator(t *tag.Tag) (tag.Validator, error) {
	switch t.Name {
	case "required":
		return tag.CreateRequiredValidator(), nil
	case "min":
		min, err := tag.ParseMin(t)
		if err != nil {
			return nil, err
		}
		return tag.CreateMinValidator(min), nil
	case "max":
		max, err := tag.ParseMax(t)
		if err != nil {
			return nil, err
		}
		return tag.CreateMaxValidator(max), nil
	case "length":
		length, err := tag.ParseLength(t)
		if err != nil {
			return nil, err
		}
		return tag.CreateLengthValidator(length), nil
	case "email":
		return &tag.EmailValidator{}, nil
	case "ip":
		return &tag.IPValidator{}, nil
	case "ipv4":
		return &tag.IPv4Validator{}, nil
	case "ipv6":
		return &tag.IPv6Validator{}, nil
	case "url":
		return &tag.URLValidator{}, nil
	case "json":
		return &tag.JSONValidator{}, nil
	case "integer":
		return &tag.IntegerValidator{}, nil
	case "float":
		return &tag.FloatValidator{}, nil
	case "boolean":
		return &tag.BooleanValidator{}, nil
	case "regex":
		pattern, err := tag.ParsePattern(t)
		if err != nil {
			return nil, err
		}
		return &tag.RegexValidator{Pattern: pattern}, nil
	case "not-regex":
		pattern, err := tag.ParsePattern(t)
		if err != nil {
			return nil, err
		}
		return &tag.NotRegexValidator{Pattern: pattern}, nil
	case "in":
		return &tag.InValidator{Values: strings.Split(t.Value, ",")}, nil
	case "not-in":
		return &tag.NotInValidator{Values: strings.Split(t.Value, ",")}, nil
	case "between":
		values := strings.Split(t.Value, ",")
		if len(values) != 2 {
			return nil, fmt.Errorf("between requires two values")
		}
		min, err1 := strconv.Atoi(values[0])
		max, err2 := strconv.Atoi(values[1])
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("between requires numeric values")
		}
		return &tag.BetweenValidator{Min: min, Max: max}, nil
	case "size":
		size, err := strconv.Atoi(t.Value)
		if err != nil {
			return nil, err
		}
		return &tag.SizeValidator{Size: size}, nil
	default:
		return nil, nil
	}
}

// convertBasicType 转换基本类型
func convertBasicType(source reflect.Value, targetType reflect.Type) (reflect.Value, error) {
	sourceType := source.Type()

	if sourceType == targetType {
		return source, nil
	}

	// 如果源是interface{}，先获取其实际值
	if sourceType.Kind() == reflect.Interface {
		if source.IsNil() {
			return reflect.Zero(targetType), nil
		}
		source = reflect.ValueOf(source.Interface())
		sourceType = source.Type()
		if sourceType == targetType {
			return source, nil
		}
	}

	// 检查类型兼容性，避免不正确的自动转换
	if sourceType.ConvertibleTo(targetType) {
		// 对于数值类型和字符串之间的转换，使用手动转换而不是自动转换
		if (isNumericType(sourceType) && targetType.Kind() == reflect.String) ||
			(sourceType.Kind() == reflect.String && isNumericType(targetType)) {
			// 跳过自动转换，使用下面的手动转换逻辑
		} else {
			return source.Convert(targetType), nil
		}
	}

	sourceValue := source.Interface()
	targetKind := targetType.Kind()

	switch targetKind {
	case reflect.String:
		return reflect.ValueOf(fmt.Sprintf("%v", sourceValue)), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		switch v := sourceValue.(type) {
		case string:
			i, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("cannot convert string '%s' to int", v)
			}
			return reflect.ValueOf(i).Convert(targetType), nil
		case float64:
			return reflect.ValueOf(int64(v)).Convert(targetType), nil
		case float32:
			return reflect.ValueOf(int64(v)).Convert(targetType), nil
		case bool:
			if v {
				return reflect.ValueOf(int64(1)).Convert(targetType), nil
			}
			return reflect.ValueOf(int64(0)).Convert(targetType), nil
		case int, int8, int16, int32, int64:
			return reflect.ValueOf(v).Convert(targetType), nil
		case uint, uint8, uint16, uint32, uint64:
			return reflect.ValueOf(int64(reflect.ValueOf(v).Uint())).Convert(targetType), nil
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		switch v := sourceValue.(type) {
		case string:
			u, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("cannot convert string '%s' to uint", v)
			}
			return reflect.ValueOf(u).Convert(targetType), nil
		case float64:
			return reflect.ValueOf(uint64(v)).Convert(targetType), nil
		case bool:
			if v {
				return reflect.ValueOf(uint64(1)).Convert(targetType), nil
			}
			return reflect.ValueOf(uint64(0)).Convert(targetType), nil
		}

	case reflect.Float32, reflect.Float64:
		switch v := sourceValue.(type) {
		case string:
			f, err := strconv.ParseFloat(v, 64)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("cannot convert string '%s' to float", v)
			}
			return reflect.ValueOf(f).Convert(targetType), nil
		case int, int8, int16, int32, int64:
			return reflect.ValueOf(float64(reflect.ValueOf(v).Int())).Convert(targetType), nil
		case uint, uint8, uint16, uint32, uint64:
			return reflect.ValueOf(float64(reflect.ValueOf(v).Uint())).Convert(targetType), nil
		case bool:
			if v {
				return reflect.ValueOf(1.0).Convert(targetType), nil
			}
			return reflect.ValueOf(0.0).Convert(targetType), nil
		case float32, float64:
			return reflect.ValueOf(v).Convert(targetType), nil
		}

	case reflect.Bool:
		switch v := sourceValue.(type) {
		case string:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("cannot convert string '%s' to bool", v)
			}
			return reflect.ValueOf(b), nil
		case int, int8, int16, int32, int64:
			return reflect.ValueOf(reflect.ValueOf(v).Int() != 0), nil
		case uint, uint8, uint16, uint32, uint64:
			return reflect.ValueOf(reflect.ValueOf(v).Uint() != 0), nil
		case float32, float64:
			return reflect.ValueOf(reflect.ValueOf(v).Float() != 0), nil
		case bool:
			return reflect.ValueOf(v), nil
		}
	}

	return reflect.Value{}, fmt.Errorf("cannot convert %v to %v", sourceType, targetType)
}

// isNumericType 检查类型是否为数值类型
func isNumericType(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	default:
		return false
	}
}

// isEmpty 检查值是否为空
func isEmpty(value reflect.Value) bool {
	if !value.IsValid() {
		return true
	}

	switch value.Kind() {
	case reflect.String:
		return strings.TrimSpace(value.String()) == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return value.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return value.Float() == 0
	case reflect.Bool:
		return !value.Bool()
	case reflect.Slice, reflect.Array, reflect.Map:
		return value.Len() == 0
	case reflect.Ptr, reflect.Interface:
		return value.IsNil()
	}

	return false
}

// isExported 检查字段是否为导出字段
func isExported(name string) bool {
	return name[0] >= 'A' && name[0] <= 'Z'
}

// ValueOf 获取值的反射值
func ValueOf(value interface{}) reflect.Value {
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Ptr {
		return val.Elem()
	}
	return val
}
