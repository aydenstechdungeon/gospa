package main

import (
	"context"
	"log"

	"github.com/aydenstechdungeon/gospa"
	"github.com/aydenstechdungeon/gospa/routing"
	_ "github.com/gofiber/fiber/v2"
	"test/lib"
	_ "test/routes" // Import routes to trigger init()
)

func main() {
	app := gospa.New(gospa.Config{
		RoutesDir: "./routes",
		DevMode:   true,
		AppName:   "test",
		DefaultState: map[string]interface{}{
			"count": lib.GlobalCounter.Count,
		},
	})

	if err := app.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}

func init() {
	routing.RegisterRemoteAction("greet", func(ctx context.Context, input any) (any, error) {
		return "World", nil
	})
}
