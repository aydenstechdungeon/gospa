# Fiber to GoSPA Migration Checklist

Use this checklist when integrating GoSPA into an existing Fiber app.

## 1) Preflight

- Run `gospa verify` and resolve blocking issues.
- Ensure `SecurityHeadersMiddleware(...)` policy includes `{nonce}`.
- If `Prefork: true`, configure both `Storage` and `PubSub`.

## 2) Route Surface

- Keep existing Fiber APIs and static routes unchanged.
- Move UI routes into `routes/` as `.templ` or `.gospa`.
- Generate route artifacts with `gospa generate`.

## 3) SFC Syntax Migration

- Convert legacy template event syntax to GoSPA directives.
- Dry run:

```bash
go run ./scripts/codemod-sfc-events
```

- Apply changes:

```bash
go run ./scripts/codemod-sfc-events --write
```

- Supported rewrites:
  - `@click={...}` -> `on:click={...}`
  - `on-click={...}` -> `on:click={...}`
  - `x-on:click={...}` -> `on:click={...}`

## 4) Runtime Integration

- Add GoSPA middleware before registering routes.
- Keep websocket path consistent between server config and runtime.
- Run `gospa doctor --strict` before `gospa dev` and CI builds.

## 5) Verification Gate

- Compile and generate:

```bash
gospa generate
```

- Strict diagnostics:

```bash
gospa doctor --strict
```

- Confirm no `.gospa` diagnostics and no runtime preflight warnings.
