# plugins/diagram-performance

| | |
|---|---|
| Status | done |
| Target version | 0.8.0 (PATCH-sized fix inside a MINOR: no syntax or output change) |
| Branch | `feature/plugins-diagram-performance` |
| Wave | 65 (merged before plugins/diagrams, whose branch touches one line of each plugin's JS; the lead resolves that overlap at merge) |
| Depends on | plugins/diagram, plugins/lini, the diagram placeholder (`data-drawn`), runtime/print (`whenReady`, `data-zen-pending`) |
| Owns | `builtins/plugins/lini/assets/lini.js`, `builtins/plugins/lini/assets/lini.worker.js` (new), `builtins/plugins/lini/assets/lini.test.ts`, the embed list in `builtins/plugins/lini/lini.go` (+ `lini_test.go` asset test), `builtins/plugins/diagram/assets/diagram.js`, `builtins/plugins/diagram/assets/diagram.test.mjs`, `builtins/plugins/diagram/assets/queue.js` (new, shared shape copied into lini as `assets/queue.js`), `examples/demo/plugins/stress.md` (new, hidden from the sidebar), this spec |

## Purpose

A big diagram must never freeze the window. Today both plugins draw on the
page's main thread, which also handles clicks and scrolling:
- Lini compiles each figure synchronously in WebAssembly.
- Mermaid lays each diagram out synchronously.
- Both draw every figure on the page at once, even the ones far below the
  fold.

A large schematic or flowchart, or several of them, blocks the app until all
are done. After this change the window always stays responsive: Lini compiles
in a background worker, and figures draw one at a time, only when they come
near the screen.

## Behaviour

### Lini in a worker

1. Lini compiles in a Web Worker, `assets/lini.worker.js` (a module worker).
   - It loads `lini_wasm.js` and the `.wasm` from URLs the page passes in,
     from `ctx.assetUrl`.
   - It answers `{ id, svg }` or `{ id, error }` for each `{ id, src }` it
     receives.
   - The page creates one worker per page, and reuses it for every figure.
2. **Time limit.** A compile that takes longer than `LINI_TIMEOUT` (10 s)
   stops:
   - the worker is terminated and a fresh one starts for the next figure;
   - the figure shows its source and the error box "lini: the figure took
     too long to draw (over 10 s)";
   - one console warning is written.
3. **Fallback.** If a worker cannot start, Lini compiles on the main thread
   as now, under the one-at-a-time rule of behaviour 5. Fallback cases:
   - the WebView refuses it;
   - the `Worker` constructor is missing;
   - the worker's first message is an error loading the module or the
     WebAssembly.

   The fallback is chosen once per page and logged once to the console:
   "lini: drawing on the page (no worker)".
4. **Safety stays the same.**
   - The worker only returns SVG text.
   - The page still cleans it with `cleanSvg` and inserts it with
     `importNode`.
   - The worker never touches the page.
   - It loads only the plugin's own files.

### Both plugins: near the screen, one at a time

5. **One at a time.** Each plugin keeps one queue per page. A figure is drawn
   only after the previous one has finished, and the plugin yields to the
   window between figures (`setTimeout(0)`, or `requestIdleCallback` where it
   exists), so input and painting happen in between.
6. **Near the screen.** A figure is queued only when it comes within
   `NEAR` (1000 px) of the viewport, through `IntersectionObserver`.
   - Figures far below wait behind the placeholder until scrolled towards.
   - Without `IntersectionObserver` every figure is queued at once, but still
     one at a time.
7. **Print and PDF draw everything.** Before printing, every figure must be
   drawn:
   - every waiting figure carries `data-zen-pending`, and the attribute is
     removed when it is drawn or fails, so `whenReady` waits for it;
   - in the print preview (inside `.zen-print-preview`), or when the
     document prints (`beforeprint`), every waiting figure is queued at once,
     whatever its position.
8. **Order.** Figures near the screen draw top to bottom. A figure scrolled
   into view jumps ahead of queued figures that are still off screen.
9. **Gone with the page.** When the page is replaced:
   - queued figures are dropped and observers disconnected;
   - a Lini compile in flight is ignored when it answers (its figure is
     detached);
   - the worker lives for the page and is ended on unmount.
10. **Mermaid's size guard.** A mermaid source over `MAX_SOURCE` (200 KB),
    the same limit as Lini's, is not drawn and shows "diagram: the diagram is
    too large to draw".
11. **Nothing else changes.** It keeps:
    - the placeholder, the error boxes and the theme behaviour;
    - `data-drawn`, and the `data-zen-zoomable` of plugins/diagrams;
    - the Go HTML.

    There is no setting or flag (`NEAR`, `LINI_TIMEOUT` and `MAX_SOURCE` are
    constants).

### A page to prove it

12. `examples/demo/plugins/stress.md`, hidden from the sidebar
    (`nav.hidden`), holds:
    - one large mermaid flowchart (about 150 nodes in subgraphs);
    - one large Lini schematic (about 60 parts), checked to compile;
    - several small figures above and below them.

    It is the page the lead opens in the app to check the window stays
    responsive.

## Architecture

- **`assets/queue.js`** (diagram) and its copy in lini (plugins cannot import
  each other) export `createQueue({ near, yield })`. It returns
  `{ add(el, draw), drawAll(), destroy() }`:
  - an `IntersectionObserver` feeds a priority list;
  - one async loop runs `draw(el)` with a yield between;
  - `data-zen-pending` is set on add and cleared after draw.

  The pure ordering logic is exported and tested without a DOM.
- **`lini.js`:**
  - `loadLini` becomes `startLini(moduleUrl, wasmUrl)`, which returns
    `{ compile(src): Promise<string>, end() }`, backed by the worker or the
    fallback.
  - `renderOne` awaits `compile` and keeps everything after it as now.
  - `mount` queues the figure instead of drawing it.
- **`diagram.js`:** `mount` queues `renderOne`, with the same mermaid calls.
- **`lini.worker.js`:**
  - about 30 lines: `import init, { compile } from` the module URL, sent in
    the first message;
  - `init({ module_or_path })`;
  - then compile on each message.

  It runs nothing else.
- **The worker in the app:** it is started from the plugin's asset URL (the
  app's own origin). Plugin files are served with
  `Content-Security-Policy: sandbox`. A worker takes its CSP from its own
  response, and browsers do not apply `sandbox` to workers, so WebAssembly
  is allowed there. Behaviour 3 covers a WebView that disagrees.

## Interfaces

```js
// builtins/plugins/lini/assets/lini.js
export const LINI_TIMEOUT = 10_000;
export function startLini(moduleUrl, wasmUrl, opts /* { workerUrl?, Worker? } */) /* : { compile(src): Promise<string>, end(): void, readonly mode: "worker" | "page" } */;
// loadLini stays exported as the page-thread fallback.

// builtins/plugins/{diagram,lini}/assets/queue.js
export const NEAR = 1000;
export function createQueue(opts /* { doc?, near?, yield?: () => Promise<void> } */) /* : { add(el, draw), drawAll(), destroy(), readonly size } */;
export function nextIndex(items /* [{ visible, top }] */) /* : number */; // pure: visible first, then top to bottom

// builtins/plugins/lini/assets/lini.worker.js (a module worker)
// in:  { type: "init", module, wasm } | { type: "compile", id, src }
// out: { type: "ready" } | { type: "init-error", message } | { id, svg } | { id, error }
```

## Tests

| # | Behaviour | Test |
|---|---|---|
| 1 | compile goes through the worker; one worker per page; ids match answers | `lini.test.ts: compiles in a worker` (a fake Worker that runs the real compiler in-process) |
| 2 | timeout terminates, restarts and shows the error | `a figure past the time limit` (fake timers, a worker that never answers) |
| 3 | no Worker, a refused worker, an init error: page-thread fallback, logged once | `falls back to the page` |
| 4 | worker output is cleaned and imported as before | existing `cleanSvg` tests and `the worker never gets the page` (source check: no `document` in lini.worker.js) |
| 5 | one at a time, with a yield between | `queue: one at a time`, `queue: yields between figures` (both plugins) |
| 6 | only near the screen; all at once without IntersectionObserver | `queue: waits for the screen`, `queue: no IntersectionObserver` |
| 7 | `data-zen-pending` while waiting; drawAll in the print preview and on beforeprint; `whenReady` resolves | `pending until drawn`, `print draws everything` (with `runtime/src/print/ready.ts` `whenReady`) |
| 8 | visible first, then top to bottom | `nextIndex` table |
| 9 | unmount drops the queue, ignores late answers, ends the worker | `unmount mid-queue` |
| 10 | mermaid source over 200 KB | `diagram.test.mjs: too large` |
| 11 | placeholder, errors and themes unchanged | the existing plugin tests pass unchanged |
| 12 | the stress page compiles (Lini) and renders without warnings | `lini_test.go: TestStressPageCompiles` (via the vendored compiler under node, like TestHelpSnippets), and the demo tests |
| — | the worker file is embedded and served | `lini_test.go: TestAssetsListed` |

Edge cases to cover:
- two figures with the same source;
- a figure that errors in the middle of the queue;
- scrolling fast past twenty figures;
- a page with 50 small figures;
- unmount while the worker is initialising;
- the theme changed while a figure waits.

## UI

None new: the existing "Drawing diagram…" placeholder shows while a figure
waits or draws. A figure below the fold keeps its placeholder until scrolled
near.

**UI consistency:** the plugin states from PLUGIN-UI.md rule 4, unchanged.

## Security

| Risk | Mitigation | Test |
|---|---|---|
| The worker loads code from elsewhere | It imports only the module and `.wasm` URLs the page passes, which come from `ctx.assetUrl`. A URL that is not same-origin is refused | `the worker only loads plugin files` |
| The worker's output reaches the page unchecked | It returns text only; the page cleans it with `cleanSvg` and inserts it with `importNode` | existing `cleanSvg` tests |
| A hostile or huge figure hangs the app | The worker, the 10 s limit and termination; the 200 KB source limits | `a figure past the time limit`, `too large` |
| The worker outlives the page | It ends on unmount | `unmount mid-queue` |

## Notes

- **Out of scope:**
  - moving mermaid to a worker, which needs the DOM to measure text;
  - caching compiled SVGs between page loads;
  - a progress indicator per figure.
- **Integration (done by the lead at merge):**
  - `engine/session/testdata/demo-build.golden`: regenerate.
  - Merge before `feature/plugins-diagrams`. Keep its `data-zen-zoomable`
    line in both plugins' `renderOne`, after `data-drawn`.
  - `docs/wiki/PLUGIN-UI.md`, rule 4 (States): "a plugin that draws figures
    queues them near the screen, one at a time, and marks waiting ones
    `data-zen-pending`".
- **Verification by the lead:**
  - open `plugins/stress.md` in the app;
  - scroll while the figures draw: the window must stay responsive;
  - check the console line that says whether Lini runs in a worker or on the
    page.
- **App Store / sandbox:** none.
- **Changelog (Fixed):** Big diagrams no longer freeze the window: Lini
  figures compile in a background worker with a 10-second limit, and
  diagrams draw one at a time as they come near the screen
  (plugins/diagram-performance).

## Prompt

> Implement `specs/plugins/diagram-performance/spec.md` following the rules
> in `specs/README.md` and docs/wiki/PLUGIN-UI.md. You work only in the
> worktree `/Users/dragosh/Developer/zen-worktrees/plugins-diagram-performance`
> on branch `feature/plugins-diagram-performance`, and only in the paths
> under *Owns*.
>
> **Context:**
> - `builtins/plugins/lini/assets/lini.js` (`loadLini`, `renderOne`, the
>   default `mount`, `setImporter`) and `builtins/plugins/diagram/assets/diagram.js`
>   (`loadMermaid`, `renderOne`, `mount`) are what you change.
> - `builtins/plugins/lini/VENDOR.md` explains the wasm-bindgen glue:
>   `init({ module_or_path })` and `initSync({ module })`.
> - `runtime/src/print/ready.ts` waits on `data-zen-pending`.
> - The tests: `lini.test.ts` (vitest `plugins` project, the real compiler
>   loaded with `initSync`) and `diagram.test.mjs` (`node --test`, a tiny
>   fake DOM, run by `TestDiagramJS`).
> - To check the stress schematic compiles, compile it with the vendored
>   compiler the way `lini.test.ts` does.
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
> - every test in the table passes;
> - `go vet ./... && go test ./...` passes;
> - `cd apps/zenmark/frontend && npx tsc --noEmit && npx vitest run` passes;
> - your report says how the worker was tested and what only the app can
>   show (the WebView starting the worker, the window staying responsive).
