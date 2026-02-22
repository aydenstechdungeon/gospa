package lib

import (
	"encoding/json"
	"fmt"

	"github.com/aydenstechdungeon/gospa/fiber"
)

// CounterState holds the counter data
type CounterState struct {
	Count int `json:"count"`
}

var GlobalCounter = &CounterState{Count: 0}

// RegisterHandlers registers action handlers for the counter
func RegisterHandlers(hub *fiber.WSHub) {
	fiber.RegisterActionHandler("increment", func(client *fiber.WSClient, payload json.RawMessage) {
		GlobalCounter.Count++
		fmt.Printf("Counter incremented: %d\n", GlobalCounter.Count)
		// Broadcast new count to all clients
		_ = fiber.BroadcastState(hub, "count", GlobalCounter.Count)
	})

	fiber.RegisterActionHandler("decrement", func(client *fiber.WSClient, payload json.RawMessage) {
		GlobalCounter.Count--
		fmt.Printf("Counter decremented: %d\n", GlobalCounter.Count)
		// Broadcast new count to all clients
		_ = fiber.BroadcastState(hub, "count", GlobalCounter.Count)
	})
}
