# DOM Bindings

How to connect your reactive state to the DOM using data attributes and programmatic APIs.

## DOM Attributes Reference

| Attribute | Description |
|-----------|-------------|
| `data-bind` | State binding (`key:type`) |
| `data-model` | Two-way binding (`key`) |
| `data-on` | Event handler (`event:action:args`) |
| `data-gospa-component` | Component ID |

## Binding Types

| Type | Description | Example |
|------|-------------|---------|
| `text` | `textContent` update | `data-bind="message:text"` |
| `html` | `innerHTML` (sanitized) | `data-bind="content:html"` |
| `style` | Inline style property | `data-bind="color:style:color"` |
| `class` | Toggle class name | `data-bind="isActive:class:active"` |
| `attr` | Any attribute | `data-bind="href:attr:href"` |
| `prop` | DOM property | `data-bind="checked:prop:checked"` |

## Programmatic Bindings

Bind runes to DOM elements manually for more complex scenarios.

```typescript
// TypeScript
import { bindElement, bindTwoWay, rune } from '/_gospa/runtime.js';

const element = document.getElementById('count');
const textRune = rune('Hello');

// One-way bindings
bindElement(element, textRune);
bindElement(element, htmlRune, { type: 'html' });
bindElement(element, colorRune, { type: 'style', attribute: 'color' });

// Two-way binding
bindTwoWay(inputElement, textRune);
```

### bindElement()

Creates a one-way binding from a rune to a DOM element.

```typescript
import { bindElement, rune } from '/_gospa/runtime.js';

const count = rune(0);

// Basic text binding
bindElement(document.getElementById('count'), count);

// Style binding
const color = rune('red');
bindElement(document.getElementById('box'), color, {
  type: 'style',
  attribute: 'backgroundColor'
});

// Class binding
const isActive = rune(true);
bindElement(document.getElementById('item'), isActive, {
  type: 'class',
  className: 'active'
});
```

### bindTwoWay()

Creates a two-way binding between an input element and a rune.

```typescript
import { bindTwoWay, rune } from '/_gospa/runtime.js';

const name = rune('');

// Two-way binding with input
const input = document.querySelector('input[name="name"]');
bindTwoWay(input, name);

// Now name.get() reflects input value
// And input.value updates when name.set() is called
```

## Conditional Rendering

Render elements conditionally based on rune state.

```typescript
import { renderIf, rune } from '/_gospa/runtime.js';

const isLoggedIn = rune(false);

// Show element only when condition is true
renderIf(document.getElementById('admin-panel'), isLoggedIn);

// With inverse condition
renderIf(document.getElementById('login-form'), isLoggedIn, { inverse: true });
```

## List Rendering

Render lists from reactive arrays with efficient DOM updates.

```typescript
import { renderList, rune } from '/_gospa/runtime.js';

interface Todo {
  id: number;
  text: string;
}

const todos = rune<Todo[]>([]);

// Render list with key-based reconciliation
renderList(
  document.getElementById('todo-list'),
  todos,
  {
    key: (todo) => todo.id,
    render: (todo, index) => {
      const li = document.createElement('li');
      li.textContent = todo.text;
      return li;
    }
  }
);
```

## Sanitization

Configure HTML sanitization for safe innerHTML bindings.

```typescript
import { setSanitizer, getSanitizer } from '/_gospa/runtime.js';

// Set custom sanitizer
setSanitizer((html: string) => {
  // Your sanitization logic
  return sanitizedHtml;
});

// Get current sanitizer
const sanitize = getSanitizer();
```

## Binding Registry

Internal binding management for cleanup and debugging.

```typescript
import { registerBinding, unregisterBinding } from '/_gospa/runtime.js';

// Register a custom binding
const bindingId = registerBinding(element, rune, config);

// Unregister when cleaning up
unregisterBinding(bindingId);
```
