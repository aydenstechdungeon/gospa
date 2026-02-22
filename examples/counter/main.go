package main

import (
	"log"

	"counter/lib"
	_ "counter/routes" // Import routes to trigger init()

	"github.com/aydenstechdungeon/gospa"
)

func main() {
	app := gospa.New(gospa.Config{
		RoutesDir: "./routes",
		DevMode:   true,
		AppName:   "counter",
		DefaultState: map[string]interface{}{
			"count": lib.GlobalCounter.Count,
		},
	})

	// Register counter handlers
	lib.RegisterHandlers(app.Hub)

	if err := app.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}
