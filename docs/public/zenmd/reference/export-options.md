# Export options

Every option of `zenmd export site` and `zenmd export pdf`, its config key
and its default. Settings › Export in Zenmark has the same options; a flag
wins over the folder's `export:` settings, which win over `config.yml`.

## Website (`zenmd export site`)

| Flag | Config key | Default | What it does |
|---|---|---|---|
| `-o`, `--out <dir>` | `export.out` | `exports/html` | The output folder. |
| `--portable` / `--hosted` | `export.portable` | portable | Portable opens from disk and from any web server; hosted loads scripts as files, for a web server only. |
| `--drafts` / `--no-drafts` | `export.drafts` | off | Include pages with `draft: true`. |
| `--clean` | none | off | Empty the output folder first (it must hold a `.zen-build` file). |
| `--print` / `--no-print` | none | on | Also write `print.html`, the whole site as one page. |
| `--scheme <auto, light or dark>` | `export.theme.scheme` | `auto` | The colours: auto follows the reader's system. |
| `--save-defaults` | none | off | Write the options given into `.zenmd.yml` under `export:`, then export. |
| `--dry-run` | none | off | With `--save-defaults`: show the change and write nothing. |

## PDF (`zenmd export pdf`)

| Flag | Config key | Default | What it does |
|---|---|---|---|
| `-o`, `--out <file>` | `export.pdf.out` | `exports/pdf/<folder>.pdf` | The output file. |
| `--drafts` / `--no-drafts` | `export.pdf.drafts` | off | Include pages with `draft: true`. |
| `--section <page>` / `--all` | none | all | Only this page and the pages under it. |
| `--cover` / `--no-cover` | `export.pdf.cover` | on | The cover and the contents. |
| `--open` / `--no-open` | `export.pdf.open` | off in the CLI, on in the app | Open the PDF when it is written. |
| `--save-defaults`, `--dry-run` | none | off | As for the website. |

An output path in a folder's `.zenmd.yml` must stay inside the folder.
