// Package validation provides form validation for GoSPA projects.
// Uses Valibot on the client-side (~1.5KB gzipped) and Go validator on server-side.
package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/aydenstechdungeon/gospa/plugin"
)

// Plugin provides form validation capabilities.
type Plugin struct {
	config *Config
}

// Config holds validation plugin configuration.
type Config struct {
	// SchemasDir is where validation schemas are stored.
	SchemasDir string `yaml:"schemas_dir" json:"schemasDir"`

	// OutputDir is where generated validation code is written.
	OutputDir string `yaml:"output_dir" json:"outputDir"`

	// GenerateTypes generates TypeScript types from schemas.
	GenerateTypes bool `yaml:"generate_types" json:"generateTypes"`

	// GenerateServer generates Go server-side validation.
	GenerateServer bool `yaml:"generate_server" json:"generateServer"`

	// GenerateClient generates Valibot client-side validation.
	GenerateClient bool `yaml:"generate_client" json:"generateClient"`

	// StrictMode enables strict validation (no unknown fields).
	StrictMode bool `yaml:"strict_mode" json:"strictMode"`

	// CustomValidators is a list of custom validator names.
	CustomValidators []string `yaml:"custom_validators" json:"customValidators"`
}

// Schema represents a validation schema definition.
type Schema struct {
	Name   string                 `json:"name"`
	Fields map[string]FieldSchema `json:"fields"`
}

// FieldSchema represents a field in a validation schema.
type FieldSchema struct {
	Type     string `json:"type"`     // string, number, boolean, date, email, url, uuid, etc.
	Required bool   `json:"required"` // whether field is required
	Min      any    `json:"min"`      // minimum value (number) or length (string)
	Max      any    `json:"max"`      // maximum value (number) or length (string)
	Pattern  string `json:"pattern"`  // regex pattern for strings
	Message  string `json:"message"`  // custom error message
	Default  any    `json:"default"`  // default value
}

// DefaultConfig returns the default validation configuration.
func DefaultConfig() *Config {
	return &Config{
		SchemasDir:       "schemas",
		OutputDir:        "generated/validation",
		GenerateTypes:    true,
		GenerateServer:   true,
		GenerateClient:   true,
		StrictMode:       true,
		CustomValidators: []string{},
	}
}

// New creates a new Validation plugin.
func New(cfg *Config) *Plugin {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &Plugin{config: cfg}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "validation"
}

// Init initializes the validation plugin.
func (p *Plugin) Init() error {
	// Create directories
	dirs := []string{p.config.SchemasDir, p.config.OutputDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// Dependencies returns required dependencies.
func (p *Plugin) Dependencies() []plugin.Dependency {
	deps := []plugin.Dependency{
		// Go validator for server-side
		{Type: plugin.DepGo, Name: "github.com/go-playground/validator/v10", Version: "latest"},
	}

	// Valibot for client-side (lightweight ~1.5KB gzipped)
	if p.config.GenerateClient {
		deps = append(deps, plugin.Dependency{
			Type: plugin.DepBun, Name: "valibot", Version: "latest",
		})
	}

	return deps
}

// OnHook handles lifecycle hooks.
func (p *Plugin) OnHook(hook plugin.Hook, ctx map[string]interface{}) error {
	switch hook {
	case plugin.BeforeBuild, plugin.BeforeDev:
		projectDir, _ := ctx["project_dir"].(string)
		if projectDir == "" {
			projectDir = "."
		}
		return p.generateValidation(projectDir)

	case plugin.AfterGenerate:
		projectDir, _ := ctx["project_dir"].(string)
		if projectDir == "" {
			projectDir = "."
		}
		return p.generateValidation(projectDir)
	}
	return nil
}

// Commands returns custom CLI commands.
func (p *Plugin) Commands() []plugin.Command {
	return []plugin.Command{
		{
			Name:        "validation:generate",
			Alias:       "vg",
			Description: "Generate validation code from schemas",
			Action: func(args []string) error {
				projectDir := "."
				if len(args) > 0 {
					projectDir = args[0]
				}
				return p.generateValidation(projectDir)
			},
		},
		{
			Name:        "validation:create",
			Alias:       "vc",
			Description: "Create a new validation schema",
			Action: func(args []string) error {
				if len(args) == 0 {
					return fmt.Errorf("schema name required")
				}
				return p.createSchema(args[0])
			},
		},
		{
			Name:        "validation:list",
			Alias:       "vl",
			Description: "List all validation schemas",
			Action: func(args []string) error {
				projectDir := "."
				if len(args) > 0 {
					projectDir = args[0]
				}
				return p.listSchemas(projectDir)
			},
		},
	}
}

// generateValidation generates validation code from schemas.
func (p *Plugin) generateValidation(projectDir string) error {
	schemasDir := filepath.Join(projectDir, p.config.SchemasDir)
	outputDir := filepath.Join(projectDir, p.config.OutputDir)

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return err
	}

	// Read all schema files
	entries, err := os.ReadDir(schemasDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No schemas directory found, skipping validation generation")
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		// Read schema file
		schemaPath := filepath.Join(schemasDir, entry.Name())
		data, err := os.ReadFile(filepath.Clean(schemaPath))
		if err != nil {
			return fmt.Errorf("failed to read schema %s: %w", entry.Name(), err)
		}

		var schema Schema
		if err := json.Unmarshal(data, &schema); err != nil {
			return fmt.Errorf("failed to parse schema %s: %w", entry.Name(), err)
		}

		validID := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
		if !validID.MatchString(schema.Name) {
			return fmt.Errorf("invalid schema name: %s in %s", schema.Name, entry.Name())
		}

		// Generate TypeScript types
		if p.config.GenerateTypes {
			if err := p.generateTypes(schema, outputDir); err != nil {
				return err
			}
		}

		// Generate Valibot client validation
		if p.config.GenerateClient {
			if err := p.generateValibotSchema(schema, outputDir); err != nil {
				return err
			}
		}

		// Generate Go server validation
		if p.config.GenerateServer {
			if err := p.generateGoValidation(schema, outputDir); err != nil {
				return err
			}
		}
	}

	return nil
}

// generateTypes generates TypeScript type definitions.
func (p *Plugin) generateTypes(schema Schema, outputDir string) error {
	var sb strings.Builder
	sb.WriteString("// Auto-generated TypeScript types from schema\n")
	sb.WriteString("// Do not edit manually\n\n")

	fmt.Fprintf(&sb, "export interface %s {\n", schema.Name)

	// Sort keys for deterministic output
	var keys []string
	for name := range schema.Fields {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	for _, name := range keys {
		field := schema.Fields[name]
		tsType := p.goTypeToTS(field.Type)
		optional := ""
		if !field.Required {
			optional = "?"
		}
		fmt.Fprintf(&sb, "  %s%s: %s;\n", name, optional, tsType)
	}
	sb.WriteString("}\n")

	filename := fmt.Sprintf("%s.types.ts", schema.Name)
	return os.WriteFile(filepath.Join(outputDir, filename), []byte(sb.String()), 0600)
}

// generateValibotSchema generates Valibot validation schema.
func (p *Plugin) generateValibotSchema(schema Schema, outputDir string) error {
	var sb strings.Builder
	sb.WriteString("// Auto-generated Valibot validation schema\n")
	sb.WriteString("// Do not edit manually\n\n")
	sb.WriteString("import * as v from 'valibot';\n\n")

	fmt.Fprintf(&sb, "export const %sSchema = v.object({\n", schema.Name)

	// Sort keys for deterministic output
	var keys []string
	for name := range schema.Fields {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	for _, name := range keys {
		field := schema.Fields[name]
		validator := p.fieldToValibot(field)
		if field.Required {
			fmt.Fprintf(&sb, "  %s: %s,\n", name, validator)
		} else {
			fmt.Fprintf(&sb, "  %s: v.optional(%s),\n", name, validator)
		}
	}
	sb.WriteString("});\n\n")

	fmt.Fprintf(&sb, "export type %s = v.InferOutput<typeof %sSchema>;\n", schema.Name, schema.Name)

	filename := fmt.Sprintf("%s.schema.ts", schema.Name)
	return os.WriteFile(filepath.Join(outputDir, filename), []byte(sb.String()), 0600)
}

// generateGoValidation generates Go validation structs.
func (p *Plugin) generateGoValidation(schema Schema, outputDir string) error {
	var sb strings.Builder
	sb.WriteString("// Auto-generated Go validation structs\n")
	sb.WriteString("// Do not edit manually\n\n")
	sb.WriteString("package validation\n\n")

	fmt.Fprintf(&sb, "type %s struct {\n", schema.Name)

	// Sort keys for deterministic output
	var keys []string
	for name := range schema.Fields {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	for _, name := range keys {
		field := schema.Fields[name]
		goType := p.tsTypeToGo(field.Type)
		tags := p.generateValidateTags(field)

		jsonTag := name
		if !field.Required {
			jsonTag += ",omitempty"
		}

		fmt.Fprintf(&sb, "  %s %s `json:\"%s\" validate:\"%s\"`\n",
			p.capitalize(name), goType, jsonTag, tags)
	}
	sb.WriteString("}\n")

	filename := fmt.Sprintf("%s.go", schema.Name)
	return os.WriteFile(filepath.Join(outputDir, filename), []byte(sb.String()), 0600)
}

// fieldToValibot converts a field schema to Valibot validator.
func (p *Plugin) fieldToValibot(field FieldSchema) string {
	switch field.Type {
	case "string":
		return p.stringToValibot(field)
	case "number", "integer":
		return p.numberToValibot(field)
	case "boolean":
		return "v.boolean()"
	case "date":
		return "v.date()"
	case "email":
		return "v.pipe(v.string(), v.email())"
	case "url":
		return "v.pipe(v.string(), v.url())"
	case "uuid":
		return "v.pipe(v.string(), v.uuid())"
	default:
		return "v.any()"
	}
}

// stringToValibot generates Valibot string validator.
func (p *Plugin) stringToValibot(field FieldSchema) string {
	var parts []string
	parts = append(parts, "v.string()")

	if minVal, ok := field.Min.(float64); ok {
		parts = append(parts, fmt.Sprintf("v.minLength(%d)", int(minVal)))
	}
	if maxVal, ok := field.Max.(float64); ok {
		parts = append(parts, fmt.Sprintf("v.maxLength(%d)", int(maxVal)))
	}
	if field.Pattern != "" {
		parts = append(parts, fmt.Sprintf("v.regex(/%s/)", field.Pattern))
	}

	if len(parts) == 1 {
		return parts[0]
	}
	return fmt.Sprintf("v.pipe(%s)", strings.Join(parts, ", "))
}

// numberToValibot generates Valibot number validator.
func (p *Plugin) numberToValibot(field FieldSchema) string {
	var parts []string
	parts = append(parts, "v.number()")

	if minVal, ok := field.Min.(float64); ok {
		parts = append(parts, fmt.Sprintf("v.minValue(%d)", int(minVal)))
	}
	if maxVal, ok := field.Max.(float64); ok {
		parts = append(parts, fmt.Sprintf("v.maxValue(%d)", int(maxVal)))
	}

	if len(parts) == 1 {
		return parts[0]
	}
	return fmt.Sprintf("v.pipe(%s)", strings.Join(parts, ", "))
}

// generateValidateTags generates Go validate tags.
func (p *Plugin) generateValidateTags(field FieldSchema) string {
	var tags []string

	if field.Required {
		tags = append(tags, "required")
	}

	switch field.Type {
	case "email":
		tags = append(tags, "email")
	case "url":
		tags = append(tags, "url")
	case "uuid":
		tags = append(tags, "uuid")
	}

	if minVal, ok := field.Min.(float64); ok {
		if field.Type == "string" {
			tags = append(tags, fmt.Sprintf("min=%d", int(minVal)))
		} else {
			tags = append(tags, fmt.Sprintf("gte=%d", int(minVal)))
		}
	}

	if maxVal, ok := field.Max.(float64); ok {
		if field.Type == "string" {
			tags = append(tags, fmt.Sprintf("max=%d", int(maxVal)))
		} else {
			tags = append(tags, fmt.Sprintf("lte=%d", int(maxVal)))
		}
	}

	return strings.Join(tags, ",")
}

// goTypeToTS converts Go type to TypeScript type.
func (p *Plugin) goTypeToTS(typ string) string {
	switch typ {
	case "string", "email", "url", "uuid":
		return "string"
	case "number", "integer":
		return "number"
	case "boolean":
		return "boolean"
	case "date":
		return "Date"
	case "array":
		return "any[]"
	default:
		return "any"
	}
}

// tsTypeToGo converts TypeScript type to Go type.
func (p *Plugin) tsTypeToGo(typ string) string {
	switch typ {
	case "string", "email", "url", "uuid":
		return "string"
	case "number", "integer":
		return "int"
	case "boolean":
		return "bool"
	case "date":
		return "time.Time"
	case "array":
		return "[]any"
	default:
		return "any"
	}
}

// capitalize capitalizes the first letter of a string.
func (p *Plugin) capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// createSchema creates a new validation schema template.
func (p *Plugin) createSchema(name string) error {
	validID := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	if !validID.MatchString(name) {
		return fmt.Errorf("invalid schema name: %s", name)
	}

	schema := Schema{
		Name: name,
		Fields: map[string]FieldSchema{
			"id":        {Type: "string", Required: true},
			"name":      {Type: "string", Required: true, Min: 1, Max: 100},
			"email":     {Type: "email", Required: true},
			"active":    {Type: "boolean", Required: false, Default: true},
			"createdAt": {Type: "date", Required: false},
		},
	}

	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s.json", name)
	return os.WriteFile(filepath.Join(p.config.SchemasDir, filename), data, 0600)
}

// listSchemas lists all validation schemas.
func (p *Plugin) listSchemas(projectDir string) error {
	schemasDir := filepath.Join(projectDir, p.config.SchemasDir)

	entries, err := os.ReadDir(schemasDir)
	if err != nil {
		return err
	}

	fmt.Println("Validation schemas:")
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			name := strings.TrimSuffix(entry.Name(), ".json")
			fmt.Printf("  - %s\n", name)
		}
	}
	return nil
}

// GetConfig returns the current configuration.
func (p *Plugin) GetConfig() *Config {
	return p.config
}

// Ensure Plugin implements CLIPlugin interface.
var _ plugin.CLIPlugin = (*Plugin)(nil)
