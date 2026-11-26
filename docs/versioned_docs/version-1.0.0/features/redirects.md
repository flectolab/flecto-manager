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
   - **Query string** options, for `BASIC` and `BASIC_HOST` (see below)

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

:::warning[Host Types (BASIC_HOST, REGEX_HOST)]

For host-based redirect types, the following rules apply:

- The **source** must **not** contain the scheme (`http://` or `https://`). Only specify the host and path (e.g. `example.com/path`).
- The **target** **must** contain the scheme, and therefore the full domain (e.g. `https://example.com/path`).

:::

## Query String Options

By default a redirect compares the requested URL **with its query string**: a `BASIC`
redirect from `/old-page` does not apply to `/old-page?utm_source=newsletter`. Two
options, available on `BASIC` and `BASIC_HOST` redirects, change that. Both are off
by default, so a redirect without options behaves as it always did.

| Option | Effect |
|--------|--------|
| **Ignore query string** (`ignoreQueryString`) | The redirect applies whatever the query string of the request is |
| **Preserve query string** (`preserveQueryString`) | The query string of the request is forwarded to the target, to keep campaign tracking for example |

```
Type:    BASIC
Source:  /old-page
Target:  /new-page
Options: ignore query string, preserve query string
```

- Request: `GET /old-page` → Redirects to `/new-page`
- Request: `GET /old-page?utm_source=newsletter` → Redirects to `/new-page?utm_source=newsletter`

When the target already has a query string, it is kept as written and the parameters
of the request are appended after it, except those the target already has:
`/new-page?lang=fr` with a request carrying `?lang=en&utm_source=x` redirects to
`/new-page?lang=fr&utm_source=x`. The query string is inserted before the fragment of the
target (`/new-page#top` gives `/new-page?utm_source=x#top`).

Rules:

- The source of a redirect ignoring the query string must not contain one.
- A redirect with the exact query string of the request still wins: `/old-page?x=1`
  (without options) is preferred over `/old-page` ignoring the query string for a
  request to `/old-page?x=1`.
- Regex types (`REGEX`, `REGEX_HOST`) do not take these options: the pattern itself
  decides what to do with the query string.
- With a CDN in front of the agents, publishing a redirect ignoring the query string
  purges it by prefix, see [CDN cache purge](cdn.md#what-gets-purged).

Agents need a client built with this feature to apply the options. An older agent
ignores them and compares the URL with its query string, as before.

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
type    source    target    status    options
```

The `options` column is optional: a file with the first four columns only is still
accepted, and its rows keep the options of the redirects they overwrite (dropped if the
row turns the redirect into a regex type). With the column, a row may leave the cell
empty, which clears the options.

**Example content:**

| type | source | target | status | options |
|------|--------|--------|--------|---------|
| BASIC | /old-page | /new-page | MOVED_PERMANENT | ignoreQueryString=true,preserveQueryString=true |
| BASIC | /about | /about-us | 301 | |
| REGEX | ^/blog/(.*)$ | /articles/$1 | MOVED_PERMANENT | |
| BASIC_HOST | old.example.com/shop | https://shop.example.com | FOUND | ignoreQueryString=true |

### Columns

| Column | Required | Values |
|--------|----------|--------|
| `type` | Yes | `BASIC`, `BASIC_HOST`, `REGEX`, `REGEX_HOST` |
| `source` | Yes | Path or regex pattern |
| `target` | Yes | Target URL or path |
| `status` | Yes | `MOVED_PERMANENT`, `FOUND`, `TEMPORARY_REDIRECT`, `PERMANENT_REDIRECT` or `301`, `302`, `307`, `308` |
| `options` | No | Comma separated `key=value`: `ignoreQueryString`, `preserveQueryString`, with `true` or `false`. A key left out is not set |

The export writes the same five columns, so an exported file can be imported back.
An option explicitly set to `false` is exported as such and differs from one not
set: importing it over a redirect without that option creates a draft.

### Import Options

- **Overwrite**: If enabled, existing redirects with the same source will be updated

### Limits

- Maximum file size: **200MB** (roughly 1.2 million rows)
- The file is streamed and imported in batches, so a large import runs as a single
  long request. Expect around **7,000 rows per second**; a 700,000 row file takes
  close to two minutes.
- The whole import runs in one transaction: if it fails, nothing is imported.
- At most **1,000 errors** are returned in detail. The reported error count is always
  the exact total, so a larger count than the listed errors means the list was truncated.

## Priority

When multiple redirects could match a path, they are evaluated in order:

1. Exact matches, query string included: `BASIC_HOST`, then `BASIC`
2. `BASIC_HOST`, then `BASIC` redirects ignoring the query string, on the path alone
3. Then regex matches (`REGEX_HOST`, then `REGEX`)

Within regex matches, longer patterns take priority.
