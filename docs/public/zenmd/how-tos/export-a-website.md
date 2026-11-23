# Export a website

Turn a folder into a static website that opens from disk or from any web
server.

## Steps

1. Export:
   ```sh
   zenmd export site ~/Docs/acme
   ```
   The site is written to `exports/html` in the folder. `-o <dir>` writes it elsewhere.
2. Pick how it is served:
   - `--portable` (the default) opens from disk and from any web server;
   - `--hosted` loads scripts as files, for a web server only.
3. Keep your choices for next time:
   ```sh
   zenmd export site ~/Docs/acme --scheme light --save-defaults
   ```
   This writes them under `export:` in `.zenmd.yml`. A flag always wins over the file.

In Zenmark the same is **File › Export › Website…** (⇧⌘B), and Settings ›
Export holds the same options. Every option is in the
[export options reference](../reference/export-options.md).

## Troubleshooting

- **"will not clean …":** `--clean` empties the output folder only when it holds a `.zen-build` file from an earlier export.
- **Draft pages are missing:** pages with `draft: true` are left out unless you pass `--drafts`.
