# Basic and State Configuration

Core application settings and state configuration options for GoSPA.

## Basic Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `RoutesDir` | `string` | `"./routes"` | Directory containing route files |
| `RoutesFS` | `fs.FS` | `nil` | Filesystem for routes (takes precedence over RoutesDir) |
| `DevMode` | `bool` | `false` | Enable development features (logging, print routes) |
| `RuntimeScript` | `string` | `"/_gospa/runtime.js"` | Path to client runtime script |
| `StaticDir` | `string` | `"./static"` | Directory for static files |
| `StaticPrefix` | `string` | `"/static"` | URL prefix for static files |
| `AppName` | `string` | `"GoSPA App"` | Application name |
| `Logger` | `*slog.Logger` | `slog.Default()` | Structured logger |

## State Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `DefaultState` | `map[string]interface{}` | `{}` | Initial state for new sessions |
| `SerializationFormat` | `string` | `"json"` | Serialization for WebSocket: `"json"` or `"msgpack"` |
| `StateSerializer` | `StateSerializerFunc` | Auto | Overrides default outbound state serialization |
| `StateDeserializer` | `StateDeserializerFunc` | Auto | Overrides default inbound state deserialization |

## Example

```go
package main

import "github.com/aydenstechdungeon/gospa"

func main() {
    app := gospa.New(gospa.Config{
        RoutesDir: "./routes",
        AppName:   "My App",
        DevMode:   true,
        
        DefaultState: map[string]interface{}{
            "theme": "dark",
            "user":  nil,
        },
    })
    
    app.Run(":3000")
}
```
