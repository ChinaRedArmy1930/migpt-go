//go:generate go run tool_gen.go
package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"migpt-go/doc"
	"os"
	"text/template"
)

func main() {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, "../llm/tools/func", nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	var tools []struct {
		Name string
	}

	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, decl := range f.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || !hasToolRegister(fn.Doc) {
					continue
				}

				// 验证函数签名
				if !validateFuncSignature(fn) {
					log.Printf("函数 %s 签名不符合要求，已跳过", fn.Name.Name)
					continue
				}

				tools = append(tools, struct{ Name string }{fn.Name.Name})
			}
		}
	}

	generateRegistry(tools)
}

func validateFuncSignature(fn *ast.FuncDecl) bool {
	// 验证参数：必须是 (json.RawMessage)
	if len(fn.Type.Params.List) != 1 {
		return false
	}
	selectorExpr, ok := fn.Type.Params.List[0].Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	// 提取包名和类型名
	pkgIdent, ok := selectorExpr.X.(*ast.Ident)
	if !ok {
		return false
	}
	pkgName := pkgIdent.Name          // 包名
	typeName := selectorExpr.Sel.Name // 类型名

	if pkgName != "json" || typeName != "RawMessage" {
		return false
	}

	// 验证返回值：必须是 (string, error)
	if len(fn.Type.Results.List) != 2 {
		return false
	}

	if ret1, ok1 := fn.Type.Results.List[0].Type.(*ast.Ident); !ok1 || ret1.Name != "string" {
		return false
	}
	if ret2, ok2 := fn.Type.Results.List[1].Type.(*ast.Ident); !ok2 || ret2.Name != "error" {
		return false
	}

	return true
}

func generateRegistry(tools []struct{ Name string }) {
	tmpl := template.Must(template.New("").Parse(doc.ToolGenTemplate))
	f, err := os.Create("../llm/tools/gen/tools_registry.gen.go")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	if err := tmpl.Execute(f, tools); err != nil {
		log.Fatal(err)
	}
}

func hasToolRegister(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, c := range doc.List {
		if c.Text == "//tool:register" {
			return true
		}
	}
	return false
}
