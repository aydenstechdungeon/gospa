package main

import (
	"log"

	_ "todo/routes" // Import routes to trigger init()

	"github.com/aydenstechdungeon/gospa"
)

func main() {
	app := gospa.New(gospa.Config{
		RoutesDir: "./routes",
		DevMode:   true,
		AppName:   "todo",
	})

	if err := app.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}
