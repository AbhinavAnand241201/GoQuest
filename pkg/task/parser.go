package task

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
)

// ParserError represents an error that occurred during parsing.
type ParserError struct {
	File    string
	Line    int
	Message string
}

func (e *ParserError) Error() string {
	return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Message)
}

// Parser parses Go scripts to extract task definitions.
type Parser struct {
	fset *token.FileSet
}

// NewParser creates a new Parser instance.
func NewParser() *Parser {
	return &Parser{
		fset: token.NewFileSet(),
	}
}

// ParseScript parses a Go script and extracts task definitions.
func (p *Parser) ParseScript(filename string) ([]TaskSpec, error) {
	// Parse the file
	f, err := parser.ParseFile(p.fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse file: %w", err)
	}

	var tasks []TaskSpec

	// Look for task structs
	for _, decl := range f.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			// Check if this is a task struct
			specs, err := p.extractTaskSpecs(structType, typeSpec.Name.Name)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, specs...)
		}
	}

	if len(tasks) == 0 {
		return nil, &ParserError{
			File:    filename,
			Line:    1,
			Message: "no task definitions found",
		}
	}

	return tasks, nil
}

// extractTaskSpecs extracts TaskSpecs from a struct type (one per field with a task name tag).
func (p *Parser) extractTaskSpecs(structType *ast.StructType, structName string) ([]TaskSpec, error) {
	var specs []TaskSpec
	for _, field := range structType.Fields.List {
		if field.Tag == nil {
			continue
		}
		tag := strings.Trim(field.Tag.Value, "`")
		if !strings.HasPrefix(tag, "task:") {
			continue
		}
		tagParts := strings.Split(tag, "task:")
		if len(tagParts) != 2 {
			continue
		}
		tagValue := strings.Trim(tagParts[1], "\"")
		var spec TaskSpec
		if strings.HasPrefix(tagValue, "name=") {
			spec.Name = strings.TrimPrefix(tagValue, "name=")
		}
		if strings.Contains(tagValue, "depends=") {
			// supports both name=... depends=... or just depends=...
			for _, part := range strings.Split(tagValue, " ") {
				if strings.HasPrefix(part, "depends=") {
					deps := strings.TrimPrefix(part, "depends=")
					if deps != "" {
						spec.Depends = strings.Split(deps, ",")
					}
				}
			}
		}
		if spec.Name != "" {
			for _, dep := range spec.Depends {
				if dep == "" {
					return nil, &ParserError{
						File:    p.fset.File(structType.Pos()).Name(),
						Line:    p.fset.Position(structType.Pos()).Line,
						Message: "dependency name cannot be empty",
					}
				}
			}
			specs = append(specs, spec)
		}
	}
	return specs, nil
}

// ValidateTaskStruct validates a task struct at runtime.
func ValidateTaskStruct(task interface{}) error {
	v := reflect.ValueOf(task)
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %v", v.Kind())
	}
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("task")
		if tag == "" {
			continue
		}
		if strings.HasPrefix(tag, "name=") {
			name := strings.TrimPrefix(tag, "name=")
			if name == "" {
				return fmt.Errorf("field %s: missing task:name tag", field.Name)
			}
			// Check for Run field
			runField, ok := field.Type.FieldByName("Run")
			if !ok {
				return fmt.Errorf("task %s: missing Run field", name)
			}
			if runField.Type.Kind() != reflect.Func {
				return fmt.Errorf("task %s: Run field must be a function", name)
			}
			// Check function signature: func(context.Context) (interface{}, error)
			if runField.Type.NumIn() != 1 || runField.Type.NumOut() != 2 {
				return fmt.Errorf("task %s: Run function must have signature func(context.Context) (interface{}, error)", name)
			}
			if runField.Type.In(0).String() != "context.Context" {
				return fmt.Errorf("task %s: Run function must take context.Context as first parameter", name)
			}
			if runField.Type.Out(1).String() != "error" {
				return fmt.Errorf("task %s: Run function must return error as second return value", name)
			}
			// Check that the value is non-nil
			valField := v.Field(i).FieldByName("Run")
			if valField.IsNil() {
				return fmt.Errorf("task %s: Run function is nil", name)
			}
		}
		if strings.HasPrefix(tag, "depends=") {
			deps := strings.TrimPrefix(tag, "depends=")
			if deps != "" {
				for _, dep := range strings.Split(deps, ",") {
					if dep == "" {
						return fmt.Errorf("field %s: empty dependency name", field.Name)
					}
				}
			}
		}
	}
	return nil
}
