package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/LandcLi/landc-go/mvc/pkg/meta"
	"github.com/gin-gonic/gin"
)

// Info OpenAPI 文档信息
type Info struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`
}

// Server 服务器信息
type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

// Document OpenAPI 3.0 文档
type Document struct {
	OpenAPI    string                `json:"openapi"`
	Info       Info                  `json:"info"`
	Servers    []Server              `json:"servers,omitempty"`
	Paths      map[string]*PathItem  `json:"paths"`
	Components *Components           `json:"components,omitempty"`
	Tags       []Tag                 `json:"tags,omitempty"`
}

// Components 组件定义
type Components struct {
	Schemas         map[string]*Schema        `json:"schemas,omitempty"`
	SecuritySchemes map[string]*SecurityScheme `json:"securitySchemes,omitempty"`
}

// SecurityScheme 安全方案
type SecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
	In           string `json:"in,omitempty"`
	Name         string `json:"name,omitempty"`
}

// Tag 标签
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// PathItem 路径项
type PathItem struct {
	Get     *Operation `json:"get,omitempty"`
	Post    *Operation `json:"post,omitempty"`
	Put     *Operation `json:"put,omitempty"`
	Delete  *Operation `json:"delete,omitempty"`
	Patch   *Operation `json:"patch,omitempty"`
	Options *Operation `json:"options,omitempty"`
	Head    *Operation `json:"head,omitempty"`
}

// Operation 操作
type Operation struct {
	Tags        []string             `json:"tags,omitempty"`
	Summary     string               `json:"summary,omitempty"`
	Description string               `json:"description,omitempty"`
	OperationID string               `json:"operationId,omitempty"`
	Parameters  []Parameter          `json:"parameters,omitempty"`
	RequestBody *RequestBody         `json:"requestBody,omitempty"`
	Responses   map[string]*Response `json:"responses"`
	Security    []map[string][]string `json:"security,omitempty"`
}

// Parameter 参数
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // query, path, header, cookie
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema"`
}

// RequestBody 请求体
type RequestBody struct {
	Description string               `json:"description,omitempty"`
	Required    bool                 `json:"required,omitempty"`
	Content     map[string]*MediaType `json:"content"`
}

// MediaType 媒体类型
type MediaType struct {
	Schema *Schema `json:"schema"`
}

// Response 响应
type Response struct {
	Description string               `json:"description"`
	Content     map[string]*MediaType `json:"content,omitempty"`
}

// Schema 数据模式
type Schema struct {
	Type        string             `json:"type,omitempty"`
	Format      string             `json:"format,omitempty"`
	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Enum        []interface{}      `json:"enum,omitempty"`
	Default     interface{}        `json:"default,omitempty"`
	Ref         string             `json:"$ref,omitempty"`
	Example     interface{}        `json:"example,omitempty"`
}

// Generator OpenAPI 文档生成器
type Generator struct {
	doc        *Document
	schemas    map[string]*Schema
	tagMap     map[string]bool
}

// NewGenerator 创建文档生成器
func NewGenerator(info Info) *Generator {
	return &Generator{
		doc: &Document{
			OpenAPI: "3.0.3",
			Info:    info,
			Paths:   make(map[string]*PathItem),
			Components: &Components{
				Schemas:         make(map[string]*Schema),
				SecuritySchemes: make(map[string]*SecurityScheme),
			},
		},
		schemas: make(map[string]*Schema),
		tagMap:  make(map[string]bool),
	}
}

// AddServer 添加服务器
func (g *Generator) AddServer(url, description string) *Generator {
	g.doc.Servers = append(g.doc.Servers, Server{URL: url, Description: description})
	return g
}

// AddSecurityScheme 添加安全方案
func (g *Generator) AddSecurityScheme(name string, scheme *SecurityScheme) *Generator {
	g.doc.Components.SecuritySchemes[name] = scheme
	return g
}

// AddBearerAuth 添加 Bearer Token 认证方案
func (g *Generator) AddBearerAuth() *Generator {
	return g.AddSecurityScheme("bearerAuth", &SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
	})
}

// RegisterController 注册 controller 并从中提取路由/参数文档
func (g *Generator) RegisterController(instance interface{}) {
	instanceType := reflect.TypeOf(instance)
	instanceValue := reflect.ValueOf(instance)
	if instanceValue.Kind() == reflect.Ptr {
		instanceValue = instanceValue.Elem()
	}

	groupPath := getGroupPath(instance)
	tagName := getTagName(instance, groupPath)

	if tagName != "" && !g.tagMap[tagName] {
		g.tagMap[tagName] = true
		g.doc.Tags = append(g.doc.Tags, Tag{Name: tagName})
	}

	for i := 0; i < instanceType.NumMethod(); i++ {
		method := instanceType.Method(i)
		if !isExported(method.Name) {
			continue
		}

		methodMeta := parseMethodMetaForDoc(method)
		if methodMeta.HTTPMethod == "" {
			continue
		}

		fullPath := joinPath(groupPath, methodMeta.Path)
		operation := g.buildOperation(method, methodMeta, tagName)

		g.addPathOperation(fullPath, methodMeta.HTTPMethod, operation)
	}
}

// Generate 生成文档
func (g *Generator) Generate() *Document {
	// 把收集的 schemas 放入 components
	for name, schema := range g.schemas {
		g.doc.Components.Schemas[name] = schema
	}
	return g.doc
}

// JSON 输出 JSON 格式文档
func (g *Generator) JSON() ([]byte, error) {
	doc := g.Generate()
	return json.MarshalIndent(doc, "", "  ")
}

// Handler 返回 Gin handler，提供 OpenAPI JSON 文档
func (g *Generator) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := g.JSON()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Data(http.StatusOK, "application/json", data)
	}
}

// SwaggerUIHandler 返回 Swagger UI 页面
func (g *Generator) SwaggerUIHandler(specPath string) gin.HandlerFunc {
	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <title>%s - Swagger UI</title>
  <meta charset="utf-8"/>
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({url: "%s", dom_id: '#swagger-ui', presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset]})
  </script>
</body>
</html>`, g.doc.Info.Title, specPath)

	return func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
	}
}

// --- 内部方法 ---

func (g *Generator) buildOperation(method reflect.Method, methodMeta *methodMetaInfo, tagName string) *Operation {
	op := &Operation{
		Summary:     methodMeta.Description,
		OperationID: method.Name,
		Responses: map[string]*Response{
			"200": {Description: "Successful response"},
		},
	}

	if tagName != "" {
		op.Tags = []string{tagName}
	}

	// 解析参数
	methodType := method.Type
	paramIndex := 1
	if methodType.NumIn() > 1 && methodType.In(1).String() == "*web.LandcContext" {
		paramIndex = 2
	}

	if methodType.NumIn() > paramIndex {
		paramType := methodType.In(paramIndex)
		if paramType.Kind() == reflect.Ptr {
			paramType = paramType.Elem()
		}
		if paramType.Kind() == reflect.Struct {
			g.extractParameters(op, paramType, methodMeta.HTTPMethod)
		}
	}

	// 解析返回值
	if methodType.NumOut() > 0 {
		outType := methodType.Out(0)
		if outType.Kind() == reflect.Ptr {
			outType = outType.Elem()
		}
		if outType.Kind() == reflect.Struct {
			schema := g.typeToSchema(outType)
			op.Responses["200"] = &Response{
				Description: "Successful response",
				Content: map[string]*MediaType{
					"application/json": {Schema: schema},
				},
			}
		}
	}

	return op
}

func (g *Generator) extractParameters(op *Operation, paramType reflect.Type, httpMethod string) {
	var bodyFields []reflect.StructField

	for i := 0; i < paramType.NumField(); i++ {
		field := paramType.Field(i)

		if field.Type.Name() == "Meta" {
			continue
		}

		source := field.Tag.Get("source")
		name := field.Tag.Get("name")
		if name == "" {
			name = field.Tag.Get("json")
			if name == "" {
				name = strings.ToLower(field.Name)
			}
		}
		// 去掉 json tag 中的 omitempty 等选项
		if idx := strings.Index(name, ","); idx != -1 {
			name = name[:idx]
		}

		desc := field.Tag.Get("description")
		required := strings.Contains(field.Tag.Get("binding"), "required")
		defaultVal := field.Tag.Get("d")

		switch source {
		case "query":
			param := Parameter{
				Name:        name,
				In:          "query",
				Description: desc,
				Required:    required,
				Schema:      g.fieldToSchema(field),
			}
			if defaultVal != "" {
				param.Schema.Default = defaultVal
			}
			op.Parameters = append(op.Parameters, param)
		case "path":
			op.Parameters = append(op.Parameters, Parameter{
				Name:        name,
				In:          "path",
				Description: desc,
				Required:    true,
				Schema:      g.fieldToSchema(field),
			})
		case "header":
			op.Parameters = append(op.Parameters, Parameter{
				Name:        name,
				In:          "header",
				Description: desc,
				Required:    required,
				Schema:      g.fieldToSchema(field),
			})
		default:
			// body 字段
			bodyFields = append(bodyFields, field)
		}
	}

	// 如果有 body 字段，生成 requestBody
	if len(bodyFields) > 0 && (httpMethod == "POST" || httpMethod == "PUT" || httpMethod == "PATCH") {
		schema := &Schema{
			Type:       "object",
			Properties: make(map[string]*Schema),
		}
		for _, field := range bodyFields {
			name := field.Tag.Get("json")
			if name == "" {
				name = strings.ToLower(field.Name)
			}
			if idx := strings.Index(name, ","); idx != -1 {
				name = name[:idx]
			}
			schema.Properties[name] = g.fieldToSchema(field)
			if strings.Contains(field.Tag.Get("binding"), "required") {
				schema.Required = append(schema.Required, name)
			}
		}

		op.RequestBody = &RequestBody{
			Required: true,
			Content: map[string]*MediaType{
				"application/json": {Schema: schema},
			},
		}
	}
}

func (g *Generator) fieldToSchema(field reflect.StructField) *Schema {
	schema := g.goTypeToSchema(field.Type)
	if desc := field.Tag.Get("description"); desc != "" {
		schema.Description = desc
	}
	if enum := field.Tag.Get("enum"); enum != "" {
		values := strings.Split(enum, ",")
		for _, v := range values {
			schema.Enum = append(schema.Enum, v)
		}
	}
	return schema
}

func (g *Generator) typeToSchema(t reflect.Type) *Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	name := t.Name()
	if name != "" {
		// 注册到 schemas 并使用 $ref
		if _, exists := g.schemas[name]; !exists {
			schema := g.buildStructSchema(t)
			g.schemas[name] = schema
		}
		return &Schema{Ref: "#/components/schemas/" + name}
	}

	return g.goTypeToSchema(t)
}

func (g *Generator) buildStructSchema(t reflect.Type) *Schema {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !isExported(field.Name) {
			continue
		}
		if field.Type.Name() == "Meta" {
			continue
		}

		name := field.Tag.Get("json")
		if name == "" {
			name = strings.ToLower(field.Name)
		}
		if name == "-" {
			continue
		}
		if idx := strings.Index(name, ","); idx != -1 {
			name = name[:idx]
		}

		propSchema := g.goTypeToSchema(field.Type)
		if desc := field.Tag.Get("description"); desc != "" {
			propSchema.Description = desc
		}
		schema.Properties[name] = propSchema

		if strings.Contains(field.Tag.Get("binding"), "required") {
			schema.Required = append(schema.Required, name)
		}
	}

	return schema
}

func (g *Generator) goTypeToSchema(t reflect.Type) *Schema {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return &Schema{Type: "integer", Format: "int32"}
	case reflect.Int64:
		return &Schema{Type: "integer", Format: "int64"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return &Schema{Type: "integer", Format: "int32"}
	case reflect.Uint64:
		return &Schema{Type: "integer", Format: "int64"}
	case reflect.Float32:
		return &Schema{Type: "number", Format: "float"}
	case reflect.Float64:
		return &Schema{Type: "number", Format: "double"}
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.Slice, reflect.Array:
		return &Schema{Type: "array", Items: g.goTypeToSchema(t.Elem())}
	case reflect.Map:
		return &Schema{Type: "object"}
	case reflect.Struct:
		// 特殊类型处理
		if t.String() == "time.Time" {
			return &Schema{Type: "string", Format: "date-time"}
		}
		return g.typeToSchema(t)
	case reflect.Interface:
		return &Schema{Type: "object"}
	default:
		return &Schema{Type: "string"}
	}
}

func (g *Generator) addPathOperation(path, httpMethod string, op *Operation) {
	if g.doc.Paths[path] == nil {
		g.doc.Paths[path] = &PathItem{}
	}

	pathItem := g.doc.Paths[path]
	switch strings.ToUpper(httpMethod) {
	case "GET":
		pathItem.Get = op
	case "POST":
		pathItem.Post = op
	case "PUT":
		pathItem.Put = op
	case "DELETE":
		pathItem.Delete = op
	case "PATCH":
		pathItem.Patch = op
	case "OPTIONS":
		pathItem.Options = op
	case "HEAD":
		pathItem.Head = op
	}
}

// --- 辅助函数 ---

type methodMetaInfo struct {
	Path        string
	HTTPMethod  string
	Description string
}

func parseMethodMetaForDoc(method reflect.Method) *methodMetaInfo {
	info := &methodMetaInfo{}
	methodType := method.Type

	paramIndex := 1
	if methodType.NumIn() > 1 && methodType.In(1).String() == "*web.LandcContext" {
		paramIndex = 2
	}

	if methodType.NumIn() <= paramIndex {
		info.HTTPMethod = "GET"
		info.Path = "/" + strings.ToLower(method.Name)
		return info
	}

	paramType := methodType.In(paramIndex)
	if paramType.Kind() == reflect.Ptr {
		paramType = paramType.Elem()
	}

	if paramType.Kind() != reflect.Struct {
		info.HTTPMethod = "GET"
		info.Path = "/" + strings.ToLower(method.Name)
		return info
	}

	metaData := meta.Data(reflect.New(paramType).Elem().Interface())

	if path, ok := metaData["path"].(string); ok {
		info.Path = path
	}
	if httpMethod, ok := metaData["method"].(string); ok {
		info.HTTPMethod = httpMethod
	}
	if description, ok := metaData["description"].(string); ok {
		info.Description = description
	}

	if info.Path == "" {
		info.Path = "/" + strings.ToLower(method.Name)
	}
	if info.HTTPMethod == "" {
		info.HTTPMethod = "GET"
	}

	return info
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

func getTagName(instance interface{}, groupPath string) string {
	instanceValue := reflect.ValueOf(instance)
	if instanceValue.Kind() == reflect.Ptr {
		instanceValue = instanceValue.Elem()
	}
	if instanceValue.Kind() != reflect.Struct {
		return ""
	}

	// 尝试从 Meta tag 中获取 group
	metaData := meta.Data(instanceValue.Interface())
	if group, ok := metaData["group"].(string); ok && group != "" {
		return group
	}

	// 使用类型名
	typeName := instanceValue.Type().Name()
	typeName = strings.TrimSuffix(typeName, "Controller")
	typeName = strings.TrimSuffix(typeName, "Handler")
	typeName = strings.TrimSuffix(typeName, "Api")
	if typeName != "" {
		return typeName
	}

	// 使用 group path
	if groupPath != "" {
		return strings.Trim(groupPath, "/")
	}

	return ""
}

func joinPath(group, path string) string {
	group = strings.TrimRight(group, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return group + path
}

func isExported(name string) bool {
	return name != "" && name[0] >= 'A' && name[0] <= 'Z'
}
