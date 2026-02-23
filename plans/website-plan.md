# GoSPA Website Architecture Plan

## Overview

Build a documentation website for GoSPA framework **using GoSPA itself** (dogfooding). The site will showcase the framework's capabilities while providing comprehensive documentation.

## Deep Reasoning Chain

### Psychological Analysis
- **Target Audience**: Go developers seeking reactive SPA solutions, likely coming from JavaScript frameworks
- **Cognitive Load**: Developers want quick answers - code samples first, explanations second
- **Trust Signals**: Performance benchmarks, type safety, comparison table build credibility
- **Differentiation**: "Svelte-like reactivity in Go" is the key hook - must be prominent

### Technical Analysis
- **Framework**: GoSPA with Fiber + Templ (the framework documenting itself)
- **Rendering**: SSR for SEO, client-side hydration for interactivity
- **Performance**: Sub-15KB runtime aligns with "lightweight" positioning
- **State Management**: Use GoSPA's own Rune/Derived/Effect primitives for interactive components

### Accessibility Analysis
- Semantic HTML via Templ components
- Keyboard navigation for sidebar and code examples
- High contrast dark theme (Go/terminal aesthetic)
- Skip links and proper heading hierarchy

### Scalability Analysis
- File-based routing scales naturally with docs growth
- Component-based architecture for reusable UI elements
- Markdown content can be migrated to CMS later if needed

---

## Project Structure

```
website/
├── main.go                    # GoSPA app entry point
├── go.mod                     # Go module
├── go.sum
├── routes/                    # File-based routing
│   ├── root_layout.templ      # Base HTML shell with nav
│   ├── page.templ             # Homepage
│   ├── docs/
│   │   ├── layout.templ       # Docs layout with sidebar
│   │   ├── page.templ         # /docs redirect or overview
│   │   ├── get-started/
│   │   │   └── page.templ     # Getting started guide
│   │   ├── reactive-primitives/
│   │   │   └── page.templ     # Rune, Derived, Effect docs
│   │   ├── routing/
│   │   │   └── page.templ     # File-based routing docs
│   │   ├── state-management/
│   │   │   └── page.templ     # State sync, sessions
│   │   ├── websocket/
│   │   │   └── page.templ     # WebSocket integration
│   │   ├── remote-actions/
│   │   │   └── page.templ     # Remote actions docs
│   │   ├── security/
│   │   │   └── page.templ     # CSRF, CORS, XSS
│   │   ├── client-runtime/
│   │   │   └── page.templ     # JS API reference
│   │   └── api/
│   │       └── page.templ     # Go API reference
│   └── components/            # Shared Templ components
│       ├── sidebar.templ
│       ├── code_block.templ
│       ├── benchmark_card.templ
│       └── nav.templ
├── static/
│   ├── css/
│   │   └── styles.css         # Tailwind v4 via CDN or custom
│   └── js/                    # Any additional client JS
└── content/                   # Markdown content (optional)
```

---

## Routing Architecture

```mermaid
graph TD
    A[/] --> B[Homepage]
    A --> C[/docs]
    C --> D[/docs/get-started]
    C --> E[/docs/reactive-primitives]
    C --> F[/docs/routing]
    C --> G[/docs/state-management]
    C --> H[/docs/websocket]
    C --> I[/docs/remote-actions]
    C --> J[/docs/security]
    C --> K[/docs/client-runtime]
    C --> L[/docs/api]
```

---

## Homepage Design

### Hero Section
- **Headline**: "Svelte-like Reactivity for Go"
- **Subheadline**: "Build reactive SPAs with server-side rendering. Type-safe. Lightweight. Fast."
- **CTA Buttons**: "Get Started" | "View on GitHub"
- **Visual**: Animated code comparison (Go + Templ vs React/Svelte)

### Benchmark Visualization Section
Transform the benchmark data into an interactive, visually compelling display:

```
┌─────────────────────────────────────────────────────────────────┐
│  PERFORMANCE BENCHMARK                                           │
│  ─────────────────────────────────────────────────────────────── │
│                                                                  │
│  Peak RPS: 24,345                                               │
│  ████████████████████████████████████████  24k                  │
│                                                                  │
│  Latency @ 150 concurrent: 6.03ms avg                           │
│  ▓▓▓▓▓▓░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  6ms                  │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │  RPS vs Concurrency                                       │   │
│  │                                                           │   │
│  │  25k ┤                                    ●──●──●         │   │
│  │  20k ┤                           ●──●──●                  │   │
│  │  15k ┤                    ●──●                            │   │
│  │  10k ┤              ●──●                                  │   │
│  │   5k ┤        ●──●                                        │   │
│  │      └─────────────────────────────────────────────────  │   │
│  │         5    10    25    50   75   100  150  200         │   │
│  │                     Concurrency                          │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ✓ 100% Success Rate  ✓ 307,600 Total Requests  ✓ 14.6s Total  │
└─────────────────────────────────────────────────────────────────┘
```

**Interactive Elements**:
- Hover over data points for exact values
- Animated chart on scroll into view
- Toggle between RPS and Latency views

### Features Grid
Six feature cards with icons:
1. **Reactive Primitives** - Rune, Derived, Effect
2. **File-Based Routing** - SvelteKit-style
3. **WebSocket Sync** - Real-time state
4. **Type Safety** - Compile-time validation
5. **Lightweight Runtime** - <15KB gzipped
6. **Security Built-in** - CSRF, CORS, XSS protection

### Comparison Table
| Feature | GoSPA | HTMX | Alpine | SvelteKit |
|---------|-------|------|--------|-----------|
| Language | Go | HTML | JS | JS/TS |
| Runtime Size | <15KB | ~14KB | ~15KB | Varies |
| SSR | ✅ | ✅ | ❌ | ✅ |
| Type Safety | ✅ | ❌ | ❌ | ✅ |
| WebSocket | ✅ | ❌ | ❌ | ✅ |
| File Routing | ✅ | ❌ | ❌ | ✅ |
| Reactivity | ✅ | ❌ | ✅ | ✅ |

### Code Example Section
Show a complete counter example with syntax highlighting:
- Server-side Go code
- Templ template
- Client-side reactivity

---

## Documentation Section Design

### Sidebar Navigation
```
docs/
├── Getting Started
│   └── Quick Start
├── Core Concepts
│   ├── Reactive Primitives
│   ├── Routing
│   └── State Management
├── Features
│   ├── WebSocket Integration
│   ├── Remote Actions
│   └── Security
└── Reference
    ├── Client Runtime API
    └── Go API
```

### Page Layout
```
┌─────────────────────────────────────────────────────────────────┐
│  HEADER: Logo | Search | GitHub | Theme Toggle                  │
├────────────┬────────────────────────────────────────────────────┤
│            │                                                    │
│  SIDEBAR   │   MAIN CONTENT                                     │
│            │                                                    │
│  ▸ Getting │   # Page Title                                     │
│    Started │   ─────────────────────────────────────            │
│            │                                                    │
│  ▸ Core    │   Introduction paragraph...                        │
│    Concepts│                                                    │
│            │   ## Section                                       │
│    ▸ Primit│   ────────────────                                 │
│    ▸ Routin│                                                    │
│    ▸ State │   ```go                                            │
│            │   // Code example                                  │
│  ▸ Features│   ```                                              │
│    ▸ WebSoc│                                                    │
│    ▸ Remote│   > Note: Important information                    │
│    ▸ Securi│                                                    │
│            │   ## Next Section                                  │
│  ▸ Referenc│   ...                                              │
│    ▸ Client│                                                    │
│    ▸ Go API│   ─────────────────────────────────────            │
│            │   ← Previous Page    Next Page →                   │
└────────────┴────────────────────────────────────────────────────┘
```

---

## Components to Build

### Go/Templ Components

| Component | Purpose |
|-----------|---------|
| `Sidebar` | Collapsible navigation with active state |
| `CodeBlock` | Syntax-highlighted code with copy button |
| `BenchmarkChart` | Interactive SVG/Canvas chart |
| `FeatureCard` | Icon + title + description |
| `ComparisonTable` | Feature comparison matrix |
| `Nav` | Top navigation with mobile menu |
| `Footer` | Links and copyright |
| `Breadcrumbs` | Docs navigation path |
| `Pagination` | Previous/Next page links |
| `SearchModal` | Command+K search (future) |
| `ThemeProvider` | Dark/light mode toggle |

### Client-Side Interactive Components

| Component | Purpose | GoSPA Feature Used |
|-----------|---------|-------------------|
| `BenchmarkViz` | Animated benchmark charts | Rune for state, Effect for animations |
| `CodeTabs` | Tabbed code examples | Rune for active tab |
| `ThemeToggle` | Dark/light mode | Local state (data-gospa-local) |
| `SidebarCollapse` | Expand/collapse sections | Rune for expanded state |
| `CopyButton` | Copy code to clipboard | Effect for clipboard API |

---

## Benchmark Data Structure

```go
type BenchmarkData struct {
    Tests []BenchmarkTest
    Summary BenchmarkSummary
}

type BenchmarkTest struct {
    Name        string
    Concurrency int
    Requests    int
    RPS         float64
    AvgLatency  string
    P95Latency  string
    SuccessRate float64
}

type BenchmarkSummary struct {
    PeakRPS          float64
    PeakConcurrency  int
    BestLatency      string
    BestLatencyConc  int
    TotalRequests    int
    TotalDuration    string
    SuccessRate      float64
}
```

---

## Visual Design Direction

### Aesthetic: **Terminal-Inspired Modern Dark**

**Rationale**: Go developers often work in terminals. A dark, terminal-inspired aesthetic feels native while remaining modern and readable.

**Typography**:
- Display: JetBrains Mono or Fira Code (monospace for code feel)
- Body: Inter or system-ui (readable for long docs)

**Color Palette**:
```css
:root {
  --bg-primary: #0a0a0f;      /* Deep black */
  --bg-secondary: #12121a;    /* Slightly lighter */
  --bg-tertiary: #1a1a24;     /* Cards, code blocks */
  --text-primary: #e4e4e7;    /* Off-white */
  --text-secondary: #a1a1aa;  /* Muted */
  --text-muted: #71717a;      /* Very muted */
  --accent-primary: #22d3ee;  /* Cyan - Go-ish */
  --accent-secondary: #a78bfa;/* Purple - secondary */
  --accent-success: #22c55e;  /* Green - success */
  --accent-warning: #f59e0b;  /* Amber - warning */
  --border: #27272a;          /* Subtle borders */
}
```

**Visual Elements**:
- Subtle grid pattern background (like graph paper)
- Glowing accent borders on interactive elements
- Code blocks with line numbers and syntax highlighting
- Smooth transitions between pages (SPA navigation)

---

## Implementation Phases

### Phase 1: Foundation
1. Initialize GoSPA project in `website/`
2. Create root layout with navigation
3. Set up Tailwind CSS v4 via CDN
4. Create homepage hero section

### Phase 2: Homepage
1. Build benchmark visualization component
2. Create features grid
3. Add comparison table
4. Implement code example section

### Phase 3: Docs Infrastructure
1. Create docs layout with sidebar
2. Implement sidebar navigation
3. Add breadcrumbs and pagination
4. Create code block component with copy

### Phase 4: Documentation Pages
1. Write Getting Started guide
2. Create Reactive Primitives docs
3. Create Routing docs
4. Create remaining feature docs
5. Create API reference pages

### Phase 5: Polish
1. Add theme toggle (dark/light)
2. Implement smooth page transitions
3. Add search functionality (optional)
4. Optimize for SEO

---

## Technical Considerations

### SEO
- Use semantic HTML in Templ components
- Add meta tags in root layout
- Create sitemap endpoint
- Use proper heading hierarchy

### Performance
- Leverage GoSPA's SSR for fast initial load
- Use client-side navigation for SPA feel
- Lazy load interactive components
- Optimize images and assets

### Accessibility
- ARIA labels for interactive elements
- Keyboard navigation support
- Skip links for main content
- Proper focus management

---

## File Dependencies

The website will need to reference:
- `../docs/API.md` - Source content for API reference
- `../tests/benchmark.txt` - Performance data for visualization
- `../README.md` - Feature descriptions and comparison table

---

## Next Steps

1. **Approve this plan** - Review and confirm architecture
2. **Switch to Code mode** - Begin implementation
3. **Start with Phase 1** - Foundation setup
