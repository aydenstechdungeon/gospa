# Client Runtime API

## Navigation

```typescript
import { navigate } from '@gospa/client';

await navigate('/dashboard', { replace: true, scroll: false });
// Deprecated alias: { scrollToTop: false }
```

## Invalidation

```typescript
import { invalidate, invalidateTag, invalidateKey } from '@gospa/client';

await invalidate('/dashboard');
await invalidateTag('route:/dashboard');
await invalidateKey('path:/dashboard');
```

## Progressive Form Actions

```typescript
import { enhanceForms } from '@gospa/client';

enhanceForms('form[data-gospa-enhance]', {
  onValidation(validation) {
    console.log(validation.fieldErrors, validation.formError);
  },
  onSuccess(result) {
    console.log(result.data);
  },
});
```

Structured server responses use `routing.ActionResponse` and may include
`data`, `validation`, `redirect`, `error`, and `revalidate*` hints.

## Route Helper Surface

```typescript
import {
  loadRouteData,
  callRouteAction,
  preloadData,
  preloadCode,
  goto,
  refresh,
  prefetchOnHover,
  invalidateAll,
  beforeNavigate,
  afterNavigate,
} from '@gospa/client';
```

- `loadRouteData(path)` fetches `?__data=1` payloads.
- `callRouteAction(path, action, body)` posts with progressive enhancement headers and `_action` query key.
- `callRouteAction` returns parsed JSON payload and throws `RouteActionError` on non-2xx by default.
- Pass `{ throwOnError: false }` in `init` to handle non-2xx responses as payloads.
- `preloadData` / `preloadCode` / `goto` / `refresh` / `invalidateAll` provide convenience wrappers.
- `prefetchOnHover(selector, options?)` binds hover-based prefetch behavior and returns a cleanup function.
