# Configuration

The keys of `.zenmd.yml`, a folder's settings file. The same keys work in the
global `config.yml` in Zenmark's settings folder; a folder's file wins over
it, and a page's frontmatter wins over both.

## Folder keys

| Name | Type | Default | What it does |
|---|---|---|---|
| `version` | number | `1` | The file format version. |
| `logo` | path | none | An image in the folder, shown beside its title in the sidebar (`.svg`, `.png`, `.jpg`, `.gif`, `.webp`). |
| `layout` | layout id | `article` | The default layout of every page. |
| `plugins.<id>` | `true`, `false` or options | on | Turns a plugin on or off, or sets its options. |
| `markdown.dialects` | list | `[]` | Dialects read besides CommonMark and GFM, such as `[myst]`. |
| `templates` | map | none | Path rules: which page template new pages in a folder start from, such as `"how-tos/**": how-to`. |
| `export` | map | see [export options](export-options.md) | Website and PDF export settings. |
| `limits.files` | number | `2000` | Markdown files a folder may have, up to 100000. Global `config.yml` only. |
| `limits.entries` | number | `50000` | Files and folders scanned, up to 5000000. Global `config.yml` only. |
| `limits.openTimeout` | seconds | `30` | How long opening a folder may take, up to 600. Global `config.yml` only. |

## Page frontmatter

| Name | Type | What it does |
|---|---|---|
| `title` | text | The title, only for a page with no H1, such as slides. Otherwise the H1 is the title. |
| `slug` | words and `/` | A stable address for links: `slug: guides/install`. |
| `layout` | layout id | This page's layout. |
| `nav.title` | text | The page's name in the sidebar. |
| `nav.order` | number | Its place among its siblings. |
| `nav.parent` | path or slug | The page it sits under, in the place of that page's link to it (before `nav.order`). |
| `nav.hidden` | `true`/`false` | Leave it out of the sidebar. |
| `nav.children` | `folder` | List every page of the folder under this index. |
| `nav.sort` | `title` or `newest-first` | How those children are sorted. |
| `draft` | `true`/`false` | Leave it out of exports unless `--drafts` is given. |
| `template` | id or `false` | The page template it follows, or none. |
| `plugins.<id>` | `true`/`false`/options | A plugin for this page only. |

## Example

```yaml
version: 1
layout: docs
plugins:
  codeblock: true
  diagram: { theme: neutral }
markdown:
  dialects: [myst]
templates:
  "how-tos/**": how-to
export:
  out: exports/html
  theme: { scheme: light }
```

> **Missing:** every plugin's own options, such as `codeblock: { lineNumbers: true }`,
> in one table. They are spread over the plugin pages of the demo guide.
