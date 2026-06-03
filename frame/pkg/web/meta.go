package web

import (
	"reflect"
	"strings"

	"github.com/LandcLi/landc-go/frame/pkg/meta"
)

type MethodMeta struct {
	Name        string
	Path        string
	HTTPMethod  string
	Description string
}

func parseMethodMeta(method reflect.Method) (*MethodMeta, error) {
	metaInfo := &MethodMeta{
		Name: method.Name,
	}

	methodType := method.Type

	if methodType.NumIn() < 2 {
		metaInfo.HTTPMethod = "GET"
		metaInfo.Path = "/" + strings.ToLower(method.Name)
		return metaInfo, nil
	}

	paramIndex := 1
	if methodType.NumIn() > 1 && methodType.In(1).String() == "*web.LandcContext" {
		paramIndex = 2
	}

	if methodType.NumIn() < paramIndex+1 {
		metaInfo.HTTPMethod = "GET"
		metaInfo.Path = "/" + strings.ToLower(method.Name)
		return metaInfo, nil
	}

	paramType := methodType.In(paramIndex)
	if paramType.Kind() == reflect.Ptr {
		paramType = paramType.Elem()
	}

	if paramType.Kind() != reflect.Struct {
		metaInfo.HTTPMethod = "GET"
		metaInfo.Path = "/" + strings.ToLower(method.Name)
		return metaInfo, nil
	}

	metaData := meta.Data(reflect.New(paramType).Elem().Interface())

	if path, ok := metaData["path"].(string); ok {
		metaInfo.Path = path
	}

	if httpMethod, ok := metaData["method"].(string); ok {
		metaInfo.HTTPMethod = httpMethod
	}

	if description, ok := metaData["description"].(string); ok {
		metaInfo.Description = description
	}

	if metaInfo.Path == "" {
		metaInfo.Path = "/" + strings.ToLower(method.Name)
	}

	if metaInfo.HTTPMethod == "" {
		metaInfo.HTTPMethod = "GET"
	}

	return metaInfo, nil
}

func getGroupPath(instance interface{}) string {
	instanceValue := reflect.ValueOf(instance)
	if instanceValue.Kind() == reflect.Ptr {
		instanceValue = instanceValue.Elem()
	}

	if instanceValue.Kind() != reflect.Struct {
		return ""
	}

	instanceType := instanceValue.Type()

	for i := 0; i < instanceType.NumField(); i++ {
		field := instanceType.Field(i)

		if field.Type.Name() == "Meta" {
			metaData := meta.Data(instanceValue.Interface())
			if path, ok := metaData["path"].(string); ok {
				return path
			}
			break
		}
	}

	return ""
}

func isExported(name string) bool {
	return name != "" && name[0] >= 'A' && name[0] <= 'Z'
}
