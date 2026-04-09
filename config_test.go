package gospa

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.RoutesDir != "./routes" {
		t.Errorf("expected RoutesDir to be './routes', got %s", config.RoutesDir)
	}
	if config.DevMode != true {
		t.Errorf("expected DevMode to be true by default")
	}
	if config.RuntimeScript != "/_gospa/runtime.js" {
		t.Errorf("expected RuntimeScript to be '/_gospa/runtime.js', got %s", config.RuntimeScript)
	}
	if config.StaticDir != "./static" {
		t.Errorf("expected StaticDir to be './static', got %s", config.StaticDir)
	}
	if config.StaticPrefix != "/static" {
		t.Errorf("expected StaticPrefix to be '/static', got %s", config.StaticPrefix)
	}
	if !config.EnableWebSocket {
		t.Errorf("expected EnableWebSocket to be true")
	}
	if config.WebSocketPath != "/_gospa/ws" {
		t.Errorf("expected WebSocketPath to be '/_gospa/ws', got %s", config.WebSocketPath)
	}
	if config.RemotePrefix != "/_gospa/remote" {
		t.Errorf("expected RemotePrefix to be '/_gospa/remote', got %s", config.RemotePrefix)
	}
	if config.ISRSemaphoreLimit != 10 {
		t.Errorf("expected ISRSemaphoreLimit to be 10, got %d", config.ISRSemaphoreLimit)
	}
	if config.ISRTimeout != 60*time.Second {
		t.Errorf("expected ISRTimeout to be 60s, got %v", config.ISRTimeout)
	}
	if config.AllowInsecureWS != false {
		t.Errorf("expected AllowInsecureWS to be false in DefaultConfig (env var processed in New())")
	}
	if len(config.AllowPortsWithInsecureWS) != 1 || config.AllowPortsWithInsecureWS[0] != 3000 {
		t.Errorf("expected AllowPortsWithInsecureWS to be [3000], got %v", config.AllowPortsWithInsecureWS)
	}
}

func TestGOSPAWSInsecureHonoredInDevMode(t *testing.T) {
	_ = os.Setenv("GOSPA_WS_INSECURE", "1")
	defer func() { _ = os.Unsetenv("GOSPA_WS_INSECURE") }()

	config := DefaultConfig()
	config.DevMode = true
	app := New(config)

	if !app.Config.AllowInsecureWS {
		t.Errorf("expected AllowInsecureWS to be true in DevMode when GOSPA_WS_INSECURE=1")
	}
}

func TestGOSPAWSInsecureIgnoredInProduction(t *testing.T) {
	_ = os.Setenv("GOSPA_WS_INSECURE", "1")
	defer func() { _ = os.Unsetenv("GOSPA_WS_INSECURE") }()

	config := ProductionConfig()
	app := New(config)

	if app.Config.AllowInsecureWS {
		t.Errorf("expected AllowInsecureWS to be false in production even when GOSPA_WS_INSECURE=1")
	}
}

func TestProductionConfig(t *testing.T) {
	config := ProductionConfig()

	if config.DevMode != false {
		t.Errorf("expected DevMode to be false")
	}
	if config.CacheTemplates != true {
		t.Errorf("expected CacheTemplates to be true")
	}
	if config.WSReconnectDelay != time.Second {
		t.Errorf("expected WSReconnectDelay to be 1s, got %v", config.WSReconnectDelay)
	}
	if config.WSMaxReconnect != 10 {
		t.Errorf("expected WSMaxReconnect to be 10, got %d", config.WSMaxReconnect)
	}
	if config.WSHeartbeat != 30*time.Second {
		t.Errorf("expected WSHeartbeat to be 30s, got %v", config.WSHeartbeat)
	}
	if config.SSGCacheMaxEntries != 500 {
		t.Errorf("expected SSGCacheMaxEntries to be 500, got %d", config.SSGCacheMaxEntries)
	}
}

func TestMinimalConfig(t *testing.T) {
	config := MinimalConfig()

	if config.EnableWebSocket {
		t.Errorf("expected EnableWebSocket to be false")
	}
	if config.CompressState {
		t.Errorf("expected CompressState to be false")
	}
	if config.StateDiffing {
		t.Errorf("expected StateDiffing to be false")
	}
	if config.WSReconnectDelay != 0 {
		t.Errorf("expected WSReconnectDelay to be 0, got %v", config.WSReconnectDelay)
	}
	if config.WSMaxReconnect != 0 {
		t.Errorf("expected WSMaxReconnect to be 0, got %d", config.WSMaxReconnect)
	}
	if config.WSHeartbeat != 0 {
		t.Errorf("expected WSHeartbeat to be 0, got %v", config.WSHeartbeat)
	}
}
