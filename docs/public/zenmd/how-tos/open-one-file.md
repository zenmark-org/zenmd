# Open one file

Read, edit or print a single markdown file without opening its folder.

## Steps

1. Open the file:
   ```sh
   zenmd ~/Downloads/notes.md
   ```
   In Zenmark: **File › Open…** (⌘O), or **Open With › Zenmark** in Finder.
2. Read it. The sidebar shows the page's outline, not a tree of the folder.
3. To work on the folder it is in, choose **File › Open Enclosing Folder**
   (⇧⌘O), or run `zenmd ~/Downloads/notes.md --folder`.

Nothing else in the folder is scanned, so this is the quick way to look at a
file in a large folder such as Downloads.

## When it opens the folder instead

A file inside a project, with a `.zenmd.yml` or a `.git` in a
folder above it (below your home folder), opens that folder at the page. To
open the file alone anyway: `zenmd notes.md --file`.

## What is not there

- **New Page, New from Blueprint, Export › Website and the folder's settings** need a folder and are greyed out. Export › PDF works.
- **Only `config.yml` applies.** A `.zenmd.yml` next to the file is ignored; the page's frontmatter still applies.
- A link to another `.md` file in the same folder or below opens it in the same window. A link to a file above the folder does nothing.

> **Missing:** Save As…, a new untitled file and several files in one window.
