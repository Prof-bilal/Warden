# Deploying the docs to GitHub Pages

This project uses MkDocs with the Material theme for documentation generation.
Two deployment options are described below.

## Prerequisites

- A GitHub repository with the docs pushed.
- A GitHub account with permissions to create Actions and enable Pages.

## Option 1: MkDocs Material with GitHub Actions (recommended)

### Install dependencies locally

```bash
pip install mkdocs-material
```

### Create `mkdocs.yml` at the project root

```yaml
site_name: Warden
theme:
  name: material
  palette:
    primary: blue
  icon:
    repo: fontawesome/brands/github
repo_url: https://github.com/<owner>/warden
repo_name: warden

nav:
  - Home: index.md
  - Schema Reference: schema.md
  - Security Review: security.md
  - Architecture: architecture.md
  - Examples: ../examples/policy.example.yaml

markdown_extensions:
  - toc:
      permalink: true
  - pymdownx.highlight:
      anchor_linenums: true
  - pymdownx.inlinehilite
  - pymdownx.snippets
  - pymdownx.superfences
```

### Create the GitHub Actions workflow

Create `.github/workflows/docs.yml`:

```yaml
name: Deploy docs to GitHub Pages

on:
  push:
    branches:
      - main
    paths:
      - "docs/**"
      - "mkdocs.yml"
      - ".github/workflows/docs.yml"

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: pages
  cancel-in-progress: false

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: 3.x
      - run: pip install mkdocs-material
      - run: mkdocs build --site-dir docs/site
      - uses: actions/upload-pages-artifact@v3
        with:
          path: docs/site

  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - id: deployment
        uses: actions/deploy-pages@v4
```

### Enable GitHub Pages

In your repository settings, go to **Pages** and select:
- **Source:** GitHub Actions
- **Branch:** `main` (or your default branch)

### Local build test

```bash
mkdocs build
mkdocs serve   # preview at http://localhost:8000
```

## Option 2: Static generation with Jekyll

If you prefer not to use MkDocs, GitHub Pages supports Jekyll out of the box.

### Add a `_config.yml` at the project root:

```yaml
title: Warden
description: A lightweight sandbox runtime for MCP servers
remote_theme: just-the-docs/just-the-docs
plugins:
  - jekyll-remote-theme
```

### Push directly to `main`

GitHub Pages will build the site automatically from the `docs/` folder.
Add a `README.md` at the repo root linking to `docs/index.md`.

## Comparison

| | MkDocs Material | Jekyll + just-the-docs |
|---|---|---|
| Setup | Install Python + pip | Push and let GitHub build |
| Customisation | High (themes, plugins) | Moderate |
| Build speed | Fast | Fast |
| Source control | Any branch | Must be on default branch |

## Notes

- The `docs/` directory is separate from the MkDocs `docs/` convention — adjust
  the `docs_dir` setting in `mkdocs.yml` if you place source files elsewhere.
- All internal links in the docs use relative paths (e.g., `[Schema Reference](schema.md)`)
  so they work both locally and on the published site.
