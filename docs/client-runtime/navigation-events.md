# Navigation & Events

Managing SPA navigation and advanced event handling in the GoSPA client runtime.

## Navigation API

Programmatic navigation for SPA behavior.

```typescript
import { navigate, back, forward, go } from '@gospa/client';

// Basic navigation
await navigate('/about');

// With options
await navigate('/dashboard', {
  replace: true,      // Replace history entry
  scroll: true        // Scroll to top after navigation
});

// Deprecated alias (still supported for one minor release)
await navigate('/dashboard', { scrollToTop: true });

// History navigation
back();              // Go back one page
forward();           // Go forward one page
go(-2);              // Go back 2 pages
```

## Prefetching

Preload pages for instant navigation.

```typescript
import { prefetch, prefetchLinks } from '@gospa/client';

// Prefetch a specific page
prefetch('/blog/hello-world');

// Prefetch multiple pages
prefetch(['/blog/post-1', '/blog/post-2']);

// Prefetch all links matching a selector
prefetchLinks('a[data-prefetch]');
```

## Cache Invalidation

Explicitly invalidate client/server navigation caches.

```typescript
import { invalidate, invalidateTag, invalidateKey } from '@gospa/client';

await invalidate('/dashboard');
await invalidateTag('route:/dashboard');
await invalidateKey('path:/dashboard');
```

## Navigation State

Track and manage navigation state reactively.

```typescript
import { createNavigationState } from '@gospa/client';

const nav = createNavigationState();

// Reactive properties
nav.currentPath;     // Current URL path
nav.isNavigating;    // True during navigation
nav.historyLength;   // Number of history entries
```

## Lifecycle Callbacks

Register callbacks that run before or after navigation.

```typescript
import { onBeforeNavigate, onAfterNavigate } from '@gospa/client';

// Before navigation (can cancel)
const unsubBefore = onBeforeNavigate((path) => {
  console.log('Starting navigation to:', path);
  if (path === '/admin' && !isLoggedIn) return false;
});

// After navigation
const unsubAfter = onAfterNavigate((path) => {
  console.log('Finished navigation to:', path);
});
```

## Global DOM Events

The runtime dispatches custom events on the document.

| Event | Detail | Description |
|-------|--------|-------------|
| `gospa:navigated` | `{path, state}` | After successful navigation |
| `gospa:navigation-start` | `{from, to}` | Before navigation starts |
| `gospa:navigation-error` | `{error, path}` | Navigation failed |

## Event Handling

Advanced event handling with modifiers and delegation.

```typescript
import { on, delegate, debounce, throttle } from '@gospa/client';

// Event with modifiers
on(form, 'submit:prevent', (e) => {
  console.log('Form submitted');
});

// Modifiers: :prevent, :stop, :once, :capture, :passive

// Event delegation
delegate(document.body, '.item', 'click', (e, target) => {
  console.log('Item clicked:', target);
});
```

## Keyboard Events

Keyboard shortcuts and key combinations.

```typescript
import { onKey, keys } from '@gospa/client';

// Single key
onKey(document, 'Escape', () => closeModal());

// Key combination
onKey(document, 'Ctrl+s', (e) => {
  e.preventDefault();
  saveDocument();
});
```

## Navigation Options

Configure navigation behavior at runtime via data attributes or JavaScript.

| Option | Default | Description |
|--------|---------|-------------|
| `data-gospa-progress` | `true` | Show/hide the progress bar |
| `data-gospa-prefetch` | `Enabled` | Prefetch links on hover/viewport entry |
| `data-gospa-view-transition` | `Enabled` | Enable native View Transitions |
