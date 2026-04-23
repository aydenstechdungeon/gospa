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
`data`, `validation`, `redirect`, and `revalidate*` hints.
