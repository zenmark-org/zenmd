# Export a PDF

Print a whole folder, or one part of it, as a single PDF with a cover and
contents.

## Steps

1. Export:
   ```sh
   zenmd export pdf ~/Docs/acme
   ```
   The PDF is written to `exports/pdf/acme.pdf`.
2. Only one part: `--section how-tos/index.md` prints that page and the pages under it.
3. Without the cover and contents: `--no-cover`.

In Zenmark: **File › Export › PDF…** (⇧⌘P).

## Troubleshooting

- **It only works on a Mac:** the PDF is printed by macOS's web view.
- **A diagram is blank:** a block that isn't ready in time is named in the output; export again.

> **Missing:** page size, margins and headers. They aren't options yet.
