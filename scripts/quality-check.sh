#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
 
# Check for CGO dependencies
if command -v pkg-config >/dev/null 2>&1; then
  if ! pkg-config --exists libwebp libheif; then
    echo "Warning: libwebp or libheif not found. Image plugin compilation will fail."
    echo "To fix on Arch Linux: sudo pacman -S libwebp libheif"
    echo "To fix on Ubuntu: sudo apt-get install libwebp-dev libheif-dev"
    echo ""
  fi
fi

RUN_FMT=1
RUN_VET=1
RUN_STATICCHECK=1
RUN_LINT=1
RUN_VULN=1
RUN_BUILD=1
RUN_TEST=1
RUN_RACE=0
RUN_BUN_CHECK=1
RUN_EXAMPLES=1
GO_TEST_PKGS="./..."
GO_BUILD_PKGS="./..."
GOFMT_TARGETS="."
GOLANGCI_ARGS="--timeout=5m"

usage() {
  cat <<USAGE
Usage: $0 [options]

Options:
  --with-race               Run go test -race (default: off)
  --without-race            Disable go test -race
  --skip-fmt                Skip gofmt check
  --skip-vet                Skip go vet
  --skip-staticcheck        Skip staticcheck
  --skip-golangci           Skip golangci-lint
  --skip-vulncheck          Skip govulncheck
  --skip-build              Skip go build
  --skip-test               Skip go test
  --skip-bun-check          Skip bun check (runs root package script)
  --skip-examples           Skip example build validation
  --go-test-pkgs <pkgs>     Package pattern for go test (default: ./...)
  --go-build-pkgs <pkgs>    Package pattern for go build (default: ./...)
  --gofmt-targets <paths>   Space-delimited targets for gofmt check (default: .)
  -h, --help                Show help
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --with-race) RUN_RACE=1 ;;
    --without-race) RUN_RACE=0 ;;
    --skip-fmt) RUN_FMT=0 ;;
    --skip-vet) RUN_VET=0 ;;
    --skip-staticcheck) RUN_STATICCHECK=0 ;;
    --skip-golangci) RUN_LINT=0 ;;
    --skip-vulncheck) RUN_VULN=0 ;;
    --skip-build) RUN_BUILD=0 ;;
    --skip-test) RUN_TEST=0 ;;
    --skip-bun-check) RUN_BUN_CHECK=0 ;;
    --skip-examples) RUN_EXAMPLES=0 ;;
    --go-test-pkgs) shift; GO_TEST_PKGS="${1:-}" ;;
    --go-build-pkgs) shift; GO_BUILD_PKGS="${1:-}" ;;
    --gofmt-targets) shift; GOFMT_TARGETS="${1:-}" ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1"; usage; exit 2 ;;
  esac
  shift
done

# Exclude nested Go packages under client/node_modules (Bun-installed deps with go.mod).
if [[ "$GO_TEST_PKGS" == "./..." ]]; then
  GO_TEST_PKGS="$(go list ./... | grep -v node_modules | tr '\n' ' ')"
fi
if [[ "$GO_BUILD_PKGS" == "./..." ]]; then
  GO_BUILD_PKGS="$(go list ./... | grep -v node_modules | tr '\n' ' ')"
fi

run_step() {
  local label="$1"
  shift
  echo "==> ${label}"
  "$@"
}

if [[ $RUN_BUN_CHECK -eq 1 ]]; then
  run_step "bun check (root script)" bun check
fi

if [[ $RUN_FMT -eq 1 ]]; then
  echo "==> gofmt check"
  mapfile -t gofiles < <(rg --files $GOFMT_TARGETS -g '*.go' -g '!**/vendor/**')
  if [[ ${#gofiles[@]} -gt 0 ]]; then
    unformatted="$(gofmt -l "${gofiles[@]}")"
    if [[ -n "$unformatted" ]]; then
      echo "Unformatted Go files detected:" >&2
      echo "$unformatted" >&2
      exit 1
    fi
  fi
fi

if [[ $RUN_VET -eq 1 ]]; then
  run_step "go vet" go vet $GO_TEST_PKGS
fi

if [[ $RUN_STATICCHECK -eq 1 ]]; then
  run_step "staticcheck" staticcheck $GO_TEST_PKGS
fi

if [[ $RUN_LINT -eq 1 ]]; then
  run_step "golangci-lint" golangci-lint run $GOLANGCI_ARGS
fi

if [[ $RUN_VULN -eq 1 ]]; then
  run_step "govulncheck" govulncheck $GO_TEST_PKGS
fi

if [[ $RUN_BUILD -eq 1 ]]; then
  run_step "go build" go build $GO_BUILD_PKGS
fi

if [[ $RUN_TEST -eq 1 ]]; then
  run_step "go test" go test $GO_TEST_PKGS
fi

if [[ $RUN_EXAMPLES -eq 1 ]]; then
  run_step "validate examples" ./scripts/validate-examples.sh
fi

if [[ $RUN_RACE -eq 1 ]]; then
  run_step "go test -race" go test -race $GO_TEST_PKGS
fi

echo "All requested checks passed."
