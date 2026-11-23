# What's missing: the zenmd docs made from the techdocs blueprint

`docs/public/zenmd` was made with
`zenmd init --blueprint techdocs docs/public/zenmd --title zenmd --set product=zenmd`,
then filled in from the repository (README, CHANGELOG, docs/ENGINE.md, the
demo guide, `zenmd help`, `apps/zenmark/export-options.json`). This file lists
what is still missing: first the content, then what the blueprint itself
lacked while writing it.

## Content still missing (marked "Missing" in the pages)

| Page | What's missing |
|---|---|
| `index.md` | A screenshot of Zenmark with a folder open, and one of an exported site. |
| `getting-started/installation.md` | The public download page (zenmark.org), and an "Install command line tool" item in the app (planned, not built). |
| `explanation/architecture.md` | The Go API for embedding the engine (`engine/session`), with an example. |
| `explanation/plugin-trust.md` | The plugin author guide (docs/PLUGINS.md isn't published as user docs). |
| `reference/cli.md` | Exit codes, and which commands work without the app. |
| `reference/configuration.md` | Every plugin's own options in one table. |
| `reference/builtins.md` | One page per plugin and layout, as the demo guide has. |
| `how-tos/export-a-pdf.md` | Page size, margins and headers (not options yet). |
| `releases/index.md` | The next release (a lot has landed since 0.5.1). |

## What the blueprint lacked

1. **A place for images.** There's no `images/` folder and no convention for screenshots, so the home page can't show the app.
2. **The home's link to CONTRIBUTING.md** reads "CONTRIBUTING", the file name, not the page's H1 ("Contributing to zenmd"). I fixed it by hand.
3. **Example pages to delete.** `how-tos/example-task.md`, `glossary/example-term.md` and `releases/v0.1.0.md` (dated today) have to be removed and unlinked by hand. An option to make the sections without examples, or examples marked as such, would save that.
4. **Section indexes are link lists kept by hand.** Every new how-to needs a line in `how-tos/index.md`. The glossary already uses `nav: { children: folder }`; the other sections could do the same, with the index made from the folder.
5. **Releases from the changelog.** The three release pages were copied by hand from CHANGELOG.md. A `zenmd release import CHANGELOG.md` (one page per version) would keep them in step. There's also no page for the unreleased changes.
6. **Reference that writes itself.** The CLI and export tables were copied from `zenmd help` and `export-options.json`. The blueprint could generate them (`zenmd help --markdown`, the options table), so they never go stale.
7. **A "missing" marker.** I used a blockquote, "**Missing:** …". A `todo` block (a plugin or a MyST admonition) that Problems lists would let a writer find every gap.
8. **No developer section.** Diátaxis covers it under Reference, but an engine has an API too; a blueprint for a library might want an "API" section.
9. **No FAQ or troubleshooting section.** Troubleshooting lives at the end of each how-to, with nothing that collects it.

## A bug found on the way

`zenmd export site` warns on every page that the `sandbox` plugin's `images`
option "can only be set in the global config; ignored", although it *is* set
in the global `config.yml`. A user-scoped option in the global config should
be silent, or just say it isn't used by an export.
