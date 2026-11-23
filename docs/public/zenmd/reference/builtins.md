# Built-in plugins and layouts

What every build of zenmd includes. `zenmd version --verbose` lists what a
given build has.

## Plugins

| Plugin | What it does | Written as |
|---|---|---|
| `codeblock` | Colours fenced code, with optional line numbers and wrapping. | a fence with a language, such as `go` |
| `diagram` | Draws Mermaid diagrams. | a `mermaid` fence |
| `check` | Checklists that remember what you ticked. | `- [ ] item` |
| `quiz` | Questions with answers and feedback. | `:::quiz` |
| `sandbox` | Code you can run, in a container. | `:::sandbox` |
| `rating` | One to five stars (or up to ten), kept on this Mac. | `:::rating` |

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

> **Missing:** a page per plugin and layout with its options and examples, as
> the demo guide has.
