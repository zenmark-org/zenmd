# The zenmd command

Every `zenmd` command and what it does. `zenmd help` prints the same.

## Commands

| Command | What it does |
|---|---|
| `zenmd [path]` | Open a folder or markdown file in Zenmark (default: the current folder). macOS app only. |
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
| `zenmd glossary add "<Term>"` | Add a term from the term template. `--folder`, `--definition`, `--open`. |
| `zenmd glossary list [folder]` | List the terms A to Z with their definitions. |
| `zenmd version` | Print the version; `--verbose` adds what the build includes. |
| `zenmd help` | Show the help. |

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
| 1 | It failed: nothing usable was written, or the command needs the app. |
| 2 | The command line is wrong: an unknown option or command. The message ends with a hint. |

> **Missing:** which commands work without the app on macOS, when only the
> `zenmd` link is installed.
