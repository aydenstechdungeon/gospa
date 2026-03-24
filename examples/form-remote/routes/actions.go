// Package routes defines remote actions for the guestbook example.
package routes

import (
	"context"
	"fmt"

	"github.com/aydenstechdungeon/gospa"
	"github.com/aydenstechdungeon/gospa/routing"
)

const pageSize = 5

// MessageInput represents the input for submitting a message
type MessageInput struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func init() {
	// Register remote action for submitting messages
	routing.RegisterRemoteAction("submitMessage", func(_ context.Context, _ routing.RemoteContext, input any) (any, error) {
		data, ok := input.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid input")
		}

		name, _ := data["name"].(string)
		content, _ := data["content"].(string)

		if name == "" || content == "" {
			return nil, fmt.Errorf("name and content are required")
		}

		msg := GetStore().AddMessage(name, content)

		// Broadcast to all connected clients via WebSocket
		if err := gospa.Broadcast(map[string]any{
			"type":    "new_message",
			"message": msg,
		}); err != nil {
			// In a real app, you might want to log this but continue
			fmt.Printf("Broadcast error: %v\n", err)
		}

		return msg, nil
	})

	// Register remote action for getting messages with pagination
	routing.RegisterRemoteAction("getMessages", func(_ context.Context, _ routing.RemoteContext, input any) (any, error) {
		page := 1
		if data, ok := input.(map[string]any); ok {
			if p, ok := data["page"].(float64); ok {
				page = int(p)
			}
		}
		return GetStore().GetMessages(page, pageSize), nil
	})
}
