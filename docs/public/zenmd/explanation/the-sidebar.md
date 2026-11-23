# Why the sidebar follows your links

This page explains why zenmd builds the sidebar from links between pages,
not from a list in a config file.

## The idea

zenmd starts at the folder's home page (`index.md`, or `README.md` when there
is no `index.md`) and follows its links. A page appears under the first page
that links to it, in the order of the links. A page nobody links to goes under
**Other pages**.

Frontmatter can still steer it: `nav: { title: … }` renames an entry,
`nav: { order: 2 }` moves it, `nav: { hidden: true }` hides it, and
`nav: { children: folder, sort: title }` lists a whole folder under its index.

## Why it is this way

- **One source of truth.** The links you write for readers are the structure. There is no second list to keep in step.
- **Any folder works.** A folder written for GitHub, or for another tool, gets a sensible sidebar with no setup.
- **The rejected option** was a `nav:` list in config, as many site generators have. It goes stale as pages are added, and it is one more file to learn.
