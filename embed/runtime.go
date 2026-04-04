// Package embed provides embedded static assets for the GoSPA framework.
package embed

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"strings"

	"github.com/aydenstechdungeon/gospa/compiler"
)

//go:embed *.js
var runtimeFS embed.FS

// RuntimeJS returns the embedded runtime JavaScript based on the tier.
func RuntimeJS(tier compiler.RuntimeTier) ([]byte, error) {
	name := "runtime.js"
	switch tier {
	case compiler.RuntimeTierMicro:
		name = "runtime-micro.js"
	case compiler.RuntimeTierCore:
		name = "runtime-core.js"
	case compiler.RuntimeTierFull:
		name = "runtime.js"
	}
	return runtimeFS.ReadFile(name)
}

// RuntimeHash returns a truncated SHA256 hash of the runtime JavaScript.
func RuntimeHash(tier compiler.RuntimeTier) (string, error) {
	content, err := RuntimeJS(tier)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(content)
	return fmt.Sprintf("%x", h[:8]), nil
}

// RuntimeFS returns the embedded filesystem for the runtime.
func RuntimeFS() fs.FS {
	sub, _ := fs.Sub(runtimeFS, ".")
	return sub
}

// RuntimeChunks returns a list of all JavaScript files in the runtime filesystem.
func RuntimeChunks() []string {
	var chunks []string
	entries, _ := runtimeFS.ReadDir(".")
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".js") {
			chunks = append(chunks, entry.Name())
		}
	}
	return chunks
}
