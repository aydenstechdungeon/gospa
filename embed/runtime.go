// Package embed provides embedded static assets for the GoSPA framework.
package embed

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
)

//go:embed runtime.js
var runtimeFS embed.FS

// RuntimeJS returns the embedded runtime JavaScript.
func RuntimeJS() ([]byte, error) {
	return runtimeFS.ReadFile("runtime.js")
}

// RuntimeHash returns a truncated SHA256 hash of the runtime JavaScript.
func RuntimeHash() (string, error) {
	content, err := RuntimeJS()
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
