---
sidebar_position: 1
slug: /
---

# Introduction

**Flecto Manager** is a centralized management platform for HTTP redirections and static pages like `robots.txt` and `sitemap.xml`.

## Why Flecto Manager?

Managing HTTP redirections across multiple domains and environments can quickly become complex:

- Scattered configurations across different servers
- No version control or audit trail
- Difficult to coordinate changes across teams
- No easy way to test changes before deployment

Flecto Manager solves these problems by providing:

- **Centralized configuration** - Manage all redirections from a single dashboard
- **Draft system** - Validate changes before publishing to production
- **Distributed agents** - Deploy lightweight agents that sync automatically
- **Role-based access** - Control who can view and modify configurations

## Architecture

```
┌─────────────────┐     ┌─────────────────┐
│   Flecto        │     │   Flecto        │
│   Manager       │────▶│   Agent         │────▶ Users
│   (Central)     │     │   (Edge)        │
└─────────────────┘     └─────────────────┘
        │                       │
        │                       │
        ▼                       ▼
┌─────────────────┐     ┌─────────────────┐
│   Database      │     │   Local Cache   │
│   (MySql)       │     │                 │
└─────────────────┘     └─────────────────┘
```

The **Manager** is the central component where you configure redirections and pages. **Agents** are lightweight processes that sync configurations from the Manager and serve requests.

## Next Steps

- [Getting Started](./getting-started) - Install and run Flecto Manager
- [Configuration](./configuration) - Configure the Manager and Agents
- [Features](./features/redirects) - Learn about redirections, pages, and agents
