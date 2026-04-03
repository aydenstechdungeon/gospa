# PostCSS Plugin

PostCSS processing with Tailwind CSS v4 integration and additional plugins.

## Installation

```bash
gospa add:postcss
```

## Configuration

```yaml
plugins:
  postcss:
    input: ./styles/main.css
    output: ./static/css/main.css
    plugins:
      typography: true
      forms: true
      autoprefixer: true
```

## Critical CSS

The PostCSS plugin supports critical CSS extraction to improve page load performance:

```yaml
plugins:
  postcss:
    criticalCSS:
      enabled: true
      criticalOutput: ./static/css/critical.css
      nonCriticalOutput: ./static/css/non-critical.css
```

### Usage in Layout

```templ
@templ.Raw("<style>" + postcss.CriticalCSS("./static/css/critical.css") + "</style>")
@templ.Raw(postcss.AsyncCSS("/static/css/non-critical.css"))
```

## Bundle Splitting

Split CSS into separate bundles for multi-page applications:

```yaml
plugins:
  postcss:
    bundles:
      - name: marketing
        input: ./styles/marketing.css
        content: [./routes/marketing/**/*.templ]
```
