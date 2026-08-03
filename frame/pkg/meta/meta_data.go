package meta

import (
	"reflect"
)

func Data(object interface{}) map[string]interface{} {
	if object == nil {
		return make(map[string]interface{})
	}

	val := reflect.ValueOf(object)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return make(map[string]interface{})
	}

	typ := val.Type()
	result := make(map[string]interface{})

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)

		if field.Type.Name() == "Meta" {
			for _, tagName := range []string{"meta", "db", "table", "orm", "json", "yaml", "path", "method", "description", "group"} {
				tagValue := field.Tag.Get(tagName)
				if tagValue != "" {
					result[tagName] = tagValue
				}
			}
		}

		if fieldValue.Kind() == reflect.Struct && !fieldValue.CanAddr() {
			embedded := Data(fieldValue.Interface())
			for k, v := range embedded {
				result[k] = v
			}
		}
	}

	return result
}

func Get(object interface{}, key string) interface{} {
	if object == nil || key == "" {
		return nil
	}

	data := Data(object)
	return data[key]
}
