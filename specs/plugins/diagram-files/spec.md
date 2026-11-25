# plugins/diagram-files

| | |
|---|---|
| Status | done |
| Target version | 0.8.0 (MINOR: new ways to write a block; existing blocks render as before) |
| Branch | `feature/plugins-diagram-files` |
| Wave | 61 |
| Depends on | plugins/diagram, core/plugins (the plugin API), plugins/myst-builtins (`include`, `csv-table :file:`) |
| Owns | `engine/plugins/files.go` (new), `engine/plugins/files_test.go` (new), the `ReadFile` field in `engine/plugins/plugins.go` (`Context`), `engine/render/plugins.go` (wiring `ReadFile`), `engine/render/nodes.go` (the fence argument), `engine/render/files_test.go` (new), `builtins/plugins/diagram/diagram.go`, `builtins/plugins/diagram/diagram_test.go`, `docs/public/zenmd/reference/plugins/diagram.md`, `examples/demo/plugins/diagram.md`, `examples/demo/plugins/diagrams/flow.mmd` (new), this spec |

## Purpose

A mermaid diagram, and later a Lini figure (plugins/lini), can live in its own
file and be drawn on a page. Writing it in the fence's first line or in the
block itself points at a file of the open folder:

````markdown
```mermaid {file=diagrams/flow.mmd}
```

```mermaid diagrams/flow.mmd
```

```mermaid
:file: diagrams/flow.mmd
```
````

A path that starts with `/` is from the open folder's root
(`/diagrams/flow.mmd`). Any other path is from the page's own folder, as a
link is.

## Behaviour

### Reading a file of the folder (engine)

1. `plugins.Context` gains `ReadFile(path string) ([]byte, error)`. It reads
   a file of the open folder for the page being rendered:
   - a path that starts with `/` is from the folder's root;
   - any other path is from the page's folder, and `..` may climb only as far
     as the root;
   - the path is cleaned the way a link is.
2. It refuses:
   - a path that leaves the folder after cleaning;
   - a symlink whose target is outside the folder, a dangling one included;
   - anything that is not a regular file;
   - a file over 1 MB;
   - a file with a NUL byte (not text);
   - a path with `\`, NUL or a newline, or longer than 1,024 bytes.

   Each refusal is an error whose message names the path as the page wrote
   it, never the absolute path on disk: "diagrams/flow.mmd was not found in
   the folder", "… is outside the folder", "… is larger than 1 MB", "… is not
   a text file".
3. `ReadFile` is set for block and inline hooks while a page renders with a
   folder open. It is nil without a folder (a one-file window, `zenmd` on a
   lone file): a plugin then says "needs an open folder".

### Where the path is written (engine and plugin)

4. **The fence's first line.** In ```` ```mermaid diagrams/flow.mmd ````, the
   text after the language that does not start with `{` reaches the plugin as
   `Attrs["arg"]`, the same attribute MyST already gives a directive's
   argument (```` ```{mermaid} diagrams/flow.mmd ````). This holds for every
   plugin-owned fence, and a plugin that does not use `arg` ignores it, as it
   does under MyST today.
5. **Braces.** ```` ```mermaid {file=diagrams/flow.mmd} ```` gives
   `Attrs["file"]`, as now.
6. **A `:file:` line.** A block whose body starts with `:file: <path>` gives
   the path, and that line is not part of the source. Blank lines before it
   are allowed. Under MyST, ```` ```{mermaid} ```` with `:file:` already
   arrives as `Attrs["file"]`.
7. `plugins.FileRef(b Block) (path, body string, ok bool)` finds the path in
   that order: `Attrs["file"]`, then a leading `:file:` line, then
   `Attrs["arg"]`. It returns the body without the `:file:` line. `ok` is
   false when there is no path.

### The diagram plugin

8. A mermaid fence or `:::diagram` block with a path draws the file's text
   instead of the block's own. The output is the same placeholder as an
   inline diagram, with the file's text escaped in it. The diagram therefore
   works in the app, the editor preview, `zenmd export site` and portable
   sites alike, with no script change.
9. **Problems with the file:**
   - A file that cannot be read keeps the block on the page as an error box:
     `diagram: <message>` (behaviour 2). It also gives one page warning, so
     `zenmd check`, Problems and the build's warning lines list it.
   - When the block has its own text too, the file wins and one warning says
     "diagram: the block's own text is ignored because it names a file".
10. An empty file draws nothing and warns "diagram: <path> is empty".
11. The file's name is not checked. `.mmd` and `.mermaid` are the usual ones,
    and the docs say so.
12. Settings › Plugins gets one more snippet, "A diagram from a file", with
    the `:file:` form and a short note on `/`.

### Lini

13. plugins/lini uses the same `FileRef` and `ReadFile` with the same
    messages (`lini: …`). That spec carries the behaviour and its tests, and
    runs after this one.

## Architecture

- **`engine/plugins/files.go`** holds `FileRef` and `ResolvePath(page, p
  string) (string, error)`, the pure part of the path rule (`/` from the root,
  else from the page's folder, `..` capped at the root, the forbidden bytes).
  It has no file access, so it is tested on its own.
- **`engine/render/plugins.go`** sets `c.ReadFile` in `state.ctx` next to
  `PageTitle`. It joins `ResolvePath`'s result to the root from the resolver
  (`rooter`), resolves symlinks with `filepath.EvalSymlinks`, checks that the
  real path is inside the real root, opens the file, checks it is regular,
  and reads at most 1 MB + 1 bytes. This is the same check as
  `include.readFile`, in one place.
- **`engine/render/nodes.go`**, in `fenced`, puts the rest of the info string
  in `attrs["arg"]` when it is not empty and does not start with `{`.
- **`builtins/plugins/diagram/diagram.go`** calls `plugins.FileRef(b)`, then
  `ctx.ReadFile`, and renders the text as before.
- **Watcher:** a changed `.mmd` file does not reload the page by itself; ⌘R
  does. See Out of scope.

## Interfaces

```go
// engine/plugins (plugins.go): a new Context field
ReadFile func(path string) ([]byte, error) // a text file of the open folder; "/x" from the root, "x" from the page's folder; nil without a folder

// engine/plugins/files.go
// FileRef finds the file a block names: Attrs["file"], else a leading ":file: <path>" line of the body, else
// Attrs["arg"]. body is the block's text without that line. ok is false when the block names no file.
func FileRef(b Block) (path, body string, ok bool)
// ResolvePath turns a path written on page into a folder-relative one: "/a/b" -> "a/b"; "b" on "docs/p.md" ->
// "docs/b"; "../b" on "docs/p.md" -> "b"; an error when it leaves the folder or holds \, NUL or a newline.
func ResolvePath(page, p string) (string, error)
const MaxReadFile = 1 << 20
```

```text
CLI and config: none. The path is written in the page; there is no flag or
.zenmd.yml key for it.
```

`docs/wiki/PLUGINS.md` gets the field and its paragraph. That edit is part of
this spec's commit, because the contract changes before the build.

## Tests

| # | Behaviour | Test |
|---|---|---|
| 1 | `/` from the root, else from the page's folder; `..` capped | `engine/plugins/files_test.go: TestResolvePath` (table: `/a.mmd`, `a.mmd` on `docs/p.md`, `../a.mmd`, `../../a.mmd` refused, `/../a` refused, `a\b`, NUL, newline, 1,025 bytes) |
| 2 | refusals and their messages, which name only the written path | `engine/render/files_test.go: TestReadFileOutsideByLink`, `TestReadFileDanglingLink`, `TestReadFileNotRegular`, `TestReadFileTooLarge`, `TestReadFileBinary`, `TestReadFileMessagesNoAbsolutePath` |
| 3 | nil without a folder | `TestReadFileNilWithoutFolder` |
| 4 | the fence argument reaches `Attrs["arg"]`, braces are unchanged | `engine/render/files_test.go: TestFenceArgument`, `TestFenceBracesUnchanged`; existing fence tests pass |
| 5–7 | `FileRef` order and the `:file:` line | `files_test.go: TestFileRefOrder`, `TestFileRefLineStripped`, `TestFileRefBlankLinesBefore`, `TestFileRefNone` |
| 8 | the file's text is drawn, escaped, in the same placeholder | `builtins/plugins/diagram/diagram_test.go: TestFileThreeForms`, `TestFileRootPath`, `TestFileEscaped` |
| 8 | it works under MyST too | `TestFileMyST` (`{mermaid}` with `:file:` and with an argument) |
| 9 | a missing file is an error box and a warning; own text plus file warns | `TestFileMissing`, `TestFileAndBody` |
| 10 | an empty file warns | `TestFileEmpty` |
| 12 | the new snippet renders | `TestHelpSnippets` (existing, extended) |
| — | a built site carries the drawn file | `engine/session` demo golden (the demo page uses `diagrams/flow.mmd`), regenerated at merge |

Edge cases to cover:
- a path with spaces, written in braces as `{file="my flows/a.mmd"}`;
- a non-ASCII name;
- `.MMD` in capitals;
- `/` alone or a folder path, which is "not a file";
- a page at the root and a page two folders down;
- a fence argument that is more than one word;
- `:file:` with no path;
- a CRLF file.

## UI

None: nothing new on screen. A diagram from a file looks exactly like one
written in the page. Errors use the existing `.zen-plugin-error` box and page
warnings (Problems).

## Security

| Risk | Mitigation | Test |
|---|---|---|
| A page reads a file outside the folder (`/etc/passwd`, `../../`) | `/` means the folder's root, never the disk's; `..` is capped at the root; symlinks are resolved and must stay inside the real root | `TestResolvePath`, `TestReadFileOutsideByLink`, `TestReadFileDanglingLink` |
| A page reads secrets inside the folder (`.env`) into a published site | the file's text goes into a diagram the author asked for; dot folders are not special-cased (the author owns the folder), but a NUL byte refuses binaries | `TestReadFileBinary`; documented in the diagram docs |
| A huge or special file (FIFO, device) stalls the render | regular files only, 1 MB cap, a capped read | `TestReadFileNotRegular`, `TestReadFileTooLarge` |
| Error messages leak the disk layout into a published site | messages name the path as written in the page only | `TestReadFileMessagesNoAbsolutePath` |
| The file's text injects markup | it is escaped into the `<pre>` exactly as inline text is; mermaid renders it in strict mode | `TestFileEscaped` |

## Notes

- **Out of scope:**
  - reloading the page when only the `.mmd` file changes (the watcher looks
    at markdown and images; ⌘R reloads);
  - moving `include` and `csv-table :file:` onto `ReadFile`. Both refuse a
    leading `/` today and keep doing so. A follow-up issue moves them, so
    every `:file:` follows the same `/` rule;
  - other plugins (code blocks, `literalinclude`).
- **Integration (done by the lead at merge):**
  - `engine/session/testdata/demo-build.golden`: regenerate.
  - `specs/plugins/lini/spec.md`: move it to wave 62, and add its file
    behaviour and tests, mirroring 8–10 here with `lini:` messages (the lead
    edits it before lini's wave).
  - File the follow-up issue for `include` and `csv-table`.
- **App Store / sandbox:** none. It reads files inside the folder the user
  already opened.
- **Changelog (Added):** A mermaid diagram can come from a file of the
  folder: ```` ```mermaid {file=diagrams/flow.mmd} ````, ```` ```mermaid
  diagrams/flow.mmd ````, or a `:file:` line in the block. A path starting
  with `/` is from the folder's root, any other from the page's folder
  (plugins/diagram-files).

## Prompt

> Implement `specs/plugins/diagram-files/spec.md` following the rules in
> `specs/README.md`. You work only in the worktree
> `/Users/dragosh/Developer/zen-worktrees/plugins-diagram-files` on branch
> `feature/plugins-diagram-files`, and only in the paths under *Owns*.
>
> **Context:**
> - `include.readFile` (`builtins/plugins/include/include.go`) is the file
>   check to copy into `engine/render` as `Context.ReadFile`. The difference:
>   a leading `/` means the folder's root, and `..` is allowed while it stays
>   inside the folder.
> - `Context.PageTitle` in `engine/render/plugins.go` is how a resolver-backed
>   helper is wired into `state.ctx`.
> - `fenced` in `engine/render/nodes.go` already parses `{…}` after the
>   language.
> - MyST (`builtins/dialects/myst/myst.go`, `build`) already sets
>   `Attrs["arg"]` and `Attrs["file"]`.
> - `docs/wiki/PLUGINS.md` already describes `ReadFile`.
>
> **Rules:**
> - tests first, committed failing;
> - no new dependencies or downloads;
> - never launch the app, run `task install`, `task package` or
>   `task release`, start servers, push, or touch `main`;
> - run go commands with a throwaway `HOME` and `XDG_CONFIG_HOME`, keeping
>   the real `GOMODCACHE`, `GOPATH` and `GOCACHE`.
>
> **Done when:**
> - every test in the table passes, and the edge cases are covered;
> - `go vet ./... && go test ./...` passes;
> - `cd apps/zenmark/frontend && npx tsc --noEmit && npx vitest run` passes;
> - your report lists the Integration lines and anything unverified.
