package main

import (
	"log"

	_ "form-remote/routes" // Import routes to trigger init()

	"github.com/aydenstechdungeon/gospa"
)

func main() {
	app := gospa.New(gospa.Config{
		RoutesDir: "./routes",
		DevMode:   true,
		AppName:   "guestbook",
		WebSocket: true, // Enable WebSocket for real-time updates
	})

	if err := app.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}
