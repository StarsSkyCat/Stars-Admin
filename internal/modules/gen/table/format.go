package table

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"path"
	"slices"
	"strconv"
	"strings"
)

// formatGenerated 仅处理本服务的固定模板，不启动 go/goimports 或读取外部模板。
func formatGenerated(filename string, source []byte) ([]byte, error) {
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, filename, source, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return nil, err
	}
	used := map[string]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		if selector, ok := node.(*ast.SelectorExpr); ok {
			if name, ok := selector.X.(*ast.Ident); ok {
				used[name.Name] = true
			}
		}
		return true
	})
	standard, external := []string{}, []string{}
	start, end := -1, -1
	for _, decl := range file.Decls {
		group, ok := decl.(*ast.GenDecl)
		if !ok || group.Tok != token.IMPORT {
			continue
		}
		if start < 0 {
			start = set.Position(group.Pos()).Offset
		}
		end = set.Position(group.End()).Offset
		for _, item := range group.Specs {
			spec := item.(*ast.ImportSpec)
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return nil, err
			}
			name := path.Base(importPath)
			prefix := ""
			if spec.Name != nil {
				name = spec.Name.Name
				prefix = name + " "
			}
			if name != "_" && name != "." && !used[name] {
				continue
			}
			line := prefix + strconv.Quote(importPath)
			first, _, _ := strings.Cut(importPath, "/")
			if strings.Contains(first, ".") {
				external = append(external, line)
			} else {
				standard = append(standard, line)
			}
		}
	}
	if start < 0 {
		return format.Source(source)
	}
	slices.Sort(standard)
	slices.Sort(external)
	var imports bytes.Buffer
	if len(standard)+len(external) > 0 {
		imports.WriteString("import (\n")
		for _, line := range standard {
			imports.WriteString(line + "\n")
		}
		if len(standard) > 0 && len(external) > 0 {
			imports.WriteByte('\n')
		}
		for _, line := range external {
			imports.WriteString(line + "\n")
		}
		imports.WriteString(")")
	}
	var result bytes.Buffer
	result.Write(source[:start])
	result.Write(imports.Bytes())
	result.Write(source[end:])
	return format.Source(result.Bytes())
}
