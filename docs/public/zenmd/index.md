# zenmd

zenmd is the engine behind Zenmark: it turns a folder of markdown files into
pages, a website or one printed document. It reads the folder as it is, with
nothing imported or uploaded, and builds the sidebar from the links between
pages. It comes as a command line tool, `zenmd`, and as Go packages, and
Zenmark, the macOS app, is built on it. These pages are for writers who keep
docs in markdown and for developers who build on the engine.

## Key features

- **A folder is a site.** Pages render with frontmatter, plugins and layouts, and the sidebar follows your links.
- **Export without a build step.** `zenmd export site` writes a static website that opens from disk; `zenmd export pdf` writes the whole folder as one PDF.
- **Plugins and layouts.** Diagrams, checklists, quizzes and runnable code blocks; layouts for docs, posts, slides, courses and glossaries.
- **Blueprints and page templates.** Start a folder from a blueprint such as Technical documentation, and every new page from the right page template.
- **More than one markdown.** CommonMark and GFM always, and MyST when a folder turns it on.

> **Missing:** a screenshot of Zenmark showing a folder, and one of an exported site.

## Where to go next

- [Getting started](getting-started/index.md)
- [How-tos](how-tos/index.md)
- [Explanation](explanation/index.md)
- [Reference](reference/index.md)
- [Releases](releases/index.md)
- [Glossary](glossary/index.md)
- [Contributing to these docs](CONTRIBUTING.md)
