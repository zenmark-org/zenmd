# plugins/table

| | |
|---|---|
| Status | specified |
| Target version | 0.8.0 (MINOR: a new reader feature, nothing changes for authors) |
| Branch | `feature/plugins-table` |
| Wave | 59 (with plugins/diagrams: disjoint paths, and the same overlay event) |
| Depends on | ui/image-zoom (its overlay and bar), plugins/myst-builtins phase 2 (the `table` built-in) |
| Owns | `runtime/src/tableview/`; the table view rules appended to `runtime/src/css/page.css` (one block after the image zoom rules); the `expand` icon in `runtime/src/icons.ts`; `builtins/plugins/table/assets/table.css` (only to make room for the button); a "Wide tables" section in `examples/demo/features/reader.md`; this spec |

## Purpose

Pages keep a 68-character reading column, so a wide table is cut off and scrolls
sideways. A reader can open any table that does not fit over the whole window,
the way an image opens on a click, and read it at full width with its header
row in view.

## Behaviour

1. **Which tables.** Every `table` in the page (`.zen-page`): plain markdown
   tables and the table built-in's (`.zt table`, from `table`, `list-table` and
   `csv-table`). Left out: tables inside another plugin's output
   (`[data-zen-plugin]` other than the table built-in), inside a link or a
   button, nested tables, and a table marked `data-zen-nozoom`.
2. **The button shows only when the table does not fit.** A table counts as not
   fitting when its scroll box (the table itself for a markdown table, which is
   `display: block; overflow-x: auto`, or `.zt-scroll` for the built-in) is
   wider inside than out (`scrollWidth > clientWidth + 1`). The check runs after
   each render and again when the column changes width (window resize, sidebar or
   Inspector shown or hidden). A table that fits has no button. When a table
   stops fitting, its button appears, and when it fits again the button goes.
3. **The button.** A `.zen-btn.zen-btn-quiet` button, `⤢ Expand` (the `expand`
   line icon and the word), right-aligned in a row just above the table
   (`.zen-table-tools`). Its title is "Expand table" and its `aria-label` is
   "Expand table" followed by the caption when there is one ("Expand table:
   Stock levels"). It is in the Tab order. Find, the outline, copy and the word
   count never see it. Print and the PDF export never show it.
4. **The view.** Clicking the button opens the table over the whole window,
   inside the image zoom's frame (`.zen-zoom`: the blurred backdrop, the
   `role="dialog"`, `aria-modal`, and a label "Table" or "Table: <caption>").
   The table is a copy of the page's table (`cloneNode(true)`, ids removed),
   with its caption, drawn at its natural width: no 68ch limit, and cells wrap
   at 40ch at most. It is centred when narrower than the window.
5. **Header and row headers stay in view.** When the view scrolls, the
   header rows (`thead`) stay at the top, and the row headers
   (`th[scope=row]`, the built-in's stub columns) stay on the left. Both keep an
   opaque `--bg` background.
6. **Scrolling.** The view scrolls in both directions: wheel, trackpad,
   scrollbars, arrow keys, Page Up and Page Down, Home and End. It opens at the
   top left.
7. **The bar.** The image zoom's bar (`.zen-zoom-bar`, `.zen-zoom-btn`) shows
   −, the text size in %, +, Fit and ✕ (Close). + and − change the text size by
   ×1.25 each, between 50% and 200%. Fit picks the largest size, at most 100%,
   at which the table fits the window's width, and never goes below 50%. The
   view opens at 100%. The keys are the image zoom's: `+`/`=`, `-`, `0` (Fit)
   and `1` (100%). A pinch (wheel with ⌃) changes the size as well.
8. **Closing.** Esc, ✕, or a click on the backdrop outside the table closes the
   view, and focus goes back to the Expand button. A click inside the table does
   not close it, so text can be selected and copied.
9. **Links.** A link in the copy acts like the same link in the page (internal,
   heading, external and broken links alike): the view closes, then the
   matching link in the page gets the click.
10. **Focus.** The view opens with focus on ✕. Tab and ⇧Tab cycle through
    the bar's buttons, the table's links and the scroll stage, and never reach
    the page behind it.
11. **One view at a time.** The table view and the image zoom share one
    overlay. Opening one closes the other, and going to another page closes
    both.
12. **Everywhere the page is read.** The app's reader, the editor's preview
    (both through the page feature), and exported websites (`static.ts`) all
    have it. A PDF and print never do.
13. **Nothing to set.** There is no setting, `zenmd` flag or config key. The
    feature is a reader control with no output, and the HTML from `zenmd build`
    is the same as before. An author leaves a table out with
    `data-zen-nozoom`, the attribute image zoom already uses.
14. **Reduced motion.** Opening and closing are not animated.

## Architecture

- `runtime/src/tableview/tableview.ts` gives `installTableView(root)`:
  - A `MutationObserver` on `root` (childList, subtree, coalesced per animation
    frame) and a `ResizeObserver` on each candidate table's scroll box keep the
    buttons in step (behaviour 2).
  - One delegated click listener on `root` handles the buttons.
  - The view is plain DOM appended to `document.body`, reusing the zoom's
    classes. No `innerHTML`: the copy is a `cloneNode` of a table the sanitiser
    already allowed.
- **Sharing the overlay with image zoom (behaviour 11).**
  - Views over the window close each other through one document event,
    `zen:overlay-open`: a `CustomEvent` whose `detail.owner` is a unique
    string. Opening a view sends it, and every open view closes when it sees
    another owner's event.
  - plugins/diagrams uses the same event from plugin JS, which can't import
    the runtime, so the event is the contract, not a function.
  - `runtime/src/tableview/overlay.ts` holds the helper (`registerOverlay`).
  - Image zoom calls the same helper: a one-line change in `zoom.ts` (see
    Integration).
- **The button row.**
  - The row carries `data-zen-ui`: Find's `skip`, the outline, the word count
    and copy (via `user-select: none`) ignore it.
  - The row is put right before the scroll box and is removed when the table
    fits again.
  - Re-renders replace the article's HTML, which drops the row; the observer
    adds it again.
- **Who calls it.**
  - `kit/ui/src/page/index.ts` installs it on the article, next to image zoom.
  - `runtime/src/static/static.ts` installs it on `.zen-page`.
  - Both destroy it when they are torn down.

## Interfaces

```ts
// runtime/src/tableview/tableview.ts
export const TABLE_SIZE_MIN = 0.5;
export const TABLE_SIZE_MAX = 2;
export interface TableView {
  open(table: HTMLTableElement): void;
  close(): void;
  readonly isOpen: boolean;
  /** Re-checks every table now (the observers call it; tests too). */
  refresh(): void;
  destroy(): void;
}
/** The table in root that may get an Expand button, or null (behaviour 1). */
export function expandableTable(el: Element, root: HTMLElement): HTMLTableElement | null;
/** Whether the table overflows its scroll box (behaviour 2). */
export function overflows(table: HTMLTableElement): boolean;
/** The size Fit picks for a table of natural width w in a stage of width avail (behaviour 7). */
export function fitSize(w: number, avail: number): number;
export function installTableView(root: HTMLElement): TableView;

// runtime/src/tableview/overlay.ts
export const OVERLAY_EVENT = "zen:overlay-open";
/** Call when a view opens: sends OVERLAY_EVENT with a new owner, and calls close when another owner's event comes.
 *  Returns the remover; call it when the view closes. */
export function registerOverlay(doc: Document, close: () => void): () => void;

// runtime/src/index.ts (Integration)
export { installTableView, expandableTable, TABLE_SIZE_MIN, TABLE_SIZE_MAX } from "./tableview/tableview";
export type { TableView } from "./tableview/tableview";

// runtime/src/icons.ts: IconName gains "expand" (two arrows to opposite corners).
```

- **Markup**: `<div class="zen-table-tools" data-zen-ui><button type="button" class="zen-btn zen-btn-quiet zen-table-expand" title="Expand table" aria-label="…">[svg] Expand</button></div>`.
- **View**: `.zen-zoom.zen-tableview > .zen-zoom-stage.zen-tableview-stage > table.zen-tableview-table`, followed by `.zen-zoom-bar`.
- **CLI and config**: none (behaviour 13).

## Tests

Written first and committed failing. They live in
`runtime/src/tableview/tableview.test.ts` (happy-dom).

| # | Behaviour | Test |
|---|---|---|
| 1 | markdown and `.zt` tables count; other plugins' tables, tables in a link or button, nested and `data-zen-nozoom` tables do not | `expandableTable: which tables` |
| 2 | button only when the table overflows; it appears and goes as the width changes; it comes back after a re-render | `the button follows overflow`, `a re-render keeps the buttons` (scrollWidth/clientWidth stubbed) |
| 3 | the button's classes, text, title, aria-label with the caption, the `data-zen-ui` row before the scroll box | `the Expand button` |
| 3 | Find skips the row | `kit/ui/src/find/find.test.ts: skips data-zen-ui` (if Find's skip does not cover it yet, add `[data-zen-ui]` to the page's skip; see Integration) |
| 4 | the view: dialog role and label, a copy with no ids, the caption, inside `.zen-zoom` | `opening shows a copy of the table` |
| 5 | thead and `th[scope=row]` get the sticky classes | `headers stay in view` |
| 7 | + − 0 1, the bar's buttons and the % text; clamped to 50–200%; `fitSize` | `the bar sets the size`, `fitSize` |
| 8 | Esc, ✕ and a backdrop click close it, and focus returns to the button; a click in the table does not close it | `closing` |
| 9 | a link click in the copy closes the view and clicks the page's link | `links act like the page's` |
| 10 | focus starts on ✕; Tab cycles inside the view | `focus stays in the view` |
| 11 | opening the table view closes an image zoom and the other way round; `destroy` closes and disconnects | `one overlay at a time` (with `installImageZoom`) |
| 12 | the page feature and static.ts install it | `kit/ui/src/page/page.test.ts: installs the table view`, `runtime/src/static/static.test.ts: installs the table view` |
| 13 | `zenmd build` output is unchanged for a page with a wide table | existing golden tests stay green (no new HTML from Go) |
| 3, 12 | print hides `.zen-table-tools` and the view | `kit/ui/src/page/css.test.ts: print hides the table tools` |
| 14 | no transition or animation on `.zen-tableview` | `css.test.ts: the table view has no animation` |

Edge cases to cover:
- an empty table;
- a table with no `thead`;
- a table with `colspan` and `rowspan`;
- a table whose caption holds markdown;
- a table that fits on a wide window but not after the sidebar opens;
- two tables on one page, where opening the second closes the first;
- a table in the `wide` layout, which has no button when it fits;
- the view open when the page is re-rendered (it closes);
- a narrow window (375px), where the bar still fits.

## UI

**Reuses** the image zoom view (docs/wiki/UI.md Principles: one style per
state):
- the same `.zen-zoom` frame, backdrop and bottom bar;
- the same keys (`+ - 0 1`, Esc);
- the same close rules.

The button is the plugin kit's quiet button (docs/wiki/PLUGIN-UI.md), and
the icon comes from the one icon set (`runtime/src/icons.ts`).

**States**
- **Fits:** no button.
- **Overflowing:** the button.
- **Open:** the view.
- There is no loading or error state. The copy is synchronous, and if the
  table is gone when the button is clicked, nothing happens.

**Tokens**
- `--bg` for the sticky cells, `--rule` for their edge, and `--fg-muted` for
  the % text.
- The button row is 13px UI font, with `margin: 0 0 var(--space-1)`.
- Black and white only. No hex.

**Accessibility**
- The view is a real `dialog` with a label, and focus stays inside it.
- Every control is a native `button` with an `aria-label`.
- The button names the caption.
- The % is announced with `aria-live="polite"`, as in the zoom.

**Narrow windows:** the bar wraps to one row of icon buttons. Under 600px the
text size starts at Fit.

## Security

| Risk | Mitigation | Test |
|---|---|---|
| The copy carries scripts or handlers | The table comes from the sanitised page. `cloneNode` copies no listeners, and nothing is parsed from strings: no `innerHTML`, no `insertAdjacentHTML` | `opening shows a copy of the table` checks that the module source has no `innerHTML` (a grep test, as zoom does) |
| Duplicate ids make links or anchors jump to the copy | ids are removed from the copy and its descendants | `opening shows a copy of the table` |
| A link in the copy goes around the app's link handling (external links opening in the webview) | Clicks on links in the copy are prevented, and the page's own link is clicked instead | `links act like the page's` |
| Observers leak across pages and keep old articles alive | `destroy` disconnects both observers and closes the view, and the page feature and static.ts call it on teardown | `one overlay at a time` (destroy part) |

## Notes

- **Out of scope:**
  - widening a table in place inside the page (the user chose the
    over-the-window view);
  - sorting and filtering;
  - an author option to open a table wide by default;
  - tables inside other plugins (a quiz, the sandbox).
- **Integration (done by the lead at merge):**
  - `kit/ui/src/page/index.ts`: after `const zoom = installImageZoom(article);`
    add `const tables = installTableView(article); // a wide table opens over the window (plugins/table)`,
    and `tables.destroy();` next to `zoom.destroy();`. Import it from
    `@zenmd/runtime/tableview/tableview`.
  - `runtime/src/static/static.ts`: the same two lines after
    `installImageZoom(root)` and its `destroy`.
  - `runtime/src/index.ts`: the two export lines from Interfaces.
  - `runtime/src/zoom/zoom.ts`: in `open`, call `registerOverlay(doc, close)`
    after building, and call its remover in `close`.
  - Check that plugins/diagrams' `viewer.js` uses the same event name and
    `detail.owner`, whichever branch merges first.
  - Find's `skip` in the page feature also skips `[data-zen-ui]`, if it does
    not already.
  - The print rule `.zen-table-tools, .zen-tableview { display: none !important; }`
    lives in the page.css block the agent owns, so it needs no change in
    print.css.
- **App Store / sandbox:** none.
- **Changelog:** `Added: a table too wide for the reading column gets an Expand button that opens it over the whole window, with the header row and row headers in view, text size and Fit, in the app and in exported websites (plugins/table).`

## Prompt

> Implement `specs/plugins/table/spec.md` following the rules in
> `specs/README.md` ("Rules for every feature agent") and docs/wiki/PLUGIN-UI.md
> (read its review checklist). You work only in the worktree
> `/Users/dragosh/Developer/zen-worktrees/plugins-table` on branch
> `feature/plugins-table`, and only in the paths under *Owns*.
>
> Context:
> - Read `runtime/src/zoom/zoom.ts` and `specs/ui/image-zoom/spec.md` first.
>   The table view is that view's sibling: the same overlay classes, the same
>   bar, the same keys and close rules.
> - Read `builtins/plugins/table/table.go`: its output is
>   `.zen-block.zt > .zen-block-body > .zt-scroll > table`.
> - Read `runtime/src/css/page.css`: markdown tables are
>   `display: block; overflow-x: auto`.
> - Read `kit/ui/src/find/index.ts` for `skip`.
> - Do the changes listed under Integration only if a test can't be written
>   without them. In that case stop and say so: the lead makes those changes
>   at merge. Your tests for behaviour 12 may fail until then, so mark them
>   `test.todo` with the reason.
>
> Steps:
> 1. Write the tests in the Tests table and commit them failing.
> 2. Build `runtime/src/tableview/`, the CSS block in page.css, the icon, and
>    the demo section in `examples/demo/features/reader.md`: a "Wide tables"
>    section with one markdown table of about 10 columns and one `csv-table`
>    with a caption and `stub-columns: 1`, and a sentence on the Expand button.
>
> Rules:
> - No new dependencies or downloads.
> - Never launch the app, run `task install`, `task package` or
>   `task release`, start servers, push, or touch `main`.
> - Run any go command with `HOME` and `XDG_CONFIG_HOME` set to a throwaway
>   temp folder.
>
> Done when, in the worktree:
> - `cd apps/zenmark/frontend && npx tsc --noEmit && npx vitest run` passes
>   (it runs the app, kit and runtime projects);
> - `go vet ./... && go test ./...` passes (with a throwaway HOME).
>
> Report the files you changed, the tests, the test summaries, and the
> Integration lines that are still to do.
