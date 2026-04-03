# SEO Optimization Plugin

Generate SEO assets including sitemap, meta tags, and structured data.

## Installation

```bash
gospa add seo
```

## Configuration

```yaml
plugins:
  seo:
    site_url: https://example.com
    site_name: My GoSPA Site
    generate_sitemap: true
    generate_robots: true
```

## CLI Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `seo:generate` | `sg` | Generate sitemap and robots.txt |
| `seo:meta` | `sm` | Generate meta tags |
| `seo:structured` | `ss` | Generate JSON-LD |
