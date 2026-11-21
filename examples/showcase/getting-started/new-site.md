---
slug: new-site
nav:
  parent: getting-started/opening-folders.md
  title: Start a new site
  order: 1
---
# Start a new site from a blueprint

A **blueprint** is a ready-made folder: a documentation site, a learning
path or a planner. zen copies it into a folder of yours, fills in your title,
and opens it. Nothing is ever overwritten. (Blueprints used to be called site
templates; the old names still work.)

## The three blueprints that come with zen

| Blueprint | Id | For |
|---|---|---|
| Technical documentation | `techdocs` | an overview, getting started, how-tos, explanation, reference, a glossary and releases, after [Diátaxis](https://diataxis.fr) |
| Learning path | `learning` | courses with steps, quizzes and checklists, and a glossary |
| Planner | `planner` | a roadmap, weekly plans, decision records and meeting notes |

## From the app

- On the welcome screen, pick a card under **Start from a blueprint**.
- Or choose **File › New from Blueprint…** (<kbd>⇧⌘N</kbd>). It works from the welcome screen and from an open folder.

Zen asks for the values the blueprint needs, starting with the site title. Then
press **Choose folder…**: the panel lets you pick an existing folder or make a
new one. **Create** makes the site and opens it. If any file would be
overwritten, zen writes nothing and lists the files in the way.

## From a terminal

```sh
zenmd init --blueprint techdocs docs --title "Acme docs"
zenmd init --blueprint learning course --author "Ana"
zenmd blueprint list
```

| Command | Does |
|---|---|
| `zenmd init --blueprint <id> [folder]` | creates the site; the folder defaults to the current one |
| `zenmd init site --<id> [folder]` | the same, with the old spelling (`--template <id>` works too) |
| `--title`, `--author`, `--set key=value` | give the values; the others take their defaults |
| `zenmd blueprint list` | lists the blueprints (`zenmd init --list` does too) |
| `zenmd blueprint new <id> [--from <id>]` | starts a blueprint of your own in the blueprints folder |
| `zenmd init --update <id>` | writes a newer built-in next to your copy, as `<id>.new` |
| `zenmd init --dir` | prints the blueprints folder |

The command never asks questions, so it works in scripts. It exits with 1 when
files are in the way, and with 2 for a mistake in the command.

## Blueprints are folders you can edit

The first time zen lists blueprints, it copies the built-in ones into a
**blueprints folder**, next to `config.yml`. Edit them, copy them, or add your
own: every folder in `sites/` with a `blueprint.yml` is a blueprint. Zen never
overwrites a blueprint you changed, and does not bring back one you deleted.

To keep the blueprints somewhere else, set the folder in the global
`config.yml` (a folder's `.zenmd.yml` cannot):

```yaml
blueprints:
  dir: ~/Documents/zen blueprints
```

(The old key `templates:` is still read, and noted.)

A blueprint folder holds a `blueprint.yml` and the files to copy:

```yaml
id: notes
name: Team notes
description: A home page and a place for meeting notes
version: 1.0.0
order: 40
ask:
  title: { label: Site title, default: "{{folder}}" }
  team: { label: Team, default: "" }
```

In files that end in `.md`, `.yml`, `.yaml`, `.txt` or `.css`, and in file
names, zen replaces `{{title}}`, `{{author}}`, `{{date}}`, `{{folder}}` and
each key you ask for. This is plain text replacement: nothing is run, an unknown
name stays as it is, and images are copied unchanged.

## Sections instead of files

Instead of writing every folder by hand, a blueprint can list **sections**.
zen then makes each section's folder and `index.md`, a `README.md` that links
the sections in order, and a path rule in `.zenmd.yml` so that pages made later
in that folder follow the section's page template:

```yaml
layout: docs
sections:
  - id: how-tos
    title: How-tos
    folder: how-tos
    template: how-to
    pages: [ { file: example-task.md, title: Do one task } ]
templates: [how-to]
```

`zenmd blueprint new mine --from techdocs` starts from the built-in one.
