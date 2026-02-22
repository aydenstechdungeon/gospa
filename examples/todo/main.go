package main

import (
	"log"

	"todo/lib"
	_ "todo/routes" // Import routes to trigger init()

	"github.com/aydenstechdungeon/gospa"
)

func main() {
	app := gospa.New(gospa.Config{
		RoutesDir: "./routes",
		DevMode:   true,
		AppName:   "todo",
		DefaultState: map[string]interface{}{
			"todos": lib.GlobalTodoState.Todos,
		},
	})

	// Register todo handlers
	lib.RegisterHandlers(app.Hub)

	if err := app.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}
