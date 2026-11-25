# engine/comment-dialect

| | |
|---|---|
| Status | specified |
| Target version | 0.8.0 (MINOR: a new dialect, off unless a folder turns it on) |
| Branch | `feature/engine-comment-dialect` |
| Wave | 62 (after plugins/diagram-files, which also edits `engine/render/plugins.go`; beside plugins/lini, whose paths are disjoint) |
| Depends on | engine/dialects, core/plugins, plugins/diagram-files (`Attrs["arg"]`, merged first) |
| Owns | `builtins/dialects/comment/` (new: `comment.go`, `comment_test.go`), the `KnownScanner` interface in `engine/dialects/dialects.go` (+ `dialects_test.go`), the call in `replaceWithDialects` in `engine/render/plugins.go`, `engine/render/comment_test.go` (new), `docs/public/zenmd/how-tos/write-blocks-other-viewers-hide.md` (new), this spec |

## Purpose

A folder can write any plugin block inside an HTML comment:

```markdown
<!-- toc -->

<!-- mermaid /diagrams/flow.mmd -->

<!-- note Heads up
This only shows in Zenmark.
-->
```

Zenmark draws the block. GitHub, VS Code's preview and every other markdown
viewer see an ordinary comment and show nothing, so a page read in both places
never shows a broken block. It is a dialect: off by default and turned on per
folder with `markdown: dialects: [comment]`, like MyST.

## Behaviour

### What counts as a block

1. With the dialect on, an HTML comment is a block when all of these hold:
   - It starts a line: `<!--` with at most three spaces before it, the way
     CommonMark starts an HTML block.
   - A space follows `<!--`, then a name.
   - The name is a block a plugin of the page owns, or one a turned-off
     plugin owns (behaviour 9).
   - The comment is not inside a code fence, indented code or another
     comment block.
2. Any other comment stays a comment, untouched and without a warning:
   `<!-- TODO fix this -->`, `<!--note-->` (no space), a comment in the middle
   of a paragraph, and a first word that no plugin owns.
3. **One line.** `<!-- name rest -->` on a single line is a block with an
   empty body.
4. **Several lines.** `<!-- name rest` with no `-->` on that line opens a
   block. The body runs until the first line that is `-->` alone (spaces
   allowed around it).
   - A `-->` that ends a line with other text before it is where HTML would
     end the comment. The comment is then not a block: it stays a comment and
     the page gets one warning, "comment: a block must end with --> on its
     own line".
   - A comment that never closes stays as it is, with no block and no
     warning, as CommonMark reads it.

### The parts of a block

5. `rest` is either `{attrs}` (the same braces as a fence:
   `{title="Heads up"}`) or words. Words become `Attrs["arg"]`, and
   `Attrs["title"]` when no title is set, exactly as MyST passes a
   directive's argument. So `<!-- note Heads up -->` titles the note, and
   `<!-- mermaid /diagrams/flow.mmd -->` names the file (plugins/diagram-files).
6. Lines at the top of the body written `:key: value` are options, as in
   MyST: they become `Attrs[key]` (`:file: diagrams/flow.mmd`, `:class:
   dropdown`) and are not part of the body. A bad key is dropped with a
   warning, as MyST does.
7. The body is markdown or the plugin's own text (a diagram's source), as in
   a `:::` container. Nested blocks inside it (`:::`, fences, MyST when on)
   work as they do in a container.

### Rendering

8. A block renders exactly like the same block written as `:::name` or a
   fence: same plugin call, same placeholder, same warnings. A page renders
   the same in the app, the editor preview, `zenmd export site`, print and
   PDF.
9. A block of a turned-off plugin (`plugins: { diagram: false }`) gives the
   usual "turned off" notice, as a fence of that plugin does.
10. With the dialect off, every comment stays a comment, which keeps today's
    behaviour.

### Settings and the CLI (CLI first)

11. The config key already exists: `markdown.dialects` in `.zenmd.yml` takes
    `comment` beside `myst`. Settings › Reader lists "HTML comments" with its
    summary and the same switch as MyST. The Inspector's Build tab lists it
    under Dialects. No new flag is needed: `zenmd export site`, `zenmd check`
    and `zenmd tree` read the folder's `.zenmd.yml`.
12. `Detect` returns false: there is no file that tells a folder was written
    for it, so it never shows the "this folder looks written for…" hint.

## Architecture

- **The scanner needs the block names.** A dialect's `Scan(src)` does not
  know which names plugins own. For this dialect, an unknown first word must
  be left alone, not turned into a code block with a "not supported" note as
  MyST's unknown directives are.
  - `engine/dialects` therefore gains an optional
    `KnownScanner` interface.
  - `replaceWithDialects` calls `ScanKnown(src, known)` for a dialect that
    has it, with `known(name) = st.set.Block(name) != nil ||
    st.set.Denied("block", name) != ""`, and `Scan(src)` otherwise.
  - MyST does not change.
- **`builtins/dialects/comment/comment.go`:**
  - `ID "comment"`, `Name "HTML comments"`, a one-line `Summary`;
  - `Scan` returns nil (it is always called through `ScanKnown`);
  - `ScanKnown` walks the lines, skipping fenced and indented code with the
    same fence rules as `plugins.ScanContainers`, and builds `dialects.Span`s
    with `Name`, `Attrs` and `Body`;
  - for options it reuses `plugins.ParseAttrs`, and MyST's option-line rule
    re-implemented locally, about 20 lines. It does not import MyST.
  - It has no `Rewriter`, `InlineScanner` or `Extensioner`.
- **Registry** (Integration): `builtins/builtins.go` adds `"comment"` to
  `DialectIDs` and the `New` switch. The assembly's `dialects: all` picks it
  up. Then regenerate `cliset/assembly_gen.go`.
- **Order with containers:** spans merge with `:::` spans in
  `replaceWithDialects` as MyST's do (outermost first). A `:::` container
  inside a comment block's body is reached by the body's own render.

## Interfaces

```go
// engine/dialects
// KnownScanner is implemented by a dialect whose syntax only counts for names a plugin owns: the renderer calls
// ScanKnown instead of Scan, and spans for other names are never returned.
type KnownScanner interface {
    ScanKnown(src []byte, known func(name string) bool) []Span
}

// builtins/dialects/comment
func New() dialects.Dialect // ID "comment", Name "HTML comments", Detect false
```

```yaml
# .zenmd.yml (existing key)
markdown:
  dialects: [comment]   # with myst too: [myst, comment]
```

## Tests

| # | Behaviour | Test |
|---|---|---|
| 1–2 | only line-start comments with a space and a known name; others untouched | `comment_test.go: TestOnlyKnownNames`, `TestNoSpaceNoBlock`, `TestMidParagraphStays`, `TestTodoStays`, `TestIndentFourIsCode` |
| 1 | not inside code fences, indented code or another comment block | `TestSkipsCode` (backtick and tilde fences, longer fences, indented) |
| 3 | one-line form | `TestOneLine` |
| 4 | multi-line form; `-->` must be alone; never-closed comment | `TestMultiLine`, `TestCloserNotAlone` (warning, stays a comment), `TestNeverClosed` |
| 5 | words are `arg` and `title`; braces are attrs | `TestArgAndTitle`, `TestBraces`, `TestTitleNotOverridden` |
| 6 | `:key:` option lines, bad key dropped with a warning | `TestOptionLines`, `TestBadOptionKey` |
| 7 | nested `:::` and fences in the body | `engine/render/comment_test.go: TestNestedInBody` |
| 8 | renders like `:::` and a fence | `TestSameAsContainer` (note), `TestSameAsFence` (mermaid with a file, after plugins/diagram-files) |
| 9 | turned-off plugin notice | `TestTurnedOffPlugin` |
| 10 | dialect off: comments untouched | `TestDialectOff` |
| — | `KnownScanner` is used and MyST unchanged | `engine/render/comment_test.go: TestKnownScannerCalled`; existing MyST tests pass |
| 11 | listed in Settings and the build | `engine/session/dialects_test.go` (existing: lists every dialect of the build), extended at Integration |
| 12 | no hint | `TestDetectFalse` |

Edge cases to cover:
- CRLF line endings;
- `<!-- note -->` at the very end of a file with no newline;
- a comment block right after a list or a heading, with no blank line;
- two blocks on consecutive lines;
- a body that holds `<!-- an ordinary comment -->` on one line, which ends
  the block early: test what happens and document it;
- a name with a hyphen (`list-table`);
- a huge comment (1 MB) to check the scan stays linear;
- MyST and comment on together, a page using both.

## UI

None new: the dialect appears in Settings › Reader's existing dialect list,
using the existing row and switch, and in the Build tab's Dialects row.

**UI consistency:** Settings › Reader's dialect rows, unchanged.

## Security

| Risk | Mitigation | Test |
|---|---|---|
| A comment someone meant as a private note runs a plugin | only with the dialect on, which the folder turns on; only names a plugin owns; the docs say so plainly | `TestDialectOff`, `TestOnlyKnownNames` |
| Hidden content: a block GitHub readers never see | by design and documented ("for folders read in both places"); nothing new can run, because a comment block is the same plugin call as a visible one, with the same sanitiser | `TestSameAsContainer` |
| A crafted comment makes the scan slow | one pass over the lines, no backtracking regex; a 1 MB test | `TestLinearScan` |
| Attribute injection | attrs go through `plugins.ParseAttrs` and option keys through the MyST key rule, as every other block's do | `TestBraces`, `TestBadOptionKey` |

## Notes

- **Out of scope:**
  - inline comments (`text <!-- role --> text`);
  - comments inside other comments;
  - the editor highlighting comment blocks differently, since CodeMirror
    shows them as comments;
  - structure checks (`engine/structure`) seeing blocks written as comments.
    The same holds for MyST directives today, and a follow-up covers both.
- **Integration (done by the lead at merge):**
  - `builtins/builtins.go`: import `builtins/dialects/comment`, add
    `"comment"` to `DialectIDs` and `case "comment": return comment.New()`.
  - `go run ./kit/assembly/cmd/gen apps/zenmark`.
  - `docs/public/zenmd/reference/configuration.md`: the `markdown.dialects`
    row lists `comment`.
  - `docs/public/zenmd/glossary/dialect.md`: one line on the comment dialect.
  - `engine/session/dialects_test.go`: expect it in the list.
  - `docs/wiki/PLUGINS.md`, the dialects section: `KnownScanner`.
- **App Store / sandbox:** none.
- **Changelog (Added):** The `comment` dialect: with `markdown: dialects:
  [comment]`, any plugin block can be written as an HTML comment
  (`<!-- toc -->`, `<!-- note Heads up … -->`), so it draws in Zenmark and
  stays hidden on GitHub and in other viewers (engine/comment-dialect).

## Prompt

> Implement `specs/engine/comment-dialect/spec.md` following the rules in
> `specs/README.md`. You work only in the worktree
> `/Users/dragosh/Developer/zen-worktrees/engine-comment-dialect` on branch
> `feature/engine-comment-dialect`, and only in the paths under *Owns*.
>
> **Context:**
> - Read `engine/dialects/dialects.go` and `builtins/dialects/myst/`: MyST's
>   `Scan`, `build` (the option lines and the `arg`/`title` rule) and its
>   tests are the model.
> - `engine/render/plugins.go` `replaceWithDialects` and `replaceSpans` show
>   how dialect spans become plugin blocks. Note that an unknown MyST name
>   becomes code: that is why this dialect gets `KnownScanner`.
> - `plugins.ScanContainers` (`engine/plugins/syntax.go`) has the
>   code-fence skipping to follow.
> - Registering the dialect in `builtins/builtins.go` is Integration. Your
>   render tests can build a dialect list directly.
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
> - your report lists the Integration lines and anything unverified.
