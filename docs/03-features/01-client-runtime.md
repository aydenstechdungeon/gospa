# GoSPA Runtime Selection Guide

GoSPA provides multiple runtime variants to balance security, performance, and bundle size. This guide helps you choose the right one.

## Runtime Variants

### Default Runtime (`gospa`) — Recommended

The default runtime trusts server-rendered HTML (Templ auto-escapes all content). No client-side sanitization is included by default.

**File:** `runtime.js`

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
import { init, Rune, navigate } from 'gospa';

init();
```

### Secure Runtime (`gospa/runtime-secure`)

The secure runtime includes DOMPurify for HTML sanitization. Use this when displaying user-generated content.

**File:** `runtime-secure.js`

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
import { init, sanitize } from 'gospa/runtime-secure';

init();

// Sanitize user content
const cleanHtml = await sanitize(userComment);
```

### Core Runtime (`gospa/core`)

Minimal runtime without extra features. Use for custom implementations.

**File:** `runtime-core.js`

**Features:**
- Reactive primitives only
- No WebSocket, Navigation, or Transitions
- No sanitization

**Size:**
- Uncompressed: ~12 KB
- Gzipped: ~5 KB

**When to use:**
- Custom runtime implementations
- Library authors
- When you need only reactive state

### Micro Runtime (`gospa/micro`)

Ultra-lightweight runtime for state-only applications.

**File:** `runtime-micro.js`

**Features:**
- Reactive primitives only
- No DOM operations
- No sanitization

**Size:**
- Uncompressed: ~3 KB
- Gzipped: ~1.5 KB

**When to use:**
- Web Workers
- Node.js scripts
- State-only applications

### Simple Runtime (`gospa/simple`) — Deprecated

⚠️ **DEPRECATED**: This runtime is deprecated and will be removed in v2.0. Use `gospa` (default) instead.

---

## Security Model Comparison

| Runtime | Sanitizer | Trust Model | Use Case |
|---------|-----------|-------------|----------|
| `gospa` (default) | None | Trust server (Templ) | Most apps with CSP |
| `gospa/runtime-secure` | DOMPurify | Sanitize UGC | Apps with user-generated HTML |
| `gospa/core` | None | Custom | Library authors |
| `gospa/micro` | None | Custom | State-only |

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
import { init, setSanitizer } from 'gospa';
import DOMPurify from 'dompurify';

init();

// Set custom sanitizer for UGC sections
setSanitizer((html) => DOMPurify.sanitize(html));
```

Or use the secure runtime import:

```typescript
import { sanitize } from 'gospa/runtime-secure';

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
} from 'gospa'; // or 'gospa/runtime-secure'
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
import { init } from 'gospa';  // Default runtime (trusts server)
init();

// Only change if you have user-generated content:
import { init } from 'gospa/runtime-secure';
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
