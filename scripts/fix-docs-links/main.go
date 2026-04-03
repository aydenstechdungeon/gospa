// Package main provides a script to fix legacy documentation links in markdown files.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var replacements = map[string]string{
	"01-getting-started/01-quick-start.md":            "getstarted/quickstart",
	"01-getting-started/02-tutorial.md":               "getstarted/tutorial",
	"01-getting-started/03-scratch-guide.md":          "getstarted/structure",
	"02-core-concepts/02-rendering.md":                "rendering",
	"02-core-concepts/03-state.md":                    "state-management/server",
	"02-core-concepts/04-components.md":               "components",
	"02-core-concepts/05-islands.md":                  "islands",
	"02-core-concepts/06-routing.md":                  "routing",
	"03-features/01-client-runtime.md":                "client-runtime/overview",
	"03-features/02-runtime-api.md":                   "reactive-primitives/js",
	"03-features/03-realtime.md":                      "websocket",
	"03-features/04-security.md":                      "configuration/scaling",
	"03-features/05-dev-tools.md":                     "devtools",
	"03-features/06-deployment.md":                    "configuration/scaling",
	"03-features/07-gospa-sfc.md":                     "gospasfc",
	"03-features/08-production-checklist.md":          "troubleshooting",
	"04-api-reference/01-core-api.md":                 "api/core",
	"04-api-reference/02-configuration.md":            "configuration",
	"04-api-reference/03-cli.md":                      "cli",
	"04-api-reference/04-plugins.md":                  "plugins",
	"05-advanced/01-error-handling.md":                "errors",
	"05-advanced/02-state-pruning.md":                 "state-management/patterns",
	"06-migration/01-v1-to-v2.md":                     "faq",
	"07-troubleshooting/01-runtime-initialization.md": "troubleshooting",
	"07-troubleshooting/02-remote-actions.md":         "remote-actions",
	"07-troubleshooting/03-websocket-connections.md":  "websocket",
	"07-troubleshooting/04-hmr-dev-server.md":         "hmr",
	"07-troubleshooting/05-island-hydration.md":       "troubleshooting",
	"07-troubleshooting/06-state-synchronization.md":  "troubleshooting",
	"07-troubleshooting/07-build-deployment.md":       "troubleshooting",
}

func main() {
	err := filepath.Walk("docs", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		//nolint:gosec // path is provided by filepath.Walk which is constrained to docs/
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		newContent := string(content)
		changed := false

		for old, new := range replacements {
			if strings.Contains(newContent, old) {
				newContent = strings.ReplaceAll(newContent, old, new)
				changed = true
			}
		}

		// Also handle some relatives like ../02-core-concepts/
		// This is a bit more complex but let's try a simple version
		for old, new := range replacements {
			relOld := "../" + old
			if strings.Contains(newContent, relOld) {
				// Determine if we need to adjust the relative level
				// For now, let's just replace the path part
				newContent = strings.ReplaceAll(newContent, relOld, "../"+new)
				changed = true
			}
		}

		if changed {
			fmt.Printf("Updating %s\n", path)
			//nolint:gosec // path is provided by filepath.Walk and constrained to project docs/
			return os.WriteFile(path, []byte(newContent), info.Mode())
		}
		return nil
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}
