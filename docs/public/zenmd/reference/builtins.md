# Built-in plugins and layouts

What every build of zenmd includes. `zenmd version --verbose` lists what a
given build has.

## Plugins

Each plugin has a page with its names, options and an example; the list is in
[Plugins](plugins/index.md).

| Plugin | What it does | Written as | MyST names |
|---|---|---|---|
| [`codeblock`](plugins/codeblock.md) | Colours fenced code, with optional line numbers and wrapping. | a fence with a language, such as `go` | `code-block`, `code` |
| [`diagram`](plugins/diagram.md) | Draws Mermaid diagrams. | a `mermaid` fence | `mermaid` |
| [`check`](plugins/check.md) | Checklists that remember what you ticked. | `- [ ] item` in `:::check` | `check` |
| [`quiz`](plugins/quiz.md) | Questions with answers and feedback. | `:::quiz` | `quiz` |
| [`sandbox`](plugins/sandbox.md) | Code you can run, in a container. | `:::sandbox` | `sandbox` |
| [`rating`](plugins/rating.md) | One to five stars (or up to ten), kept on this Mac. | `:::rating` | `rating` |
| [`callout`](plugins/callout.md) | Notes, tips, warnings and version notes, which can fold. | `:::note` | `note`, `tip`, `hint`, `important`, `seealso`, `warning`, `caution`, `attention`, `danger`, `error`, `admonition`, `versionadded`, `versionchanged`, `deprecated` |
| [`figure`](plugins/figure.md) | A picture with a numbered caption, or the picture alone. | `:::figure` | `figure`, `image` |
| [`table`](plugins/table.md) | A table with a numbered caption, from markdown, a list or CSV. | `:::table` | `table`, `list-table`, `csv-table` |
| [`math`](plugins/math.md) | Display and inline equations, drawn with KaTeX. | `:::math`, `$$` | `math`, roles `{math}`, `{eq}` |
| [`tabs`](plugins/tabs.md) | Tabs, one panel for each item. | `:::tab-set` | `tab-set`, `tab-item` |
| [`dropdown`](plugins/dropdown.md) | A section that folds. | `:::dropdown` | `dropdown` |
| [`cards`](plugins/cards.md) | Cards, grids of cards, link buttons and badges. | `:::card` | `card`, `grid`, `grid-item`, `grid-item-card`, `button-link`, `button-ref`, roles `{bdg-…}` |
| [`xref`](plugins/xref.md) | Links to labels, figures, pages and files, and a list of pages. | `:ref[…]` | roles `{ref}`, `{numref}`, `{doc}`, `{download}`, directive `toctree` |
| [`text`](plugins/text.md) | Small marks in a line: keys, abbreviations, subscripts and superscripts. | `:kbd[Ctrl+C]` | roles `{kbd}`, `{abbr}`, `{sub}`, `{sup}` |
| [`quote`](plugins/quote.md) | Epigraphs, pull quotes and highlights. | `:::epigraph` | `epigraph`, `pull-quote`, `highlights` |
| [`aside`](plugins/aside.md) | Text set beside the page: sidebars, margin notes and topics. | `:::sidebar` | `sidebar`, `margin`, `topic` |
| [`include`](plugins/include.md) | Another file of the folder, as markdown or as code. | `:::include` | `include`, `literalinclude` |
| [`terms`](plugins/terms.md) | A glossary and links to its terms. | `:::glossary` | `glossary`, role `{term}` |

## Layouts

| Layout | For |
|---|---|
| `article` | The reading view, the default. |
| `docs` | Documentation, with a contents rail and previous and next links. |
| `post` | Blog posts, with a cover, date, author and tags. |
| `landing` | A folder's home, with cards for its pages. |
| `wide` | The whole pane, for wide tables and diagrams. |
| `handout` | A one-pager on a paper sheet. |
| `slides` | A page shown as slides. |
| `course` | Steps with progress, for a course. |
| `term` | One glossary term. |
| `glossary` | A glossary index, A to Z, built from the term pages. |
| `release` | One release; its title is its version. |
| `troubleshooting` | One problem and its fix, with the error message under the title. |

## Dialects

| Dialect | What it adds |
|---|---|
| CommonMark and GFM | Always on: tables, task lists, strikethrough, autolinks. |
| `myst` | MyST directives and roles, when a folder turns it on. |
