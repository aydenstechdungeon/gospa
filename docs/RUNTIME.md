# GoSPA Runtime Selection Guide

GoSPA provides two runtime variants to balance security and performance. This guide helps you choose the right one.

## Runtime Variants

### Full Runtime (Default)

The full runtime includes DOMPurify for HTML sanitization. Use this when security is a priority.

**File:** `runtime.js` (embedded)

**Features:**
- DOMPurify HTML sanitization
- Protection against XSS attacks
- Safe rendering of user-generated content
- All core features (WebSocket, Navigation, Transitions)

**Size:**
- Uncompressed: 25.2 KB
- Gzipped: 9.8 KB

**When to use:**
- Rendering user-generated content
- Public-facing applications
- Applications handling untrusted data
- Default choice for most applications

### Simple Runtime

The simple runtime uses a basic HTML sanitizer for higher performance. Use this when you control all content.

**File:** `runtime-simple.js` (embedded)

**Features:**
- Basic HTML sanitization (tag stripping only)
- Smaller bundle size
- All core features (WebSocket, Navigation, Transitions)
- Higher performance

**Size:**
- Uncompressed: 1.7 KB (wrapper) + 7.6 KB (core) = 9.3 KB total
- Gzipped: 0.9 KB (wrapper) + 3.0 KB (core) = 3.9 KB total

**When to use:**
- Internal tools and admin panels
- Applications where you control all HTML content
- Performance-critical applications
- No user-generated content

---

## How to Select Runtime

### Server-Side Configuration

Set `SimpleRuntime: true` in your `gospa.Config`:

```go
app := gospa.New(gospa.Config{
    RoutesDir:     "./routes",
    SimpleRuntime: true,  // Use simple runtime
})
```

The framework automatically serves the correct runtime file based on this setting.

### How It Works

1. When `SimpleRuntime: false` (default):
   - Serves `runtime.js` with DOMPurify
   
2. When `SimpleRuntime: true`:
   - Serves `runtime-simple.js` with basic sanitizer

The runtime files are embedded in the binary with content hashes for cache busting.

---

## Security Considerations

### Full Runtime Security

The full runtime uses DOMPurify to sanitize HTML before rendering:

```typescript
// DOMPurify removes dangerous content
const dirty = '<script>alert("xss")</script><p>Safe content</p>';
const clean = DOMPurify.sanitize(dirty);
// Result: '<p>Safe content</p>'
```

**Protected against:**
- XSS (Cross-Site Scripting)
- HTML injection
- JavaScript URL injection
- Event handler injection

### Simple Runtime Security

The simple runtime uses a basic sanitizer that strips HTML tags:

```typescript
// Basic sanitizer - strips tags only
const dirty = '<script>alert("xss")</script><p>Safe content</p>';
const clean = simpleSanitizer(dirty);
// Result: 'Safe content' (tags removed)
```

**Limitations:**
- Does not validate attribute values
- Does not handle all edge cases
- Not suitable for untrusted content

---

## Size Comparison

| Runtime | Uncompressed | Gzipped |
|---------|--------------|---------|
| Full (runtime.js) | 25.2 KB | 9.8 KB |
| Simple (wrapper + core) | 9.3 KB | 3.9 KB |
| Savings | 16 KB (63%) | 5.9 KB (60%) |

---

## Client-Side Usage

Both runtimes export the same API:

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
} from '@gospa/runtime';
```

---

## Manual Runtime Selection

If you need to manually specify the runtime script:

```go
app := gospa.New(gospa.Config{
    RuntimeScript: "/static/js/custom-runtime.js",
})
```

This overrides the automatic selection.

---

## Hybrid Approach

For applications with mixed security requirements, you can:

1. Use simple runtime for the main application
2. Use DOMPurify directly for specific user-generated content:

```typescript
import DOMPurify from 'dompurify';

// In your component
const safeContent = DOMPurify.sanitize(userContent);
element.innerHTML = safeContent;
```

This gives you performance for most operations while maintaining security where needed.

---

## Decision Matrix

| Scenario | Recommended Runtime |
|----------|---------------------|
| Public website with user comments | Full |
| E-commerce product pages | Full |
| Admin dashboard (internal) | Simple |
| Data visualization app | Simple |
| Blog with markdown posts | Simple (if sanitized server-side) |
| Forum or social app | Full |
| Single-player game | Simple |
| Real-time trading dashboard | Simple |
| CMS with rich text editing | Full |

---

## Migration

To switch between runtimes, simply change the config:

```go
// Before (full runtime)
app := gospa.New(gospa.Config{
    RoutesDir: "./routes",
    // SimpleRuntime defaults to false
})

// After (simple runtime)
app := gospa.New(gospa.Config{
    RoutesDir:     "./routes",
    SimpleRuntime: true,
})
```

No client-side code changes required - the API is identical.

---

## Troubleshooting

### Content Not Rendering Correctly

If content is being over-sanitized in simple runtime:

1. Check if content contains valid HTML
2. Consider switching to full runtime
3. Pre-sanitize content server-side

### Security Warnings

If security tools flag content:

1. Ensure you're using full runtime for user content
2. Add Content-Security-Policy headers
3. Audit content sources
