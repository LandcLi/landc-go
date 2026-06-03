package web

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
)

func registerHandlers(router gin.IRouter, instance interface{}) error {
	instanceValue := reflect.ValueOf(instance)
	instanceType := reflect.TypeOf(instance)

	groupPath := getGroupPath(instance)

	if groupPath != "" {
		router = router.Group(groupPath)
	}

	for i := 0; i < instanceType.NumMethod(); i++ {
		method := instanceType.Method(i)
		if !isExported(method.Name) {
			continue
		}

		meta, err := parseMethodMeta(method)
		if err != nil {
			return fmt.Errorf("failed to parse method %s: %w", method.Name, err)
		}

		if meta.HTTPMethod == "" {
			continue
		}

		handler := createHandler(instanceValue, method)

		switch strings.ToUpper(meta.HTTPMethod) {
		case "GET":
			router.GET(meta.Path, handler)
		case "POST":
			router.POST(meta.Path, handler)
		case "PUT":
			router.PUT(meta.Path, handler)
		case "DELETE":
			router.DELETE(meta.Path, handler)
		case "PATCH":
			router.PATCH(meta.Path, handler)
		case "OPTIONS":
			router.OPTIONS(meta.Path, handler)
		case "HEAD":
			router.HEAD(meta.Path, handler)
		default:
			return fmt.Errorf("unsupported HTTP method: %s", meta.HTTPMethod)
		}
	}

	return nil
}
