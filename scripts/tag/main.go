// Package main provides a script to bump version and create a new git tag.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	skipTag, releaseBranchPrefix, newTag, err := parseTagArgs(os.Args[1:])
	if err != nil {
		fmt.Println("Usage: go run scripts/tag/main.go [-skip-tag] [-release-branch-prefix <prefix>] <tag>")
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	if !strings.HasPrefix(newTag, "v") {
		fmt.Println("Error: tag must start with 'v' (e.g. v0.1.0)")
		os.Exit(1)
	}
	newVersion := strings.TrimPrefix(newTag, "v")

	// Check if git is clean
	cleanOut, _ := exec.Command("git", "status", "--porcelain").Output()
	if !skipTag && len(cleanOut) > 0 {
		fmt.Println("Error: git working directory is not clean. Please commit or stash changes before tagging.")
		os.Exit(1)
	}

	modData, err := os.ReadFile("go.mod")
	if err != nil {
		fmt.Println("Error: Could not read go.mod:", err)
		os.Exit(1)
	}
	modLines := strings.Split(string(modData), "\n")
	if len(modLines) == 0 || !strings.HasPrefix(modLines[0], "module ") {
		fmt.Println("Error: Could not determine module name from go.mod")
		os.Exit(1)
	}
	moduleName := strings.TrimSpace(strings.TrimPrefix(modLines[0], "module "))
	fmt.Println("Detected module:", moduleName)

	out, _ := exec.Command("git", "tag", "--sort=-v:refname").Output()
	var oldTag string
	tags := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(tags) > 0 && tags[0] != "" {
		oldTag = tags[0]
	}

	if oldTag != "" && oldTag != newTag {
		oldVersion := strings.TrimPrefix(oldTag, "v")
		fmt.Printf("Bumping version from %s to %s...\n", oldTag, newTag)

		updateVersionFile(oldVersion, newVersion)

		dirsWithMod := make(map[string]bool)

		err = filepath.WalkDir(".", func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if d.IsDir() {
				if d.Name() == "vendor" || d.Name() == ".git" {
					return filepath.SkipDir
				}
				return nil
			}

			if d.Name() == "go.mod" {
				if updateModFile(path, moduleName, oldVersion, newVersion, oldTag, newTag) {
					dirsWithMod[filepath.Dir(path)] = true
				}
				return nil
			}

			if d.Name() == "go.sum" {
				return nil
			}

			ext := filepath.Ext(d.Name())
			if ext == ".go" || ext == ".md" || ext == ".templ" {
				if filepath.Base(path) == "gospa.go" {
					return nil
				}
				if updateOtherFile(path, moduleName, oldVersion, newVersion, oldTag, newTag) {
					fmt.Println("Updating references in", path)
				}
			}
			return nil
		})

		if err != nil {
			fmt.Println("Error walking directory:", err)
		}

		if len(dirsWithMod) > 0 {
			fmt.Println("\nRegenerating go.sum files...")
			for dir := range dirsWithMod {
				fmt.Println("  Running go mod tidy in", dir)
				cmd := exec.Command("go", "mod", "tidy")
				cmd.Dir = dir
				if runErr := cmd.Run(); runErr != nil {
					fmt.Printf("Warning: go mod tidy failed in %s: %v\n", dir, runErr)
				}
			}
		}

		// Run in root just in case
		if err := exec.Command("go", "mod", "tidy").Run(); err != nil {
			fmt.Println("Warning: go mod tidy failed:", err)
		}

		if skipTag {
			fmt.Println("\nSkipping git commit/tag/push as requested (-skip-tag).")
			return
		}

		// Commit
		if err := exec.Command("git", "add", "-A").Run(); err != nil {
			fmt.Println("Error: git add failed:", err)
			os.Exit(1)
		}
		err = exec.Command("git", "diff", "--cached", "--quiet").Run()
		if err != nil {
			if err := exec.Command("git", "commit", "-m", "chore: bump version to "+newTag).Run(); err != nil { // #nosec //nolint:gosec
				fmt.Println("Error: git commit failed:", err)
				os.Exit(1)
			}
		} else {
			fmt.Println("No changes to commit")
		}
	} else if oldTag == newTag {
		fmt.Println("Tag", newTag, "is already the latest tag.")
	}

	if skipTag {
		return
	}

	branchOut, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	branch := strings.TrimSpace(string(branchOut))

	fmt.Printf("\nTagging %s and pushing to %s...\n", newTag, branch)
	if err := exec.Command("git", "tag", "-f", newTag).Run(); err != nil { // #nosec //nolint:gosec
		fmt.Println("Error: git tag failed:", err)
		os.Exit(1)
	}

	cmd1 := exec.Command("git", "push", "origin", branch) // #nosec //nolint:gosec
	cmd1.Stdout = os.Stdout
	cmd1.Stderr = os.Stderr
	if err := cmd1.Run(); err != nil {
		fmt.Printf("\nPush to '%s' failed (likely protected branch). Attempting to push to a release branch...\n", branch)
		releaseBranch := releaseBranchPrefix + newTag
		// Check if branch already exists and delete it if it's different
		_ = exec.Command("git", "branch", "-D", releaseBranch).Run() //nolint:gosec // ignore error if it doesn't exist

		if err := exec.Command("git", "checkout", "-b", releaseBranch).Run(); err != nil { // #nosec G204 G702
			fmt.Printf("Error creating release branch: %v\n", err)
			os.Exit(1)
		}

		cmd3 := exec.Command("git", "push", "-u", "origin", releaseBranch) //nolint:gosec
		cmd3.Stdout = os.Stdout
		cmd3.Stderr = os.Stderr
		if err := cmd3.Run(); err != nil {
			fmt.Printf("Error pushing release branch: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nSuccessfully pushed to branch '%s'. Please open a pull request.\n", releaseBranch)
		pushTag(newTag)
		os.Exit(0)
	}

	pushTag(newTag)
	fmt.Println("\nSuccessfully updated tag", newTag)
}

func parseTagArgs(args []string) (bool, string, string, error) {
	fs := flag.NewFlagSet("tag", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var skipTag bool
	var releaseBranchPrefix string
	fs.BoolVar(&skipTag, "skip-tag", false, "only update version in code, skip git commit, tag and push")
	fs.StringVar(&releaseBranchPrefix, "release-branch-prefix", "release/", "prefix for fallback branch when direct push fails")

	ordered := make([]string, 0, len(args))
	positionals := make([]string, 0, 1)
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			ordered = append(ordered, arg)
			continue
		}
		positionals = append(positionals, arg)
	}
	ordered = append(ordered, positionals...)

	if err := fs.Parse(ordered); err != nil {
		return false, "", "", err
	}
	if fs.NArg() != 1 {
		return false, "", "", errors.New("expected exactly one <tag> argument")
	}

	// Ensure user-provided branch prefix always creates a nested branch name by default.
	if releaseBranchPrefix != "" && !strings.HasSuffix(releaseBranchPrefix, "/") {
		releaseBranchPrefix += "/"
	}
	return skipTag, releaseBranchPrefix, fs.Arg(0), nil
}

func pushTag(newTag string) {
	fmt.Printf("Pushing tag %s to origin...\n", newTag)
	cmd := exec.Command("git", "push", "-f", "origin", newTag) // #nosec //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: pushing tag '%s' failed (this is expected if tags are also protected). you can push it manually with: git push origin %s\n", newTag, newTag)
	}
}

func updateVersionFile(oldVersion, newVersion string) {
	data, err := os.ReadFile("config.go")
	if err != nil {
		return
	}
	content := string(data)
	oldLine := fmt.Sprintf("const Version = %q", oldVersion)
	newLine := fmt.Sprintf("const Version = %q", newVersion)
	if strings.Contains(content, oldLine) {
		fmt.Println("Updating config.go version...")
		content = strings.Replace(content, oldLine, newLine, 1)
		// #nosec //nolint:gosec
		if err := os.WriteFile("config.go", []byte(content), 0600); err != nil {
			fmt.Println("Error writing config.go:", err)
		}
	}
}

func updateModFile(path, moduleName, oldVersion, newVersion, oldTag, newTag string) bool {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return false
	}
	content := string(data)
	changed := false

	oldStr := moduleName + " v" + oldVersion
	newStr := moduleName + " v" + newVersion
	if strings.Contains(content, oldStr) {
		content = strings.ReplaceAll(content, oldStr, newStr)
		changed = true
	}

	oldStr2 := moduleName + " " + oldTag
	newStr2 := moduleName + " " + newTag
	if strings.Contains(content, oldStr2) {
		content = strings.ReplaceAll(content, oldStr2, newStr2)
		changed = true
	}

	if changed {
		fmt.Println("Updating", path)
		// Use filepath.Clean to prevent path traversal - path is from WalkDir so already constrained
		// #nosec //nolint:gosec // path is constrained to project files in WalkDir callback
		if err := os.WriteFile(filepath.Clean(path), []byte(content), 0600); err != nil {
			fmt.Println("Error writing", path, ":", err)
		}
	}
	return changed
}

func updateOtherFile(path, moduleName, oldVersion, newVersion, oldTag, newTag string) bool {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return false
	}
	content := string(data)

	// regex to find `moduleName... oldTag` or `moduleName... vOldVersion`
	modRegexPart := regexp.QuoteMeta(moduleName)
	pattern := fmt.Sprintf(`(%s(?:/[A-Za-z0-9_.-]+)*)[ @]v?%s\b`, modRegexPart, regexp.QuoteMeta(oldVersion))
	re := regexp.MustCompile(pattern)

	newContent := re.ReplaceAllStringFunc(content, func(match string) string {
		return strings.Replace(strings.Replace(match, oldTag, newTag, 1), oldVersion, newVersion, 1)
	})

	if newContent != content {
		// Use filepath.Clean to prevent path traversal - path is from WalkDir so already constrained
		// #nosec //nolint:gosec // path is constrained to project files in WalkDir callback
		if err := os.WriteFile(filepath.Clean(path), []byte(newContent), 0600); err != nil {
			fmt.Println("Error writing", path, ":", err)
			return false
		}
		return true
	}
	return false
}
