# app/single-file

| | |
|---|---|
| Status | done |
| Target version | 0.8.0 |
| Branch | `feature/app-single-file` |
| Wave | 57 |
| Depends on | none |
| Owns | `engine/workspace/single.go` (new) and its tests, `engine/session/openfile.go` (new) and its tests, `internal/cli/cli.go` (`resolve`, the `--file` flag, usage text), `apps/zenmark/openfile.go` (new), `apps/zenmark/menu.go` (the File menu items), `kit/desktop/run.go` (the Open With handler only), `kit/ui/src/shell/**` (the one-file state), `apps/zenmark/frontend/src/shell/**`, `examples/demo/features/open-a-file.md` (new), `docs/public/zenmd/how-tos/open-one-file.md` (new) |

## Purpose

Open one markdown file on its own, without its folder. You get a window
with that page to read, edit, print or export as PDF. There's no folder
tree and nothing else in the folder is scanned.

## Behaviour

1. **What opens a single file.**
   - `zenmd <file.md>` and `zenmd edit <file.md>` open the file alone when no
     folder above it (up to the home folder) has `.zenmd.yml`, `.zen.yml` or
     `.git`. Today such a file opens its whole parent folder, so
     `zenmd ~/Downloads/notes.md` scans all of Downloads; that stops.
   - Inside a project folder the CLI keeps opening the folder at that page, as
     today. `--file` forces one file: `zenmd notes.md --file`.
   - `--folder` forces the folder, even with no marker.
2. **How the app opens one.**
   - **File › Open…** (⌘O) accepts a folder or a `.md`/`.markdown` file. A
     file opens alone. It replaces Open Folder…, keeping its shortcut.
   - **Finder.** Open With, a double-click on a `.md` file when Zenmark is
     the default, or dropping a file on the Dock icon also opens the file
     alone.
   - **Recents.** File › Open Recent lists such files with a document icon,
     next to folders. *Not built in 0.8.0: the app has no Open Recent menu
     yet; it comes with that menu.*
3. **The one-file workspace.** The session opens a workspace whose only page
   is that file. `Files()` returns one entry. The folder is not walked: the
   page count is 1 and the folder limits don't apply. Assets the page links
   (images, relative to the file) are read as today, and only inside the
   file's folder (`Resolve` keeps the same containment rule, rooted at the
   file's folder).
4. **What the window shows.**
   - The sidebar has no tree. It shows the file's name and its outline (the
     page's headings), and a quiet **Open Enclosing Folder** button.
   - The reader, editor, find in page, print, Export › PDF and the Inspector
     work as with a folder.
   - Export › Website, New Page, New from Blueprint, folder search and the
     folder-only Settings items (This Folder) are disabled. Each greyed item
     has a tooltip: "Open a folder to use this".
5. **Links to other pages.** A link to another `.md` file in the same folder
   or below opens that file in the same one-file window. Back and Forward
   work across them. A link outside the file's folder shows the usual
   "outside the folder" notice. A missing target shows the broken-link mark,
   as today.
6. **Config.** One file uses the global `config.yml` only. A `.zenmd.yml`
   next to it is ignored unless the folder is opened. The page's frontmatter
   (`layout:`, `template:`) still applies. A built-in layout renders it; with
   no `layout:` it's `article`.
7. **Editing.** Saving writes the one file atomically, as today. **New
   Page** is off, and **Save As…** is out of scope. The file watcher watches
   the file alone, so an outside edit reloads it.
8. **Open Enclosing Folder** (⌘⇧O, also in the File menu) opens the file's
   project folder at this page: the nearest marker folder, else the file's
   own folder. It's the same as `zenmd <file> --folder`.
9. **The title.** The window title is the page's title (its H1), then
   `— <file name>`. Home is the page itself.
10. **Trust is unchanged.** Plugin trust and the script/external plugin
    rules apply as they do in a folder, keyed by the file's folder.

## Architecture

- **`engine/workspace`** adds `OpenFile(path string) (*Workspace, error)`: a
  workspace rooted at the file's folder with a one-entry file set. `Files`,
  `Opened`, `Home` and `Entries` answer from that set. The walker is never
  started. `Resolve`, `Read`, `Write` and `Asset` work unchanged, so the
  containment checks stay.
- **`engine/session`** adds `OpenFile(path string) error`. It mirrors
  `Open(dir)` (newest open wins, the `workspace:opened` event) and sets
  `Single bool` in the opened payload, so the UI knows the state.
- **`internal/cli`**:
  - `resolve` returns a new `single bool` (marker found or not, plus
    `--file`/`--folder`);
  - `Launch` gains `File bool`;
  - the app's `Launch` calls `OpenFile` when it's set.
- **`apps/zenmark`** handles ⌘O (folder or file), Open With, Open Enclosing
  Folder and Recents entries for files. All of it goes through
  `Zen.OpenFile` and `Zen.OpenEnclosingFolder`.
- **`kit/ui` shell** gets a `single` state. In it, the sidebar renders the
  outline in place of the tree, and the disabled items follow
  `host.features` plus `single`.

## Interfaces

```go
// engine/workspace
func OpenFile(path string) (*Workspace, error) // ErrNotMarkdown, ErrTooLarge, os errors
func (w *Workspace) Single() bool

// engine/session
func (s *Session) OpenFile(path string) error
type Opened struct { /* existing fields */ Single bool `json:"single,omitempty"`; File string `json:"file,omitempty"` }

// internal/cli
type Launch struct { Dir, Page, Mode string; File bool }

// apps/zenmark (bound)
func (z *Zen) Open(path string) error          // folder or file, the ⌘O dialog's result
func (z *Zen) OpenEnclosingFolder() error
```

```ts
// kit/ui host
interface Opened { /* … */ single?: boolean; file?: string }
```

**CLI:**
- `zenmd [edit] <file.md> [--file | --folder]`. There are no config keys.
  Help lists both flags under "Opening".
- `--file` with a folder argument is a usage error (exit 2).

## Tests

| # | Behaviour | Test |
|---|---|---|
| 1 | Marker rules, `--file`, `--folder`, folder + `--file` usage error | `internal/cli/open_test.go: TestOpenSingleFile` (fake home; files in Downloads-like folders, inside `.git` and `.zenmd.yml` folders) |
| 2 | ⌘O accepts both; Open With opens a file alone; Recents shows files | `apps/zenmark/openfile_test.go: TestOpenPicksMode`, `kit/desktop/run_test.go: TestOpenWithFile` |
| 3 | One entry, no walk, assets contained | `engine/workspace/single_test.go: TestOpenFileOneEntry`, `TestOpenFileNoWalk` (a sibling folder of 5,000 files: opens in under 50 ms, `Entries()` is 1), `TestOpenFileAssetContainment` |
| 4 | Sidebar outline, disabled items with the tooltip | `kit/ui/src/shell/single.test.ts` |
| 5 | Links open in the same window, back/forward, outside notice | `engine/session/openfile_test.go: TestSingleFileLinks`, `kit/ui/src/shell/single.test.ts` |
| 6 | Global config only; frontmatter layout applies | `engine/session/openfile_test.go: TestSingleFileConfig` |
| 7 | Save and watch the one file | `engine/session/openfile_test.go: TestSingleFileSaveAndWatch` |
| 8 | Open Enclosing Folder finds the marker | `apps/zenmark/openfile_test.go: TestOpenEnclosingFolder` |
| 9 | Title | `kit/ui/src/shell/single.test.ts` |
| 10 | Trust keyed by the file's folder | `engine/session/openfile_test.go: TestSingleFileTrust` |

**Edge cases:**
- a symlinked file whose target is in another folder: the root is the
  target's folder, as `Resolve` does today;
- a file deleted while open: the existing "file is gone" state;
- a file over the size limit: `ErrTooLarge`, one line;
- `README.md` at the root of a git repo: opens the repo folder, unless
  `--file` is given;
- opening a file, then a folder: the newest open wins.

## UI

**UI consistency:**
- The sidebar reuses the panel column and the outline list the Inspector
  already shows. No new component.
- Disabled items use the existing greyed state and tooltip.
- Open Enclosing Folder is a `.zen-sidebar-btn`, like Reload and Close.

**States:**
- loading: the existing opening state;
- error: the existing one-line error;
- empty file: the article layout with no text, and the outline's empty text
  "No headings".

Keyboard: ⌘O, ⌘⇧O, and the outline reachable by Tab. Black-and-white
tokens only.

## Security

| Risk | Mitigation | Test |
|---|---|---|
| One file becomes a way to read anything next to or above it | Root is the file's folder; `Resolve` and `Asset` keep the containment and symlink checks; links outside show the notice | `TestOpenFileAssetContainment`, `TestSingleFileLinks` |
| A `.zenmd.yml` next to a downloaded file changes behaviour (export paths, plugins) | Ignored in one-file mode; only the global config applies | `TestSingleFileConfig` |
| A file's plugins run without the folder's trust prompt | Trust is keyed by the file's folder exactly as for a folder | `TestSingleFileTrust` |
| Huge sibling folders slow or hang the open | No walk at all | `TestOpenFileNoWalk` |

## Notes

- **Out of scope:**
  - Save As…, new untitled files, several files in one window (tabs), and
    single-file Export › Website.
  - A `zenmd export site <file.md>` for one page. Export PDF of one file
    already works through the app.
- **Integration (done by the lead at merge):**
  - `docs/wiki/UI.md`: one line on the one-file sidebar state.
  - `examples/demo/README.md`: link the new demo page under Features.
  - `engine/session/demo_test.go`: the expected order.
  - Regenerate the Wails bindings.
- **App Store / sandbox:**
  - A sandboxed build gets access to the chosen file only, not its folder.
    One-file mode therefore reads images next to it only through a security
    scoped bookmark on the folder, which the App Store build asks for on the
    first image. Note this in specs/packaging/appstore.
  - The direct-download build is not sandboxed.
- **Changelog:** `Added: open one markdown file on its own (File › Open…, Finder's Open With, zenmd notes.md) without scanning its folder; Open Enclosing Folder opens the folder when you want it.`

## Prompt

> Implement `specs/app/single-file/spec.md` following the rules in
> `specs/README.md` and docs/wiki/UI.md, in `../zen-worktrees/app-single-file`
> on `feature/app-single-file`.
>
> **Context:**
> - `internal/cli/cli.go` `resolve` maps a `.md` argument to the nearest
>   marker folder, else the file's own folder, and the app opens that folder.
> - `engine/workspace` walks the whole folder on open.
> - `kit/desktop/run.go` already gets Finder's Open With events, through
>   `OpenWith`.
>
> **Steps:**
> 1. Write the Tests table's tests and commit them failing.
> 2. Add `workspace.OpenFile` and `session.OpenFile`.
> 3. Add the CLI flags, then the app's Open…, Open With, Recents and Open
>    Enclosing Folder.
> 4. Add the shell's one-file state, the demo page and the how-to.
>
> **Rules:**
> - Only the Owns paths.
> - No new dependencies, no downloads.
> - Don't launch the app, run `task install`, `task package` or
>   `task release`, start servers, push or touch `main`.
> - Regenerate the bindings as CLAUDE.md says.
>
> **Done when:**
> - `go vet ./... && go test ./...` pass;
> - the frontend `npx tsc --noEmit && npx vitest run` pass;
> - your report lists the Integration lines and the changelog line, and says
>   what needs a look in the real app.
