# Blueprints

A **blueprint** is a ready-made folder to start from: its sections, their
first pages and the rules that shape the pages you add later. Choose
**File › New from Blueprint…** (<kbd>⇧⌘N</kbd>), pick one, answer a few
questions (the title, the author) and pick an empty folder.

| Blueprint | Gives you |
|---|---|
| Technical documentation | An overview home page, Getting started, How-tos, Explanation, Reference, Glossary and Releases |
| Learning path | courses with steps, quizzes and checklists, and a glossary |
| Planner | a roadmap, meeting notes and decisions |

## Technical documentation

A fresh folder from this blueprint looks like this:

```text
index.md                  the home page: what the product is and who it's
                          for, key features, and a link to each section
presentation.md           the same, as slides
getting-started/          Installation, Your first steps
how-tos/                  one page per task
explanation/              how it works, and why
reference/                facts to look up
glossary/
  index.md                every term A–Z, built from the term files
  example-term.md         one term per file
releases/                 one page per release, newest first
.zen/templates/           the page templates the sections use
```

Each section has a **page template**: a new page made in `how-tos/` starts
as a how-to, with its sections already in place. The folder's `.zenmd.yml`
says which template each section uses:

```yaml
templates:
  "how-tos/**": how-to
  "glossary/**": term
```

## From a terminal

```sh
zenmd init --blueprint techdocs ~/Docs/my-product
zenmd blueprint list
zenmd blueprint new my-docs --from techdocs   # your own, to edit
```

A blueprint of your own is a folder with a `blueprint.yml` in the blueprints
folder (Settings › Blueprints shows where). It lists its sections, and
Zenmark makes the folders, their first pages and the links between them.
