package fiber

import (
	"fmt"
	"testing"

	"github.com/aydenstechdungeon/gospa/store"
)

func BenchmarkSSEBrokerBroadcastToTopic(b *testing.B) {
	broker := NewSSEBroker(&SSEConfig{PubSub: store.NewMemoryPubSub()})
	for i := 0; i < 1024; i++ {
		id := fmt.Sprintf("client-%d", i)
		broker.Connect(id)
		if i%2 == 0 {
			broker.Subscribe(id, "orders")
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broker.BroadcastToTopic("orders", SSEEvent{Event: "tick", Data: i})
	}
}

func BenchmarkWSHubDispatchBroadcast(b *testing.B) {
	hub := NewWSHub(store.NewMemoryPubSub())
	clients := make([]*WSClient, 0, 2048)
	for i := 0; i < 2048; i++ {
		clients = append(clients, &WSClient{Send: make(chan []byte, 8)})
	}
	msg := []byte(`{"type":"state","payload":{"k":"v"}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.dispatchBroadcast(clients, msg)
	}
}
