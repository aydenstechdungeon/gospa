# GoSPA Language Server Protocol (LSP)

The `gospa-lsp` is a Go-based language server implementation designed for `.gospa` Single File Components (SFC). It provides structure-aware diagnostics and IntelliSense to enhance the developer experience.

## Features

- **Structural Diagnostics**: Real-time validation of SFC block conventions (e.g., preventing multiple `<template>` or `<script lang="go">` blocks).
- **GoSPA Rune Completion**: IntelliSense for reactive primitives:
  - `$state()`
  - `$derived()`
  - `$effect()`
  - `$props()`
  - `$inspect()`
- **Document Tracking**: Robust synchronization with the LSP client for high-performance updates.
- **Protocol Support**:
  - `textDocument/didOpen`
  - `textDocument/didChange` (Full sync)
  - `textDocument/completion`
  - `textDocument/publishDiagnostics`

## Building

To build the LSP binary:

```bash
go build -o ../../bin/gospa-lsp ./main.go
```

## Integration

This server communicates over `stdio` and is designed to be launched by language clients like the GoSPA VS Code extension.

## License

MIT
