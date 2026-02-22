package lib

import (
	"encoding/json"
	"fmt"

	"github.com/aydenstechdungeon/gospa/fiber"
)

// Todo represents a single todo item
type Todo struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

// TodoState holds the list of todos
type TodoState struct {
	Todos []Todo `json:"todos"`
}

var GlobalTodoState = &TodoState{
	Todos: []Todo{
		{ID: "1", Text: "Learn GoSPA", Completed: false},
		{ID: "2", Text: "Build something awesome", Completed: false},
	},
}

// RegisterHandlers registers action handlers for the todo list
func RegisterHandlers(hub *fiber.WSHub) {
	fiber.RegisterActionHandler("add_todo", func(client *fiber.WSClient, payload json.RawMessage) {
		var data struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(payload, &data); err != nil {
			return
		}
		if data.Text == "" {
			return
		}

		newTodo := Todo{
			ID:        fmt.Sprintf("%d", len(GlobalTodoState.Todos)+1),
			Text:      data.Text,
			Completed: false,
		}
		GlobalTodoState.Todos = append(GlobalTodoState.Todos, newTodo)

		fmt.Printf("Added todo: %s\n", data.Text)
		_ = fiber.BroadcastState(hub, "todos", GlobalTodoState.Todos)
	})

	fiber.RegisterActionHandler("toggle_todo", func(client *fiber.WSClient, payload json.RawMessage) {
		var data struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(payload, &data); err != nil {
			return
		}

		for i, todo := range GlobalTodoState.Todos {
			if todo.ID == data.ID {
				GlobalTodoState.Todos[i].Completed = !GlobalTodoState.Todos[i].Completed
				break
			}
		}

		_ = fiber.BroadcastState(hub, "todos", GlobalTodoState.Todos)
	})

	fiber.RegisterActionHandler("remove_todo", func(client *fiber.WSClient, payload json.RawMessage) {
		var data struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(payload, &data); err != nil {
			return
		}

		newTodos := []Todo{}
		for _, todo := range GlobalTodoState.Todos {
			if todo.ID != data.ID {
				newTodos = append(newTodos, todo)
			}
		}
		GlobalTodoState.Todos = newTodos

		_ = fiber.BroadcastState(nil, "todos", GlobalTodoState.Todos)
	})
}
