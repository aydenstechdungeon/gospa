# Client Runtime Overview

GoSPA provides multiple runtime variants to balance security, performance, and bundle size.

## Runtime Distribution

The runtime module served at **`/_gospa/runtime.js`** is built from the client source at [`client/package.json`](https://github.com/aydenstechdungeon/gospa/blob/main/client/package.json):

- `/_gospa/runtime.js` → default runtime (`dist/runtime.js`)

Additional bundles (`runtime-core.js`, `runtime-micro.js`, `runtime-simple.js`) are built into `dist/` for **embedding** in the Go binary. The runtime is not published as a standalone npm package; use the server-served module path (`/_gospa/runtime.js`) or vendor files from `dist/`.

## Runtime Variants

### Default Runtime (`/_gospa/runtime.js`) — Recommended

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

// ES module import (browser or bundler)
import { init, Rune, navigate } from '/_gospa/runtime.js';
init();
```

## Security Model Comparison

| Import / bundle | Sanitizer | Trust model | Use case |
|-----------------|-----------|-------------|----------|
| `/_gospa/runtime.js` | None (no bundled sanitizer) | Trust server (Templ) + escaped dynamic HTML by default | Most apps with CSP |
| Embedded `runtime-core` / `micro` | None | Custom | Workers, embeds |
