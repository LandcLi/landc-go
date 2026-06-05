// Package proxygen generates HTTP proxy code for controller interfaces.
//
// Usage (CLI):
//
//	landc gen proxy -type UserController -gateway-name user.controller
//
// This generates a file in the same package containing a proxy struct
// that implements the interface using di.ProxyClient.
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
	Dir           string // Package directory (default: current directory)
	Output        string // Output file (default: {dir}/{type}_proxy_gen.go)
}

// typeString converts an ast.Expr to its string representation.
func typeString(expr ast.Expr, pkgAliases map[string]string) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + typeString(t.X, pkgAliases)
	case *ast.SelectorExpr:
		pkgName := typeString(t.X, pkgAliases)
		return pkgName + "." + t.Sel.Name
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + typeString(t.Elt, pkgAliases)
		}
		return "[...]" + typeString(t.Elt, pkgAliases)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", typeString(t.Key, pkgAliases), typeString(t.Value, pkgAliases))
	default:
		var buf strings.Builder
		_ = printer.Fprint(&buf, token.NewFileSet(), expr)
		return buf.String()
	}
}

// methodType extracts the method signature from a field in an interface.
type methodType struct {
	Name       string
	Params     string // parameter list as string, e.g. "ctx context.Context, req *v1.LoginRequest"
	Results    string // return list as string, e.g. "*v1.LoginResponse, error"
	ReqType    string // request type expression, e.g. "*v1.LoginRequest"
	RespType   string // response type expression, e.g. "*v1.LoginResponse"
}

// checkTypeRefs checks if a type expression references any import aliases.
func checkTypeRefs(typeExpr string, aliases map[string]string, used map[string]bool) {
	for alias := range aliases {
		if strings.Contains(typeExpr, alias+".") {
			used[alias] = true
		}
	}
}

// Generate generates proxy code for the given interface.
func Generate(cfg Config) error {
	dir := cfg.Dir
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("get current directory: %w", err)
		}
	}

	// Parse the directory
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, 0)
	if err != nil {
		return fmt.Errorf("parse directory %s: %w", dir, err)
	}

	// Find the package
	var pkg *ast.Package
	for _, p := range pkgs {
		pkg = p
		break
	}
	if pkg == nil {
		return fmt.Errorf("no Go package found in %s", dir)
	}

	// Build import alias map and find the interface
	importAliases := make(map[string]string) // alias -> full path
	var ifaceMethods []methodType
	var pkgName string

	for _, file := range pkg.Files {
		pkgName = file.Name.Name

		// Collect import aliases
		for _, imp := range file.Imports {
			path := strings.Trim(imp.Path.Value, "\"")
			alias := ""
			if imp.Name != nil {
				alias = imp.Name.Name
			} else {
				// Default alias is the last component
				parts := strings.Split(path, "/")
				alias = parts[len(parts)-1]
			}
			importAliases[alias] = path
		}

		// Find the interface type
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

				// Extract methods
				for _, method := range iface.Methods.List {
					if len(method.Names) == 0 {
						continue // embedded interface, skip for now
					}
					mt := methodType{Name: method.Names[0].Name}

					funcType, ok := method.Type.(*ast.FuncType)
					if !ok {
						continue
					}

					// Extract request type (second param)
					if funcType.Params != nil && len(funcType.Params.List) >= 2 {
						paramType := funcType.Params.List[1].Type
						mt.ReqType = typeString(paramType, importAliases)

						// Build params string
						paramParts := make([]string, 0, len(funcType.Params.List))
						for _, p := range funcType.Params.List {
							t := typeString(p.Type, importAliases)
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

					// Extract response type (first return value)
					if funcType.Results != nil && len(funcType.Results.List) >= 1 {
						respType := funcType.Results.List[0].Type
						mt.RespType = typeString(respType, importAliases)

						// Build results string
						resultParts := make([]string, 0, len(funcType.Results.List))
						for _, r := range funcType.Results.List {
							t := typeString(r.Type, importAliases)
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
		outputPath = filepath.Join(dir, strings.ToLower(cfg.InterfaceName)+"_proxy_gen.go")
	}

	// Generate code
	proxyStructName := cfg.InterfaceName + "Proxy"

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf(`// Code generated by landc gen proxy. DO NOT EDIT.
// Source: %s interface

package %s

`, cfg.InterfaceName, pkgName))

	// Collect used imports: standard libs + aliases referenced in method types
	standardImports := []string{"context"}
	usedAliases := make(map[string]bool)

	for _, m := range ifaceMethods {
		checkTypeRefs(m.RespType, importAliases, usedAliases)
		checkTypeRefs(m.ReqType, importAliases, usedAliases)
	}

	// Build import block
	buf.WriteString("import (\n")
	for _, std := range standardImports {
		buf.WriteString(fmt.Sprintf("\t\"%s\"\n", std))
	}
	for alias := range usedAliases {
		if path, ok := importAliases[alias]; ok {
			if alias == filepath.Base(path) {
				buf.WriteString(fmt.Sprintf("\t\"%s\"\n", path))
			} else {
				buf.WriteString(fmt.Sprintf("\t%s \"%s\"\n", alias, path))
			}
		}
	}
	buf.WriteString("\t\"github.com/LandcLi/landc-go/frame/pkg/di\"\n")
	buf.WriteString(")\n\n")

	// init() registering the proxy factory
	buf.WriteString(fmt.Sprintf(`func init() {
	di.RegisterProxyFactory("%s", func(client *di.ProxyClient) %s {
		return &%s{client: client}
	})
}

`, cfg.GatewayName, cfg.InterfaceName, proxyStructName))

	// Proxy struct
	buf.WriteString(fmt.Sprintf(`type %s struct {
	client *di.ProxyClient
}

`, proxyStructName))

	// Methods
	for _, m := range ifaceMethods {
		// di.Call[Resp] returns *Resp, so if the interface returns *T, use T as type param
		callRespType := m.RespType
		if strings.HasPrefix(callRespType, "*") {
			callRespType = callRespType[1:]
		}
		buf.WriteString(fmt.Sprintf(`func (p *%s) %s(%s) (%s) {
	return di.Call[%s](p.client, ctx, "%s", req)
}

`, proxyStructName, m.Name, m.Params, m.Results, callRespType, m.Name))
	}

	code := buf.String()

	// Write file
	if err := os.WriteFile(outputPath, []byte(code), 0644); err != nil {
		return fmt.Errorf("write output file %s: %w", outputPath, err)
	}

	fmt.Printf("Generated proxy: %s\n", outputPath)
	return nil
}
