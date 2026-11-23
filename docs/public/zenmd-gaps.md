# What's missing: the zenmd docs made from the techdocs blueprint

`docs/public/zenmd` was made with
`zenmd init --blueprint techdocs docs/public/zenmd --title zenmd --set product=zenmd`,
then filled in from the repository (README, CHANGELOG, docs/wiki/ENGINE.md, the
demo guide, `zenmd help`, `apps/zenmark/export-options.json`). This file lists
what is still missing: first the content, then what the blueprint itself
lacked while writing it. Each gap links the issue that tracks it (milestone 0.8.0);
the content is tracked as a whole in [#44](https://github.com/zenmark-org/zenmd/issues/44).

## Content still missing (marked "Missing" in the pages)

| Page | What's missing |
|---|---|
| `index.md` | A screenshot of Zenmark with a folder open, and one of an exported site. ([#36](https://github.com/zenmark-org/zenmd/issues/36)) |
| `getting-started/installation.md` | The public download page (zenmark.org), and an "Install command line tool" item in the app (planned, not built). ([#30](https://github.com/zenmark-org/zenmd/issues/30), [#43](https://github.com/zenmark-org/zenmd/issues/43)) |
| `explanation/architecture.md` | The Go API for embedding the engine (`engine/session`), with an example. |
| `explanation/plugin-trust.md` | The plugin author guide (docs/wiki/PLUGINS.md isn't published as user docs). |
| `reference/cli.md` | Exit codes, and which commands work without the app. ([#39](https://github.com/zenmark-org/zenmd/issues/39)) |
| `reference/configuration.md` | Every plugin's own options in one table. |
| `reference/builtins.md` | One page per plugin and layout, as the demo guide has. |
| `how-tos/export-a-pdf.md` | Page size, margins and headers (not options yet). ([#42](https://github.com/zenmark-org/zenmd/issues/42)) |

## What the blueprint lacked

1. **A place for images.** There's no `images/` folder and no convention for screenshots, so the home page can't show the app. Tracked in [#36](https://github.com/zenmark-org/zenmd/issues/36).
2. **A home link that used the file name.** The home linked CONTRIBUTING.md as "CONTRIBUTING", not by its H1. The page was placeholder text, so it is gone from techdocs (2.6.0) and from these docs; a real contributing guide is still to write.
3. **Example pages to delete.** `how-tos/example-task.md`, `glossary/example-term.md` and `releases/v0.1.0.md` (dated today) have to be removed and unlinked by hand. An option to make the sections without examples, or examples marked as such, would save that. Tracked in [#37](https://github.com/zenmark-org/zenmd/issues/37).
4. **Section indexes are link lists kept by hand.** Every new how-to needs a line in `how-tos/index.md`. The glossary already uses `nav: { children: folder }`; the other sections could do the same, with the index made from the folder. Tracked in the [blueprint-hierarchy spec](../../specs/engine/blueprint-hierarchy/spec.md).
5. **Releases from the changelog.** The three release pages were copied by hand from CHANGELOG.md. A `zenmd release import CHANGELOG.md` (one page per version) would keep them in step. There's also no page for the unreleased changes. Tracked in [#38](https://github.com/zenmark-org/zenmd/issues/38).
6. **Reference that writes itself.** The CLI and export tables were copied from `zenmd help` and `export-options.json`. The blueprint could generate them (`zenmd help --markdown`, the options table), so they never go stale. Tracked in [#39](https://github.com/zenmark-org/zenmd/issues/39).
7. **A "missing" marker.** I used a blockquote, "**Missing:** …". A `todo` block (a plugin or a MyST admonition) that Problems lists would let a writer find every gap. Tracked in [#40](https://github.com/zenmark-org/zenmd/issues/40).
8. **No developer section.** Diátaxis covers it under Reference, but an engine has an API too; a blueprint for a library might want an "API" section. Tracked in [#41](https://github.com/zenmark-org/zenmd/issues/41).
9. ~~**No troubleshooting section.**~~ Added in techdocs 2.6.0: a Troubleshooting section with its own page template and layout, one problem per page. These docs have four such pages.

## A bug found on the way

`zenmd export site` warned on every page that the `sandbox` plugin's `images`
option could only be set in the global config, although it was. Fixed in 0.7.0
([#28](https://github.com/zenmark-org/zenmd/issues/28)): the export notes it once.
