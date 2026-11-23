# Blueprints (formerly site templates)

A blueprint is a folder that `zenmd init --blueprint` (or File › New from
Blueprint…, ⇧⌘N) copies into a new or existing folder. Spec:
[specs/core/init](../../specs/core/init/spec.md).

```sh
zenmd init --blueprint techdocs docs  # Technical documentation (Diátaxis)
zenmd init --blueprint learning course # A learning path
zenmd init --blueprint planner plans  # A planner
zenmd blueprint list                  # what is there (zenmd init --list too)
zenmd blueprint new mine --from techdocs  # start your own, in the blueprints folder
zenmd init --dir                      # where the blueprints folder is
zenmd init --update techdocs          # a newer built-in, written next to yours as techdocs.new
```

The old spellings still work: `zenmd init site --techdocs`, `--template <id>`,
`template.yml` and the assembly key `templates:`.

## Where templates live

One folder holds both kinds of template, next to `config.yml`:

```
<zen config dir>/templates/
  blueprints/   one folder per blueprint (techdocs/, learning/, planner/, your own)
  pages/        one file per page template (how-to.yml, term.yml, ..., your own)
```

Move it in the global `config.yml` (a folder's `.zenmd.yml` `templates:` is the
path rules, a different thing):

```yaml
templates:
  dir: ~/Documents/zen templates
```

`zenmd templates --dir` prints it and `zenmd init --dir` prints `blueprints/`.
The built-ins are copied there the first time they are listed, so you can edit
them. Each copy is recorded with its version and a SHA-256 fingerprint in
`.zen-templates.yml`; zen never installs one again after you delete it.

- **Updates:** `zenmd init --update <id>` (a blueprint) and `zenmd templates
  update <id>` (a page template) replace a copy you never edited in place. A copy
  you edited, or one installed before fingerprints, is left alone and the newer
  one is written beside it as `<id>.new` (`<id>.yml.new` for a page template).
  Settings › Blueprints and Settings › Templates have the same **Update** button.
- **Page templates:** `zenmd templates list` and `zenmd templates show <id>`.
  A folder's `.zen/templates/<id>.yml` wins over `templates/pages/<id>.yml`,
  which wins over the built-in one.
- **The old folder:** an earlier build kept blueprints in `templates/sites/`. The
  first use renames it to `templates/blueprints/` (with its record) and notes it
  under Problems. It stays where it is when `blueprints/` already exists, when
  either is a link, or when the templates folder is outside the settings folder.

## A blueprint

```
my-blueprint/
  blueprint.yml
  index.md
  guides/{{date}}-first.md
  .zenmd.yml
```

```yaml
id: my-blueprint
name: My blueprint
description: One line shown on the card
version: 1.0.0
order: 50
ask:
  title: { label: Site title, default: "{{folder}}" }
  author: { label: Author, default: the team }
ignore: [".DS_Store"]
```

In `.md`, `.yml`, `.yaml`, `.txt` and `.css` files and in file names,
`{{title}}`, `{{author}}`, `{{date}}`, `{{folder}}` and every `ask:` key are
replaced with their values. This is plain text replacement: nothing in a
blueprint runs. Other files are copied as they are.

An `ask:` default may use `{{folder}}`, `{{date}}` and `{{key}}` of any ask
before it, for example `product: { default: "{{title}}" }`. It follows that
answer: in the dialog live until you edit the field, in `zenmd init` from the
given or defaulted value.

Creating a site never overwrites anything. If one file would clash, nothing is
written and the clashes are listed. A blueprint is limited to 2,000 files and 50 MB.

## Sections

`blueprint.yml` can list `sections:` instead of folders and files. zen builds
them:

```yaml
layout: docs                 # .zenmd.yml layout:
home:                        # index.md: links each section, in order
  title: "{{title}}"
  template: overview         # optional: the home has this page template's structure
  pages:                     # optional: extra root pages, linked from the home
    - { file: faq.md, title: "Questions and answers" }
sections:
  - id: releases
    title: Releases
    folder: releases         # releases/index.md, a landing page with cards
    template: release-notes  # path rule in .zenmd.yml: "releases/**": release-notes
    order: newest-first      # nav: { sort: newest-first } on the index
    pages:                   # starting pages, from the page template's skeleton
      - { file: v0.1.0.md, title: v0.1.0, set: { version: 0.1.0, name: First release } }
  - id: about
    title: About
    file: about.md           # a file section: one page
templates: [release-notes]   # page templates copied into .zen/templates
files: [CONTRIBUTING.md]     # other files, copied as they are
```

- A folder's home is its `index.md` (any case), with `README.md` as the fallback when there is none; `index.md` wins when both exist. This holds for the root and for every folder. Blueprints write `index.md`.
- `index: { layout: landing, tagline: "..." }`, `{ template: overview }` or `{ layout: glossary }` change the index page.
- A starting page with its own `layout:` keeps it and follows no template.
- The `release-notes` page template is a release page: layout `release`, the fields `version` (required), `date` (required, today by default) and `name` (optional), and a skeleton of just Added, Changed and Fixed. It has no `title` and no H1: the version is the page's title (see `title:` in docs/wiki/LAYOUTS.md), and the file is named after it, `releases/v0.1.0.md`, titled v0.1.0 (the layout's `title: "v{{version}}"`). The releases index shows each release as a card with its version, date and name.
- The `troubleshooting` page template is one problem per page, in the `troubleshooting` layout: the H1 is the problem in the reader's words, then a paragraph on when it happens, Symptoms (a list), Cause, Solution (numbered steps), and optional Check that it's fixed and Related. Its optional fields are `error` (the exact message, shown as code under the title), `versions`, `fixed-in` and `tags`. techdocs has a Troubleshooting section (`troubleshooting/`) between Reference and Releases.
- In skeletons and titles: `{{title}}`, `{{author}}`, `{{today}}`, any `ask:` key and the `set:` fields. `{{sections}}` in a template is a link to every section.
- The built-in page templates are in `builtins/pagetemplates/`. A blueprint's own `.zen/templates/<id>.yml` wins.
- `nav.sort` is written and accepted in frontmatter, but the sidebar does not sort by it yet. Checking pages against their template comes later.
