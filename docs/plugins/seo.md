# SEO Optimization Plugin

Generate SEO assets including sitemap, meta tags, and structured data with built-in XSS protection.

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

## Security
The plugin automatically HTML-escapes all metadata including titles, descriptions, and canonical URLs to prevent Cross-Site Scripting (XSS).

## CLI Commands

| Command | Alias | Description |
|---------|-------|-------------|
| `seo:generate` | `sg` | Generate sitemap and robots.txt |
| `seo:meta` | `sm` | Generate meta tags |
| `seo:structured` | `ss` | Generate JSON-LD |
