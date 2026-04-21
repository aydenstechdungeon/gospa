# FAQ (Frequently Asked Questions)

## Core Concepts

### Is GoSPA a Single Page Application (SPA)?
Yes, GoSPA is a Single Page Application framework that uses file-based routing and real-time state synchronization to deliver an interactive user experience without full-page reloads.

### Does GoSPA support SSR?
Yes, Server-Side Rendering (SSR) is the default rendering strategy for all GoSPA routes. This ensures optimal performance and SEO out of the box.

### Can I use GoSPA with existing Go/Fiber projects?
Yes! GoSPA is designed to be integrated into any existing Go/Fiber application. You can use it for specific routes or as the foundation for your entire UI.

## Reactivity

### Is GoSPA's reactivity system similar to Svelte?
Yes, GoSPA's reactivity is inspired by Svelte 5's signal-based architecture (`$state`, `$derived`, `$effect`). It uses modern JavaScript Proxies to achieve fine-grained, auto-tracking reactivity.

### How do I prevent memory leaks?
GoSPA automatically manages reactive scopes for all island components. If you are building custom advanced logic, always use the `EffectScope` primitive to group and dispose of effects.

## Security

### Does GoSPA include CSRF protection?
Yes! GoSPA includes built-in CSRF protection that is compatible with both standard HTML forms and dynamic AJAX/Remote Action requests.

### Is GoSPA secure against XSS?
Yes, with the default runtime policy. Dynamic HTML bindings and stream HTML updates escape content by default, and raw HTML rendering requires an explicit trusted wrapper.
