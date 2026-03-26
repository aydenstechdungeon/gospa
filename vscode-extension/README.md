# GoSPA VS Code Extension

Official Visual Studio Code extension for GoSPA, providing language support for `.gospa` Single File Components (SFC).

## Features

- **Syntax Highlighting**: Sophisticated grammars for GoSPA files, including embedded Go, TypeScript, CSS, and HTML within SFC blocks.
- **Language Server Integration**: Connects seamlessly with `gospa-lsp` for real-time validation and diagnostics.
- **GoSPA Primitives**: Support for GoSPA reactive primitives like `$state()`, `$derived()`, and `$effect()`.

## Configuration

The extension can be configured via VS Code settings:

- `gospa.lsp.path`: Custom path to the `gospa-lsp` binary. Default is `gospa-lsp`.

## Development

This extension is built using [Bun](https://bun.sh).

### Dependencies

```bash
bun install
```

### Build

To compile the extension:

```bash
bun run compile
```

To watch for changes during development:

```bash
bun run watch
```

## License

MIT
