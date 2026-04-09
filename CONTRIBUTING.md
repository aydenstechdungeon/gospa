# Contributing to GoSPA

First off, thank you for considering contributing to GoSPA! We welcome all contributions from the community. Let's make this the best SPA framework for Go.

## 1. Getting Started

1. Fork the repository
2. Clone your fork locally
3. Write/Fix code
4. Format using `gofmt`
5. Ensure all unit tests pass with `go test ./...`
6. Run `golangci-lint run` if possible to maintain code quality

## 2. Issues & Bug Reports

Please use the GitHub Issue tracker. When creating a bug report, it is highly recommended you attach an executable code block reproducing the bug. Mention your runtime environment (browser + OS + Go version).

## 3. Pull Request Guidelines

*   **Branch naming**: Try to keep feature branches structured, e.g., `feature/awesome-new-router` or `fix/derived-deadlock`.
*   **Documentation**: If you are adding a new core feature, you must also add accompanying documentation in the `docs/` directory. Reference existing docs for structure.
*   **Tests**: Every bug fix should have an accompanying unit test guarding against regressions. All new critical paths need at least moderate `testing.T` coverage in Go.

## 4. Coding conventions

*   Follow standard idiomatic Go styles (as guided by Effective Go).
*   Prioritize explicit error checking over panicking. Only panic if you are within a `Must...` initialization sequence or standard library execution fails completely.
*   Client-side scripts are written in Typescript and compiled via Bun. Maintain strong ESLint/Prettier defaults on all TS assets. 

Thanks again!
