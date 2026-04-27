# Client-side Navigation

GoSPA implements an "instant" navigation pattern by default using a high-performance client-side router.

## Instant Navigation

When a user clicks a `data-gospa-link`, the following happens immediately:

1.  **URL Update**: The browser's URL bar is updated via `pushState` or `replaceState` before any network request is made.
2.  **Active State Transformation**: Links with `data-gospa-active` are updated immediately.
3.  **Loading Indicator**: The main page container receives `data-gospa-loading="true"`.

## Navigation Configuration

Configure navigation behavior in `gospa.Config`:

```go
NavigationOptions: gospa.NavigationOptions{
    SpeculativePrefetching: &gospa.NavigationSpeculativePrefetchingConfig{
        Enabled: ptr(true),
        HoverDelay: ptr(80),
    },
    ViewTransitions: &gospa.NavigationViewTransitionsConfig{
        Enabled: ptr(true),
    },
    ProgressBar: &gospa.NavigationProgressBarConfig{
        Enabled: ptr(true),
        Color:   ptr("#22d3ee"),
    },
}
```

## Persistent Elements

Use `data-gospa-permanent` to preserve elements across navigations (e.g., video players, sidebar scroll state).

```html
<ul data-gospa-permanent>
    <!-- Managed by client-side JS -->
</ul>
```

## Programmatic Navigation

```typescript
import { navigate } from '/_gospa/runtime.js';

navigate('/new-path', { 
    replace: true,
    scroll: false 
});
```

`scrollToTop` remains supported as a deprecated alias for `scroll` during migration.

## Invalidation API

```typescript
import { invalidate, invalidateTag, invalidateKey } from '/_gospa/runtime.js';

await invalidate('/blog/hello-world');
await invalidateTag('route:/blog/hello-world');
await invalidateKey('path:/blog/hello-world');
```
