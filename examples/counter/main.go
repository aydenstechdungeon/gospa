// Package main provides a simple counter example using GoSPA.
package main

import (
	"log"

	_ "github.com/aydenstechdungeon/gospa/examples/counter/routes" // Import routes to trigger init()

	"github.com/aydenstechdungeon/gospa"
)

func main() {
	app := gospa.New(gospa.Config{
		RoutesDir: "./routes",
		DevMode:   true,
		AppName:   "counter",
	})

	if err := app.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}
