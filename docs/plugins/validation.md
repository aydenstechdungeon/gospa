# Form Validation Plugin

Client and server-side form validation with Valibot and Go validator.

## Installation

```bash
gospa add validation
```

## Configuration

```yaml
plugins:
  validation:
    schemas_dir: ./schemas
    output_dir: ./generated/validation
```

## Regex Validation
When using the `Pattern` field in your schema, the plugin generates a `regexp` tag in the Go struct. 

> [!IMPORTANT]
> **Server-side Requirement:** If you are using `github.com/go-playground/validator/v10` for server-side validation, you must register a custom "regexp" validator as it is not a built-in tag.

## CLI Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `validation:generate` | `vg` | Generate validation code |
| `validation:create` | `vc` | Create schema file |
| `validation:list` | `vl` | List all schemas |
