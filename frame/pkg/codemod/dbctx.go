// Package codemod 提供 landc-go 的代码迁移工具。
//
// 当前支持迁移"资源访问上下文化"：为 DAO / service 接口与方法注入 ctx 参数，
// 并把 db.GetDB() / cache.GetCache() / context.Background() 迁移为
// db.GetDBFrom(ctx) / cache.GetCacheFrom(ctx) / ctx，以支持命名资源作用域。
package codemod

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// MigrateDBContext 迁移目录下所有 .go 文件（含测试），返回修改过的文件相对路径。
//
// 迁移规则：
//  1. 接口方法（InterfaceType）与方法定义（FuncDecl，非 New* 构造函数）的第一个参数
//     若不是 context.Context，则在参数列表最前插入 `ctx context.Context`
//  2. 方法体内 `db.GetDB()` → `db.GetDBFrom(ctx)`、`cache.GetCache()` → `cache.GetCacheFrom(ctx)`
//  3. 方法体内 `context.Background()` → `ctx`
//
// 调用点（如 service 调用 dao）不在本工具范围内——迁移后请以 `go build` 的错误列表
// 逐个补齐调用处的 ctx 参数。
func MigrateDBContext(dir string) ([]string, error) {
	var modified []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		changed, err := migrateFile(path)
		if err != nil {
			return err
		}
		if changed {
			rel, relErr := filepath.Rel(dir, path)
			if relErr != nil {
				rel = path
			}
			modified = append(modified, rel)
		}
		return nil
	})
	return modified, err
}

func migrateFile(path string) (bool, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return false, err
	}

	m := &migrator{fset: fset}
	ast.Inspect(f, m.inspect)

	// 用到了 ctx 的方法若未 import context，补上
	if m.usesCtx && !importsContext(f) {
		ensureContextImport(f)
		m.changed = true
	}

	if !m.changed {
		return false, nil
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, f); err != nil {
		return false, err
	}
	//nolint:gosec // 源码文件无敏感信息，0644 便于团队共享
	return true, os.WriteFile(path, buf.Bytes(), 0o644)
}

type migrator struct {
	fset    *token.FileSet
	changed bool
	usesCtx bool // 文件内是否有方法使用了 ctx 标识符
}

func (m *migrator) inspect(n ast.Node) bool {
	switch node := n.(type) {
	case *ast.FuncDecl:
		if node.Recv == nil {
			return false // 顶层函数：不迁移
		}
		if strings.HasPrefix(node.Name.Name, "New") {
			return false // 构造函数：不加 ctx
		}
		// 仅当方法体访问资源（GetDB/GetCache/context.Background）时才加 ctx，
		// 避免误迁移不访问资源的辅助方法（纯配置解析、测试辅助等）。
		if node.Body != nil && bodyUsesResource(node.Body) &&
			node.Type.Params != nil && !hasContextParam(node.Type.Params) {
			addCtxParam(node.Type.Params, node.Type.Params.Opening+1)
			m.changed = true
		}
		if node.Body != nil {
			m.usesCtx = true
			m.migrateBody(node.Body)
		}
		return false
	case *ast.InterfaceType:
		m.migrateInterface(node)
	}
	return true
}

func (m *migrator) migrateInterface(it *ast.InterfaceType) {
	if it.Methods == nil {
		return
	}
	for _, field := range it.Methods.List {
		ft, ok := field.Type.(*ast.FuncType)
		if !ok || ft.Params == nil {
			continue
		}
		name := ""
		if len(field.Names) > 0 {
			name = field.Names[0].Name
		}
		if strings.HasPrefix(name, "New") || hasContextParam(ft.Params) {
			continue
		}
		addCtxParam(ft.Params, ft.Params.Opening+1)
		m.changed = true
	}
}

// migrateBody 处理方法体内：GetDB/GetCache 调用替换 + context.Background() 替换。
func (m *migrator) migrateBody(body *ast.BlockStmt) {
	// GetDB/GetCache 原地改写（无需替换节点）
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		switch sel.Sel.Name {
		case "GetDB":
			if len(call.Args) == 0 {
				sel.Sel.Name = "GetDBFrom"
				call.Args = []ast.Expr{ast.NewIdent("ctx")}
				m.changed = true
			}
		case "GetCache":
			if len(call.Args) == 0 {
				sel.Sel.Name = "GetCacheFrom"
				call.Args = []ast.Expr{ast.NewIdent("ctx")}
				m.changed = true
			}
		}
		return true
	})

	// context.Background() 替换为 ctx（需替换节点，自定义递归）；
	// ctx := context.Background() 等赋值被删除（返回 nil）
	var list []ast.Stmt
	for _, stmt := range body.List {
		if r := rewriteStmt(stmt, &m.changed); r != nil {
			list = append(list, r)
		}
	}
	body.List = list
}

// bodyUsesResource 报告方法体是否访问框架资源（GetDB/GetCache/context.Background）。
func bodyUsesResource(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		switch sel.Sel.Name {
		case "GetDB", "GetCache", "Background":
			found = true
			return false
		}
		return true
	})
	return found
}

// hasContextParam 报告参数列表第一个参数是否为 context.Context。
func hasContextParam(params *ast.FieldList) bool {
	if params == nil || len(params.List) == 0 {
		return false
	}
	first := params.List[0]
	if sel, ok := first.Type.(*ast.SelectorExpr); ok {
		if x, ok := sel.X.(*ast.Ident); ok && x.Name == "context" && sel.Sel.Name == "Context" {
			return true
		}
	}
	return false
}

// addCtxParam 在参数列表最前插入 ctx context.Context。
// pos 为插入位置的 token 位置（使用 Opening+1）。必须为整个 Field（含 Type 的
// context.Context 选择器）设置有效位置，否则 go/printer 会用 NoPos 算出错误的
// End() 行号，误判列表跨行而输出尾逗号（如 `func F(ctx context.Context,)`）。
func addCtxParam(params *ast.FieldList, pos token.Pos) {
	sel := &ast.SelectorExpr{
		X:   ast.NewIdent("context"),
		Sel: ast.NewIdent("Context"),
	}
	sel.X.(*ast.Ident).NamePos = pos
	sel.Sel.NamePos = pos

	ctxField := &ast.Field{
		Names: []*ast.Ident{ast.NewIdent("ctx")},
		Type:  sel,
	}
	ctxField.Names[0].NamePos = pos
	params.List = append([]*ast.Field{ctxField}, params.List...)
}

func importsContext(f *ast.File) bool {
	for _, imp := range f.Imports {
		if strings.Trim(imp.Path.Value, `"`) == "context" {
			return true
		}
	}
	return false
}

func ensureContextImport(f *ast.File) {
	imp := &ast.ImportSpec{Path: &ast.BasicLit{Kind: token.STRING, Value: `"context"`}}
	if len(f.Imports) == 0 {
		decl := &ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{imp}}
		f.Decls = append([]ast.Decl{decl}, f.Decls...)
		return
	}
	f.Imports = append(f.Imports, imp)
}

// rewriteStmt 递归改写语句中的 context.Background() 为 ctx。
// 返回 nil 表示该语句被删除（如 ctx := context.Background()，因 ctx 已为方法参数）。
//
//nolint:gocyclo // AST 遍历器的 switch 分支结构，豁免复杂度告警
func rewriteStmt(stmt ast.Stmt, changed *bool) ast.Stmt {
	switch s := stmt.(type) {
	case *ast.BlockStmt:
		var list []ast.Stmt
		for _, st := range s.List {
			if r := rewriteStmt(st, changed); r != nil {
				list = append(list, r)
			}
		}
		s.List = list
	case *ast.ExprStmt:
		s.X = rewriteExpr(s.X, changed)
	case *ast.AssignStmt:
		// ctx := context.Background() / ctx = context.Background()：
		// 方法已加 ctx 参数，删除该赋值，避免自引用 ctx := ctx。
		if isCtxBackgroundAssign(s) {
			*changed = true
			return nil
		}
		for i, r := range s.Rhs {
			s.Rhs[i] = rewriteExpr(r, changed)
		}
	case *ast.IncDecStmt:
		s.X = rewriteExpr(s.X, changed)
	case *ast.ReturnStmt:
		for i, r := range s.Results {
			s.Results[i] = rewriteExpr(r, changed)
		}
	case *ast.IfStmt:
		if s.Init != nil {
			s.Init = rewriteStmt(s.Init, changed)
		}
		s.Cond = rewriteExpr(s.Cond, changed)
		s.Body = rewriteStmt(s.Body, changed).(*ast.BlockStmt)
		if s.Else != nil {
			s.Else = rewriteStmt(s.Else, changed)
		}
	case *ast.ForStmt:
		if s.Init != nil {
			s.Init = rewriteStmt(s.Init, changed)
		}
		if s.Cond != nil {
			s.Cond = rewriteExpr(s.Cond, changed)
		}
		if s.Post != nil {
			s.Post = rewriteStmt(s.Post, changed)
		}
		s.Body = rewriteStmt(s.Body, changed).(*ast.BlockStmt)
	case *ast.RangeStmt:
		s.Key = rewriteExpr(s.Key, changed)
		s.Value = rewriteExpr(s.Value, changed)
		s.X = rewriteExpr(s.X, changed)
		s.Body = rewriteStmt(s.Body, changed).(*ast.BlockStmt)
	case *ast.DeferStmt:
		// defer/go 后的调用保持 CallExpr（defer context.Background() 属无意义代码，不处理）
		s.Call = rewriteCall(s.Call, changed).(*ast.CallExpr)
	case *ast.GoStmt:
		s.Call = rewriteCall(s.Call, changed).(*ast.CallExpr)
	case *ast.SwitchStmt:
		if s.Init != nil {
			s.Init = rewriteStmt(s.Init, changed)
		}
		s.Tag = rewriteExpr(s.Tag, changed)
		s.Body = rewriteStmt(s.Body, changed).(*ast.BlockStmt)
	case *ast.CaseClause:
		for i, e := range s.List {
			s.List[i] = rewriteExpr(e, changed)
		}
		for i, st := range s.Body {
			s.Body[i] = rewriteStmt(st, changed)
		}
	case *ast.DeclStmt:
		// var ctx = context.Background()：ctx 已为参数，删除。
		if gd, ok := s.Decl.(*ast.GenDecl); ok && gd.Tok == token.VAR {
			for _, spec := range gd.Specs {
				if vs, ok := spec.(*ast.ValueSpec); ok && len(vs.Names) == 1 &&
					vs.Names[0].Name == "ctx" && len(vs.Values) == 1 && isBackgroundCall(vs.Values[0]) {
					*changed = true
					return nil
				}
			}
		}
	case *ast.EmptyStmt, *ast.BranchStmt, *ast.LabeledStmt, *ast.SelectStmt, *ast.TypeSwitchStmt, *ast.CommClause:
		// 保持默认（覆盖常见场景即可）
	}
	return stmt
}

// isCtxBackgroundAssign 判断是否为 ctx := context.Background() 或 ctx = context.Background()。
func isCtxBackgroundAssign(s *ast.AssignStmt) bool {
	if len(s.Lhs) != 1 || len(s.Rhs) != 1 {
		return false
	}
	id, ok := s.Lhs[0].(*ast.Ident)
	if !ok || id.Name != "ctx" {
		return false
	}
	return isBackgroundCall(s.Rhs[0])
}

// isBackgroundCall 判断表达式是否为 context.Background() 调用。
func isBackgroundCall(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	x, ok := sel.X.(*ast.Ident)
	return ok && x.Name == "context" && sel.Sel.Name == "Background" && len(call.Args) == 0
}

// rewriteExpr 递归改写表达式中的 context.Background() 为 ctx。
func rewriteExpr(e ast.Expr, changed *bool) ast.Expr {
	if call, ok := e.(*ast.CallExpr); ok {
		return rewriteCall(call, changed)
	}
	switch expr := e.(type) {
	case *ast.BinaryExpr:
		expr.X = rewriteExpr(expr.X, changed)
		expr.Y = rewriteExpr(expr.Y, changed)
	case *ast.UnaryExpr:
		expr.X = rewriteExpr(expr.X, changed)
	case *ast.ParenExpr:
		expr.X = rewriteExpr(expr.X, changed)
	case *ast.IndexExpr:
		expr.X = rewriteExpr(expr.X, changed)
		expr.Index = rewriteExpr(expr.Index, changed)
	case *ast.SelectorExpr:
		expr.X = rewriteExpr(expr.X, changed)
	case *ast.CompositeLit:
		for i, elt := range expr.Elts {
			expr.Elts[i] = rewriteExpr(elt, changed)
		}
	case *ast.KeyValueExpr:
		expr.Key = rewriteExpr(expr.Key, changed)
		expr.Value = rewriteExpr(expr.Value, changed)
	case *ast.StarExpr:
		expr.X = rewriteExpr(expr.X, changed)
	case *ast.FuncLit:
		if expr.Body != nil {
			expr.Body = rewriteStmt(expr.Body, changed).(*ast.BlockStmt)
		}
	}
	return e
}

// rewriteCall 处理调用表达式：context.Background() → ctx，其余递归参数。
func rewriteCall(call *ast.CallExpr, changed *bool) ast.Expr {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		if x, ok := sel.X.(*ast.Ident); ok && x.Name == "context" && sel.Sel.Name == "Background" && len(call.Args) == 0 {
			*changed = true
			return ast.NewIdent("ctx")
		}
	}
	call.Fun = rewriteExpr(call.Fun, changed)
	for i, arg := range call.Args {
		call.Args[i] = rewriteExpr(arg, changed)
	}
	return call
}
