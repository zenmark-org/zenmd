# Open one file

You can open a single markdown file without its folder. The window has that
page to read, edit, print or export as a PDF. Nothing else in the folder is
scanned, so a file in `~/Downloads` does not make Zenmark read all of Downloads.

## How to open one

- Choose **File › Open…** (<kbd>⌘O</kbd>) and pick a `.md` or `.markdown` file.
- In Finder, use **Open With › Zenmark**, double-click the file when Zenmark is
  the default app for it, or drop it on the Dock icon.
- In a terminal: `zenmd notes.md`.

A file inside a project folder, one with a `.zenmd.yml` or a `.git` above it, still opens that folder at the page, as before. In a terminal
`zenmd notes.md --file` forces the file alone and `zenmd notes.md --folder`
forces the folder.

## What the window shows

The sidebar has no tree. It shows the file's name, the page's **outline** (its
headings; a click scrolls to one, and <kbd>↑</kbd> <kbd>↓</kbd> move between
them) and an **Open Enclosing Folder** button.

[Reading](reader.md), [editing](editor.md), [find](find.md), [printing](printing.md)
and the [Inspector](inspector.md) work as they do for a folder. Images and
links next to the file work too, and a link to another `.md` file in the same
folder or below opens it in the same window: <kbd>⌘[</kbd> and <kbd>⌘]</kbd>
go back and forward. A link to a file above the file's folder does nothing.

## What it leaves off

These need a folder, so they are greyed out, with the tip *Open a folder to
use this*:

- **New Page** and **New from Blueprint**
- **Export › Website** (Export › PDF works)
- the folder's `.zenmd.yml` in Settings

Only the global `config.yml` applies. A `.zenmd.yml` next to the file is
ignored unless you open the folder. The page's own frontmatter (`layout:`,
`template:`) still applies; with no `layout:` it is an `article`.

## Open the folder

**File › Open Enclosing Folder** (<kbd>⇧⌘O</kbd>), or the button in the
sidebar, opens the file's project folder at this page: the nearest folder above
it with a `.zenmd.yml` or `.git`, else the file's own folder. It is
the same as `zenmd notes.md --folder`.
