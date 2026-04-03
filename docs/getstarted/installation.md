# Installation

Install the GoSPA CLI and set up your environment for high-performance Go web development.

## Prerequisites

Before installing GoSPA, ensure you have the following tools installed:

- **Go 1.25+**: Required for the backend and Templ generation. [Download Go](https://go.dev/dl/)
- **Bun**: Required for client-side asset processing and fast TypeScript transpilation. [Install Bun](https://bun.sh/)
- **Templ CLI**: The core template engine for GoSPA.
  ```bash
  go install github.com/a-h/templ/cmd/templ@latest
  ```

## 1. Install GoSPA CLI

The GoSPA CLI is the central hub for your project lifecycle, from scaffolding to production builds.

```bash
go install github.com/aydenstechdungeon/gospa/cmd/gospa@latest
```

## 2. Verify Installation

Check if the CLI is correctly installed by running:

```bash
gospa version
```

## 3. Next Steps

Now that you have the CLI installed, you can:

- [Quick Start Guide](quickstart) - Build your first app in 5 minutes.
- [Project Structure](structure) - Learn about the file hierarchy.
- [CLI Reference](../cli) - Explore all available commands.
