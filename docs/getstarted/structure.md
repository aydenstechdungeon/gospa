# Getting Started from Scratch

While the easiest way to start a new GoSPA project is using the `gospa create` command, understanding how to construct a GoSPA application from scratch is crucial for diagnosing issues, customizing the build pipeline, and understanding the framework's architecture.

This guide outlines exactly what files and configurations are strictly required to boot a fully functional GoSPA App.

## 1. The Project Structure 

Your minimum project requires the Go module, an entry point (`main.go`), and a `routes/` directory containing your basic views. With GoSPA, you also need to set up a `package.json` to handle client-side tooling (Bun and Tailwind).

```bash
myapp/
├── go.mod
├── main.go
├── package.json
├── static/
│   └── css/
│       └── style.css
└── routes/
    ├── _error.templ
    ├── _middleware.go
    ├── root_layout.templ
    └── page.templ
```

## 2. Server Configuration (`main.go`)

The core server file initializes GoSPA with the file-based router.

```go
package main

import (
	"log"
	"os"

	_ "yourmodule/routes" // Import routes to trigger init()
	"github.com/aydenstechdungeon/gospa"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	config := gospa.DefaultConfig()
	config.RoutesDir = "./routes"
	config.DevMode = true
	config.AppName = "My GoSPA App"

	app := gospa.New(config)
	if err := app.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
```

## 3. The Required Root Layout (`root_layout.templ`)
This file is critically important. It defines the `<html>` wrapper for your application and **must** insert three core pieces to enable GoSPA's reactivity engine:
- The base `runtime.js` script.
- The `data-gospa-islands` script block marking where dynamic island mounting happens.

```go
package routes

templ RootLayout(title string) {
	<!DOCTYPE html>
	<html lang="en" data-gospa-auto>
	<head>
		<meta charset="UTF-8"/>
		<meta name="viewport" content="width=device-width, initial-scale=1.0"/>
		<title>{ title }</title>
		<link rel="stylesheet" href="/static/css/style.css"/>
	</head>
	<body>
		{ children... }

		<!-- GoSPA core engine -->
		<script src="/_gospa/runtime.js"></script>
		<!-- Island hydration hook -->
		<script data-gospa-islands></script>
	</body>
	</html>
}
```
**Important:** If these tags are missing, your application will only render as raw HTML. Reactivity (*islands*) will fail to mount.

## 4. Frontend Tooling & Asset Bundling (`package.json`)

To compile TypeScript and CSS locally, you need a bundler. Bun is our native recommendation.

```json
{
	"name": "myapp",
	"type": "module",
	"scripts": {
		"build": "bun run build:css",
		"build:css": "tailwindcss -i ./static/css/style.css -o ./static/css/main.css"
	},
	"devDependencies": {
		"tailwindcss": "^4.0.0",
		"@tailwindcss/cli": "^4.0.0"
	}
}
```

### Common Gotcha: MIME Type Errors
If you see MIME type errors when the browser attempts to fetch island modules (e.g. `Refused to execute script from '/islands/button.js' because its MIME type ('text/plain') is not executable`), this usually indicates the server failed to interpret `.js` or `.ts` assets properly.
GoSPA natively patches this by setting `Content-Type: application/javascript` on the `/islands/` route locally, but you must ensure your `./generated` folder is transpiled properly if overriding the bundler pipeline.
