package cmd

import (
	"context"
	"fmt"
	"reflect"

	"github.com/LandcLi/landc-go/tools/str"
	"github.com/LandcLi/landc-go/tools/tag"
)

// NewFromObject 从对象创建命令
func NewFromObject(obj interface{}) (*Command, error) {
	// 获取对象类型
	typ := reflect.TypeOf(obj)
	val := reflect.ValueOf(obj)

	// 处理指针类型
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		val = val.Elem()
	}

	// 检查是否是结构体
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("object must be a struct")
	}

	// 解析主命令元数据
	cmd, err := parseCommandFromStruct(typ, val)
	if err != nil {
		return nil, err
	}

	// 解析子命令（从方法）
	err = parseSubCommandsFromMethods(cmd, typ, val)
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

// parseCommandFromStruct 从结构体解析命令
func parseCommandFromStruct(typ reflect.Type, val reflect.Value) (*Command, error) {
	// 解析元数据
	meta, err := parseMetaFromStruct(typ, val)
	if err != nil {
		return nil, err
	}

	// 创建命令
	cmd := &Command{
		Name:             meta["name"],
		Brief:            meta["brief"],
		supportedOptions: make(map[string]bool),
		optionAliases:    make(map[string]string),
	}

	return cmd, nil
}

// parseSubCommandsFromMethods 从方法解析子命令
func parseSubCommandsFromMethods(parent *Command, typ reflect.Type, val reflect.Value) error {
	// 遍历所有方法
	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)

		// 检查方法签名
		if !isValidCommandMethod(method.Type) {
			continue
		}

		// 解析子命令
		subCmd, err := parseCommandFromMethod(method, val)
		if err != nil {
			continue
		}

		// 添加子命令
		err = parent.AddCommand(subCmd)
		if err != nil {
			return err
		}
	}

	return nil
}

// parseCommandFromMethod 从方法解析命令
func parseCommandFromMethod(method reflect.Method, val reflect.Value) (*Command, error) {
	// 获取方法输入参数类型
	inputType := method.Type.In(2) // 第一个参数是接收者，第二个是上下文，第三个是输入参数
	if inputType.Kind() == reflect.Pointer {
		inputType = inputType.Elem()
	}

	// 检查输入参数是否是结构体
	if inputType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("method input must be a struct")
	}

	// 解析输入参数结构体的元数据
	meta, err := parseMetaFromStruct(inputType, reflect.Zero(inputType))
	if err != nil {
		return nil, err
	}

	// 创建命令
	cmd := &Command{
		Name:             meta["name"],
		Brief:            meta["brief"],
		supportedOptions: make(map[string]bool),
		optionAliases:    make(map[string]string),
	}

	// 解析输入参数结构体的字段作为选项或参数
	err = parseFieldsFromStruct(cmd, inputType)
	if err != nil {
		return nil, err
	}

	// 设置命令执行函数
	cmd.Func = func(ctx context.Context, parser *Parser) error {
		// 创建输入参数实例
		inputVal := reflect.New(inputType)
		inputStruct := inputVal.Elem()

		// 填充输入参数
		err := fillInputFromParser(inputStruct, parser)
		if err != nil {
			return err
		}

		// 调用方法
		args := []reflect.Value{
			val,                  // 接收者
			reflect.ValueOf(ctx), // 上下文
			inputVal,             // 输入参数
		}

		// 执行方法
		results := method.Func.Call(args)

		// 处理返回值
		if len(results) > 1 {
			errVal := results[1]
			if !errVal.IsNil() {
				return errVal.Interface().(error)
			}
		}

		return nil
	}

	return cmd, nil
}

// parseMetaFromStruct 从结构体解析元数据
func parseMetaFromStruct(typ reflect.Type, val reflect.Value) (map[string]string, error) {
	meta := make(map[string]string)

	// 遍历结构体字段
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// 检查是否是 Meta 字段
		if field.Name == "Meta" {
			// 解析 Meta 字段的内部字段
			for j := 0; j < field.Type.NumField(); j++ {
				metaField := field.Type.Field(j)
				metaFieldVal := fieldVal.Field(j)

				// 解析标签
				switch metaField.Name {
				case "Name":
					tagValue := tag.GetTagValue(metaField, "name")
					if tagValue != "" {
						meta["name"] = tagValue
					} else if metaFieldVal.IsValid() && metaFieldVal.Kind() == reflect.String {
						meta["name"] = metaFieldVal.String()
					}
				case "Brief":
					tagValue := tag.GetTagValue(metaField, "brief")
					if tagValue != "" {
						meta["brief"] = tagValue
					} else if metaFieldVal.IsValid() && metaFieldVal.Kind() == reflect.String {
						meta["brief"] = metaFieldVal.String()
					}
				}
			}
		}
	}

	// 如果没有指定名称，使用结构体名称（首字母小写）
	if meta["name"] == "" {
		structName := typ.Name()
		if structName != "" {
			meta["name"] = str.Uncapitalize(structName)
		}
	}

	return meta, nil
}

// parseFieldsFromStruct 从结构体字段解析参数和选项
func parseFieldsFromStruct(cmd *Command, typ reflect.Type) error {
	// 遍历结构体字段
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// 跳过 Meta 字段
		if field.Name == "Meta" {
			continue
		}

		// 解析字段标签
		isArg := tag.GetTagValue(field, "arg") == "true"
		short := tag.GetTagValue(field, "short")
		name := tag.GetTagValue(field, "name")
		brief := tag.GetTagValue(field, "brief")

		// 如果没有指定名称，使用字段名称
		if name == "" {
			name = field.Name
		}

		// 处理参数
		if isArg {
			// 添加参数配置
			arg := Argument{
				Name:     name,
				Brief:    brief,
				Optional: tag.GetTagValue(field, "optional") == "true",
			}
			cmd.AddArgument(arg)
		} else {
			// 处理选项
			optionName := name
			if short != "" {
				optionName = short + "," + name
			}
			cmd.AddOption(optionName, true) // 默认需要参数
		}
	}

	return nil
}

// fillInputFromParser 从解析器填充输入参数
func fillInputFromParser(input reflect.Value, parser *Parser) error {
	typ := input.Type()

	// 遍历结构体字段
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := input.Field(i)

		// 跳过 Meta 字段
		if field.Name == "Meta" {
			continue
		}

		// 解析字段标签
		isArg := tag.GetTagValue(field, "arg") == "true"
		short := tag.GetTagValue(field, "short")
		name := tag.GetTagValue(field, "name")

		// 如果没有指定名称，使用字段名称
		if name == "" {
			name = field.Name
		}

		// 填充参数
		if isArg {
			// 从参数获取值
			argValue := parser.GetArg(0) // 简化处理，只获取第一个参数
			if argValue != "" {
				// 根据字段类型设置值
				setFieldValue(fieldVal, argValue)
			}
		} else {
			// 填充选项
			optionValue := ""
			if short != "" {
				optionValue = parser.GetOpt(short)
			}
			if optionValue == "" {
				optionValue = parser.GetOpt(name)
			}
			if optionValue != "" {
				// 根据字段类型设置值
				setFieldValue(fieldVal, optionValue)
			}
		}
	}

	return nil
}

// setFieldValue 根据字段类型设置值
func setFieldValue(field reflect.Value, value string) {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var intValue int64
		_, _ = fmt.Sscanf(value, "%d", &intValue)
		field.SetInt(intValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var uintValue uint64
		_, _ = fmt.Sscanf(value, "%d", &uintValue)
		field.SetUint(uintValue)
	case reflect.Float32, reflect.Float64:
		var floatValue float64
		_, _ = fmt.Sscanf(value, "%f", &floatValue)
		field.SetFloat(floatValue)
	case reflect.Bool:
		var boolValue bool
		_, _ = fmt.Sscanf(value, "%t", &boolValue)
		field.SetBool(boolValue)
	}
}

// isValidCommandMethod 检查方法是否是有效的命令方法
func isValidCommandMethod(methodType reflect.Type) bool {
	// 检查方法参数数量
	if methodType.NumIn() != 3 {
		return false
	}

	// 检查方法返回值数量
	if methodType.NumOut() < 1 {
		return false
	}

	// 检查第一个参数是否是接收者
	// 检查第二个参数是否是 context.Context
	ctxType := methodType.In(1)
	if ctxType != reflect.TypeOf((*context.Context)(nil)).Elem() {
		return false
	}

	// 检查第三个参数是否是结构体指针
	inputType := methodType.In(2)
	if inputType.Kind() != reflect.Pointer {
		return false
	}
	inputType = inputType.Elem()
	if inputType.Kind() != reflect.Struct {
		return false
	}

	// 检查返回值
	if methodType.NumOut() > 1 {
		// 第二个返回值应该是 error
		errType := methodType.Out(1)
		if errType != reflect.TypeOf((*error)(nil)).Elem() {
			return false
		}
	}

	return true
}
