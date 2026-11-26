---
sidebar_position: 2
---

# Static Pages

Serve static files like `robots.txt`, `sitemap.xml`, and other text-based content.

## Creating a Page

1. Navigate to your project
2. Click **Pages** in the sidebar
3. Click **Add Page**
4. Configure the page:
   - **Type**: The page type (see below)
   - **Path**: The URL path (e.g., `/robots.txt`)
   - **Content Type**: The MIME type
   - **Content**: The file content

## Page Types

### BASIC

Exact path matching. The path must match exactly.

```
Type:         BASIC
Path:         /robots.txt
Content Type: TEXT_PLAIN
Content:      User-agent: *
              Allow: /
```

- Request: `GET /robots.txt` → Returns the content
- Request: `GET /other.txt` → No match

### BASIC_HOST

Exact path matching with host detection. The path must include the host to match against.

```
Type:         BASIC_HOST
Path:         example.com/robots.txt
Content Type: TEXT_PLAIN
Content:      User-agent: *
              Allow: /
```

- Request: `GET example.com/robots.txt` → Returns the content
- Request: `GET other.com/robots.txt` → No match (different host)

## Content Types

| Content Type | MIME Type | Description |
|--------------|-----------|-------------|
| `TEXT_PLAIN` | `text/plain` | Plain text files (robots.txt, .txt) |
| `XML` | `application/xml` | XML files (sitemap.xml, .xml) |

## Common Use Cases

### robots.txt

```
Type:         BASIC
Path:         /robots.txt
Content Type: TEXT_PLAIN
```

```text
User-agent: *
Allow: /

Sitemap: https://example.com/sitemap.xml
```

### sitemap.xml

```
Type:         BASIC
Path:         /sitemap.xml
Content Type: XML
```

```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/</loc>
    <lastmod>2024-01-01</lastmod>
    <priority>1.0</priority>
  </url>
</urlset>
```

### security.txt

```
Type:         BASIC
Path:         /.well-known/security.txt
Content Type: TEXT_PLAIN
```

```text
Contact: security@example.com
Expires: 2025-12-31T23:59:59.000Z
```

### Multi-host configuration

When you need different content per host:

```
Type:         BASIC_HOST
Path:         shop.example.com/robots.txt
Content Type: TEXT_PLAIN
```

```text
User-agent: *
Disallow: /checkout/
Disallow: /cart/
```

## Draft System

Like redirects, pages support drafts:

1. Edit the content
2. Save as draft
3. Preview changes
4. Publish when ready

## Content Limits

Default limits (configurable):

| Limit | Default |
|-------|---------|
| Max page size | 1 MB |
| Total size per project | 100 MB |
