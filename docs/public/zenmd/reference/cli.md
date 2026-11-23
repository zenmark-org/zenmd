# The zenmd command

Every `zenmd` command and what it does. `zenmd help` prints the same.

## Commands

| Command | What it does |
|---|---|
| `zenmd [path]` | Open a folder or markdown file in Zenmark (default: the current folder). A file outside a project folder (no `.zenmd.yml` or `.git` above it) opens alone; `--file` forces one file, `--folder` the folder. macOS app only. |
| `zenmd edit [path]` | Open a page in the editor; a new `.md` file is created on save. macOS app only. |
| `zenmd export site [folder]` | Write a static website. See [export options](export-options.md). |
| `zenmd export pdf [folder] [-o file.pdf]` | Write the whole folder as one PDF. macOS app only. |
| `zenmd build [folder]` | The same as `export site`. |
| `zenmd init --blueprint <id> [folder]` | Start a folder from a blueprint. `--title`, `--author` and `--set key=value` answer its questions. |
| `zenmd init --list` | List the blueprints. `--update <id>` updates one; `--dir` prints their folder. |
| `zenmd blueprint list` | List the blueprints. |
| `zenmd blueprint new <id> [--from <id>]` | Write a new blueprint into the blueprints folder. |
| `zenmd templates list` | List the page templates, where each comes from, and its version. |
| `zenmd templates show <id>` | Print a page template's fields, structure and skeleton. |
| `zenmd templates update <id>` | Bring a built-in page template to its newer version. |
| `zenmd templates --dir` | Print the templates folder (`blueprints/`, `pages/`). |
| `zenmd check [folder]` | Check every page against its page template and its layout's structure, as Problems › Structure does. [Details below](#zenmd-check). |
| `zenmd glossary add "<Term>"` | Add a term from the term template. `--folder`, `--definition`, `--open`. |
| `zenmd glossary list [folder]` | List the terms A to Z with their definitions. |
| `zenmd version` | Print the version; `--verbose` adds what the build includes. |
| `zenmd help` | Show the help. |

## zenmd check

```
zenmd check [folder] [--format text|json] [--severity error|warning|suggestion] [--template <id>]
```

It reads every page of the folder and checks it against:

1. the page template it follows: `template: <id>` in its frontmatter (`template: false` opts out), else the folder's `templates:` path rule in `.zenmd.yml`;
2. the `structure:` of its layout.

It also lists what the page's render notes, such as a `title:` that an H1 makes useless. Nothing is written.

| Option | What it does |
|---|---|
| `--format text` | One line per problem on stdout (the default). |
| `--format json` | One JSON document: `{ "pages": 12, "problems": [ { "page", "line", "severity", "rule", "message", "fix", "source" } ] }`. |
| `--severity <s>` | The lowest severity shown: `error`, `warning` (the default) or `suggestion`. Errors are always shown. |
| `--template <id>` | Check every page against this page template, whatever the pages and path rules say. |

The text format is fixed, so a script or a CI step can read it:

```
<page>:<line>: <severity>: <message>
```

`<page>` is the path from the folder, `<line>` the line of the page file (1 for a problem of the whole page), and `<severity>` is `error`, `warning` or `suggestion`. A page with no problem prints nothing. The summary (`Checked 12 pages: 1 error, 3 warnings, 0 suggestions`) goes to stderr, so stdout holds the problem lines only.

```
guides/install.md:1: error: Missing section 'Steps'
blog/hello.md:1: error: date is required by the post layout
```

It exits with 1 when any problem is an error, 0 otherwise, and 2 when the command line is wrong (an unknown `--template` is one). There is no `check:` key in `.zenmd.yml` yet; the options are flags only.

## Options for every command

| Option | What it does |
|---|---|
| `-h`, `--help` | Show the help. |
| `-v`, `--version` | Show the version. |
| `--` | Treat the arguments after it as paths. |

## The command-line build

On Linux, `zenmd` is a command-line-only build with no app
([installation](../getting-started/installation.md#install-zenmd-on-linux)).
`zenmd version` says `(command line)` after the version. Every command works
except the three marked "macOS app only". Those print one line and exit with 1:

- opening a folder or page (`zenmd`, `zenmd <path>`, `zenmd edit`): `the Zenmark app isn't part of this build; zenmd export site <folder> makes a website`;
- `zenmd export pdf`: `export pdf needs the Zenmark app on macOS`.

`zenmd templates --dir` and `zenmd init --dir` print the folder on both builds.
On Linux the settings folder is `$XDG_CONFIG_HOME/zenmark`, or `~/.config/zenmark`
when that isn't set.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | It worked. |
| 1 | It failed: nothing usable was written, or the command needs the app. `zenmd check` also exits with 1 when a page has an error. |
| 2 | The command line is wrong: an unknown option or command. The message ends with a hint. |

> **Missing:** which commands work without the app on macOS, when only the
> `zenmd` link is installed.
