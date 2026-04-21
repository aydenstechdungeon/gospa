package fiber

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	gofiber "github.com/gofiber/fiber/v3"
)

func TestSSESubscribeHandler_Authorization(t *testing.T) {
	broker := NewSSEBroker(&SSEConfig{})
	app := gofiber.New()

	// Mock endpoint for SSESubscribeHandler
	app.Post("/subscribe", broker.SSESubscribeHandler())

	// 1. Setup clients
	targetClientID := "target-client-id"
	legitClientID := "legit-client-id"

	legitToken, _ := globalSessionStore.CreateSession(legitClientID)
	targetToken, _ := globalSessionStore.CreateSession(targetClientID)

	// Register clients in broker (internal state)
	// We need to bypass the streaming Connect for testing the handler in isolation
	broker.mutex.Lock()
	broker.clients[targetClientID] = &SSEClient{ID: targetClientID, Topics: make(map[string]bool)}
	broker.clients[legitClientID] = &SSEClient{ID: legitClientID, Topics: make(map[string]bool)}
	broker.mutex.Unlock()

	// 2. Attempt unauthorized subscription (legit-client trying to sub to target-client)
	reqBody, _ := json.Marshal(map[string]interface{}{
		"clientId": targetClientID,
		"topics":   []string{"secret-topic"},
	})

	req := httptest.NewRequest("POST", "/subscribe", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", "gospa_session="+legitToken)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp.StatusCode != gofiber.StatusForbidden {
		t.Errorf("expected 403 Forbidden for unauthorized subscription, got %d", resp.StatusCode)
	}

	// 3. Attempt authorized subscription (target-client sub to itself)
	req2 := httptest.NewRequest("POST", "/subscribe", bytes.NewReader(reqBody))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Cookie", "gospa_session="+targetToken)

	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if resp2.StatusCode != gofiber.StatusOK {
		t.Errorf("expected 200 OK for authorized subscription, got %d", resp2.StatusCode)
	}
}

func TestSSEBrokerConnect_DuplicateClientIDDoesNotOverwrite(t *testing.T) {
	broker := NewSSEBroker(&SSEConfig{})

	first := broker.Connect("client-fixed")
	second := broker.Connect("client-fixed")

	if first.ID != "client-fixed" {
		t.Fatalf("expected first ID to be preserved, got %q", first.ID)
	}
	if second.ID == "client-fixed" {
		t.Fatal("expected duplicate ID to be remapped to a server-minted ID")
	}
	if second.ID == "" {
		t.Fatal("expected second client to have a non-empty ID")
	}

	if broker.ClientCount() != 2 {
		t.Fatalf("expected 2 connected clients, got %d", broker.ClientCount())
	}

	connected, ok := broker.GetClient("client-fixed")
	if !ok {
		t.Fatal("expected original client to remain connected under original ID")
	}
	if connected != first {
		t.Fatal("expected original client to remain mapped to original ID")
	}
}

func TestSSEBroker_BroadcastWithConcurrentConnectDisconnect(t *testing.T) {
	broker := NewSSEBroker(&SSEConfig{})

	for i := 0; i < 32; i++ {
		id := fmt.Sprintf("seed-%d", i)
		broker.Connect(id)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 300; i++ {
			broker.Broadcast(SSEEvent{Event: "tick", Data: i})
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 300; i++ {
			id := fmt.Sprintf("temp-%d", i)
			client := broker.Connect(id)
			broker.Disconnect(client.ID)
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for concurrent broadcast/connect-disconnect workload")
	}
}
