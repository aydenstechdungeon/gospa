# GoSPA SFC API Reference

This page documents the SFC API contracts as implemented in code (`compiler/module_exports.go`, `routing/registry.go`, `gospa.go`, `client/src/forms.ts`, `client/src/route-helpers.ts`).

For a complete parser/compiler/runtime walkthrough, see [SFC System Reference](system-reference.md).

Client runtime module reference:

- Runtime module path: `/_gospa/runtime.js`

## SFC file contract (`.gospa`)

Parser/compiler support:

- At most one `<template>` block
- At most one `<style>` block
- At most one `<script lang="go">` block
- At most one `<script lang="ts">` / `<script lang="js">` block
- At most one `<script context="module" lang="go">` block
- SFC max input size: 2 MB

Supported script languages:

- `lang="go"` (instance or module)
- `lang="ts"` (instance)
- `lang="js"` (normalized to `ts` handling)

Supported component types (`frontmatter type`):

- `island`
- `page`
- `layout`
- `static`
- `server`

Hydration modes:

- `immediate`
- `visible`
- `idle`
- `interaction`

## SFC event directive contract

- Template syntax: `on:<event>={handlerName}`
- Compiler lowering: `on:click={increment}` -> `data-gospa-on="click:increment"`
- Runtime resolution looks up handler names on island handler registries.
- Current generation path derives handler registries from transformed Go-script function names.
- Use named function handlers. Inline function literals are not part of the current delegated handler lookup contract.

Frontmatter keys consumed by compiler:

- `type`
- `hydrate`
- `hydrate_mode`
- `server_only`
- `package`

## Route module exports (`<script context="module" lang="go">`)

Supported top-level exports:

- `Load(c routing.LoadContext) (map[string]interface{}, error)`
- `Load(c routing.LoadContext) (map[string]any, error)`
- `ActionDefault(c routing.LoadContext) (interface{}, error)` (or `any`)
- `Action<Name>(c routing.LoadContext) (interface{}, error)` (or `any`)

Validation rules:

- Module script must be `lang="go"`.
- Exactly one module script block is allowed.
- Functions must be top-level (methods are rejected).
- At least one valid export is required if a module script exists.
- `Action<Name>` suffix must begin with uppercase (`ActionSave`).

## `routing.LoadContext`

`Load` and action functions receive:

- `Param(key string) string`
- `Params() map[string]string`
- `Query(key string, defaultValue ...string) string`
- `QueryValues() map[string][]string`
- `Header(key string) string`
- `Headers() map[string]string`
- `SetHeader(key, value string)`
- `Cookie(key string) string`
- `SetCookie(key, value string, maxAge int, path string, httpOnly, secure bool)`
- `FormValue(key string, defaultValue ...string) string`
- `Method() string`
- `Path() string`
- `Local(key string) interface{}`

## Action response contract (`routing.ActionResponse`)

Server-side struct:

```go
type ActionResponse struct {
  Data           interface{}            `json:"data,omitempty"`
  Code           string                 `json:"code,omitempty"`
  Error          string                 `json:"error,omitempty"`
  Redirect       *ActionRedirect        `json:"redirect,omitempty"`
  Validation     *ActionValidationError `json:"validation,omitempty"`
  Revalidate     []string               `json:"revalidate,omitempty"`
  RevalidateTags []string               `json:"revalidateTags,omitempty"`
  RevalidateKeys []string               `json:"revalidateKeys,omitempty"`
}
```

Related types:

```go
type ActionRedirect struct {
  To     string `json:"to"`
  Status int    `json:"status,omitempty"`
}

type ActionValidationError struct {
  FieldErrors map[string]string `json:"fieldErrors,omitempty"`
  FormError   string            `json:"formError,omitempty"`
}
```

Enhanced form behavior (`X-Gospa-Enhance: 1`):

- Success returns JSON (default `code: "SUCCESS"` when omitted).
- `kit.Redirect` returns `code: "REDIRECT"` with `redirect`.
- `kit.Fail(status, data)` returns HTTP status + `code: "FAIL"` and validation when coercible.
- Revalidation hints are applied server-side after successful actions.

Non-enhanced form behavior:

- Redirect/fail use redirect-based browser flow (`303` default).
- If `ActionResponse.Redirect` is set, server redirects to `redirect.to`.

## Client action helpers

### `enhanceForm(form, options)`

Key semantics:

- Resolves `_action` from `data-gospa-action` / submitter value / form dataset / `"default"`.
- Sends `X-Gospa-Enhance: 1` and `Accept: application/json`.
- GET/HEAD submits serialize form data into query params; other methods send `FormData` body.
- Concurrent submits abort stale requests; only latest response is applied.
- Validation errors populate `aria-invalid` and `data-gospa-error`.
- Revalidation hints (`revalidate*`) are applied before `onSuccess`.

`options` callbacks:

- `optimistic(form, formData)`
- `onPending(form)`
- `onSuccess(result, form, response)`
- `onValidation(validation, form, response)`
- `onRedirect(redirect, form, response)`
- `onError(error, form, response?)`

### `callRouteAction(path, action, body?, init?)`

- Appends `_action=<action>` query param.
- Sends enhancement headers (`X-Gospa-Enhance: 1`, `Accept: application/json`).
- Returns parsed JSON payload (`ActionEnhanceSuccess` shape).
- Throws `RouteActionError` on non-2xx by default.
- Set `init.throwOnError = false` to opt into payload-first handling.

### `loadRouteData(path, init?)`

- Fetches `?__data=1` endpoint with `Accept: application/json`.
- Throws if response is not OK.

### Additional route helper exports (`client/src/route-helpers.ts`)

```ts
function preloadData<T = Record<string, unknown>>(path: string, init?: RequestInit): Promise<T>;
function preloadCode(path: string): Promise<void>;
function goto(to: string, options?: NavigateOptions): Promise<boolean>;
function refresh(init?: RequestInit): Promise<void>;
function prefetchOnHover(
  selector: string,
  options?: { delay?: number; preloadCode?: boolean; preloadData?: boolean }
): () => void;

// re-exported aliases from navigation
const beforeNavigate: typeof onBeforeNavigate;
const afterNavigate: typeof onAfterNavigate;
function invalidateAll(): Promise<void>;
```

## M1 helper APIs (stable)

These helpers are implemented and part of the supported parity surface.

### Server helpers (`routing/kit`)

```go
// Track data dependency keys for invalidation affinity.
func Depends(keys ...string)

// Execute work outside dependency tracking.
func Untrack(fn func() error) error

// Read immediate parent layout data with a typed contract.
func Parent[T any](c routing.LoadContext) (T, error)

// Return typed HTTP error control-flow from load/actions.
func Error(status int, body interface{}) error
```

- `kit.Depends` keys are indexed as both cache tags and cache keys using the `dep:<key>` namespace.
- `kit.Untrack` suppresses dependency capture for wrapped operations.
- `kit.Parent[T]` reads only the nearest parent layout load payload.
- `kit.Error` is serialized for `?__data=1` and enhanced actions, and mapped to SSR status rendering.

### Client helpers (`/_gospa/runtime.js`)

```ts
// Reload current route data in place.
function refresh(init?: RequestInit): Promise<void>;

// Declaratively bind hover-based prefetch to links.
function prefetchOnHover(
  selector: string,
  options?: { delay?: number; preloadCode?: boolean; preloadData?: boolean }
): () => void;
```

### Deterministic coverage references

- `kit.Depends` / `kit.Untrack` / `kit.Parent[T]`: `render_load_helpers_test.go`
- `kit.Error`: `render_load_helpers_test.go`, `gospa_form_action_test.go`
- `refresh` / `prefetchOnHover`: `client/src/route-helpers.test.ts`
- dependency invalidation indexing: `render_invalidate_test.go`
