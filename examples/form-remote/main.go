package main

import (
	"log"

	_ "github.com/aydenstechdungeon/gospa/examples/form-remote/routes" // Import routes to trigger init()

	"github.com/aydenstechdungeon/gospa"
)

func main() {
	config := gospa.DefaultConfig()
	config.RoutesDir = "./routes"
	config.DevMode = true
	config.AppName = "guestbook"
	config.EnableWebSocket = true

	app := gospa.New(config)

	if err := app.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}
