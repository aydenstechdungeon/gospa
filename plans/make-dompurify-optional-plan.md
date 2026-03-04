# Make DOMPurify Optional - Implementation Plan

## Overview

Shift GoSPA from "sanitization-by-default" to "trust-the-server-by-default" model, similar to modern frameworks. DOMPurify becomes an opt-in dependency rather than bundled by default.

## Architecture Changes

### Current State
- `runtime.ts` - Auto-configures DOMPurify sanitizer
- `runtime-simple.ts` - Auto-configures simple sanitizer  
- `dom.ts` - Warns if no sanitizer configured
- Navigation and HTML bindings always sanitize

### Target State
- `runtime.ts` - No automatic sanitizer (trust server)
- `runtime-simple.ts` - Removed (no longer needed)
- `dom.ts` - Silent passthrough by default
- DOMPurify available as explicit import
- New `runtime-secure.ts` - Opt-in DOMPurify runtime

## Implementation Steps

### Phase 1: Client Code Changes

#### 1.1 Modify `client/src/dom.ts`
```typescript
// Remove the warning, default to passthrough
export let sanitizeHtml: (html: string) => string | Promise<string> = (html) => html;
```

#### 1.2 Modify `client/src/navigation.ts`
```typescript
// Remove safeSanitize wrapper, use content directly
const content = data.content;
contentEl.innerHTML = content;
```

#### 1.3 Modify `client/src/runtime.ts`
```typescript
// Remove automatic DOMPurify setup
// Keep exports for backward compatibility
export { domPurifySanitizer, setSanitizer } from './sanitize.ts';
// No longer calls setSanitizer()
```

#### 1.4 Create `client/src/runtime-secure.ts` (New)
```typescript
// Opt-in secure runtime with DOMPurify
import { domPurifySanitizer, preloadSanitizer } from './sanitize.ts';
import { setSanitizer } from './dom.ts';

setSanitizer(domPurifySanitizer);
preloadSanitizer();

// Re-export everything from runtime
export * from './runtime.ts';
```

#### 1.5 Deprecate `client/src/runtime-simple.ts`
- Mark as deprecated
- Point users to new model

#### 1.6 Update `client/package.json`
```json
{
  "optionalDependencies": {
    "dompurify": "^3.3.1"
  }
}
```

### Phase 2: Documentation Updates

#### 2.1 Update `docs/SECURITY.md`
- Remove "DOMPurify by default" messaging
- Document "Trust the Server" philosophy
- Add CSP-first security recommendations
- Document opt-in sanitization for user-generated content

#### 2.2 Update `docs/CLIENT_RUNTIME.md`
- Update sanitizer configuration section
- Document new `runtime-secure.ts` import
- Migration guide from old to new model

#### 2.3 Update `docs/RUNTIME.md`
- New runtime variants table
- `runtime.js` - Standard (no sanitizer)
- `runtime-secure.js` - With DOMPurify
- Bundle size comparisons

#### 2.4 Update `README.md`
- Security section: Trust Templ + CSP
- Bundle size section: ~20KB smaller
- Installation: `npm install dompurify` for user content

### Phase 3: Website Documentation

#### 3.1 Update `website/routes/docs/security/page.templ`
- New "Trust the Server" section
- CSP recommendations
- Opt-in sanitization documentation

#### 3.2 Update `website/routes/docs/runtime/page.templ`
- New runtime comparison table
- Remove "Full vs Simple" distinction
- Add "Standard vs Secure" distinction

#### 3.3 Create migration guide on website
- For users upgrading from v1
- How to enable sanitization if needed

## New Developer Experience

### Standard Usage (No User Content)
```typescript
import { init } from 'gospa/runtime';
init();
```
No sanitization, ~15KB bundle, CSP headers recommended.

### With User-Generated Content
```typescript
import { init } from 'gospa/runtime-secure';
init();
```
With DOMPurify, ~35KB bundle, XSS protection enabled.

### Manual Configuration
```typescript
import { init } from 'gospa/runtime';
import { domPurifySanitizer, setSanitizer } from 'gospa/runtime';

setSanitizer(domPurifySanitizer);
init();
```

## Breaking Changes

| Before | After |
|--------|-------|
| `runtime.ts` has DOMPurify | `runtime.ts` no sanitizer |
| `runtime-simple.ts` basic sanitizer | `runtime-simple.ts` deprecated |
| `DisableSanitization: true` | Default behavior |
| ~35KB default bundle | ~15KB default bundle |
| Sanitization opt-out | Sanitization opt-in |

## Security Recommendations (New Docs)

1. **Trust the Server**: Templ auto-escapes, server is trusted
2. **Use CSP Headers**: Primary XSS defense
3. **Opt-in Sanitization**: Only for user-generated HTML
4. **Validate Server-Side**: All user input validated in Go

## CSP Example (For Docs)

```go
app.Use(func(c fiber.Ctx) error {
    c.Set("Content-Security-Policy", 
        "default-src 'self'; "+
        "script-src 'self'; "+
        "style-src 'self' 'unsafe-inline'")
    return c.Next()
})
```

## Migration Path

Users upgrading from v1 who need sanitization:

```typescript
// Change this:
import { init } from 'gospa/runtime';

// To this:
import { init } from 'gospa/runtime-secure';
```

Or install DOMPurify manually:

```bash
npm install dompurify
```

```typescript
import { init } from 'gospa/runtime';
import { domPurifySanitizer, setSanitizer } from 'gospa/sanitize';

setSanitizer(domPurifySanitizer);
init();
```
