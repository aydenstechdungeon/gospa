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

## CLI Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `validation:generate` | `vg` | Generate validation code |
| `validation:create` | `vc` | Create schema file |
| `validation:list` | `vl` | List all schemas |
