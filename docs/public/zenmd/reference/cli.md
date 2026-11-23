# The zenmd command

Every `zenmd` command and what it does. `zenmd help` prints the same.

## Commands

| Command | What it does |
|---|---|
| `zenmd [path]` | Open a folder or markdown file in Zenmark (default: the current folder). |
| `zenmd edit [path]` | Open a page in the editor; a new `.md` file is created on save. |
| `zenmd export site [folder]` | Write a static website. See [export options](export-options.md). |
| `zenmd export pdf [folder] [-o file.pdf]` | Write the whole folder as one PDF. |
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

> **Missing:** exit codes, and which commands work without the app installed.
> Neither is written down anywhere yet.
