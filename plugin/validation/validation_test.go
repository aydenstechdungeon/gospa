package validation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	p := New(nil)
	if p.config.SchemasDir != "schemas" {
		t.Errorf("expected default schemas dir 'schemas', got %s", p.config.SchemasDir)
	}
}

func TestGenerateTypes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "validation-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	p := New(&Config{OutputDir: tmpDir, GenerateTypes: true})
	schema := Schema{
		Name: "User",
		Fields: map[string]FieldSchema{
			"id":    {Type: "string", Required: true},
			"email": {Type: "email", Required: true},
			"age":   {Type: "number", Required: false},
		},
	}

	err = p.generateTypes(schema, tmpDir)
	if err != nil {
		t.Fatalf("failed to generate types: %v", err)
	}

	typesPath := filepath.Join(tmpDir, "User.types.ts")
	// #nosec G304
	data, err := os.ReadFile(typesPath)
	if err != nil {
		t.Fatalf("failed to read types file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "export interface User {") {
		t.Errorf("missing interface definition")
	}
	if !strings.Contains(content, "id: string;") {
		t.Errorf("missing id field")
	}
	if !strings.Contains(content, "email: string;") {
		t.Errorf("missing email field")
	}
	if !strings.Contains(content, "age?: number;") {
		t.Errorf("missing optional age field")
	}
}

func TestGenerateValibotSchema(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "validation-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	p := New(&Config{OutputDir: tmpDir, GenerateClient: true})
	schema := Schema{
		Name: "User",
		Fields: map[string]FieldSchema{
			"email": {Type: "email", Required: true},
			"name":  {Type: "string", Required: true, Min: 2.0},
		},
	}

	err = p.generateValibotSchema(schema, tmpDir)
	if err != nil {
		t.Fatalf("failed to generate valibot schema: %v", err)
	}

	schemaPath := filepath.Join(tmpDir, "User.schema.ts")
	// #nosec G304
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("failed to read schema file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "export const UserSchema = v.object({") {
		t.Errorf("missing schema definition")
	}
	if !strings.Contains(content, "v.email()") {
		t.Errorf("missing email validator")
	}
	if !strings.Contains(content, "v.minLength(2)") {
		t.Errorf("missing minLength validator")
	}
}

func TestGenerateGoValidation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "validation-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	p := New(&Config{OutputDir: tmpDir, GenerateServer: true})
	schema := Schema{
		Name: "User",
		Fields: map[string]FieldSchema{
			"email": {Type: "email", Required: true},
			"age":   {Type: "number", Required: true, Min: 18.0},
		},
	}

	err = p.generateGoValidation(schema, tmpDir)
	if err != nil {
		t.Fatalf("failed to generate go validation: %v", err)
	}

	goPath := filepath.Join(tmpDir, "User.go")
	// #nosec G304
	data, err := os.ReadFile(goPath)
	if err != nil {
		t.Fatalf("failed to read go file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "type User struct {") {
		t.Errorf("missing struct definition")
	}
	if !strings.Contains(content, "`json:\"email\" validate:\"required,email\"`") {
		t.Errorf("missing email field with tags")
	}
	if !strings.Contains(content, "`json:\"age\" validate:\"required,gte=18\"`") {
		t.Errorf("missing age field with tags")
	}
}

func TestCreateSchema(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "validation-create-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	p := New(&Config{SchemasDir: tmpDir})
	err = p.createSchema("TestSchema")
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	schemaPath := filepath.Join(tmpDir, "TestSchema.json")
	// #nosec G304
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("failed to read schema file: %v", err)
	}

	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		t.Fatalf("failed to parse schema: %v", err)
	}

	if schema.Name != "TestSchema" {
		t.Errorf("expected schema name 'TestSchema', got %s", schema.Name)
	}
	if _, ok := schema.Fields["email"]; !ok {
		t.Errorf("missing email field in template")
	}
}
