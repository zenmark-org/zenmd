# ui/files-view

| | |
|---|---|
| Status | specified |
| Target version | 0.8.0 |
| Branch | `feature/ui-files-view` |
| Wave | 60 |
| Depends on | ui/navigation, ui/sidebar-menu, app/single-file (all done) |
| Owns | `engine/workspace/filetree.go`, `engine/workspace/filetree_test.go`, `engine/session/files.go`, `engine/session/files_test.go`, `internal/cli/tree.go`, `internal/cli/tree_test.go`, `apps/zenmark/files.go`, `apps/zenmark/files_test.go`, `kit/ui/src/nav/files.ts`, `kit/ui/src/nav/files.test.ts`, `kit/ui/src/nav/viewswitch.ts`, `kit/ui/src/nav/viewswitch.test.ts`, `kit/ui/src/nav/index.ts`, `kit/ui/src/nav/nav.test.ts`, `kit/ui/src/nav/entries.ts`, `kit/ui/src/nav/entries.test.ts`, `kit/ui/src/nav/nav.css`; in `engine/workspace/workspace.go` only the extraction of the skip rule into `skipDir` (below) |

## Purpose

The sidebar shows the folder as it is on disk, as well as the page tree that
the links build. A *Pages | Files* switch picks the view. Files lists the
folders, the markdown pages and the images where they are. It marks the pages
that the Pages tree puts under "Other pages" or hides. `zenmd tree --files`
prints the same listing.

## Behaviour

### The listing (engine)

1. `Workspace.FileTree` walks the folder with the same rules as the page
   scan, through one shared `skipDir`:
   - folders starting with `.` are skipped, and so are `node_modules`,
     `vendor` and the top-level `exports`;
   - a symlink counts only when it resolves inside the root;
   - the walk visits at most `MaxEntries` entries;
   - it is cancelled with its context.
2. It lists folders, markdown files (`.md`, `.markdown`) and images (the
   servable asset types: png, jpg, jpeg, gif, webp, svg, avif). Other files
   are left out. A folder that holds none of these, at any depth, is left
   out too.
3. Each level sorts folders first, then files. Names sort case-insensitively
   in natural order (`2-intro` comes before `10-setup`). Paths are
   workspace-relative with `/`. A folder's path ends in `/` (`guides/`), so it
   never clashes with a page path.
4. At most `MaxFileTreeEntries` (5000) files are listed.
   - Past that, the result says how many more were left out (`Omitted`), and
     the folders that hold them are still listed.
   - Folders, pages and images all count toward the cap. Pages are never
     cut while images remain: images are dropped first.
5. In one-file mode (`Single`), the tree is the one file. No folder is
   walked.

### Its marks (session)

6. `Session.FileTree` adds the Pages tree's view of each page to the listing:
   - `nav: "tree"` when the page is in the Pages tree, outside "Other pages";
   - `"other"` when it is under "Other pages";
   - `"hidden"` when it is in neither (`nav.hidden`, or under a hidden page).

   Images and folders have no mark. The page title is the same title the
   Pages tree shows. Images and folders show their file name.
7. `Session.FileTree` reads the index that the session already built. It
   never renders or reads a page again.

### The command (CLI first)

8. `zenmd tree [folder]` prints the Pages tree, indented two spaces per
   level: titles, then each path in parentheses. "Other pages" prints as a
   group line.
9. `zenmd tree --files [folder]` prints the Files listing:
   - names with a trailing `/` for folders;
   - a page's mark after it: `(other)`, or `(hidden)`;
   - a last line `… N more files not shown` when entries were left out.
10. `--json` prints either tree as JSON: the `NavNode` list, or the
    `FileTree` value. A folder that cannot be opened exits 1 with the usual
    `zenmd:` message. An unknown flag exits 2.
11. The view is a per-machine choice, not a folder option, so there is no
    config key. The CLI's `--files` is the only option.

### The sidebar (app)

12. **The switch.** With a folder open, a two-option switch, *Pages* and
    *Files*, sits at the top of the sidebar, under the header.
    - Pages is the default.
    - The choice is remembered per folder (`zen.nav.view:<root>`) and comes
      back when the folder is opened again.
    - In one-file mode the switch is not shown.
13. **The rows.** Files uses the same tree component as Pages (`renderTree`):
    the same rows, chevrons, indentation, keyboard, focus and current-page
    mark.
    - Folders are rows of kind `group`, collapsed by default except the top
      level.
    - The folders of the open page are expanded when the view first shows.
    - Expanded folders are remembered per folder
      (`zen.files.expanded:<root>`), separately from the Pages tree.
14. **The marks.** A page marked `other` or `hidden` shows a muted badge after
    its title ("other", "hidden"), using the tree's existing badge. The badge
    has a tooltip: "Under Other pages in the Pages view", or "Hidden from the
    Pages view".
15. **Opening.**
    - **Pages:** Enter or a click opens the page as in Pages, and the current
      page is marked.
    - **Images:** Enter or a click opens the image in the image view over the
      window (ui/image-zoom). Esc closes it and puts the focus back on the
      row.
    - **Folders:** Enter or a click toggles the folder.
16. **The context menu.** It is built by the same `menuEntries`.
    - **Page:** Open, Open in Editor, New Page Here… (in the page's folder),
      Copy Path, Copy Link, Reveal in Finder.
      - A `hidden` page gets **Show in Sidebar** (`SetNavHidden(path,
        false)`).
      - A page in the tree gets Hide from Sidebar, as in Pages.
      - Move Up/Down and dragging are left out, because the disk order is not
        the nav order.
    - **Folder:** New Page Here…, Copy Path, Reveal in Finder.
    - **Image:**
      - Open;
      - Copy Path;
      - Copy Link, which copies `![](<path relative to the open page>)`,
        URL-encoded the way Copy Link encodes page links;
      - Reveal in Finder.
    - Items the host lacks are left out, as now.
17. **Staying current.** On `workspace:changed` (save, new page, reorder,
    hide, the watcher, Reload ⌘R), the Files view fetches the listing again
    when it is the view shown. The focus and the expanded folders are kept.
    - The view is fetched only while it is shown.
    - Switching to Files fetches it.
    - A failed fetch shows the error in the sidebar status line (the
      existing `role="alert"`) and keeps the last listing.
18. **States.**
    - **Loading:** the first fetch of a folder shows "Loading files…" in the
      left column's muted empty-text style.
    - **Empty:** "No pages or images in this folder".
    - **Too many:** the last row, a non-interactive muted line, reads
      "N more files not shown".
19. **Read-only builds** (no `editor` feature): the Files view is there, and
    its menu has no editing items (New Page Here…, Show/Hide), as in Pages.

## Architecture

- **Engine walk.** `engine/workspace/filetree.go` walks the folder. Its skip
  rule is the one `FilesContext` uses, moved into
  `skipDir(name string, atRoot bool) bool` in `workspace.go` so the two can
  never drift apart. That extraction is the only change to `workspace.go`.
- **Session marks.** `engine/session/files.go` calls it under the session
  lock, with the session's context. It then adds the marks from `s.tree`: a
  page under the `group` node "Other pages" is `other`, a page not found is
  `hidden`, and any other page is `tree`. It does not cache. The walk is
  cheap next to the index, and it only runs while Files is shown.
- **CLI.** `internal/cli/tree.go` opens a headless session (as `check` does),
  then prints `Workspace().Tree` or `FileTree()`.
- **App binding.** `apps/zenmark/files.go` adds the Wails binding
  `(z *Zen) FileTree() (session.FileTree, error)`.
- **Frontend.** `kit/ui/src/nav/files.ts` turns a `FileTree` into `NavNode`s
  for `renderTree`: folders become `kind: "group"`, pages `"page"`, and images
  a new `"file"` kind that tree.ts treats like a page with no
  `aria-current`. It passes the marks as `badges`. `viewswitch.ts` is the
  switch. `nav/index.ts` mounts the switch, keeps one tree host per view and
  shows one at a time.
- **Opening images.** Images open through the shell's image zoom.
  `files.ts` builds an `<img src="/_ws/<path>">` and hands it to `ImageZoom.open`,
  which the reader's page feature already installs. The host exposes it as
  `ctx.zoomImage?(src: string, alt: string)` (Integration).

## Interfaces

```go
// engine/workspace
const MaxFileTreeEntries = 5000

type FileEntry struct {
    Name     string      `json:"name"`      // "intro.md", "guides"
    Path     string      `json:"path"`      // "guides/intro.md"; folders end in "/"
    Kind     string      `json:"kind"`      // "folder" | "page" | "image"
    Children []FileEntry `json:"children,omitempty"`
}

func (w *Workspace) FileTree(ctx context.Context) (entries []FileEntry, omitted int, err error)
func skipDir(name string, atRoot bool) bool // shared with FilesContext

// engine/session
type FileNode struct {
    Name     string     `json:"name"`
    Title    string     `json:"title"`         // page title, else Name
    Path     string     `json:"path"`
    Kind     string     `json:"kind"`          // "folder" | "page" | "image"
    Nav      string     `json:"nav,omitempty"` // pages: "tree" | "other" | "hidden"
    Children []FileNode `json:"children,omitempty"`
}
type FileTree struct {
    Nodes   []FileNode `json:"nodes"`
    Omitted int        `json:"omitted"`
}
func (s *Session) FileTree() (FileTree, error) // errNoWorkspace without a folder

// apps/zenmark
func (z *Zen) FileTree() (session.FileTree, error)
```

```text
zenmd tree [folder] [--files] [--json]
```

```ts
// kit/ui/src/host.ts (Integration)
export type FileNode = { name: string; title: string; path: string; kind: "folder" | "page" | "image"; nav?: "tree" | "other" | "hidden"; children?: FileNode[] };
export type FileTree = { nodes: FileNode[]; omitted: number };
// on Api:
fileTree?(): Promise<FileTree>;
// NavNode.kind gains "file" (images in the Files view)

// kit/ui/src/nav/files.ts
export function toNavNodes(t: FileTree): { tree: NavNode[]; badges: Record<string, string>; titles: Record<string, string> };
export function folderChain(path: string): string[]; // "a/b/c.md" -> ["a/", "a/b/"]

// kit/ui/src/nav/viewswitch.ts
export type NavView = "pages" | "files";
export function renderViewSwitch(host: HTMLElement, get: () => NavView, set: (v: NavView) => void): { sync(): void };
export function loadView(root: string): NavView;
export function saveView(root: string, v: NavView): void;
```

## Tests

| # | Behaviour | Test |
|---|---|---|
| 1 | skip rules shared, symlink outside left out, entry limit, cancel | `engine/workspace/filetree_test.go: TestFileTreeSkips`, `TestFileTreeSymlinkOutside`, `TestFileTreeEntryLimit`, `TestFileTreeCancel`, `TestSkipDirMatchesScan` |
| 2 | pages and images only; empty folders left out | `TestFileTreeKinds`, `TestFileTreeDropsEmptyFolders` |
| 3 | folders first, natural case-insensitive order, `/` paths, folder paths end in `/` | `TestFileTreeOrder` |
| 4 | cap at 5000, images dropped before pages, `omitted` counted | `TestFileTreeCap` (with the cap lowered) |
| 5 | one-file mode lists the one file | `TestFileTreeSingle` |
| 6 | marks tree / other / hidden (`nav.hidden` and under a hidden page); titles | `engine/session/files_test.go: TestFileTreeMarks`, `TestFileTreeTitles` |
| 7 | no page is rendered again | `TestFileTreeUsesIndex` (a counting renderer) |
| 8 | `zenmd tree` prints the Pages tree | `internal/cli/tree_test.go: TestTreePages` |
| 9 | `--files` prints names, marks and the omitted line | `TestTreeFiles`, `TestTreeFilesOmitted` |
| 10 | `--json` both ways; bad folder exits 1; unknown flag exits 2 | `TestTreeJSON`, `TestTreeErrors` |
| 11 | no config key: `tree` takes no `-o` or config options | `TestTreeFlags` |
| 12 | switch shown with a folder, hidden in one-file mode, remembered per folder | `kit/ui/src/nav/viewswitch.test.ts: radiogroup, arrows, persists per root`; `nav.test.ts: switch hidden in single mode` |
| 13 | same tree component; folders collapsed except the top; current page's folders open; separate expanded store | `files.test.ts: toNavNodes`, `folderChain`; `nav.test.ts: files view expands the open page's folders`, `files expanded state is its own` |
| 14 | badges for other and hidden with their tooltips | `files.test.ts: badges`; `nav.test.ts: badge tooltip` |
| 15 | page opens, image opens the zoom and focus returns, folder toggles | `nav.test.ts: files view opens pages`, `opens images in the zoom`, `folder row toggles` |
| 16 | menus for page, hidden page, folder, image; no Move or drag in Files; image Copy Link | `entries.test.ts: files page menu`, `hidden page has Show in Sidebar`, `folder menu`, `image menu`, `image link is relative and encoded` |
| 17 | refetch on workspace:changed only while shown; focus and expanded state kept; error in the status line | `nav.test.ts: refetches when shown`, `not fetched while Pages is shown`, `keeps focus`, `fetch error shows in status` |
| 18 | loading, empty and too-many states | `nav.test.ts: files loading`, `files empty`, `files omitted line` |
| 19 | read-only build: no editing items | `entries.test.ts: read-only files menu` |
| — | the Wails binding returns the session's tree | `apps/zenmark/files_test.go: TestFileTreeBinding` |

Edge cases to cover:
- a folder with only images;
- `README.md` and `index.md` in the same folder;
- names with spaces, `#` and non-ASCII (in Copy Link and in the CLI output);
- a page with the same name as a folder (`guides.md` and `guides/`);
- an upper-case `.MD`;
- a `.markdown` file;
- an image symlinked from outside the root (left out);
- 5001 files.

## UI

**UI consistency:** Files reuses existing patterns:
- the sidebar tree, through the same `renderTree`, rows, chevrons, badges,
  focus ring and current-page mark (docs/wiki/UI.md "Sidebar");
- the sidebar context menu (ui/sidebar-menu);
- the image view over the window (ui/image-zoom);
- the status line under the tree.

The switch reuses the Settings segment control: `role="radiogroup"`, buttons
with `role="radio"`, arrow keys move between them, and the same classes
(`.zen-settings-segment`, `.zen-settings-segment-option`). Those styles are
global, so the switch has no look of its own.

- **Placement:** under the 44px header, full width with the sidebar's
  padding. It is in the tab order between the header buttons and the tree.
- **States:** loading, empty and too many, as in Behaviour 18, all in the
  muted empty-text style of the left column.
- **Keyboard:**
  - the switch: ←/→, Space and Enter;
  - the tree: the same keys as Pages (↑/↓, ←/→, Enter, Home/End, the
    context-menu key, ⇧F10);
  - ⌥↑/⌥↓ do nothing in Files.
- **Accessibility:**
  - the tree is labelled "Files" (Pages stays "Pages");
  - a folder row's accessible name is the folder name;
  - images are announced as "image";
  - badges are text.
- **Tokens:** black and white tokens only. `prefers-reduced-motion` is
  respected: there are no new transitions.

## Security

| Risk | Mitigation | Test |
|---|---|---|
| A listing leaks files outside the folder | the walk is rooted at the workspace root; a symlink counts only when `isFile` resolves it inside the root | `TestFileTreeSymlinkOutside` |
| Hidden folders (`.git`, `.env` folders) are exposed | the same `skipDir` as the scan; dot folders are skipped and dot files are never markdown or images | `TestSkipDirMatchesScan`, `TestFileTreeSkips` |
| A huge folder stalls the app | `MaxEntries` for the walk, `MaxFileTreeEntries` for the result, cancellable context, fetched only while shown | `TestFileTreeEntryLimit`, `TestFileTreeCap`, `TestFileTreeCancel` |
| File names inject markup | rows are drawn with `textContent` only (tree.ts), and the image `alt` and `src` are set as properties | `files.test.ts: names are text` |
| Image src points outside the folder | built only from listed paths as `/_ws/<encoded path>`, which the asset route checks again (`Workspace.Asset`) | `files.test.ts: image src is /_ws/ and encoded` |

## Notes

- **Out of scope:**
  - moving, renaming or deleting files from the Files view (links would
    need rewriting: a later spec);
  - listing other file types (PDF, code);
  - a View-menu item or a shortcut for the switch;
  - the watcher seeing image changes (Reload ⌘R shows them).
- **Integration (done by the lead at merge):**
  - `internal/cli/cli.go`: add
    `case flags && first && arg == "tree": return runTree(args[i+1:], cwd, stdout, stderr)`
    next to `check`, and add `tree` to the help text.
  - `kit/ui/src/host.ts`: add the `FileNode` and `FileTree` types and
    `fileTree?(): Promise<FileTree>` to `Api`, and add `"file"` to
    `NavNode.kind`.
  - `kit/ui/src/nav/tree.ts`: a `kind: "file"` row opens like a page but
    never gets `aria-current`. One line in `build`: the `aria-current` check
    is for `n.kind === "page"` only (already so). Check that `activate`
    calls `onOpen` for `"file"`.
  - `kit/ui/src/shell/types.ts` (the feature context): add
    `zoomImage?(src: string, alt: string): void`. It is wired in the shell
    to the reader's `ImageZoom`.
  - `apps/zenmark/frontend/src/api/fake.ts`: `fileTree()` from the sample
    workspace, with one image and one hidden page. In `wails.ts`, map it to
    the `FileTree` binding.
  - Regenerate the Wails bindings.
  - `docs/wiki/UI.md` "Sidebar": one sentence on the Pages | Files switch and
    the marks.
  - `docs/public/zenmd/`: the CLI reference gets `zenmd tree`.
- **App Store / sandbox:** none. The folder is already granted, and the walk
  stays inside it.
- **Changelog (Added):** The sidebar has a Pages | Files switch: Files shows
  the folder as it is on disk, its folders, pages and images, and marks the
  pages the Pages view puts under Other pages or hides. Images open in the
  image view. `zenmd tree` prints either tree (`--files`, `--json`).

## Prompt

> Implement `specs/ui/files-view/spec.md` following the rules in
> `specs/README.md`. You are in the worktree
> `../zen-worktrees/ui-files-view` on `feature/ui-files-view`.
>
> **Context:** the sidebar's Pages tree is built from links by
> `engine/index` (`BuildWithMeta`) and drawn by `kit/ui/src/nav/tree.ts`.
> The folder scan is `Workspace.FilesContext` in
> `engine/workspace/workspace.go`. Its folder skip rule moves into
> `skipDir` and is shared with the new `FileTree` walk. The marks come from
> the session's `s.tree`: under the "Other pages" group means `other`, and
> not found means `hidden`. Reuse `renderTree`, `menuEntries`, the badges and
> the Settings segment control; do not write a second tree or menu. The CLI
> command follows `internal/cli/check.go`. Its parsing goes in the
> `case … arg == "tree"` line of `cli.go`, which is an Integration line:
> report it instead of editing `cli.go`.
>
> **Rules:**
> - only your Owns paths;
> - tests first, committed failing;
> - no new dependencies or downloads;
> - never launch the app, run `task install`, `task package` or
>   `task release`, start servers, push, or touch `main`;
> - run every go command with a throwaway `HOME` and `XDG_CONFIG_HOME`,
>   keeping the real `GOMODCACHE`, `GOPATH` and `GOCACHE`.
>
> **Done when:**
> - every test in the table passes;
> - `go vet ./... && go test ./engine/... ./internal/... ./apps/...` passes;
> - `cd apps/zenmark/frontend && npx tsc --noEmit && npx vitest run` passes,
>   after you copy the bindings from the main checkout;
> - your report lists the Integration lines, including `cli.go`, and
>   anything unverified (how it looks in the app).
