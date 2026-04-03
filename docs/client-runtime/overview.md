# Client Runtime Overview

GoSPA provides multiple runtime variants to balance security, performance, and bundle size.

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

## Security Model Comparison

| Import / bundle | Sanitizer | Trust model | Use case |
|-----------------|-----------|-------------|----------|
| `@gospa/client` | None (optional `setSanitizer`) | Trust server (Templ) | Most apps with CSP |
| `@gospa/client/runtime-secure` | DOMPurify | Sanitize UGC | User-generated HTML |
| Embedded `runtime-core` / `micro` | Varies | Custom | Workers, embeds |
