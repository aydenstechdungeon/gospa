# GoSPA Runtime Selection Guide

GoSPA provides multiple runtime variants to balance security, performance, and bundle size. This guide helps you choose the right one.

## Published npm packages

The **`@gospa/client`** package [exports](https://github.com/aydenstechdungeon/gospa/blob/main/client/package.json) only:

- `@gospa/client` → default runtime (`dist/runtime.js`)
- `@gospa/client/runtime-secure` → DOMPurify-enabled runtime (`dist/runtime-secure.js`)

Additional bundles (`runtime-core.js`, `runtime-micro.js`, `runtime-simple.js`) are built into `dist/` for **embedding** in the Go binary; they are **not** separate npm import paths unless you vendor the files.

## Runtime Variants

### Default Runtime (`@gospa/client`) — Recommended

The default runtime trusts server-rendered HTML (Templ auto-escapes all content). No DOMPurify bundle is included by default.

**File (build output):** `runtime.js`

**Features:**
- Trust-the-server security model
- All core features (WebSocket, Navigation, Transitions)
- Smallest bundle size
- CSP-first approach to security

**Size:**
- Uncompressed: ~15 KB
- Gzipped: ~6 KB

**When to use:**
- Most applications (recommended default)
- Server-rendered apps using Templ
- Apps without user-generated HTML content
- When you have a proper CSP configured

```typescript
// Browser-style (no bundler)
import * as GoSPA from "/_gospa/runtime.js";
GoSPA.init();

// npm style (with bundler)
import { init, Rune, navigate } from '@gospa/client';
init();
```

### Secure Runtime (`@gospa/client/runtime-secure`)

The secure runtime includes DOMPurify for HTML sanitization. Use this when displaying user-generated content.

**File (build output):** `runtime-secure.js`

**Features:**
- DOMPurify HTML sanitization
- Protection against XSS attacks
- Safe rendering of user-generated content
- All core features (WebSocket, Navigation, Transitions)

**Size:**
- Uncompressed: ~35 KB
- Gzipped: ~13 KB

**When to use:**
- Rendering user-generated HTML content
- Social media apps with comments
- Forums, wikis, CMS with rich text
- Any app displaying untrusted HTML

```typescript
// Browser-style (no bundler)
import * as GoSPA from "/_gospa/runtime-secure.js";
GoSPA.init();

// npm style (with bundler)
import { init, sanitize } from '@gospa/client/runtime-secure';
init();

// Sanitize user content
const cleanHtml = await GoSPA.sanitize(userComment);
```

### Core bundle (`runtime-core.js`, embed / advanced)

Built from `client/src/runtime-core.ts`. Reactive primitives and wiring used by the default runtime; **not** a separate npm export.

**Typical use:** embedded scripts, custom bundling, or advanced setups—not the default app path.

**Features (vs full default):** omits higher-level features bundled in `runtime.ts` (see source tree).

### Micro bundle (`runtime-micro.js`, embed)

Minimal state-focused bundle for **Web Workers** or experiments. **Not** a published npm entrypoint.

### Simple runtime (`runtime-simple.js`, embed)

Lightweight variant used when `SimpleRuntime: true` is configured server-side. Prefer the default runtime unless you explicitly need this embed.

---

## Security Model Comparison

| Import / bundle | Sanitizer | Trust model | Use case |
|-----------------|-----------|-------------|----------|
| `@gospa/client` | None (optional `setSanitizer`) | Trust server (Templ) | Most apps with CSP |
| `@gospa/client/runtime-secure` | DOMPurify | Sanitize UGC | User-generated HTML |
| Embedded `runtime-core` / `micro` | Varies | Custom | Workers, embeds |

## When Do You Need Each Runtime?

### Use `gospa` (default) when:
- Building a typical server-rendered app
- Using Templ for all templates
- Content comes from your database (trusted)
- You have a CSP configured
- No user-generated HTML (just text/JSON)

### Use `gospa/runtime-secure` when:
- Users can submit HTML content
- Running a forum, wiki, or social app
- Displaying rich text from untrusted sources
- Embedding third-party HTML widgets

### Example scenarios:

| App Type | Recommended Runtime |
|----------|---------------------|
| E-commerce store | `gospa` |
| Dashboard (internal) | `gospa` |
| Blog | `gospa` |
| Forum / Community | `gospa/runtime-secure` |
| Social media app | `gospa/runtime-secure` |
| Wiki | `gospa/runtime-secure` |
| SaaS app | `gospa` |
| Comment system | `gospa/runtime-secure` |

---

## Using DOMPurify with the Default Runtime

Even with the default runtime, you can add DOMPurify manually for specific components:

```typescript
import { init, setSanitizer } from '@gospa/client';
import DOMPurify from 'dompurify';

init();

// Set custom sanitizer for UGC sections
setSanitizer((html) => DOMPurify.sanitize(html));
```

Or use the secure runtime import:

```typescript
import { sanitize } from '@gospa/client/runtime-secure';

// Use only where needed
const clean = await sanitize(dirtyHtml);
```

---

## Client-Side Usage

All runtimes export the same core API:

```typescript
import {
    // Core
    init,
    createComponent,
    destroyComponent,
    getComponent,
    getState,
    setState,
    callAction,
    bind,
    autoInit,
    
    // State Primitives
    Rune,
    Derived,
    Effect,
    StateMap,
    batch,
    effect,
    watch,
    
    // DOM Bindings
    bindElement,
    bindTwoWay,
    renderIf,
    renderList,
    
    // Events
    on,
    offAll,
    debounce,
    throttle,
    delegate,
    onKey,
    
    // WebSocket
    getWebSocket,
    sendAction,
    syncedRune,
    
    // Navigation
    getNavigation,
    navigate,
    back,
    forward,
    
    // Transitions
    getTransitions,
    fade,
    fly,
    slide,
    scale,
} from '@gospa/client'; // or '@gospa/client/runtime-secure'
```

---

## Migration from v1.x

If you're upgrading from GoSPA v1.x:

### Before (v1.x):
```go
// Server config
app := gospa.New(gospa.Config{
    SimpleRuntime: true,  // Use lightweight runtime
})
```

### After (v2.x):
```typescript
// Client - no changes needed for most apps
import { init } from '@gospa/client';  // Default runtime (trusts server)
init();

// Only change if you have user-generated content:
import { init } from '@gospa/client/runtime-secure';
init();
```

### Key changes:
1. `SimpleRuntime` config option removed (no longer needed)
2. Default runtime no longer includes DOMPurify
3. Added `gospa/runtime-secure` for UGC scenarios
4. `runtime-simple.js` is deprecated

---

## Troubleshooting

### Content Not Rendering User HTML

If user HTML content is being escaped instead of rendered:

1. **Check your runtime**: Use `gospa/runtime-secure` for user-generated HTML
2. **Use `type: 'html'` binding**: `bindElement(el, content, { type: 'html' })`
3. **Sanitize manually**: Use `sanitize()` from `gospa/runtime-secure`

### Bundle Size Too Large

If your bundle is larger than expected:

1. **Check imports**: Ensure you're importing from `gospa`, not `gospa/runtime-secure`
2. **Tree shaking**: Use named imports to enable tree shaking
3. **Verify runtime**: Check Network tab to see which runtime file is loaded

### Security Warnings

If security tools flag content:

1. **Use secure runtime**: Switch to `gospa/runtime-secure` for UGC
2. **Add CSP headers**: Configure strict Content-Security-Policy
3. **Server-side validation**: Validate all user input on the server
