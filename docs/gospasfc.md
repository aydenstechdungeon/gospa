# GoSPA SFC (Single File Components)

GoSPA Single File Components (SFCs) provide a powerful way to define interactive UI components with minimal overhead. SFCs combine HTML, CSS, and TypeScript reactivity into a single, cohesive file.

## SFC Components

A GoSPA SFC component is typically defined using a standard `templ` component combined with data-attributes that GoSPA's client-side runtime recognizes.

### Example:
```templ
package components

templ Counter(initial int) {
    <div data-gospa-component="Counter" data-gospa-state={ templ.JSONString(map[string]interface{}{"count": initial}) }>
        <button data-on:click="count++">Increment</button>
        <span data-bind="text:count">{ fmt.Sprint(initial) }</span>
    </div>
}
```

## Data Bindings

GoSPA provides several data-binding attributes to connect your HTML to its reactive state:

- `data-bind="text:key"`: Bind an element's text content.
- `data-bind="html:key"`: Bind an element's HTML content (automatically sanitized).
- `data-bind="class:name:key"`: Toggle a CSS class based on a reactive boolean.
- `data-model="key"`: Two-way data binding for input elements.

## Event Handlers

Events are registered using the `data-on:*` syntax.

```templ
<button data-on:click="open = !open">Toggle</button>
<input data-on:input.debounce.500ms="search($el.value)" />
```

### Event Modifiers
GoSPA supports several event modifiers:
- `.prevent`: Calls `event.preventDefault()`.
- `.stop`: Calls `event.stopPropagation()`.
- `.debounce.XXXms`: Debounces the event handler.
- `.throttle.XXXms`: Throttles the event handler.

## Islands

To make an SFC component interactive, you can also define an "Island" setup function in TypeScript.

```typescript
import { $state, registerSetup } from "@gospa/client";

registerSetup("Counter", (el, props, state) => {
    const count = $state(state.count || 0);
    
    el.querySelector("button")?.addEventListener("click", () => {
        count.set(count.get() + 1);
    });
});
```

GoSPA will automatically hydrate your island when it enters the viewport (if configured to do so).
