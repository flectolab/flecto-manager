---
sidebar_position: 1
---

# Redirects

Manage HTTP redirections from the Manager dashboard or API.

## Creating a Redirect

1. Navigate to your project
2. Click **Redirects** in the sidebar
3. Click **Add Redirect**
4. Configure the redirect:
   - **Type**: The redirect type (see below)
   - **Source**: The path or pattern to match
   - **Target**: Where to redirect
   - **Status**: HTTP status code

## Redirect Types

### BASIC

Exact path matching. The source path must match exactly.

```
Type:   BASIC
Source: /old-page
Target: /new-page
Status: MOVED_PERMANENT (301)
```

- Request: `GET /old-page` → Redirects to `/new-page`
- Request: `GET /old-page/sub` → No match

### BASIC_HOST

Exact path matching with host detection. The source must include the host to match against.

```
Type:   BASIC_HOST
Source: example.com/old-page
Target: https://new-domain.com/new-page
Status: MOVED_PERMANENT (301)
```

- Request: `GET example.com/old-page` → Redirects to `https://new-domain.com/new-page`
- Request: `GET other.com/old-page` → No match (different host)

### REGEX

Regular expression matching on the path. Capture groups can be used in the target.

```
Type:   REGEX
Source: ^/blog/([0-9]+)/(.*)$
Target: /articles/$1/$2
Status: MOVED_PERMANENT (301)
```

- Request: `GET /blog/123/my-post` → Redirects to `/articles/123/my-post`
- Request: `GET /blog/456/another` → Redirects to `/articles/456/another`

### REGEX_HOST

Regular expression matching with host detection. The source must include the host pattern.

```
Type:   REGEX_HOST
Source: ^shop\.example\.com/products/([a-z]+)/([0-9]+)$
Target: https://newshop.example.com/$1/item/$2
Status: FOUND (302)
```

- Request: `GET shop.example.com/products/shoes/42` → Redirects to `https://newshop.example.com/shoes/item/42`
- Request: `GET other.com/products/shoes/42` → No match (different host)

## HTTP Status Codes

| Status | Code | Description |
|--------|------|-------------|
| `MOVED_PERMANENT` | 301 | Permanent redirect, cached by browsers |
| `FOUND` | 302 | Temporary redirect, not cached |
| `TEMPORARY_REDIRECT` | 307 | Temporary redirect, preserves HTTP method |
| `PERMANENT_REDIRECT` | 308 | Permanent redirect, preserves HTTP method |

## Draft System

Redirects support a draft workflow:

1. **Create Draft** - Changes are saved as drafts
2. **Review** - Preview changes before publishing
3. **Publish** - Apply changes to production

This allows you to prepare multiple changes and publish them together.

## Bulk Import

Import redirects from a TSV (tab-separated values) file.

### File Format

The file must be a `.csv` or `.tsv` file with **tab-separated** columns.

**Header row (required):**
```
type    source    target    status
```

**Example content:**

| type | source | target | status |
|------|--------|--------|--------|
| BASIC | /old-page | /new-page | MOVED_PERMANENT |
| BASIC | /about | /about-us | 301 |
| REGEX | ^/blog/(.*)$ | /articles/$1 | MOVED_PERMANENT |
| BASIC_HOST | old.example.com/shop | https://shop.example.com | FOUND |

### Columns

| Column | Required | Values |
|--------|----------|--------|
| `type` | Yes | `BASIC`, `BASIC_HOST`, `REGEX`, `REGEX_HOST` |
| `source` | Yes | Path or regex pattern |
| `target` | Yes | Target URL or path |
| `status` | Yes | `MOVED_PERMANENT`, `FOUND`, `TEMPORARY_REDIRECT`, `PERMANENT_REDIRECT` or `301`, `302`, `307`, `308` |

### Import Options

- **Overwrite**: If enabled, existing redirects with the same source will be updated

## Priority

When multiple redirects could match a path, they are evaluated in order:

1. Exact matches (`BASIC`, `BASIC_HOST`) first
2. Then regex matches (`REGEX`, `REGEX_HOST`)

Within each category, longer/more specific patterns take priority.
