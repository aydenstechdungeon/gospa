// Package main validates that promoted helper APIs have docs, migration mapping, and tests references.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type helperRule struct {
	Name          string
	CodeNeedle    string
	APINeedle     string
	MigrateNeedle string
	TestNeedle    string
}

var rules = []helperRule{
	{
		Name:          "kit.Depends",
		CodeNeedle:    "func Depends(",
		APINeedle:     "`kit.Depends`",
		MigrateNeedle: "`kit.Depends`",
		TestNeedle:    "render_load_helpers_test.go",
	},
	{
		Name:          "kit.Untrack",
		CodeNeedle:    "func Untrack(",
		APINeedle:     "`kit.Untrack`",
		MigrateNeedle: "`kit.Untrack`",
		TestNeedle:    "render_load_helpers_test.go",
	},
	{
		Name:          "kit.Parent[T]",
		CodeNeedle:    "func Parent[T any]",
		APINeedle:     "`kit.Parent[T]`",
		MigrateNeedle: "`kit.Parent`",
		TestNeedle:    "render_load_helpers_test.go",
	},
	{
		Name:          "kit.Error",
		CodeNeedle:    "func Error(",
		APINeedle:     "`kit.Error`",
		MigrateNeedle: "`kit.Error`",
		TestNeedle:    "gospa_form_action_test.go",
	},
	{
		Name:          "refresh",
		CodeNeedle:    "export async function refresh(",
		APINeedle:     "function refresh(",
		MigrateNeedle: "`refresh(",
		TestNeedle:    "client/src/route-helpers.test.ts",
	},
	{
		Name:          "prefetchOnHover",
		CodeNeedle:    "export function prefetchOnHover(",
		APINeedle:     "function prefetchOnHover(",
		MigrateNeedle: "`prefetchOnHover(",
		TestNeedle:    "client/src/route-helpers.test.ts",
	},
}

func main() {
	codeCorpus, err := readFiles(
		"routing/kit/kit.go",
		"routing/kit/scope.go",
		"client/src/route-helpers.ts",
		"client/src/navigation.ts",
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read code corpus: %v\n", err)
		os.Exit(1)
	}

	apiDoc, err := os.ReadFile("docs/gospasfc/api-reference.md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read API docs: %v\n", err)
		os.Exit(1)
	}
	migrationDoc, err := os.ReadFile("docs/migration/sveltekit-to-gospa.md")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read migration docs: %v\n", err)
		os.Exit(1)
	}

	failures := validatePromotion(string(codeCorpus), string(apiDoc), string(migrationDoc))

	if len(failures) > 0 {
		fmt.Fprintln(os.Stderr, "helper promotion validation failed:")
		for _, f := range failures {
			fmt.Fprintf(os.Stderr, "- %s\n", f)
		}
		os.Exit(1)
	}

	fmt.Printf("validated %d helper promotion rules\n", len(rules))
}

func readFiles(paths ...string) (string, error) {
	var b strings.Builder
	for _, path := range paths {
		cleanPath := filepath.Clean(path)
		f, err := os.Open(cleanPath)
		if err != nil {
			return "", err
		}
		data, err := io.ReadAll(f)
		closeErr := f.Close()
		if err != nil {
			return "", err
		}
		if closeErr != nil {
			return "", closeErr
		}
		b.Write(data)
		b.WriteByte('\n')
	}
	return b.String(), nil
}

func validatePromotion(codeCorpus, apiText, migrationText string) []string {
	var failures []string
	for _, rule := range rules {
		if !strings.Contains(codeCorpus, rule.CodeNeedle) {
			continue
		}
		if !strings.Contains(apiText, rule.APINeedle) {
			failures = append(failures, fmt.Sprintf("%s missing API docs marker %q", rule.Name, rule.APINeedle))
		}
		if !strings.Contains(migrationText, rule.MigrateNeedle) {
			failures = append(failures, fmt.Sprintf("%s missing migration marker %q", rule.Name, rule.MigrateNeedle))
		}
		if !strings.Contains(apiText, rule.TestNeedle) {
			failures = append(failures, fmt.Sprintf("%s missing test coverage marker %q in API docs", rule.Name, rule.TestNeedle))
		}
	}
	return failures
}
