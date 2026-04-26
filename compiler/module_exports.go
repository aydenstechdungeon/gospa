package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"strings"
	"unicode"

	"github.com/aydenstechdungeon/gospa/compiler/sfc"
)

func (c *GospaCompiler) validateModuleScriptExports(block sfc.Block) error {
	if strings.TrimSpace(block.Content) == "" {
		return nil
	}

	src := "package p\n" + block.Content
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("invalid <script context=\"module\" lang=\"go\">: %w", err)
	}

	var foundExports int
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil {
			continue
		}
		name := fn.Name.Name
		switch {
		case name == "Load":
			foundExports++
			if err := validateModuleLoadSignature(fn); err != nil {
				return fmt.Errorf("invalid module export Load: %w", err)
			}
		case strings.HasPrefix(name, "Action"):
			foundExports++
			if name == "Action" {
				return fmt.Errorf("invalid module export Action: use ActionDefault or Action<Name>")
			}
			if !isValidActionExportName(name) {
				return fmt.Errorf("invalid module export %s: Action<Name> suffix must start with an uppercase letter (e.g., ActionSave)", name)
			}
			if err := validateModuleActionSignature(fn); err != nil {
				return fmt.Errorf("invalid module export %s: %w", name, err)
			}
		}
	}

	if foundExports == 0 {
		return fmt.Errorf("module script must export at least one of: Load, ActionDefault, Action<Name>")
	}

	return nil
}

func validateModuleLoadSignature(fn *ast.FuncDecl) error {
	if fn.Recv != nil {
		return fmt.Errorf("load must be a top-level function, not a method")
	}
	if err := validateLoadContextParam(fn); err != nil {
		return err
	}
	resultTypes := extractFuncResultTypes(fn.Type.Results)
	if len(resultTypes) != 2 {
		return fmt.Errorf("expected signature func Load(c routing.LoadContext) (map[string]interface{}, error)")
	}
	if !isMapStringAnyType(resultTypes[0]) {
		return fmt.Errorf("first return value must be map[string]interface{} or map[string]any, got %s", resultTypes[0])
	}
	if normalizeType(resultTypes[1]) != "error" {
		return fmt.Errorf("second return value must be error, got %s", resultTypes[1])
	}
	return nil
}

func validateModuleActionSignature(fn *ast.FuncDecl) error {
	if fn.Recv != nil {
		return fmt.Errorf("%s must be a top-level function, not a method", fn.Name.Name)
	}
	if err := validateLoadContextParam(fn); err != nil {
		return err
	}
	resultTypes := extractFuncResultTypes(fn.Type.Results)
	if len(resultTypes) != 2 {
		return fmt.Errorf("expected signature func %s(c routing.LoadContext) (interface{}, error)", fn.Name.Name)
	}
	first := normalizeType(resultTypes[0])
	if first != "interface{}" && first != "any" {
		return fmt.Errorf("first return value must be interface{} or any, got %s", resultTypes[0])
	}
	if normalizeType(resultTypes[1]) != "error" {
		return fmt.Errorf("second return value must be error, got %s", resultTypes[1])
	}
	return nil
}

func validateLoadContextParam(fn *ast.FuncDecl) error {
	if fn.Type == nil || fn.Type.Params == nil {
		return fmt.Errorf("expected one parameter: c routing.LoadContext")
	}
	paramTypes := extractFieldTypes(fn.Type.Params)
	if len(paramTypes) != 1 {
		return fmt.Errorf("expected exactly one parameter of type routing.LoadContext")
	}
	paramType := normalizeType(paramTypes[0])
	if !strings.HasSuffix(paramType, ".LoadContext") {
		return fmt.Errorf("parameter must be routing.LoadContext (or aliased package), got %s", paramTypes[0])
	}
	return nil
}

func extractFieldTypes(fields *ast.FieldList) []string {
	if fields == nil {
		return nil
	}
	out := make([]string, 0, len(fields.List))
	for _, field := range fields.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		typeStr := exprToString(field.Type)
		for i := 0; i < count; i++ {
			out = append(out, typeStr)
		}
	}
	return out
}

func extractFuncResultTypes(results *ast.FieldList) []string {
	return extractFieldTypes(results)
}

func exprToString(expr ast.Expr) string {
	if expr == nil {
		return ""
	}
	var buf bytes.Buffer
	_ = printer.Fprint(&buf, token.NewFileSet(), expr)
	return buf.String()
}

func normalizeType(in string) string {
	return strings.ReplaceAll(strings.TrimSpace(in), " ", "")
}

func isMapStringAnyType(t string) bool {
	norm := normalizeType(t)
	return norm == "map[string]interface{}" || norm == "map[string]any"
}

func isValidActionExportName(name string) bool {
	if name == "ActionDefault" {
		return true
	}
	if !strings.HasPrefix(name, "Action") {
		return false
	}
	suffix := strings.TrimPrefix(name, "Action")
	if suffix == "" {
		return false
	}
	first := []rune(suffix)[0]
	return unicode.IsUpper(first)
}
