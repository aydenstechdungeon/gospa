package gospa

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestIslandsRoute_DynamicTSPathTraversalBlocked(t *testing.T) {
	tmpDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(originalWD) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}

	if err := os.MkdirAll(filepath.Join("generated", "components"), 0o755); err != nil {
		t.Fatalf("mkdir generated: %v", err)
	}
	if err := os.WriteFile(filepath.Join("generated", "components", "card.ts"), []byte(`export const card = "ok"`), 0o600); err != nil {
		t.Fatalf("write generated ts file: %v", err)
	}

	app := New(Config{DevMode: true})
	app.setupRoutes()
	defer func() { _ = app.Fiber.Shutdown() }()

	t.Run("serves valid generated module", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/islands/components/card.js", nil)
		res, err := app.Fiber.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer func() { _ = res.Body.Close() }()
		if res.StatusCode != fiber.StatusOK {
			t.Fatalf("expected status %d, got %d", fiber.StatusOK, res.StatusCode)
		}
		body, _ := io.ReadAll(res.Body)
		if !strings.Contains(string(body), `export const card = "ok"`) {
			t.Fatalf("expected generated TS module contents, got %q", string(body))
		}
	})

	t.Run("rejects traversal attempt", func(t *testing.T) {
		_, code, err := safeIslandTSPath("/islands/../../etc/passwd.js")
		if err == nil {
			t.Fatal("expected traversal attempt to return error")
		}
		if code != fiber.StatusForbidden {
			t.Fatalf("expected status %d for traversal attempt, got %d", fiber.StatusForbidden, code)
		}
	})

	t.Run("rejects absolute path escape", func(t *testing.T) {
		_, code, err := safeIslandTSPath("/islands//etc/passwd.js")
		if err == nil {
			t.Fatal("expected absolute path attempt to return error")
		}
		if code != fiber.StatusForbidden {
			t.Fatalf("expected status %d for absolute path attempt, got %d", fiber.StatusForbidden, code)
		}
	})
}
