# plugins/lini

| | |
|---|---|
| Status | specified |
| Target version | 0.8.0 (MINOR: a new built-in plugin; nothing changes for existing pages) |
| Branch | `feature/plugins-lini` |
| Wave | 62 |
| Depends on | plugins/diagram (the pattern it copies), plugins/table (the `zen:overlay-open` event, merged), plugins/diagram-files (`FileRef`, `Context.ReadFile`) |
| Owns | `builtins/plugins/lini/` (new: `lini.go`, `lini_test.go`, `assets/lini.js`, `assets/lini.css`, `assets/lini.test.ts`; the vendored files and `VENDOR.md` are added by the lead before the wave), `docs/public/zenmd/reference/plugins/lini.md` (new), `examples/demo/plugins/lini.md` (new), this spec |

## Purpose

A ```` ```lini ```` block draws a [Lini](https://lini.rs/) figure: a flow,
a tree or mindmap, a sequence, a chart, a floor plan or a circuit, written as
text, next to the mermaid diagrams Zenmark already draws. Lini compiles to an
SVG whose colours follow the light or dark theme by themselves.

## Behaviour

### The block (Go)

1. The `lini` built-in owns the block name `lini`. A fenced ```` ```lini ````
   block and a `:::lini` container render the same way. It is **off by
   default** (`Default: false`) until it has been in a release, and turned on
   with `plugins: { lini: {} }` in `.zenmd.yml` or a page's frontmatter, like
   any built-in.
2. Go emits a placeholder that holds the source as escaped text:
   `<div class="zen-lini"><pre>…</pre></div>`. The page without scripts (a
   built site with JavaScript off, print before the script ran, a feed)
   shows the source as a code block.
3. The block takes one attribute, `file`. An unknown attribute gives one page
   warning through `Context.Warn` ("lini: unknown option :x:"), as callout
   does.
3a. **From a file**, as mermaid does (plugins/diagram-files):
    - The three forms are ```` ```lini {file=figures/a.lini} ````,
      ```` ```lini figures/a.lini ```` and a leading `:file:` line, read through
      `plugins.FileRef` and `ctx.ReadFile`.
    - `/` means the folder's root; any other path is from the page's folder.
    - The file's text goes into the same placeholder, so it is drawn as an
      inline figure is.
    - A missing or refused file gives the error box `lini: <message>` and a
      page warning.
    - Text in the block as well as a file gives one warning ("the block's own
      text is ignored because it names a file").
    - An empty file warns.

### Drawing (browser)

4. `assets/lini.js` loads the vendored compiler once per page, the first time a
   lini block mounts, and compiles each block's source to SVG. Blocks that are
   not on the page never load it.
5. The SVG replaces the `<pre>` inside `.zen-lini`, wrapped in
   `.zen-lini-svg`. It is inserted only after the sanitiser of behaviour 11.
6. A source that does not compile keeps its source visible and gets an error
   box under it, `.zen-plugin-error` with "lini: <first line of the message>",
   and one `console.warn` line, as a broken mermaid diagram does.
7. A compiler that cannot load (missing file, WebAssembly refused) leaves every
   block as source and shows one error box per block: "lini: could not load the
   Lini compiler". The page keeps working.
8. The colours are Lini's own CSS variables with `light-dark()`. The plugin
   sets `color-scheme` on `.zen-lini-svg` from the app's theme
   (`data-theme="dark"`/`"light"` on the document, else the system), so a
   figure follows Settings › Appearance › Theme and not only the system.
9. A wide figure scrolls sideways inside `.zen-lini-svg`
   (`overflow-x: auto`), never past the reading column.
10. It works in the app, in the editor's preview and in sites made by
    `zenmd export site`, which copies every file of the plugin
    (`copyExtras`). In a **portable** site (`--portable`, opened from
    `file://`) browsers refuse to fetch the WebAssembly file: the blocks stay
    as source with the message of behaviour 7. This is documented, not fixed.

### Safety of the output

11. Before insertion the SVG is parsed with `DOMParser` as `image/svg+xml`
    and cleaned:
    - `script`, `foreignObject`, `iframe`, `object`, `embed`, `use` with an
      external reference, and `animate`/`set` that target `href` are removed;
    - every `on*` attribute is removed;
    - `href`/`xlink:href` are kept only when they start with `#`;
    - a parse error counts as a compile error (behaviour 6).

    The cleaned root `<svg>` is imported into the page with `importNode`,
    never through `innerHTML`.
12. The compiler is loaded only from the plugin's own asset URL
    (`ctx.assetUrl`), never from the open folder or the network.

### The page's security policy (Integration)

13. The app's pages and built sites allow WebAssembly to compile:
    `'wasm-unsafe-eval'` is added to `script-src` in `frontendCSP`
    (apps/zenmark/service.go) and in `csp()` (engine/site/shell.go). Nothing
    else changes in the policy. `'unsafe-eval'` is never added. Files of the
    open folder keep `Content-Security-Policy: sandbox`, and a workspace
    `.wasm` keeps `text/plain` (engine/session/assets.go): only the plugin's
    bundled file can be compiled.

### Help and docs

14. Settings › Plugins shows the plugin with a summary and three snippets: a
    flow, a mindmap and a sequence, each a lini block that compiles.
15. `docs/public/zenmd/reference/plugins/lini.md` describes the block, turning
    it on, the theme, the portable-site limit and a link to Lini's own
    language docs. `examples/demo/plugins/lini.md` shows the three snippets.

## Architecture

- **Go** (`builtins/plugins/lini/lini.go`) is the diagram plugin's shape:
  `Manifest`, a `RenderBlock` that escapes the body, and `Assets`/`Entry`
  from an embedded `assets/` folder (`lini.js`, `lini.css`, and the vendored
  `lini_wasm.js` and `lini_wasm_bg.wasm`, or whatever names the package ships;
  see `VENDOR.md`).
- **Browser** (`assets/lini.js`) is the diagram plugin's mount:
  1. find `.zen-lini`;
  2. `loadLini(url)` imports the vendored ES module from `ctx.assetUrl(…)` and
     calls its init with the explicit URL of the `.wasm` file (never
     `import.meta.url` guesses);
  3. `renderOne` compiles, cleans (`cleanSvg`) and inserts.

  It does not depend on mermaid's files.
- **Vendoring:**
  - Before the wave, the lead downloads the `lini-wasm` npm package (with
    the user's go-ahead), checks its licence (MIT) and size, and commits the
    browser build into `builtins/plugins/lini/assets/` with `VENDOR.md`: file,
    package, version, source URL, SHA-256 and licence, as the diagram
    plugin's.
  - The subagent never downloads.
- **Registry** (Integration): `builtins/builtins.go` registers `lini.New()`
  and adds `"lini"` to `PluginIDs`. `apps/zenmark/app-assembly.yml` adds
  `lini` to `plugins:`, followed by regenerating the assembly.
- **Later:** the diagram viewer of plugins/diagrams (zoom, pan, Expand) can
  take `.zen-lini-svg` as well. That is a line in that spec, not this one.

## Interfaces

```go
// builtins/plugins/lini
func New() plugins.Plugin // ID "lini", Blocks ["lini"], Default false
```

```js
// builtins/plugins/lini/assets/lini.js
export const LINI_MODULE = "/_plugins/lini/<vendored module>.js";
export function loadLini(moduleUrl, wasmUrl): Promise<{ compile(src: string): string }>;
export function cleanSvg(svgText: string, doc?: Document): SVGSVGElement; // throws on a parse error
export async function renderOne(el: HTMLElement, lini): Promise<void>;
export function linkScheme(el: HTMLElement, doc?: Document): void; // behaviour 8
export default { mount(wrapper: HTMLElement, ctx): Promise<void> };
```

The real compile call follows the vendored package's API (for example
`compile(src)` returning an SVG string, or a result object). `loadLini` hides
it behind `{ compile }`, and the brief says to read the package's `.d.ts` first.

```text
CLI and config: no new flag. The plugin is turned on like any built-in
(plugins: { lini: {} }); it has no options.
```

## Tests

| # | Behaviour | Test |
|---|---|---|
| 1 | block name, off by default, fenced and `:::` | `lini_test.go: TestManifest`, `TestRenderBlockFencedAndContainer` |
| 2 | placeholder escapes the source | `TestRenderBlockEscapes` |
| 3 | unknown option warns | `TestUnknownOptionWarns` |
| 3a | from a file: three forms, `/` root, missing, own text plus file, empty | `TestFileThreeForms`, `TestFileRootPath`, `TestFileMissing`, `TestFileAndBody`, `TestFileEmpty` |
| 4 | compiler loaded once, lazily | `lini.test.ts: loads once for many blocks`, `no block, no load` |
| 5 | svg replaces the pre | `renders the svg in .zen-lini-svg` |
| 6 | compile error keeps the source, error box, console line | `a wrong figure keeps its source` |
| 7 | load failure leaves sources with one message each | `a compiler that cannot load` |
| 8 | color-scheme follows the app theme | `color-scheme follows data-theme and the system` |
| 9 | wide figures scroll inside | `lini.css: overflow-x auto` (stylesheet test, tokens only) |
| 10 | built site copies the plugin's files | `lini_test.go: TestAssetsListed` (every vendored file is in `Assets()`); a site test in `engine/site` is left to Integration |
| 11 | svg cleaning | `cleanSvg drops script, foreignObject, on* and outside hrefs`, `keeps #refs`, `a parse error is a compile error`, `no innerHTML` (source check) |
| 12 | compiler only from the plugin URL | `uses ctx.assetUrl` |
| 13 | CSP allows wasm only | Integration: `apps/zenmark/middleware_test.go` and `engine/site/shell_test.go` assert `'wasm-unsafe-eval'` and no `'unsafe-eval'`; `engine/session` keeps `.wasm` from the folder as `text/plain` (existing test) |
| 14 | help snippets compile | `TestHelpSnippets` (each snippet is a lini block), and `lini.test.ts: the help snippets compile` with the real vendored compiler under Node |
| 15 | docs and demo pages exist | `builtins/docs_test.go` (existing: every built-in has a reference and a demo page) |

Edge cases to cover:
- an empty block;
- a 2,000-line figure;
- a label with `<script>`;
- a label with `"` and `&`;
- two blocks with the same `#id`, which must not clash in the page (ids
  prefixed per block when the compiler does not do it);
- dark mode switched while the page is open;
- a block inside a tab or a dropdown, drawn when shown.

## UI

**UI consistency:** the block reuses the diagram plugin's pattern:
- the source as a code block until drawn;
- the figure in place in the reading column;
- `.zen-plugin-error` for errors, as every plugin does (docs/wiki/PLUGIN-UI.md).

There is no new control. A figure is an image: the wrapper gets
`role="img"`, with `aria-label` set to the figure's title when Lini emits one,
else "Lini figure". The source stays available to screen readers through a
visually hidden `<details>` "Source". Tokens only. No animation, so
`prefers-reduced-motion` needs nothing.

States:
- **source:** before the script runs, and when it fails;
- **drawn;**
- **error:** source plus box.

## Security

| Risk | Mitigation | Test |
|---|---|---|
| Lini's SVG carries script or event handlers from a page's text | `cleanSvg` (behaviour 11), `importNode`, never `innerHTML` | `cleanSvg drops script…`, `no innerHTML` |
| `'wasm-unsafe-eval'` widens the page policy | it allows compiling WebAssembly only, not JS `eval`; scripts still come from `'self'` only; workspace files stay sandboxed and a folder's `.wasm` is served as text, so only the bundled compiler is compiled | the CSP tests in Integration; existing `engine/session` header tests |
| A vendored file is tampered with or swapped | `VENDOR.md` records the SHA-256; a Go test hashes the embedded files against it | `TestVendoredHashes` |
| A huge figure stalls the page | compile runs once per block on mount; the body is capped by the page size limits already in the renderer; a block over 200 KB is not compiled and gets the error box "lini: figure too large" | `a figure over 200 KB is not compiled` |
| The compiler reaches the network | it is loaded from the plugin URL; the CSP `default-src 'self'` blocks anything else | `uses ctx.assetUrl`, the CSP tests |

## Notes

- **Out of scope:**
  - compiling in Go for pages without scripts;
  - a `lini` CLI command;
  - zoom and pan (plugins/diagrams can take it later);
  - turning it on by default;
  - converting mermaid blocks.
- **Before the wave (the lead, with the user's go-ahead):**
  - download `lini-wasm` from npm;
  - record its version, size and SHA-256 in `VENDOR.md`;
  - commit the browser build under `builtins/plugins/lini/assets/`;
  - if it is over 3 MB, stop and ask the user before going on.
- **Integration (done by the lead at merge):**
  - `builtins/builtins.go`: import the package, add `lini.New()` to the
    registry list and `"lini"` to `PluginIDs` and the `New` switch.
  - `apps/zenmark/app-assembly.yml`: add `lini` to `plugins:`, then run
    `go run ./kit/assembly/cmd/gen apps/zenmark`.
  - `apps/zenmark/service.go` `frontendCSP` and `engine/site/shell.go`
    `csp()`: add `'wasm-unsafe-eval'` to `script-src`. Update their tests
    (`middleware_test.go`, `shell_test.go`) to expect it, and expect no
    `'unsafe-eval'`.
  - `apps/zenmark/frontend/vite.config.ts`: add `lini.test.ts` to the
    `plugins` test project if it is not picked up.
  - `docs/wiki/PLUGINS.md`: a row for `lini` in the built-ins table.
  - `engine/session/testdata/demo-build.golden`: regenerate.
- **App Store / sandbox:** none. WebAssembly runs in the WebView, and no
  file is written.
- **Changelog (Added):** A `lini` built-in draws [Lini](https://lini.rs/)
  figures (flows, mindmaps, sequences, charts, floor plans, circuits) from a
  ```` ```lini ```` block, next to mermaid. Turn it on with
  `plugins: { lini: {} }` (plugins/lini).

## Prompt

> Implement `specs/plugins/lini/spec.md` following the rules in
> `specs/README.md` and docs/wiki/PLUGIN-UI.md (read its review checklist).
> You work only in the worktree `/Users/dragosh/Developer/zen-worktrees/plugins-lini`
> on branch `feature/plugins-lini`, and only in the paths under *Owns*.
>
> **Context:**
> - Copy the shape of `builtins/plugins/diagram/`: `diagram.go`,
>   `assets/diagram.js` (`loadMermaid`, `renderOne`, `showError`, the default
>   `mount` with `ctx.assetUrl`), its tests and `VENDOR.md`.
> - The vendored Lini compiler is already in `builtins/plugins/lini/assets/`
>   with `VENDOR.md`. Read its `.d.ts` and JS glue first to learn the real
>   compile call and how its init takes the `.wasm` URL. Hide that API behind
>   `loadLini(...) → { compile }`.
> - Your browser tests can import the vendored module under Node/vitest to
>   compile the help snippets for real (behaviour 14).
> - The CSP change (behaviour 13) and the registry lines are Integration:
>   report them, don't edit those files.
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
> - your report lists the Integration lines and anything unverified (how a
>   figure looks in the app, dark mode in the real WebView).
