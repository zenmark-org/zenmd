# Blueprints (formerly site templates)

A blueprint is a folder that `zenmd init --blueprint` (or File › New from
Blueprint…, ⇧⌘N) copies into a new or existing folder. Spec:
[specs/core/init](../specs/core/init/spec.md).

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
the global `templates:` key (noted under Problems), `template.yml` and the
assembly key `templates:`.

## Where blueprints live

`<zen config dir>/blueprints/sites/<id>/` (`templates/` is still used when it is
the only one that exists), next to `config.yml`. Move it in the
global `config.yml`:

```yaml
blueprints:
  dir: ~/Documents/zen blueprints
```

The three built-ins are copied there the first time blueprints are listed, so
you can edit them. zen never overwrites your copy, and a built-in you delete
stays deleted.

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
    - { file: presentation.md, title: "{{product}} in five slides", layout: slides }
sections:
  - id: releases
    title: Releases
    folder: releases         # releases/index.md, a landing page with cards
    template: release-notes  # path rule in .zenmd.yml: "releases/**": release-notes
    order: newest-first      # nav: { sort: newest-first } on the index
    pages:                   # starting pages, from the page template's skeleton
      - { file: "{{today}}-first.md", title: First release, set: { version: 0.1.0 } }
  - id: about
    title: About
    file: about.md           # a file section: one page
templates: [release-notes]   # page templates copied into .zen/templates
files: [CONTRIBUTING.md]     # other files, copied as they are
```

- A folder's home is its `index.md` (any case), with `README.md` as the fallback when there is none; `index.md` wins when both exist. This holds for the root and for every folder. Blueprints write `index.md`.
- `index: { layout: landing, tagline: "..." }`, `{ template: overview }` or `{ layout: glossary }` change the index page.
- A starting page with its own `layout:` keeps it and follows no template.
- In skeletons and titles: `{{title}}`, `{{author}}`, `{{today}}`, any `ask:` key and the `set:` fields. `{{sections}}` in a template is a link to every section.
- The built-in page templates are in `builtins/pagetemplates/`. A blueprint's own `.zen/templates/<id>.yml` wins.
- `nav.sort` is written and accepted in frontmatter, but the sidebar does not sort by it yet. Checking pages against their template comes later.
