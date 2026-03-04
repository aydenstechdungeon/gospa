# Troubleshooting Island Hydration & Components

## Island Not Hydrating

### Problem
The island element appears in the DOM but never becomes interactive. No `gospa:hydrated` event fires.

### Causes & Solutions

#### 1. Missing `data-gospa-island` Attribute

```html
<!-- BAD - No island attribute -->
<div id="counter">
    <button>Count: 0</button>
</div>

<!-- GOOD - Properly marked island -->
<div data-gospa-island="Counter" id="counter">
    <button>Count: 0</button>
</div>
```

#### 2. Missing Island Module

Ensure the island module exists at the expected path:

```javascript
// Default: /islands/{name}.js
// For data-gospa-island="Counter", expects:
// GET /islands/Counter.js
```

Check browser Network tab for 404 errors on island module requests.

#### 3. Island Manager Not Initialized

```javascript
// Ensure islands are initialized
import { initIslands } from '@gospa/runtime';

// Auto-initialization with data-gospa-auto on <html>
// Or manual:
const manager = initIslands({
    moduleBasePath: '/islands',
    defaultTimeout: 30000,
});
```

#### 4. JavaScript Error in Island Module

Check console for errors in the island's `hydrate` function:

```javascript
// /islands/Counter.js
export default {
    hydrate(element, props, state) {
        console.log('Hydrating counter:', element, props, state);
        // Check for errors here
    }
};
```

---

## "Island Hydration Timeout"

### Problem
Island takes too long to hydrate and times out.

### Causes & Solutions

#### 1. Large Module Size

```javascript
// BAD - Heavy imports blocking hydration
import { HeavyLibrary } from 'heavy-lib';

export default {
    async hydrate(element, props, state) {
        const lib = new HeavyLibrary();  // Slow initialization
    }
};

// GOOD - Lazy load heavy code
export default {
    async hydrate(element, props, state) {
        const { HeavyLibrary } = await import('heavy-lib');
        const lib = new HeavyLibrary();
    }
};
```

#### 2. Slow Network

Increase timeout for slow connections:

```javascript
const manager = initIslands({
    defaultTimeout: 60000,  // 60 seconds
});
```

Or for specific islands:

```html
<div 
    data-gospa-island="Chart" 
    data-gospa-timeout="60000"
>
```

---

## Island Hydrates Multiple Times

### Problem
Island `hydrate` function runs more than once, causing duplicate event listeners or state issues.

### Solution
Check if already hydrated before running:

```javascript
export default {
    hydrate(element, props, state) {
        // Prevent double hydration
        if (element.dataset.hydrated) return;
        element.dataset.hydrated = 'true';
        
        // Your hydration logic
        const button = element.querySelector('button');
        let count = state.count || 0;
        
        button.addEventListener('click', () => {
            count++;
            button.textContent = `Count: ${count}`;
        });
    }
};
```

Or use GoSPA's built-in protection:

```javascript
import { hydrateIsland } from '@gospa/runtime';

// This is idempotent - only hydrates once
await hydrateIsland('counter');
```

---

## Island Not Found in Lazy Mode

### Problem
Calling `hydrateIsland()` returns `null` or throws "Island not found".

### Causes & Solutions

#### 1. Wrong ID or Name

```javascript
// HTML
<div data-gospa-island="Counter" id="my-counter">

// JavaScript - Can use either:
await hydrateIsland('my-counter');  // By ID
await hydrateIsland('Counter');      // By island name
```

#### 2. Island Not in DOM

Ensure island exists before calling:

```javascript
import { getIslandManager } from '@gospa/runtime';

const manager = getIslandManager();
const island = manager.getIsland('my-counter');

if (island) {
    await manager.hydrateIsland(island);
} else {
    console.error('Island not found');
}
```

#### 3. Discover Islands After DOM Changes

For dynamically added islands:

```javascript
// Add new island to DOM
container.innerHTML = `
    <div data-gospa-island="DynamicChart" id="chart-1">
        <canvas></canvas>
    </div>
`;

// Re-scan for new islands
const manager = getIslandManager();
manager.discoverIslands();

// Now hydrate
await hydrateIsland('chart-1');
```

---

## Hydration Mode Not Working

### Problem
`data-gospa-mode` doesn't behave as expected (e.g., `visible` hydrates immediately).

### Solutions

#### Visible Mode Not Triggering

```html
<!-- Ensure element is initially in viewport or has threshold -->
<div 
    data-gospa-island="LazyImage"
    data-gospa-mode="visible"
    data-gospa-threshold="100"  <!-- Trigger 100px before visible -->
>
```

Check if `IntersectionObserver` is supported (polyfill for older browsers).

#### Idle Mode Never Triggering

```html
<!-- Add delay if needed -->
<div 
    data-gospa-island="Analytics"
    data-gospa-mode="idle"
    data-gospa-defer="1000"  <!-- Wait 1s after idle -->
>
```

Note: `requestIdleCallback` may never fire on busy main thread. Use with caution for critical UI.

#### Interaction Mode Issues

```html
<!-- Works with: mouseenter, touchstart, focusin, click -->
<div 
    data-gospa-island="Tooltip"
    data-gospa-mode="interaction"
>
```

If not triggering, check CSS doesn't block events:

```css
/* BAD - Blocks interaction events */
[data-gospa-island] {
    pointer-events: none;
}

/* GOOD - Allow events through */
[data-gospa-island] {
    pointer-events: auto;
}
```

---

## Props Not Available in Hydrate

### Problem
`props` parameter is empty or undefined in `hydrate()`.

### Solutions

#### 1. Check JSON Format

```html
<!-- BAD - Invalid JSON (single quotes) -->
<div data-gospa-props="{'initial': 10}">

<!-- GOOD - Valid JSON -->
<div data-gospa-props='{"initial": 10}'>
```

#### 2. Escape Quotes Properly

```html
<!-- In templ templates, use proper escaping -->
<div data-gospa-props={ fmt.Sprintf(`{"message": "%s"}`, escapedMessage) }>
```

#### 3. Access Props Safely

```javascript
export default {
    hydrate(element, props = {}, state = {}) {
        const initial = props.initial ?? 0;
        const message = props.message ?? 'Default';
        
        console.log('Props:', props);
        console.log('State:', state);
    }
};
```

---

## State Not Persisting Across Navigation

### Problem
Island state resets when navigating between pages.

### Solutions

#### 1. Use State Prop

```html
<div 
    data-gospa-island="Counter"
    data-gospa-state='{"count": 5}'
>
```

#### 2. Store in URL or Session

```javascript
export default {
    hydrate(element, props, state) {
        // Restore from URL
        const params = new URLSearchParams(window.location.search);
        let count = parseInt(params.get('count')) || state.count || 0;
        
        const button = element.querySelector('button');
        button.addEventListener('click', () => {
            count++;
            updateURL();
            render();
        });
        
        function updateURL() {
            const url = new URL(window.location);
            url.searchParams.set('count', count);
            window.history.replaceState({}, '', url);
        }
        
        function render() {
            button.textContent = `Count: ${count}`;
        }
        
        render();
    }
};
```

#### 3. Use Global State

```javascript
import { getState } from '@gospa/runtime';

export default {
    hydrate(element, props, state) {
        // Access global state that persists across navigation
        const globalCount = getState('counter');
    }
};
```

---

## Server-Only Island Still Tries to Hydrate

### Problem
`data-gospa-server-only` island attempts hydration anyway.

### Solution
Verify attribute value:

```html
<!-- These all work -->
<div data-gospa-island="Static" data-gospa-server-only></div>
<div data-gospa-island="Static" data-gospa-server-only="true"></div>
<div data-gospa-island="Static" data-gospa-server-only="1"></div>

<!-- Check island manager config -->
<script>
    initIslands({
        respectServerOnly: true,  // Default
    });
</script>
```

---

## Priority Queue Not Working

### Problem
`data-gospa-priority` doesn't affect hydration order.

### Solution
Priority only affects `immediate` mode islands:

```html
<!-- High priority - hydrates first -->
<div 
    data-gospa-island="Navigation"
    data-gospa-priority="high"
    data-gospa-mode="immediate"
>

<!-- Low priority - hydrates last -->
<div 
    data-gospa-island="FooterWidget"
    data-gospa-priority="low"
    data-gospa-mode="immediate"
>
```

Non-immediate modes (`visible`, `idle`, `interaction`, `lazy`) are not affected by priority.

---

## Common Mistakes

### Mistake 1: Modifying Props During Hydration

```javascript
// BAD - Props should be immutable
export default {
    hydrate(element, props) {
        props.count = 10;  // Don't modify props!
    }
};

// GOOD - Use local state
export default {
    hydrate(element, props) {
        let count = props.count || 0;  // Copy to local variable
        count = 10;  // OK to modify local
    }
};
```

### Mistake 2: Not Cleaning Up Event Listeners

```javascript
// BAD - Memory leak on re-hydration
export default {
    hydrate(element) {
        element.addEventListener('click', handler);
    }
};

// GOOD - Track for cleanup
export default {
    hydrate(element) {
        if (element._cleanup) element._cleanup();
        
        const handler = () => console.log('clicked');
        element.addEventListener('click', handler);
        
        element._cleanup = () => {
            element.removeEventListener('click', handler);
        };
    }
};
```

### Mistake 3: Expecting Island Before DOM Ready

```javascript
// BAD - May run before island is registered
document.addEventListener('DOMContentLoaded', () => {
    hydrateIsland('counter');  // Might fail
});

// GOOD - Wait for manager ready
import { initIslands } from '@gospa/runtime';

const manager = initIslands();
manager.ready().then(() => {
    hydrateIsland('counter');
});
```
