package pathsafety

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolvePath resolves targetPath against baseDir and enforces baseDir containment unless allowEscape is true.
func ResolvePath(baseDir, targetPath string, allowEscape bool) (string, error) {
	trimmed := strings.TrimSpace(targetPath)
	if trimmed == "" {
		return "", fmt.Errorf("path is required")
	}

	baseAbs, err := filepath.Abs(filepath.Clean(baseDir))
	if err != nil {
		return "", fmt.Errorf("resolve base path: %w", err)
	}

	var resolved string
	if filepath.IsAbs(trimmed) {
		resolved = filepath.Clean(trimmed)
	} else {
		resolved = filepath.Join(baseAbs, filepath.Clean(trimmed))
	}

	resolvedAbs, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve target path: %w", err)
	}

	if allowEscape {
		return resolvedAbs, nil
	}

	rel, err := filepath.Rel(baseAbs, resolvedAbs)
	if err != nil {
		return "", fmt.Errorf("evaluate path relation: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes base directory")
	}

	return resolvedAbs, nil
}
