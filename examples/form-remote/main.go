// Package main provides a guestbook example using remote actions and WebSockets in GoSPA.
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
	config.ContentSecurityPolicy = "default-src 'self'; script-src 'self' 'nonce-{nonce}' https://unpkg.com; style-src 'self' 'nonce-{nonce}' https://unpkg.com; img-src 'self' data: https:; connect-src 'self' wss: https:;"

	app := gospa.New(config)

	if err := app.Run(":3001"); err != nil {
		log.Fatal(err)
	}
}
