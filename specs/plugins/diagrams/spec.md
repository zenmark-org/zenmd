# plugins/diagrams

| | |
|---|---|
| Status | specified |
| Target version | 0.8.0 (MINOR: a new reader feature; the diagram block's markup is unchanged) |
| Branch | `feature/plugins-diagrams` |
| Wave | 59 (with plugins/table: disjoint paths, and the same overlay protocol) |
| Depends on | plugins/diagram, ui/image-zoom (its `.zen-zoom` view and bar in page.css) |
| Owns | `builtins/plugins/diagram/assets/viewer.js` (new), `builtins/plugins/diagram/assets/viewer.test.ts` (new), `builtins/plugins/diagram/assets/diagram.js` (calls the viewer after a render), `builtins/plugins/diagram/assets/diagram.css`, `builtins/plugins/diagram/diagram.go` (Help text only), `builtins/plugins/diagram/diagram_test.go` (the asset list), `examples/demo/plugins/diagram.md`, one `plugins` project line in `apps/zenmark/frontend/vite.config.ts`, this spec |

## Purpose

A large diagram is shrunk to the reading column and its labels get too small
to read. As on GitHub, a reader can zoom and pan inside a diagram where it
sits on the page. They can also open it over the whole window and move around
it there.

## Behaviour

1. **Which diagrams.** Every diagram the diagram plugin rendered. These are
   `.zen-diagram` blocks with a `.zen-diagram-svg` holding mermaid's `svg`. A
   diagram that failed keeps its source and error, and gets no controls.
2. **The viewport.** After a render, the diagram sits in a viewport
   (`.zen-diagram-view`) with `overflow: hidden`:
   - its width is the column's;
   - its height is the diagram's fitted height, so the page doesn't move;
   - at the start it looks exactly as it does today, fitted and centred.
3. **The controls, in place.** A small bar sits in the top-right corner of the
   viewport (`.zen-diagram-tools`). It uses the image zoom's bar look
   (`.zen-zoom-bar` and `.zen-zoom-btn`), and holds these buttons:
   - Zoom in `+`;
   - Zoom out `−`;
   - Reset `⟲`;
   - Expand `⤢`;
   - four pan arrows `← ↑ → ↓`.

   Each button has a title and an `aria-label` ("Zoom in", "Pan left", …).

   **When the bar shows:**
   - It shows while the pointer is over the diagram or focus is inside it.
   - With `(hover: none)`, as on a touch screen, it is always shown.
   - It never covers the diagram when not wanted: it's hidden when it is not
     shown.
4. **Zoom in place.**
   - Zoom in and Zoom out step ×1.25 around the viewport's centre.
   - A pinch (wheel with ⌃ or ⌘) zooms around the pointer.
   - A plain wheel scrolls the page, as now: the diagram never traps the
     page's scroll.
   - The scale stays between 10% and 800% of the fitted size.
   - Reset goes back to fitted and centred.
5. **Pan in place.**
   - Dragging with the pointer pans (pointer capture, cursor `grab` and then
     `grabbing`), but only once the diagram is zoomed past fit. At fit a drag
     does nothing, so text and links keep working.
   - The arrow buttons pan by 10% of the viewport per click.
   - Panning stops at the diagram's edges. At least a quarter of the viewport
     always shows the diagram.
6. **Keyboard.**
   - The viewport is focusable (`tabindex="0"`, `role="img"`). Its label is
     the diagram's accessible title (mermaid's `accTitle`, or the svg's
     `aria-roledescription`), or "Diagram".
   - With focus on the viewport:
     - `+`/`=` zooms in, `-` zooms out, and `0` resets;
     - the arrow keys pan (with ⇧, by 50%);
     - Enter opens the expanded view.
   - The keys are taken only while the viewport has focus.
   - Tab then reaches the bar's buttons.
7. **Expanded view.**
   - Expand (or Enter) opens the diagram over the whole window, inside the
     image zoom's frame:
     - `.zen-zoom`, with its blurred backdrop;
     - `role="dialog"` and `aria-modal`;
     - the label "Diagram" or "Diagram: <title>".
   - The svg element itself moves into the view and moves back on close. It is
     not copied, so mermaid's ids, styles and arrow markers keep working, and
     there are no duplicate ids.
   - While it is away, the page keeps a placeholder of the same height, so
     nothing jumps.
   - The view opens fitted to the window, never enlarged past 100% of the
     diagram's natural size by the fit.
8. **In the expanded view.**
   - The bar is the image zoom's: −, the scale in %, +, Fit, ✕.
   - The keys are `+`/`=`, `-`, `0` (Fit), `1` (100%), and the arrow keys pan.
   - A plain wheel or two-finger scroll pans, and a pinch zooms around the
     pointer.
   - Dragging pans at any scale.
   - A click on the backdrop outside the diagram closes the view. Esc and ✕
     close it too.
   - Focus starts on ✕, stays inside the view, and goes back to the Expand
     button (or the viewport) on close.
   - The in-place zoom is kept as it was before opening.
9. **One view over the window at a time.** The diagram view, the image zoom
   and the table view (plugins/table) close each other through one document
   event. Opening one sends `zen:overlay-open` (a `CustomEvent` whose
   `detail.owner` is a unique string). Every open view closes when it sees an
   event from another owner. When the block unmounts (going to another page, a
   re-render, the plugins being reloaded), its view closes, its listeners go
   and the svg is put back.
10. **Links and text in the diagram.**
    - A click that isn't the end of a drag reaches the diagram as before:
      mermaid `click` links, and text selection at fit.
    - A drag never fires a click (any move past 4px counts as a drag).
11. **Theme and re-render.** When the plugin draws a diagram again (a new
    page render), it gets fresh controls at fit. A stale viewer of the old svg
    is dropped.
12. **Print and PDF.** The bar and the view are never printed. The diagram
    prints fitted, with its in-place zoom ignored
    (`@media print { transform: none }`).
13. **Everywhere.** The app's reader, the editor's preview and exported
    websites all have it, because the plugin's JS runs in all of them. Pages
    with no diagram load nothing new: `viewer.js` is imported by `diagram.js`
    only after a diagram rendered.
14. **Nothing to set.** There is no setting, `zenmd` flag or config key, and
    the block's HTML from Go is unchanged.
15. **Reduced motion.** Zoom and pan are never animated: each step jumps.

## Architecture

- **`viewer.js`** is a plain ES module, with no dependencies, served next to
  `diagram.js` (`ctx.assetUrl("viewer.js")`, or a relative import that works
  under `/_plugins/diagram/` and in built sites).
  - It exports `attachViewer(el, opts)`, which wraps `.zen-diagram-svg` in the
    viewport, adds the bar and returns `{ open, close, reset, destroy }`.
  - The pan/zoom state is `{ scale, x, y }`, applied as
    `transform: translate(x, y) scale(s)` on the svg, with
    `transform-origin: 0 0`.
  - The pure maths (clamping, zooming around a point, fitting, edge limits)
    goes in exported functions so it can be tested without a DOM.
- **`diagram.js`:**
  - After `renderOne` succeeds, it imports the viewer and attaches it.
  - `mount` returns a clean-up that calls `destroy` (PLUGINS.md: `mount` may
    return a `CleanUp`).
- **Styles.**
  - `diagram.css` gets the viewport, the corner bar's position and
    visibility, the cursors and the print rule.
  - The look of the bar, the buttons and the expanded view comes from the
    `.zen-zoom*` rules in `runtime/src/css/page.css`. The page CSS ships on
    every page, in the app and in built sites, so it can be reused without
    copying it.
- **Icons.** The plugin can't import `runtime/src/icons.ts`, so it uses the
  same text glyphs the zoom bar uses (`−`, `+`, `✕`), plus `⟲ ⤢ ← ↑ → ↓`.
- **Overlay protocol (behaviour 9)** is a document event, so a plugin can
  join it without importing the runtime.
  - plugins/table's `runtime/src/tableview/overlay.ts` uses the same event,
    and so does image zoom after that spec's Integration line.
  - The lead writes the protocol into docs/wiki/PLUGIN-UI.md at merge.

## Interfaces

```js
// builtins/plugins/diagram/assets/viewer.js
export const SCALE_MIN = 0.1;   // of the fitted size, in place; of natural size, expanded
export const SCALE_MAX = 8;
export const STEP = 1.25;
export const PAN_STEP = 0.1;    // of the viewport, per arrow click or key
export const DRAG_SLOP = 4;     // px before a press becomes a drag
export const OVERLAY_EVENT = "zen:overlay-open";

/** Pure: the scale after zooming by factor, clamped. */
export function clampScale(s) /* : number */;
/** Pure: the new {scale, x, y} after zooming by factor around point (px, py) of the viewport. */
export function zoomAt(state, factor, px, py) /* : {scale,x,y} */;
/** Pure: keeps at least a quarter of the viewport on the content; w,h = content size at scale 1, vw,vh = viewport. */
export function limitPan(state, w, h, vw, vh) /* : {scale,x,y} */;
/** Pure: the scale and offsets that fit content w×h centred in vw×vh, at most maxScale. */
export function fit(w, h, vw, vh, maxScale) /* : {scale,x,y} */;

/** Wraps el's .zen-diagram-svg in the viewport and adds the controls. */
export function attachViewer(el /* .zen-diagram */, opts /* { doc?: Document } */) /* : Viewer */;
// Viewer: { open(): void, close(): void, reset(): void, readonly isOpen: boolean, readonly state: {scale,x,y}, destroy(): void }
```

**Markup after attach:**

```html
.zen-diagram
  .zen-diagram-view[tabindex=0][role=img][aria-label]
    .zen-diagram-svg > svg
  .zen-diagram-tools.zen-zoom-bar
    button.zen-zoom-btn × 8
```

The expanded view is
`.zen-zoom.zen-diagram-zoom > .zen-zoom-stage > (svg)` followed by
`.zen-zoom-bar`.

**Overlay event:**
`document.dispatchEvent(new CustomEvent("zen:overlay-open", { detail: { owner } }))`.

**CLI and config:** none (behaviour 14).

## Tests

Written first and committed failing.
- `builtins/plugins/diagram/assets/viewer.test.ts` runs under vitest with
  happy-dom: a new `plugins` project, `include: ["../../../builtins/plugins/*/assets/**/*.test.ts"]`.
- The pure functions are also covered there.
- The existing `diagram.test.mjs` (`node --test`, run by `TestDiagramJS`)
  keeps passing.

| # | Behaviour | Test |
|---|---|---|
| 1 | a rendered diagram gets a viewer; a failed one doesn't | `viewer.test.ts: only rendered diagrams` |
| 2 | the viewport wraps the svg; it starts at fit, with the same height | `the viewport starts fitted` |
| 3 | the bar's eight buttons, titles and aria-labels; shown on hover and focus-within; `(hover: none)` CSS | `the corner bar` and `diagram.css: the bar shows on hover, focus and touch` (a CSS text test) |
| 4 | + − step ×1.25 around the centre; ⌃wheel zooms at the pointer; a plain wheel isn't prevented; 10–800%; Reset | `zoom in place`, `a plain wheel scrolls the page`, `zoomAt`, `clampScale` |
| 5 | drag pans only past fit; arrows pan 10%; edges are limited | `drag pans when zoomed`, `drag at fit does nothing`, `limitPan` |
| 6 | focusable viewport with its label; + - 0, arrows (⇧ ×5) and Enter; keys ignored without focus | `keyboard in place` |
| 7 | Expand moves the svg into a `.zen-zoom` dialog, leaves a placeholder of the same height, fits; the svg comes back on close, with its id the same | `expand moves the svg and back`, `fit` |
| 8 | the bar's − % + Fit ✕; keys 0 and 1; wheel pans and ⌃wheel zooms; Esc, ✕ and the backdrop close it; focus starts on ✕ and returns; the in-place state is kept | `the expanded view`, `closing the expanded view` |
| 9 | opening sends `zen:overlay-open`; another owner's event closes it; `destroy` closes it, puts the svg back and removes the listeners | `one view at a time`, `destroy cleans up` |
| 10 | a click after a drag is swallowed; a click without a drag passes through | `drags never click` |
| 11 | a re-render drops the old viewer and attaches a new one at fit | `a new render gets fresh controls` |
| 12 | print rule: bar hidden, transform none | `diagram.css: print` |
| 13 | `diagram.js` attaches only after a successful render; the asset list includes `viewer.js` | `diagram.test.mjs: attaches after render` (stub import), `diagram_test.go: TestAssets` |
| 15 | no `transition` or `animation` on the viewer's elements | `diagram.css: no motion` |
| — | no `innerHTML` in viewer.js | `viewer.test.ts: no innerHTML` (reads the source) |

Edge cases to cover:
- a diagram with no viewBox or zero size (no viewer, no error);
- a very wide, short diagram (fit is limited by width) and a tall one;
- zooming at the edge;
- the window resized while zoomed (the limits are re-applied);
- two diagrams on one page, where expanding the second closes the first;
- unmount while expanded;
- a touch screen (`(hover: none)`).

## UI

**Reuses:**
- the image zoom view for the expanded diagram (the same frame, bar, keys and
  close rules);
- the same bar look for the corner controls;
- docs/wiki/UI.md Principles: one style per state.

There is no new button style, and the black and white tokens only:
- `--bg` for the bar;
- `--rule` for its border;
- `--fg-muted` for the %;
- `--focus` for the ring on the viewport and the buttons.

**States**
- **Rendering:** as now.
- **Failed:** as now, with no controls.
- **Fitted:** the controls are hidden until hover or focus.
- **Zoomed:** the cursor is `grab`.
- **Expanded:** the view.

**Accessibility**
- The viewport is an `img` with a label, and every control is a native
  `button` with an `aria-label`.
- The % in the expanded bar is `aria-live="polite"`.
- Focus is trapped in the expanded view.
- The keys only act on the focused viewport.

**Narrow windows (375px):** the corner bar shows only Zoom in, Zoom out,
Reset and Expand. The arrows are left out, since drag and the keys cover
panning.

## Security

| Risk | Mitigation | Test |
|---|---|---|
| New HTML from page data | The viewer only moves the svg mermaid made (strict mode) and builds its controls with `createElement` and `textContent`. No `innerHTML` | `no innerHTML` |
| Duplicate ids, or markers pointing at the wrong copy | The svg is moved, not cloned | `expand moves the svg and back` (the id is unchanged, and there's one element with it) |
| Listeners or the overlay outliving the page | `destroy` from the mount clean-up; a document listener per open view only | `destroy cleans up` |
| Keys stolen from the page | The keys are handled only while the viewport has focus, or in the open view | `keyboard in place` (an arrow on the body doesn't pan) |

## Notes

- **Out of scope:**
  - zoom for other plugins' drawings;
  - a minimap;
  - remembering zoom across page loads;
  - copying the diagram as an image (a later issue).
- **Integration (done by the lead at merge):**
  - **docs/wiki/PLUGIN-UI.md:** add rule 10, "Views over the window": reuse
    `.zen-zoom`, `.zen-zoom-stage`, `.zen-zoom-bar` and `.zen-zoom-btn`; send
    `zen:overlay-open` with your owner on open; close on another owner's
    event; Esc, ✕ or the backdrop closes; focus is trapped and returned. Add
    the matching item to the review checklist.
  - **plugins/table:** its `overlay.ts` must use the same event
    (`zen:overlay-open`, `detail.owner`). Both specs say so. If either branch
    merges first, the lead checks the other matches.
  - **specs/plugins/diagram/spec.md:** add behaviour 7, "zoom and pan: see
    plugins/diagrams".
- **App Store / sandbox:** none.
- **Changelog:** `Added: diagrams can be zoomed and panned where they are (corner controls, pinch, drag, keys) and opened over the whole window, as on GitHub, in the app and in exported websites (plugins/diagrams).`

## Prompt

> Implement `specs/plugins/diagrams/spec.md` following the rules in
> `specs/README.md` ("Rules for every feature agent") and docs/wiki/PLUGIN-UI.md
> (read its checklist). You work only in the worktree
> `/Users/dragosh/Developer/zen-worktrees/plugins-diagrams` on branch
> `feature/plugins-diagrams`, and only in the paths under *Owns*.
>
> Context:
> - Read `builtins/plugins/diagram/assets/diagram.js` (how a diagram renders
>   and mounts), `diagram.test.mjs` and `diagram_test.go`.
> - Read `runtime/src/zoom/zoom.ts` and the `.zen-zoom*` rules in
>   `runtime/src/css/page.css`. The expanded view must look and behave like
>   image zoom, and must reuse those classes, never copy them.
> - Read `runtime/src/plugins.ts` for `mount`, its clean-up, and `ctx.assetUrl`.
> - Another agent is building plugins/table at the same time. It uses the same
>   `zen:overlay-open` event, so implement it exactly as in Interfaces, and
>   don't touch `runtime/`.
>
> Steps:
> 1. Add the `plugins` vitest project line to
>    `apps/zenmark/frontend/vite.config.ts`.
> 2. Write the tests in the Tests table and commit them failing.
> 3. Build `viewer.js`, wire it into `diagram.js`, add the CSS, update the Help
>    summary in `diagram.go`, and add a "Zoom and pan" section to
>    `examples/demo/plugins/diagram.md`, with a large flowchart (about 20
>    nodes) and one sentence on the controls.
>
> Rules:
> - No new dependencies or downloads.
> - Never launch the app, run `task install`, `task package` or
>   `task release`, start servers, push, or touch `main`.
> - Run every go command with `HOME` and `XDG_CONFIG_HOME` set to a throwaway
>   temp folder.
>
> Done when, in the worktree:
> - `cd apps/zenmark/frontend && npx tsc --noEmit && npx vitest run` passes;
> - `go vet ./... && go test ./...` passes.
>
> Report the files you changed, the tests, the test summaries, and anything
> you could not check: for example, the pinch on a real trackpad, which only
> the app shows.
