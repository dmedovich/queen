package cli

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
)

// prepareRegistration adds the generated migration to a conventional
// migrations.Register function. A custom registry remains under user control.
func prepareRegistration(path, variable string) ([]byte, bool, error) {
	source, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read migration registry: %w", err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, source, parser.ParseComments)
	if err != nil {
		return nil, false, fmt.Errorf("parse migration registry: %w", err)
	}
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name != "Register" || fn.Body == nil {
			continue
		}
		if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 || len(fn.Type.Params.List[0].Names) != 1 || !isQueenParameter(file, fn.Type.Params.List[0].Type) {
			return nil, false, nil
		}
		parameter := fn.Type.Params.List[0].Names[0].Name
		if parameter == "_" {
			return nil, false, nil
		}
		if len(fn.Body.List) > 0 {
			if _, ok := fn.Body.List[len(fn.Body.List)-1].(*ast.ReturnStmt); ok {
				return nil, false, nil
			}
		}
		offset := fset.File(fn.Body.Rbrace).Offset(fn.Body.Rbrace)
		var updated bytes.Buffer
		updated.Write(source[:offset])
		if offset > 0 && source[offset-1] != '\n' {
			updated.WriteByte('\n')
		}
		fmt.Fprintf(&updated, "\t%s.MustAdd(%s)\n", parameter, variable)
		updated.Write(source[offset:])
		formatted, err := format.Source(updated.Bytes())
		if err != nil {
			return nil, false, fmt.Errorf("format migration registry: %w", err)
		}
		return formatted, true, nil
	}
	return nil, false, nil
}

func isQueenParameter(file *ast.File, expr ast.Expr) bool {
	pointer, ok := expr.(*ast.StarExpr)
	if !ok {
		return false
	}
	selector, ok := pointer.X.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Queen" {
		return false
	}
	alias, ok := selector.X.(*ast.Ident)
	if !ok {
		return false
	}
	for _, imp := range file.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil || importPath != "github.com/dmedovich/queen" {
			continue
		}
		name := "queen"
		if imp.Name != nil {
			name = imp.Name.Name
		}
		return alias.Name == name
	}
	return false
}

func replaceRegistry(path string, content []byte) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".queen-registry-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temp.Name()) }()
	if err := temp.Chmod(info.Mode().Perm()); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}
