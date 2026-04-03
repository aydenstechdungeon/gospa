# Transitions

Animate elements as they enter or leave the DOM with built-in or custom transitions.

## Built-in Transitions

GoSPA provides several built-in transition functions.

| Function | Description | Example |
|----------|-------------|---------|
| `fade` | Smoothly transitions opacity from 0 to 1 | `fade(el, { duration: 400 })` |
| `fly` | Animates position (x, y) and opacity | `fly(el, { x: 100, y: 0 })` |
| `slide` | Slides the element vertically | `slide(el, { duration: 400 })` |
| `scale` | Scales the element from a starting point | `scale(el, { start: 0 })` |
| `blur` | Blurs the element in/out | `blur(el, { amount: 5 })` |

## Easing Functions

Control the timing curve of transitions. Available: `linear`, `cubicOut`, `cubicInOut`, `elasticOut`, `bounceOut`, etc.

```typescript
import { fly, cubicOut } from '@gospa/client';

fly(element, { 
  duration: 400, 
  easing: cubicOut 
});
```

## Attribute API

Use HTML attributes to declaratively apply transitions.

```html
<!-- Basic fade -->
<div data-transition="fade">I fade in/out</div>

<!-- Different in/out transitions -->
<div data-transition-in="fade" data-transition-out="slide">
  I fade in and slide out
</div>

<!-- With parameters -->
<div data-transition="fly" data-transition-params='{"x": 100, "duration": 500}'>
  I fly from the right
</div>
```

## Programmatic API

Control transitions programmatically with JavaScript.

```typescript
import { transitionIn, transitionOut, fade, fly } from '@gospa/client';

// Enter transition
transitionIn(element, fade, { duration: 300 });

// Exit transition with callback
transitionOut(element, fly, { x: -100 }, () => {
  element.remove();
});
```

## Custom Transitions

Create custom transition functions by returning an object with `css` or `tick` properties.

```typescript
function wiggle(element, { duration = 400, intensity = 10 }) {
  return {
    duration,
    css: (t) => {
      const x = Math.sin(t * Math.PI * 4) * intensity * (1 - t);
      return `transform: translateX(${x}px)`;
    }
  };
}
```
