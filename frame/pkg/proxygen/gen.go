// Package proxygen generates HTTP proxy code for controller interfaces.
//
// Usage (CLI):
//
//	landc gen proxy -type UserController -gateway-name user.controller
//
// This generates a file in the sdk/ directory of the project, producing a
// proxy struct that implements the interface using di.ProxyClient.
package proxygen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// Config configures the proxy code generator.
type Config struct {
	InterfaceName string // e.g. "UserController"
	GatewayName   string // e.g. "user.controller"
	Dir           string // Interface package directory (default: current directory)
	OutDir        string // Output directory (default: ./sdk relative to go.mod)
	SdkPkgName    string // SDK package name (default: "sdk")
	Output        string // Output file (optional, overrides OutDir+SdkPkgName)
}

// typeString converts an ast.Expr to its string representation.
func typeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X)
	case *ast.SelectorExpr:
		pkgName := typeString(t.X)
		return pkgName + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeString(t.Elt)
		}
		return "[...]" + typeString(t.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", typeString(t.Key), typeString(t.Value))
	default:
		var buf strings.Builder
		_ = printer.Fprint(&buf, token.NewFileSet(), expr)
		return buf.String()
	}
}

// methodType extracts the method signature from a field in an interface.
type methodType struct {
	Name     string
	Params   string // parameter list as string
	Results  string // return list as string
	ReqType  string // request type expression
	RespType string // response type expression
}

// checkTypeRefs checks if a type expression references any import aliases.
func checkTypeRefs(typeExpr string, aliases map[string]string, used map[string]bool) {
	for alias := range aliases {
		if strings.Contains(typeExpr, alias+".") {
			used[alias] = true
		}
	}
}

// findGoModDir walks up from dir to find the directory containing go.mod.
func findGoModDir(dir string) string {
	dir, _ = filepath.Abs(dir)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// readModuleName reads the module name from go.mod in the given directory.
func readModuleName(modDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(modDir, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(line[7:]), nil
		}
	}
	return "", fmt.Errorf("module name not found in go.mod")
}

// Generate generates proxy code for the given interface.
//
//nolint:gocyclo // 生成器配置/解析/输出分派分支多，拆分会破坏流程完整性
func Generate(cfg Config) error {
	dir := cfg.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("get current directory: %w", err)
		}
	}

	// Parse the interface package directory
	fset := token.NewFileSet()
	//nolint:staticcheck // 生成器仅需语法树，无需 go/types 类型检查；解析全部文件而非按 build tags 分组
	pkgs, err := parser.ParseDir(fset, dir, nil, 0)
	if err != nil {
		return fmt.Errorf("parse directory %s: %w", dir, err)
	}

	//nolint:staticcheck // ast.Package 仅用于按文件遍历语法树，无需 go/types 对象模型
	var pkg *ast.Package
	for _, p := range pkgs {
		pkg = p
		break
	}
	if pkg == nil {
		return fmt.Errorf("no Go package found in %s", dir)
	}

	// Collect imports and find the interface
	importAliases := make(map[string]string) // alias -> full import path
	var ifaceMethods []methodType
	var pkgName string

	for _, file := range pkg.Files {
		pkgName = file.Name.Name

		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, "\"")
			alias := ""
			if imp.Name != nil {
				alias = imp.Name.Name
			} else {
				parts := strings.Split(path, "/")
				alias = parts[len(parts)-1]
			}
			importAliases[alias] = path
		}

		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.TYPE {
				continue
			}
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if typeSpec.Name.Name != cfg.InterfaceName {
					continue
				}
				iface, ok := typeSpec.Type.(*ast.InterfaceType)
				if !ok {
					continue
				}

				for _, method := range iface.Methods.List {
					if len(method.Names) == 0 {
						continue
					}
					mt := methodType{Name: method.Names[0].Name}

					funcType, ok := method.Type.(*ast.FuncType)
					if !ok {
						continue
					}

					if funcType.Params != nil && len(funcType.Params.List) >= 2 {
						paramType := funcType.Params.List[1].Type
						mt.ReqType = typeString(paramType)

						paramParts := make([]string, 0, len(funcType.Params.List))
						for _, p := range funcType.Params.List {
							t := typeString(p.Type)
							names := make([]string, len(p.Names))
							for i, n := range p.Names {
								names[i] = n.Name
							}
							if len(names) > 0 {
								paramParts = append(paramParts, strings.Join(names, ", ")+" "+t)
							} else {
								paramParts = append(paramParts, t)
							}
						}
						mt.Params = strings.Join(paramParts, ", ")
					}

					if funcType.Results != nil && len(funcType.Results.List) >= 1 {
						respType := funcType.Results.List[0].Type
						mt.RespType = typeString(respType)

						resultParts := make([]string, 0, len(funcType.Results.List))
						for _, r := range funcType.Results.List {
							t := typeString(r.Type)
							if len(r.Names) > 0 {
								names := make([]string, len(r.Names))
								for i, n := range r.Names {
									names[i] = n.Name
								}
								resultParts = append(resultParts, strings.Join(names, ", ")+" "+t)
							} else {
								resultParts = append(resultParts, t)
							}
						}
						mt.Results = strings.Join(resultParts, ", ")
					}

					ifaceMethods = append(ifaceMethods, mt)
				}
			}
		}
	}

	if len(ifaceMethods) == 0 {
		return fmt.Errorf("interface %s not found in %s", cfg.InterfaceName, dir)
	}

	// Determine output file path
	outputPath := cfg.Output
	if outputPath == "" {
		outDir := cfg.OutDir
		if outDir == "" {
			if modDir := findGoModDir(dir); modDir != "" {
				outDir = filepath.Join(modDir, "sdk")
			} else {
				outDir = filepath.Join(dir, "..", "sdk")
			}
		}
		if err := os.MkdirAll(outDir, 0o755); err != nil {
			return fmt.Errorf("create out dir: %w", err)
		}
		// Use lowercase interface name for the file
		fileName := strings.ToLower(cfg.InterfaceName[:1]) + cfg.InterfaceName[1:] + "_proxy_gen.go"
		outputPath = filepath.Join(outDir, fileName)
	}

	// Compute interface package import path
	modDir := findGoModDir(dir)
	ifaceImportPath := ""
	if modDir != "" {
		module, err := readModuleName(modDir)
		if err == nil {
			rel, _ := filepath.Rel(modDir, dir)
			rel = filepath.ToSlash(rel)
			ifaceImportPath = module + "/" + rel
		}
	}
	if ifaceImportPath == "" {
		return fmt.Errorf("cannot determine module path for interface package (go.mod not found)")
	}

	// Generate code
	proxyStructName := strings.ToLower(cfg.InterfaceName[:1]) + cfg.InterfaceName[1:] + "Proxy"

	// Determine SDK package name
	sdkPkgName := cfg.SdkPkgName
	if sdkPkgName == "" {
		sdkPkgName = "sdk"
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, `// Code generated by landc gen proxy. DO NOT EDIT.
// Source: %s interface (%s)

package %s

import (
	"context"

	"%s"
`, cfg.InterfaceName, ifaceImportPath, sdkPkgName, ifaceImportPath)

	// Add used aliased imports (v1 types, etc.)
	usedAliases := make(map[string]bool)
	for _, m := range ifaceMethods {
		checkTypeRefs(m.RespType, importAliases, usedAliases)
		checkTypeRefs(m.ReqType, importAliases, usedAliases)
	}
	for alias := range usedAliases {
		if path, ok := importAliases[alias]; ok {
			if alias == filepath.Base(path) {
				fmt.Fprintf(&buf, "\t%q\n", path)
			} else {
				fmt.Fprintf(&buf, "\t%s %q\n", alias, path)
			}
		}
	}

	buf.WriteString("\t\"github.com/LandcLi/landc-go/frame/pkg/di\"\n")
	buf.WriteString(")\n\n")

	// init() registering the proxy factory
	fmt.Fprintf(&buf, `func init() {
	di.RegisterProxyFactory("%s", func(client *di.ProxyClient) %s.%s {
		return &%s{client: client}
	})
}

`, cfg.GatewayName, pkgName, cfg.InterfaceName, proxyStructName)

	// Proxy struct (unexported)
	fmt.Fprintf(&buf, `type %s struct {
	client *di.ProxyClient
}

`, proxyStructName)

	// Methods
	for _, m := range ifaceMethods {
		callRespType := m.RespType
		callRespType = strings.TrimPrefix(callRespType, "*")
		fmt.Fprintf(&buf, `func (p *%s) %s(%s) (%s) {
	return di.Call[%s](p.client, ctx, "%s", req)
}

`, proxyStructName, m.Name, m.Params, m.Results, callRespType, m.Name)
	}

	code := buf.String()

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	if err := os.WriteFile(outputPath, []byte(code), 0o600); err != nil {
		return fmt.Errorf("write output file %s: %w", outputPath, err)
	}

	fmt.Printf("Generated proxy: %s\n", outputPath)
	return nil
}
