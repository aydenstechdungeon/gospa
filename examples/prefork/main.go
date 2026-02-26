package main

import (
	"context"
	"log"

	"github.com/aydenstechdungeon/gospa"
	"github.com/aydenstechdungeon/gospa/store/redis"
	"github.com/gofiber/fiber/v2"
	redisclient "github.com/redis/go-redis/v9"
)

func main() {
	// 1. Create a Redis client.
	rdb := redisclient.NewClient(&redisclient.Options{
		Addr: "localhost:6379", // Make sure Redis is running locally
	})

	// 2. Ping Redis to verify connection before starting the app.
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Printf("Redis not available on localhost:6379! App starting with isolated in-memory stores.")
		rdb = nil
	}

	config := gospa.Config{
		AppName: "GoSPA Prefork Example",
		Prefork: true,
	}

	// 3. Configure GoSPA to use Redis if available.
	if rdb != nil {
		config.Storage = redis.NewStore(rdb)
		config.PubSub = redis.NewPubSub(rdb)
		log.Println("Redis configured for Session Storage and PubSub successfully.")
	} else {
		log.Println("WARNING: Prefork enabled without Redis. This will cause state inconsistencies across processes!")
	}

	app := gospa.New(config)

	// Since we are running outside the standard components example setup
	// We disable SPA for this trivial example or add a barebones route
	app.Fiber.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("GoSPA Prefork Example Running! Refresh and watch the process ID change depending on Fiber's load balancer.")
	})

	log.Fatal(app.Run(":3000"))
}
