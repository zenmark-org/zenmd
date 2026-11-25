# plugins/diagrams

| | |
|---|---|
| Status | specified |
| Target version | 0.8.0 (MINOR: a new reader feature; the blocks' HTML from Go is unchanged) |
| Branch | `feature/plugins-diagrams` |
| Wave | 64 (rewritten after plugins/table, plugins/lini and the diagram placeholder were merged) |
| Depends on | plugins/diagram, plugins/lini, plugins/table (`runtime/src/tableview/overlay.ts`, the one-view-at-a-time event), ui/image-zoom (the `.zen-zoom` view and bar in page.css) |
| Owns | `runtime/src/diagramview/` (new: `diagramview.ts`, `diagramview.test.ts`), the diagram view rules appended to `runtime/src/css/page.css` (one block after the table view's), `builtins/plugins/diagram/assets/diagram.js` and `builtins/plugins/lini/assets/lini.js` (one line each: mark the drawn figure zoomable) and their tests, `builtins/plugins/diagram/diagram.go` and `builtins/plugins/lini/lini.go` (Help text only), a "Zoom and pan" section in `examples/demo/plugins/diagram.md` and `examples/demo/plugins/lini.md`, this spec |

## Purpose

A large diagram is shrunk to the reading column, and its labels become too
small to read. As on GitHub, a reader can zoom and pan inside a diagram where
it sits on the page, or open it over the whole window and move around it there.
This works for mermaid diagrams, Lini figures, and any plugin's figure that
opts in.

## Behaviour

1. **Which figures.** A figure opts in with the attribute `data-zen-zoomable`
   on the element that holds its `svg`.
   - The diagram plugin sets it on `.zen-diagram-svg`, and Lini on
     `.zen-lini-svg`, when the figure is drawn (the same moment they set
     `data-drawn`).
   - Any plugin may set it on its own figure: it is documented in
     PLUGIN-UI.md (Integration).
   - A figure that failed keeps its source and error, is never marked, and
     gets no controls.
   - A marked element without one `svg` child, or with an svg of zero size,
     is left alone.
2. **The viewport.** The runtime watches the page and wraps each marked
   element in a viewport (`.zen-figure-view`) with `overflow: hidden`:
   - its width is the column's;
   - its height is the figure's fitted height, so the page doesn't move;
   - at the start it looks exactly as it does today, fitted and centred.

   A Lini figure wider than the column today scrolls sideways. In the viewport
   it is fitted to the width instead, and zooming gets it back to full size.
3. **The controls, in place.** A small bar in the top-right corner of the
   viewport (`.zen-figure-tools`) uses the image zoom's bar look
   (`.zen-zoom-bar`, `.zen-zoom-btn`). It holds Zoom in `+`, Zoom out `−`,
   Reset `⟲`, Expand `⤢`, and four pan arrows `← ↑ → ↓`. Each button has a
   title and an `aria-label` ("Zoom in", "Pan left", …).
   - The bar shows while the pointer is over the figure or focus is inside it.
   - With `(hover: none)` (touch) it is always shown.
   - Otherwise it is hidden, so it never covers the figure when not wanted.
4. **Zoom in place.**
   - Zoom in and Zoom out step ×1.25 around the viewport's centre.
   - A pinch (wheel with ⌃ or ⌘) zooms around the pointer.
   - A plain wheel scrolls the page, as now: the figure never traps the page's
     scroll.
   - The scale stays between 10% and 800% of the fitted size. Reset goes back
     to fitted and centred.
5. **Pan in place.**
   - Dragging pans (pointer capture, cursor `grab` then `grabbing`), but only
     once zoomed past fit. At fit a drag does nothing, so text and links keep
     working.
   - The arrow buttons pan by 10% of the viewport.
   - Panning stops at the edges, so at least a quarter of the viewport always
     shows the figure.
6. **Keyboard.**
   - The viewport is focusable (`tabindex="0"`, `role="img"`). Its label is
     the svg's `<title>` or `aria-label` (mermaid's `accTitle`, Lini's title),
     else "Diagram".
   - With focus on the viewport:
     - `+`/`=` zooms in, `-` zooms out, and `0` resets;
     - the arrows pan, by 50% with ⇧;
     - Enter opens the expanded view.
   - Keys act only while the viewport has focus. Tab then reaches the bar.
7. **Expanded view.** Expand, or Enter, opens the figure over the whole window
   in the image zoom's frame: `.zen-zoom` with its backdrop, `role="dialog"`,
   `aria-modal`, and the label "Diagram" or "Diagram: <title>".
   - The svg element itself moves into the view and back on close. It is not
     copied, so mermaid's markers and Lini's scoped styles and ids keep working,
     and no id is ever duplicated.
   - While it is away, the page keeps a placeholder of the same height.
   - The view opens fitted to the window, never past 100% of the figure's
     natural size.
8. **In the expanded view.**
   - The bar is the image zoom's: −, the scale in %, +, Fit, ✕.
   - The keys are `+`/`=`, `-`, `0` (Fit), `1` (100%), and the arrows pan.
   - A wheel or two-finger scroll pans; a pinch zooms around the pointer.
   - Dragging pans at any scale.
   - The backdrop, Esc and ✕ close it.
   - Focus starts on ✕, is kept inside the view, and returns to the Expand
     button or the viewport on close. The in-place zoom is kept.
9. **One view over the window at a time.** Opening calls the runtime's
   `registerOverlay` (`runtime/src/tableview/overlay.ts`), so the figure view,
   the image zoom and the table view close each other.
10. **Gone with the page.** When the page is replaced or the runtime is
    destroyed, an open view closes, the svg is put back, and the observers and
    listeners go. A figure drawn again (a new render, a theme change that
    redraws) gets a fresh viewer at fit, and a stale one for a detached svg is
    dropped.
11. **Links and text.**
    - A click that isn't the end of a drag reaches the figure: mermaid `click`
      links, and text selection at fit.
    - Any move past 4px counts as a drag and never fires a click.
12. **Print and PDF.** The bar and the view are never printed. The figure
    prints fitted, with no transform.
13. **Everywhere.** The app's reader, the editor's preview and exported sites
    (hosted and portable) all have it. It is runtime code installed on the
    page like the image zoom and the table view, not plugin code. A page
    without marked figures does nothing.
14. **Nothing to set.** There is no setting, `zenmd` flag or config key, and
    the blocks' HTML from Go is unchanged.
15. **Reduced motion.** Zoom and pan never animate: each step jumps.

## Architecture

- **`runtime/src/diagramview/diagramview.ts`** is a sibling of
  `runtime/src/tableview/tableview.ts`, with the same shape:
  - `installDiagramView(root)` returns a `{ refresh, destroy }`;
  - a `MutationObserver` on `root` finds `[data-zen-zoomable]` elements, and
    attaches a viewer to each one once;
  - a `ResizeObserver` re-fits on resize.
- **The pan and zoom state** is `{ scale, x, y }`, applied as
  `transform: translate(x, y) scale(s)` with `transform-origin: 0 0` on the
  svg. The pure maths (`clampScale`, `zoomAt`, `limitPan`, `fit`) is exported
  and tested without a DOM.
- **The expanded view** reuses the image zoom's classes and `registerOverlay`.
  It never copies them.
- **The plugins** only mark the drawn figure:
  - `diagram.js`: `box.setAttribute("data-zen-zoomable", "")` beside `drawn`;
  - `lini.js`: the same on `.zen-lini-svg`.

  They import nothing from the runtime.
- **Icons:** the zoom bar's text glyphs (`− + ✕`), plus `⟲ ⤢ ← ↑ → ↓`, or
  `runtime/src/icons.ts`, which the runtime may import.
- **Styles:** one block in `page.css` for:
  - the viewport and the corner bar's position and visibility;
  - the cursors and the print rule;
  - `.zen-lini-svg` inside a viewport no longer scrolling sideways.

## Interfaces

```ts
// runtime/src/diagramview/diagramview.ts
export const SCALE_MIN = 0.1;   // of the fitted size in place; of the natural size expanded
export const SCALE_MAX = 8;
export const STEP = 1.25;
export const PAN_STEP = 0.1;    // of the viewport, per arrow click or key
export const DRAG_SLOP = 4;     // px before a press becomes a drag
export type ViewState = { scale: number; x: number; y: number };
export function clampScale(s: number): number;
export function zoomAt(s: ViewState, factor: number, px: number, py: number): ViewState;
export function limitPan(s: ViewState, w: number, h: number, vw: number, vh: number): ViewState;
export function fit(w: number, h: number, vw: number, vh: number, maxScale: number): ViewState;
export interface DiagramView { refresh(): void; destroy(): void; readonly openCount: number }
export function installDiagramView(root: HTMLElement): DiagramView;

// runtime/src/index.ts (Integration)
export { installDiagramView } from "./diagramview/diagramview";
export type { DiagramView } from "./diagramview/diagramview";
```

**Markup after attach:**

```html
.zen-figure-view[tabindex=0][role=img][aria-label]
  [data-zen-zoomable] > svg
.zen-figure-tools.zen-zoom-bar
  button.zen-zoom-btn × 8
```

The expanded view is
`.zen-zoom.zen-figure-zoom > .zen-zoom-stage > svg` followed by
`.zen-zoom-bar`.

**The opt-in (for any plugin):** set `data-zen-zoomable` on the element that
holds the figure's one `svg`, once the figure is drawn.

**CLI and config:** none (behaviour 14).

## Tests

Written first and committed failing. `diagramview.test.ts` runs in the
`runtime` vitest project with happy-dom. Sizes come from stubbed
`getBoundingClientRect`, `clientWidth` and `viewBox`, as
`tableview.test.ts` stubs them.

| # | Behaviour | Test |
|---|---|---|
| 1 | only marked elements with one svg of non-zero size; failed figures untouched | `diagramview.test.ts: only zoomable figures` |
| 2 | the viewport wraps it, starts fitted, keeps the height; a wide Lini figure fits the width | `the viewport starts fitted`, `fit`, `a wide figure fits the column` |
| 3 | the bar's eight buttons, titles and aria-labels; hover, focus-within and `(hover: none)` | `the corner bar`; `page.css: the bar shows on hover, focus and touch` (CSS text test) |
| 4 | ×1.25 around the centre; ⌃wheel at the pointer; plain wheel not prevented; 10–800%; Reset | `zoom in place`, `a plain wheel scrolls the page`, `zoomAt`, `clampScale` |
| 5 | drag pans only past fit; arrows 10%; edges limited | `drag pans when zoomed`, `drag at fit does nothing`, `limitPan` |
| 6 | focusable viewport with its label; keys only with focus | `keyboard in place` |
| 7 | Expand moves the svg into a `.zen-zoom` dialog and back, with the same id and one element with it; a placeholder of the same height | `expand moves the svg and back` |
| 8 | the expanded bar and keys; wheel pans; Esc, ✕ and backdrop close; focus kept and returned; in-place state kept | `the expanded view`, `closing the expanded view` |
| 9 | the figure view and the image zoom close each other | `one view at a time` (with `installImageZoom`) |
| 10 | a page replaced or destroyed puts the svg back and drops listeners; a redraw gets a fresh viewer | `destroy cleans up`, `a new drawing gets fresh controls` |
| 11 | a click after a drag is swallowed; one without passes | `drags never click` |
| 12 | print: no bar, no transform | `page.css: print` (CSS text test) |
| 13 | the page and built sites install it | `kit/ui/src/page/page.test.ts` and `runtime/src/static/static.test.ts` (Integration, as for the table view) |
| 1 | the plugins mark only drawn figures | `diagram.test.mjs: marks the drawn svg zoomable`, `lini.test.ts: marks the drawn figure zoomable`, and neither marks a failed one |
| 15 | no `transition` or `animation` on the view's elements | `page.css: no motion` |
| — | no `innerHTML` in the view | `no innerHTML` (reads the source) |

Edge cases to cover:
- no viewBox;
- a wide, short figure and a tall one;
- zooming at an edge;
- resizing the window while zoomed;
- two figures on a page, where expanding the second closes the first;
- the page replaced while expanded;
- a mermaid diagram and a Lini figure on one page;
- touch (`(hover: none)`).

## UI

**Reuses:**
- the image zoom view for the expanded figure (frame, bar, keys, close rules);
- the same bar look for the corner controls;
- the table view's install and overlay pattern;
- docs/wiki/UI.md Principles: one style per state.

There is no new button style. Tokens only: `--bg`, `--rule`, `--fg-muted`,
`--focus`.

**States**
- **Drawing:** the placeholder, as now.
- **Failed:** as now, with no controls.
- **Fitted:** the controls are hidden until hover or focus.
- **Zoomed:** the cursor is `grab`.
- **Expanded:** the view.

**Accessibility**
- The viewport is an `img` with a label, and every control is a native
  `button` with an `aria-label`.
- The % is `aria-live="polite"`.
- Focus is trapped in the expanded view.
- The keys act only on the focused viewport.

**Narrow windows (375px):** the corner bar shows only Zoom in, Zoom out,
Reset and Expand.

## Security

| Risk | Mitigation | Test |
|---|---|---|
| New HTML from page data | The view only moves an svg a plugin drew. Its controls are built with `createElement` and `textContent`, never `innerHTML` | `no innerHTML` |
| Duplicate ids, or markers pointing at a copy | The svg is moved, never cloned | `expand moves the svg and back` |
| Listeners or the overlay outliving the page | `destroy` from the page's teardown; one document listener per open view | `destroy cleans up` |
| Keys stolen from the page | Keys only while the viewport has focus or the view is open | `keyboard in place` |
| A plugin marks an element it doesn't own | The attribute only adds the viewer around what is there. It never reads or runs anything inside | `only zoomable figures` |

## Notes

- **Out of scope:**
  - a minimap;
  - remembering the zoom across page loads;
  - copying a figure as an image (a later issue);
  - zoom for plain `<img>` (the image zoom already does it).
- **Integration (done by the lead at merge):**
  - `kit/ui/src/page/index.ts`: `const figures = installDiagramView(article);`
    next to `installTableView`, and `figures.destroy()` in the unmount.
  - `runtime/src/static/static.ts`: the same two lines on `root`.
  - `runtime/src/index.ts`: the export lines from Interfaces.
  - `docs/wiki/PLUGIN-UI.md`:
    - rule 10, "Views over the window": reuse `.zen-zoom*` and
      `registerOverlay`; Esc, ✕ or the backdrop closes; focus is trapped and
      returned;
    - the `data-zen-zoomable` opt-in;
    - the checklist item.
  - `specs/plugins/diagram/spec.md` and `specs/plugins/lini/spec.md`: one
    behaviour line, "zoom and pan: see plugins/diagrams".
  - Regenerate the demo golden.
- **Before building:** fix the open app bug where plugin figures stay on
  "Drawing diagram…" in the app. The viewer only attaches to drawn figures,
  so it cannot be checked in the app until diagrams draw there.
- **App Store / sandbox:** none.
- **Changelog:** `Added: diagrams and Lini figures can be zoomed and panned where they are (corner controls, pinch, drag, keys) and opened over the whole window, as on GitHub, in the app and in exported websites (plugins/diagrams).`

## Prompt

> Implement `specs/plugins/diagrams/spec.md` following the rules in
> `specs/README.md` ("Rules for every feature agent") and
> docs/wiki/PLUGIN-UI.md (read its checklist). You work only in the worktree
> `/Users/dragosh/Developer/zen-worktrees/plugins-diagrams` on branch
> `feature/plugins-diagrams`, and only in the paths under *Owns*.
>
> **Context:**
> - `runtime/src/tableview/` (`tableview.ts`, `overlay.ts` and their tests)
>   is the model to copy: install on a root, observers, `registerOverlay`, the
>   expanded view in the `.zen-zoom` frame, the stubbed sizes in its tests.
> - `runtime/src/zoom/zoom.ts` and the `.zen-zoom*` rules in
>   `runtime/src/css/page.css`: the expanded view must look and act like the
>   image zoom, and reuse those classes, never copy them.
> - `builtins/plugins/diagram/assets/diagram.js` and
>   `builtins/plugins/lini/assets/lini.js` set `data-drawn` when a figure is
>   drawn. Add `data-zen-zoomable` on the svg's holder at that same moment.
> - Installing the view on the page and in built sites is Integration, like
>   the table view's. Report it; don't edit those files.
>
> **Steps:**
> 1. Write the tests in the table and commit them failing.
> 2. Build `runtime/src/diagramview/`, the page.css block, and the two
>    plugin lines.
> 3. Add the Help text and the demo sections: a "Zoom and pan" section with a
>    large flowchart (about 20 nodes) in `diagram.md`, and a sentence pointing
>    at the schematic in `lini.md`.
>
> **Rules:**
> - no new dependencies or downloads;
> - never launch the app, run `task install`, `task package` or
>   `task release`, start servers, push, or touch `main`;
> - run go commands with `HOME` and `XDG_CONFIG_HOME` set to a throwaway
>   folder, keeping the real `GOMODCACHE`, `GOPATH` and `GOCACHE`.
>
> **Done when**, in the worktree:
> - `cd apps/zenmark/frontend && npx tsc --noEmit && npx vitest run` passes;
> - `go vet ./... && go test ./...` passes.
>
> Report the files changed, the tests, the summaries, the Integration lines,
> and what you could not check (a real trackpad pinch, how it looks in the
> app).
