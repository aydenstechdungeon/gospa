package main

import (
	"log"

	"github.com/aydenstechdungeon/gospa"
	_ "todo/routes" // Import routes to trigger init()
)

func main() {
	app := gospa.New(gospa.Config{
		RoutesDir:   "./routes",
		DevMode:     true,
		AppName:     "todo",
	})

	if err := app.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}
