package main

import (
	"bytes"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"
)

func main() {
	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		var buf bytes.Buffer
		buf.WriteString("<html><body><h1>Hello World</h1></body></html>")
		body := make([]byte, buf.Len())
		copy(body, buf.Bytes())
		c.Response().Header.SetContentType("text/html")
		c.Response().SetBody(body)
		c.Response().SetStatusCode(200)
		fmt.Printf("Handler: body_len=%d status=%d content_type=%s\n",
			len(c.Response().Body()), c.Response().StatusCode(), c.Response().Header.ContentType())
		return nil
	})

	go func() {
		if err := app.Listen(":3001"); err != nil {
			log.Fatal(err)
		}
	}()

	fmt.Println("Server started on :3001")
	select {}
}
