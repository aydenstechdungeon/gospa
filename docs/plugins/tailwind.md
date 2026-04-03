# Tailwind CSS Plugin

Adds Tailwind CSS v4 support with CSS-first configuration, content scanning, and watch mode.

## Installation

```bash
gospa add:tailwind
```

## Configuration (`gospa.yaml`)

```yaml
plugins:
  tailwind:
    input: ./static/css/app.css
    output: ./static/dist/app.css
    content:
      - ./routes/**/*.templ
      - ./components/**/*.templ
    minify: true
```

## CLI Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `add:tailwind` | `at` | Install Tailwind deps and create starter files |
| `tailwind:build` | `tb` | Build CSS for production |
| `tailwind:watch` | `tw` | Watch and rebuild CSS on changes |

## Usage

1. Run `gospa add:tailwind`.
2. Edit `static/css/app.css` using Tailwind v4 `@theme` syntax.
3. The plugin automatically runs during `gospa dev` and `gospa build`.
