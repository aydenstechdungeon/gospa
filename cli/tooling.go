package cli

import (
	"os/exec"
)

// PackageManager represents a Node.js package manager.
type PackageManager string

const (
	// BunPM is the bun package manager.
	BunPM PackageManager = "bun"
	// PnpmPM is the pnpm package manager.
	PnpmPM PackageManager = "pnpm"
	// NpmPM is the npm package manager.
	NpmPM PackageManager = "npm"
	// NonePM is used when no package manager is found.
	NonePM PackageManager = ""
)

func (pm PackageManager) String() string {
	return string(pm)
}

// GetPackageManager returns the best available package manager in priority order: bun, pnpm, npm.
func GetPackageManager() PackageManager {
	if _, err := exec.LookPath("bun"); err == nil {
		return BunPM
	}
	if _, err := exec.LookPath("pnpm"); err == nil {
		return PnpmPM
	}
	if _, err := exec.LookPath("npm"); err == nil {
		return NpmPM
	}
	return NonePM
}

// GetBundlerCommand returns the command to use for client-side bundling.
func GetBundlerCommand(pm PackageManager) string {
	if pm == BunPM {
		return "bun"
	}
	// Default fallback to npx/pnpx/bun x with esbuild
	return string(pm)
}

// GetExecuteCommand returns the "execute" (dlx/x) equivalent for the package manager.
func GetExecuteCommand(pm PackageManager) string {
	switch pm {
	case BunPM:
		return "bun x"
	case PnpmPM:
		return "pnpm dlx"
	case NpmPM:
		return "npx"
	default:
		return "npx"
	}
}

// GetRunCommand returns the command to run a script.
func GetRunCommand(pm PackageManager) string {
	if pm == "" {
		return "npm"
	}
	return string(pm)
}
