#!/bin/bash
set -e

# Project root setup
PROJECT_ROOT=$(cd "$(dirname "$0")/.." && pwd)
cd "$PROJECT_ROOT"

# Ensure bin directory exists
mkdir -p "$PROJECT_ROOT/bin"

echo "Step 1: Compiling GoSPA Language Server (LSP)..."
go build -o "$PROJECT_ROOT/bin/gospa-lsp" "$PROJECT_ROOT/cmd/gospa-lsp/main.go"
echo "✓ LSP binary created at: $PROJECT_ROOT/bin/gospa-lsp"

echo -e "\nStep 2: Preparing VS Code Extension..."
cd "$PROJECT_ROOT/vscode-extension"

# Install dependencies if node_modules don't exist
if [ ! -d "node_modules" ]; then
    echo "Installing Bun dependencies..."
    bun install
fi

echo "Compiling extension logic..."
bun run compile

echo "Packaging extension as VSIX..."
# vsce package might prompt for 'y' if repository is missing, so we use --no-interaction or check if we can skip it.
# Bunx usually handles this.
bunx vsce package --no-git-selection || bunx vsce package --no-git-selection --allow-missing-repository

echo -e "\n--------------------------------------------------"
echo "Setup Complete!"
echo "--------------------------------------------------"
echo "1. The LSP binary is at: $PROJECT_ROOT/bin/gospa-lsp"
echo "2. The Extension is at: $PROJECT_ROOT/vscode-extension/$(ls -t *.vsix | head -n 1)"
echo ""
echo "Please install the .vsix in VS Code and ensure '$PROJECT_ROOT/bin/gospa-lsp'"
echo "is in your PATH or configured in VS Code settings as 'gospa.lsp.path'."
echo "--------------------------------------------------"
