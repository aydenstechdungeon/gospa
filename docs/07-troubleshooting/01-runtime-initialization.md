# Troubleshooting Runtime Initialization

## "GoSPA is not defined" Error

### Problem
When calling `GoSPA.remote()` or other GoSPA functions from inline scripts or event handlers, you get:

```
Uncaught ReferenceError: GoSPA is not defined
```

### Cause
The GoSPA runtime hasn't been initialized yet. The `window.GoSPA` global is created by the runtime's `init()` function, which is called automatically when the page loads if properly configured.

### Solution

#### 1. Ensure `data-gospa-auto` is on your `<html>` tag

The runtime checks for this attribute to auto-initialize:

```html
<!DOCTYPE html>
<html lang="en" data-gospa-auto>
<head>
    <title>My App</title>
</head>
<body>
    <!-- Your content -->
</body>
</html>
```

#### 2. Use GoSPA's root layout

If using a custom layout, ensure it includes the runtime script. The default root layout handles this automatically:

```templ
package routes

templ RootLayout() {
    <!DOCTYPE html>
    <html data-gospa-auto>
        <head>
            <title>My App</title>
        </head>
        <body>
            { children... }
            <!-- Runtime script is auto-injected here by GoSPA -->
        </body>
    </html>
}
```

#### 3. Manual initialization (advanced)

If not using `data-gospa-auto`, manually initialize in a module script:

```html
<script type="module">
    import * as GoSPA from '/_gospa/runtime.js';
    GoSPA.init({
        wsUrl: 'ws://localhost:3000/_gospa/ws',
        debug: false
    });
</script>
```

#### 4. Wait for DOM ready

If calling from a regular script tag, ensure the runtime has loaded:

```html
<script>
    // Wait for GoSPA to be available
    function waitForGoSPA(callback, maxAttempts = 50) {
        let attempts = 0;
        const check = () => {
            if (typeof GoSPA !== 'undefined') {
                callback();
            } else if (attempts < maxAttempts) {
                attempts++;
                setTimeout(check, 100);
            } else {
                console.error('GoSPA failed to load');
            }
        };
        check();
    }

    waitForGoSPA(() => {
        // Now safe to use GoSPA
        GoSPA.remote('myAction', {});
    });
</script>
```

### How GoSPA is Created

The `window.GoSPA` object is created in `client/src/runtime-core.ts` when `init()` is called:

```typescript
// Create the public GoSPA global object
const GoSPA = {
    config,
    components,
    globalState,
    init,
    createComponent,
    destroyComponent,
    getComponent,
    getState,
    setState,
    callAction,
    bind,
    autoInit,
    // Remote actions
    remote,
    remoteAction,
    configureRemote,
    getRemotePrefix,
    // State primitives
    get Rune() { return Rune; },
    get Derived() { return Derived; },
    get Effect() { return Effect; },
    // Utility functions
    batch,
    effect,
    watch,
    // Events
    get on() { return on; },
    get offAll() { return offAll; },
    get debounce() { return debounce; },
    get throttle() { return throttle; }
};

// Expose to window as the primary public API
(window as any).GoSPA = GoSPA;
```

## "__GOSPA__ is not defined" vs "GoSPA is not defined"

There are two different globals:

- **`window.GoSPA`** - The public API (what you should use)
- **`window.__GOSPA__`** - Internal debugging object (same content, different name)

Always use `GoSPA` (without underscores) in your application code.

## ES Module Alternative

Instead of relying on the global, import directly from the runtime:

```html
<script type="module">
    import * as GoSPA from '/_gospa/runtime.js';
    
    // No need for GoSPA global
    const result = await GoSPA.remote('myAction', {});
</script>
```
This approach:
- Works immediately (no waiting for init)
- Is tree-shakeable
- Works better with TypeScript
- Doesn't pollute global scope

## Checking Runtime Status

To verify the runtime loaded correctly:

```javascript
// In browser console
console.log(typeof GoSPA);  // Should print "object"
console.log(Object.keys(GoSPA));  // Should list available methods
```

## Common Mistakes

### Mistake 1: Calling before DOM is ready

```javascript
// BAD - Script runs before runtime loads
goSPA.remote('action', {});  // Note: wrong case too!

// GOOD - Wait for page to load
window.addEventListener('DOMContentLoaded', () => {
    GoSPA.remote('action', {});
});
```

### Mistake 2: Wrong case

```javascript
// BAD - Wrong case
goSPA.remote('action', {});
Gospa.remote('action', {});

// GOOD
GoSPA.remote('action', {});
```

### Mistake 3: Script type="module" without import

```html
<!-- BAD - Module scripts don't see globals automatically -->
<script type="module">
    GoSPA.remote('action', {});  // May fail
</script>

<!-- GOOD - Either import or use regular script -->
<script type="module">
    import * as GoSPA from '/_gospa/runtime.js';
    GoSPA.remote('action', {});
</script>

<!-- Or -->
<script>
    GoSPA.remote('action', {});
</script>
```
