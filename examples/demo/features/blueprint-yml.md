# The blueprint.yml reference

Every blueprint is a folder with a `blueprint.yml` at its top. That file says
what a new folder gets: the questions asked when the folder is made, its home
page, its sections and the page templates they use. Every other file in the
blueprint folder is copied into the new folder as it is.

Your blueprints are in the blueprints folder of Zenmark's settings folder
(Settings › Blueprints shows where). `zenmd blueprint new my-docs` starts one
with a commented `blueprint.yml`, and `zenmd blueprint new my-docs --from techdocs`
copies a built-in one to edit.

A key Zenmark doesn't know is an error, so a typo never goes unnoticed: the
blueprint is listed with the reason, and can't be used until it is fixed.

## A whole file

```yaml
id: my-docs                      # required: lower case letters, digits and -
name: My documentation           # required: shown in New from Blueprint…
description: Guides and reference for one product
version: 1.0.0                   # SemVer; default 0.0.0
order: 20                        # where it shows in the list, lower first; default 100

requires:                        # the layouts and plugins the pages use
  layouts: [docs, landing, glossary, term]
  plugins: [codeblock]

ask:                             # questions asked when a folder is made
  title:   { label: Site title, default: "{{folder}}" }
  author:  { label: Author, default: "the team" }
  product: { label: Product name, default: "{{title}}" }

layout: docs                     # the new folder's default layout (.zenmd.yml layout:)

home:                            # index.md, the start page
  title: "{{title}}"
  template: overview             # optional: the home follows this page template
  pages:                         # optional: other pages at the top, linked from the home
    - { file: faq.md, title: Questions and answers }

sections:                        # in sidebar order
  - id: guides
    title: Guides
    folder: guides
    template: how-to
    index: { layout: landing, tagline: "Steps for one real task" }
    pages:
      - { file: first-guide.md, title: Your first guide }
  - id: releases
    title: Releases
    folder: releases
    template: release-notes
    order: newest-first
    pages:
      - { file: v0.1.0.md, title: v0.1.0, set: { version: 0.1.0, name: First release } }
  - id: glossary
    title: Glossary
    folder: glossary
    template: term
    index: { layout: glossary }
  - id: about
    title: About
    file: about.md               # a section can be one page instead of a folder

templates: [how-to, release-notes, term]   # page templates copied into .zen/templates
files: [CONTRIBUTING.md]         # files to note as part of the blueprint
ignore: [".DS_Store", drafts]    # names or paths that are not copied
```

## The top keys

| Key | What it does |
|---|---|
| `id` | **Required.** The blueprint's id, as in `zenmd init --blueprint my-docs`. Lower case letters, digits and `-`, starting with a letter. |
| `name` | **Required.** The name shown in New from Blueprint… and Settings › Blueprints. |
| `description` | One line under the name. |
| `version` | A version such as `1.0.0`. Zenmark uses it to update the built-in blueprints. |
| `order` | A number: lower ones are listed first. The default is 100. |
| `requires` | `layouts:` and `plugins:` the blueprint's pages use. Each build of Zenmark is checked to include everything its built-in blueprints need; in your own, it tells readers what the pages rely on. |
| `ask` | The questions asked when a folder is made (see below). |
| `layout` | The new folder's default layout, written to its `.zenmd.yml`. |
| `home` | The `index.md` start page (see below). |
| `sections` | The parts of the folder, in sidebar order (see below). |
| `templates` | Page template ids copied into the new folder's `.zen/templates/`, so the folder keeps working on another Mac. |
| `files` | Other files of the blueprint, listed for people reading it. Every file in the blueprint folder is copied anyway. |
| `ignore` | Names or paths in the blueprint folder that are not copied. |

## `ask`: the questions

Each key is a question; `label` is what the dialog shows, and `default` fills
the field. `title` is always asked first.

- Answers are used as `{{key}}` in every file, file name and page of the
  blueprint, and in `home` and `sections`.
- Zenmark fills in two values itself, which can't be asked: `{{folder}}`, the
  new folder's name, and `{{date}}`, today's date.
- A default may use `{{folder}}`, `{{date}}`, and any question asked before
  it, as in `product: { default: "{{title}}" }`.
- Key names start with a lower case letter and use letters, digits, `_` and `-`.

`zenmd init --blueprint my-docs --title "Acme"` answers a question from the
terminal; one left out takes its default.

## `home`: the start page

Zenmark writes `index.md` with the title, and one link to each section in
order. Those links make the sidebar order.

| Key | What it does |
|---|---|
| `title` | The home page's H1. |
| `template` | A page template, such as `overview`: the home has its sections (what it is, key features, where to go next). |
| `pages` | Other pages at the top of the folder, linked from the home after the sections. Each is a `.md` file name with a `title`, and optionally a `layout` and `set` (as for section pages). |

## `sections`

Each section is a folder with an `index.md`, or a single page.

| Key | What it does |
|---|---|
| `id` | **Required.** Lower case letters, digits and `-`; used once. |
| `title` | **Required.** The section's name in the home's links and the sidebar. |
| `folder` | A folder section: `<folder>/index.md` and the pages in it. |
| `file` | A one-page section, a `.md` file. Give either `folder` or `file`. A file section has no `index` or `pages`. |
| `template` | The page template its pages follow. It becomes a path rule in `.zenmd.yml` (`"guides/**": how-to`), so New Page in that folder starts from it. |
| `index` | How `index.md` is made (see below). |
| `order` | `newest-first` sorts the pages by version or date, newest on top; `title` sorts them A to Z. Without it, pages follow the links of the index. |
| `pages` | The starting pages, written from the template's skeleton. |

### `index`

| Key | What it does |
|---|---|
| `layout` | The index's layout. `landing` (the default) shows a card for each page; `glossary` builds an A–Z index of the term pages. |
| `tagline` | The line under the title of a landing index. |
| `template` | Use the body of this page template for the index instead. |
| `title` | The index's H1, when it should differ from the section's title. |

### `pages`

| Key | What it does |
|---|---|
| `file` | **Required.** A `.md` file name inside the section folder. |
| `title` | The page's H1. |
| `layout` | The page's layout, when it differs from the folder's. |
| `set` | Frontmatter fields to fill in, such as `{ version: 0.1.0, name: First release }`. Versions are written quoted. |

## Try it

```sh
zenmd blueprint new my-docs --from techdocs
zenmd blueprint list
zenmd init --blueprint my-docs ~/Docs/try-it
```

Back to [Blueprints](blueprints.md).
