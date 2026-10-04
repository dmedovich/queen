package cli

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/dmedovich/queen"
)

// checkRegistry compares statically declared Queen migrations with the
// migrations linked into the CLI binary. It never opens a database.
func checkRegistry(dir string, registered []queen.Migration) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations directory: %w", err)
	}

	declared := make(map[string]string)
	var sourceFiles int
	for _, entry := range files {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		sourceFiles++
		path := filepath.Join(dir, entry.Name())
		versions, err := sourceVersions(path)
		if err != nil {
			return err
		}
		for _, version := range versions {
			if previous, exists := declared[version]; exists {
				return fmt.Errorf("migration %s is declared in both %s and %s", version, previous, path)
			}
			declared[version] = path
		}
	}
	if sourceFiles == 0 {
		return fmt.Errorf("no Go source files found in %s", dir)
	}
	if len(declared) == 0 {
		return fmt.Errorf("no Queen migration declarations found in %s", dir)
	}

	linked := make(map[string]bool, len(registered))
	for _, migration := range registered {
		if (migration.UpFunc != nil || migration.DownFunc != nil) && migration.ManualChecksum == "" {
			return fmt.Errorf("go migration %s requires ManualChecksum for CI verification", migration.Version)
		}
		linked[migration.Version] = true
	}
	var missing, extra []string
	for version, path := range declared {
		if !linked[version] {
			missing = append(missing, fmt.Sprintf("%s (%s)", version, path))
		}
	}
	for version := range linked {
		if _, exists := declared[version]; !exists {
			extra = append(extra, version)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 || len(extra) > 0 {
		return fmt.Errorf("registry mismatch: unregistered source migrations: [%s]; registered without a source declaration: [%s]", strings.Join(missing, ", "), strings.Join(extra, ", "))
	}
	return nil
}

func sourceVersions(path string) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	aliases := make(map[string]bool)
	for _, imp := range file.Imports {
		importPath, err := strconv.Unquote(imp.Path.Value)
		if err != nil || importPath != "github.com/dmedovich/queen" {
			continue
		}
		alias := "queen"
		if imp.Name != nil {
			alias = imp.Name.Name
		}
		aliases[alias] = true
	}
	var versions []string
	var scanErr error
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok || !isQueenMigrationType(literal.Type, aliases) {
			return true
		}
		var versionExpr ast.Expr
		for _, element := range literal.Elts {
			field, ok := element.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			if key, ok := field.Key.(*ast.Ident); ok && key.Name == "Version" {
				versionExpr = field.Value
				break
			}
		}
		if value, ok := versionExpr.(*ast.BasicLit); ok && value.Kind == token.STRING {
			version, err := strconv.Unquote(value.Value)
			if err == nil && version != "" {
				versions = append(versions, version)
				return true
			}
		}
		position := fset.Position(literal.Pos())
		scanErr = fmt.Errorf("%s: queen migration requires a literal Version for registry verification", position)
		return false
	})
	if scanErr != nil {
		return nil, scanErr
	}
	return versions, nil
}

func isQueenMigrationType(expr ast.Expr, aliases map[string]bool) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || (selector.Sel.Name != "M" && selector.Sel.Name != "Migration") {
		return false
	}
	ident, ok := selector.X.(*ast.Ident)
	return ok && aliases[ident.Name]
}
