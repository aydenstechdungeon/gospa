package fiber

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

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
