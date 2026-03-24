// Package main provides a prefork counter test example using GoSPA.
package main

import (
	"log"

	_ "github.com/aydenstechdungeon/gospa/examples/counter-test-prefork/routes" // Import routes to trigger init()

	"github.com/aydenstechdungeon/gospa"
	"github.com/aydenstechdungeon/gospa/store/redis"
	goredis "github.com/redis/go-redis/v9"
)

func main() {
	rdb := goredis.NewClient(&goredis.Options{
		Addr: "localhost:6379",
	})

	app := gospa.New(gospa.Config{
		RoutesDir: "./routes",
		DevMode:   true,
		AppName:   "counter",
		Prefork:   true,
		Storage:   redis.NewStore(rdb),
		PubSub:    redis.NewPubSub(rdb),
	})

	if err := app.Run(":3000"); err != nil {
		log.Fatal(err)
	}
}
