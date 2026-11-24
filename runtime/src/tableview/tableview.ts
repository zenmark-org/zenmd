// Table view (specs/plugins/table): a table that does not fit the reading column gets an Expand button that opens
// a copy of it over the window, with the header row and row headers in view. Sibling of the image zoom: the same
// overlay classes, bar, keys and close rules. Plain DOM, no HTML strings: the copy is a cloneNode of a table the
// sanitiser already allowed.

import { icon } from "../icons";
import { registerOverlay } from "./overlay";

export const TABLE_SIZE_MIN = 0.5;
export const TABLE_SIZE_MAX = 2;
const STEP = 1.25;
const PAD = 32; // the stage's padding on each side
const NARROW = 600; // under this the view starts at Fit

export interface TableView {
  open(table: HTMLTableElement): void;
  close(): void;
  readonly isOpen: boolean;
  /** Re-checks every table now (the observers call it; tests too). */
  refresh(): void;
  destroy(): void;
}

const clamp = (s: number) => Math.min(TABLE_SIZE_MAX, Math.max(TABLE_SIZE_MIN, s));
const isTable = (n: unknown): n is HTMLTableElement => !!n && (n as Element).nodeType === 1 && (n as Element).tagName === "TABLE";

/** The element that scrolls sideways: .zt-scroll for the built-in's tables, else the table itself. */
const scrollBox = (table: HTMLTableElement): HTMLElement => table.closest<HTMLElement>(".zt-scroll") ?? table;

/** The table in root that may get an Expand button, or null (behaviour 1). */
export function expandableTable(el: Element, root: HTMLElement): HTMLTableElement | null {
  if (!isTable(el) || el === (root as Element) || !root.contains(el)) return null;
  if (el.closest("a[href], button, [data-zen-nozoom]")) return null;
  const plugin = el.closest("[data-zen-plugin]");
  if (plugin && plugin.getAttribute("data-zen-plugin") !== "table") return null;
  if (el.parentElement?.closest("table")) return null; // nested
  return el;
}

/** Whether the table overflows its scroll box (behaviour 2). */
export function overflows(table: HTMLTableElement): boolean {
  const box = scrollBox(table);
  return box.scrollWidth > box.clientWidth + 1;
}

/** The size Fit picks for a table of natural width w in a stage of width avail (behaviour 7). */
export function fitSize(w: number, avail: number): number {
  if (!(w > 0)) return 1;
  return clamp(Math.min(1, avail / w));
}

function viewButton(doc: Document, label: string, text: string, onClick: () => void, className = ""): HTMLButtonElement {
  const b = doc.createElement("button");
  b.type = "button";
  b.className = `zen-zoom-btn ${className}`.trim();
  b.title = label;
  b.setAttribute("aria-label", label);
  b.textContent = text;
  b.addEventListener("click", (e) => { e.stopPropagation(); onClick(); });
  return b;
}

const captionOf = (table: HTMLTableElement) => (table.querySelector("caption")?.textContent ?? "").replace(/\s+/g, " ").trim();

export function installTableView(root: HTMLElement): TableView {
  const doc = root.ownerDocument;
  const win = doc.defaultView ?? window;
  let view: { close(): void; source: HTMLTableElement } | null = null;
  let destroyed = false;

  // ---- the buttons ----
  function makeRow(table: HTMLTableElement): HTMLElement {
    const row = doc.createElement("div");
    row.className = "zen-table-tools";
    row.setAttribute("data-zen-ui", "");
    const b = doc.createElement("button");
    b.type = "button";
    b.className = "zen-btn zen-btn-quiet zen-table-expand";
    b.title = "Expand table";
    const cap = captionOf(table);
    b.setAttribute("aria-label", cap ? `Expand table: ${cap}` : "Expand table");
    b.append(icon(doc, "expand"), doc.createTextNode(" Expand"));
    row.append(b);
    return row;
  }
  const rowBefore = (box: HTMLElement): HTMLElement | null => {
    const p = box.previousElementSibling;
    return p && p.classList.contains("zen-table-tools") ? (p as HTMLElement) : null;
  };

  const watched = new Set<HTMLElement>();
  const ro = typeof ResizeObserver !== "undefined" ? new ResizeObserver(() => schedule()) : null;
  let queued = 0;
  function schedule(): void {
    if (queued || destroyed) return;
    const raf = win.requestAnimationFrame ?? ((f: FrameRequestCallback) => win.setTimeout(f, 16));
    queued = raf.call(win, () => { queued = 0; refresh(); });
  }

  function refresh(): void {
    if (destroyed) return;
    const keep = new Set<HTMLElement>();
    for (const t of Array.from(root.querySelectorAll("table"))) {
      if (!expandableTable(t, root)) continue;
      const box = scrollBox(t);
      keep.add(box);
      if (ro && !watched.has(box)) { watched.add(box); ro.observe(box); }
      const row = rowBefore(box);
      if (overflows(t)) {
        if (!row) box.before(makeRow(t));
      } else row?.remove();
    }
    for (const row of Array.from(root.querySelectorAll(".zen-table-tools"))) {
      const next = row.nextElementSibling as HTMLElement | null;
      if (!next || !keep.has(next)) row.remove();
    }
    for (const box of Array.from(watched)) {
      if (!keep.has(box) || !box.isConnected) { watched.delete(box); ro?.unobserve(box); }
    }
    if (view && !view.source.isConnected) view.close(); // the page was re-rendered
    mo.takeRecords(); // our own changes
  }
  const mo = new MutationObserver(() => schedule());
  mo.observe(root, { childList: true, subtree: true });
  win.addEventListener("resize", schedule);

  // ---- the view ----
  function stick(copy: HTMLTableElement): void {
    // Several header rows stack; several row-header columns sit side by side.
    let top = 0;
    for (const tr of Array.from(copy.querySelectorAll<HTMLElement>("thead tr"))) {
      for (const c of Array.from(tr.children) as HTMLElement[]) c.style.top = `${top}px`;
      top += tr.offsetHeight || 0;
    }
    for (const tr of Array.from(copy.rows)) {
      let left = 0;
      for (const c of Array.from(tr.children) as HTMLElement[]) {
        if (!c.classList.contains("zen-tv-stub")) continue;
        c.style.left = `${left}px`;
        left += c.offsetWidth || 0;
      }
    }
  }

  function open(source: HTMLTableElement): void {
    if (destroyed || !isTable(source) || !source.isConnected || !root.contains(source)) return;
    view?.close();
    const returnFocus = doc.activeElement as HTMLElement | null;
    const trigger = rowBefore(scrollBox(source))?.querySelector<HTMLElement>(".zen-table-expand") ?? returnFocus;
    const cap = captionOf(source);

    const overlay = doc.createElement("div");
    overlay.className = "zen-zoom zen-tableview";
    overlay.setAttribute("role", "dialog");
    overlay.setAttribute("aria-modal", "true");
    overlay.setAttribute("aria-label", cap ? `Table: ${cap}` : "Table");

    const stage = doc.createElement("div");
    stage.className = "zen-zoom-stage zen-tableview-stage";
    stage.tabIndex = 0;
    const copy = source.cloneNode(true) as HTMLTableElement;
    copy.removeAttribute("id");
    for (const n of Array.from(copy.querySelectorAll("[id]"))) n.removeAttribute("id");
    copy.classList.add("zen-tableview-table");
    for (const c of Array.from(copy.querySelectorAll("thead th, thead td"))) c.classList.add("zen-tv-head");
    for (const c of Array.from(copy.querySelectorAll('th[scope="row"]'))) c.classList.add("zen-tv-stub");
    stage.append(copy);

    const label = doc.createElement("span");
    label.className = "zen-zoom-scale";
    label.setAttribute("aria-live", "polite");
    const closeBtn = viewButton(doc, "Close (Esc)", "✕", () => close());
    const barEl = doc.createElement("div");
    barEl.className = "zen-zoom-bar";
    barEl.append(
      viewButton(doc, "Smaller text (−)", "−", () => setSize(size / STEP)),
      label,
      viewButton(doc, "Larger text (+)", "+", () => setSize(size * STEP)),
      viewButton(doc, "Fit (0)", "Fit", () => setSize(fit()), "zen-zoom-fit"),
      closeBtn,
    );
    overlay.append(stage, barEl);

    let size = 1;
    let natural = 0; // the table's width at 100%
    const fit = () => fitSize(natural, (stage.clientWidth || win.innerWidth) - 2 * PAD);
    function setSize(next: number): void {
      size = clamp(next);
      stage.style.setProperty("--zen-table-size", String(size));
      label.textContent = `${Math.round(size * 100)}%`;
      stick(copy);
    }

    const links = Array.from(copy.querySelectorAll("a"));
    const pageLinks = Array.from(source.querySelectorAll("a"));
    const onLink = (e: MouseEvent) => {
      const a = (e.target as Element | null)?.closest?.("a");
      const i = a ? links.indexOf(a) : -1;
      if (i < 0) return;
      e.preventDefault(); // never around the app's link handling: the page's own link gets the click
      const target = pageLinks[i];
      close();
      if (target?.isConnected) target.click();
    };
    stage.addEventListener("click", (e) => {
      if (e.target === stage) close(); // the backdrop outside the table
      else onLink(e);
    });

    const focusables = () =>
      Array.from(overlay.querySelectorAll<HTMLElement>("a[href], button:not([disabled]), [tabindex='0']"));
    const SCROLL_KEYS: Record<string, (s: HTMLElement) => [number, number]> = {
      ArrowDown: () => [0, 48], ArrowUp: () => [0, -48], ArrowRight: () => [48, 0], ArrowLeft: () => [-48, 0],
      PageDown: (s) => [0, s.clientHeight * 0.9], PageUp: (s) => [0, -s.clientHeight * 0.9],
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.metaKey || e.ctrlKey || e.altKey) return;
      const k = e.key;
      if (k === "Escape") close();
      else if (k === "+" || k === "=") setSize(size * STEP);
      else if (k === "-" || k === "_") setSize(size / STEP);
      else if (k === "0") setSize(fit());
      else if (k === "1") setSize(1);
      else if (k === "Tab") {
        const list = focusables();
        const i = list.indexOf(doc.activeElement as HTMLElement);
        const n = list.length;
        list[e.shiftKey ? (i <= 0 ? n - 1 : i - 1) : (i < 0 || i === n - 1 ? 0 : i + 1)]?.focus();
      } else if (doc.activeElement !== stage && (k in SCROLL_KEYS || k === "Home" || k === "End") && doc.activeElement?.tagName !== "A") {
        // The stage scrolls by itself when it has the focus; from a button the keys scroll it here.
        if (k === "Home") stage.scrollTo?.({ left: 0, top: 0 });
        else if (k === "End") stage.scrollTo?.({ left: stage.scrollWidth, top: stage.scrollHeight });
        else {
          const [dx, dy] = SCROLL_KEYS[k](stage);
          stage.scrollBy?.({ left: dx, top: dy });
        }
      } else return;
      e.preventDefault();
      e.stopPropagation();
    };
    const onWheel = (e: WheelEvent) => {
      if (!e.ctrlKey) return;
      e.preventDefault();
      setSize(size * Math.exp(-e.deltaY * 0.01));
    };
    const onResize = () => { if (stage.clientWidth) stick(copy); };

    overlay.addEventListener("wheel", onWheel, { passive: false });
    doc.addEventListener("keydown", onKey, true);
    win.addEventListener("resize", onResize);
    doc.body.append(overlay);
    const unregister = registerOverlay(doc, () => close());
    natural = Math.max(copy.scrollWidth, copy.offsetWidth);
    setSize(win.innerWidth < NARROW ? fit() : 1);
    stage.scrollLeft = 0;
    stage.scrollTop = 0;
    closeBtn.focus();

    function close(): void {
      if (view !== handle) return;
      view = null;
      unregister();
      doc.removeEventListener("keydown", onKey, true);
      win.removeEventListener("resize", onResize);
      overlay.remove();
      if (trigger?.isConnected) trigger.focus?.();
    }
    const handle = { close, source };
    view = handle;
  }

  const onClick = (e: MouseEvent) => {
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    const btn = (e.target as Element | null)?.closest?.(".zen-table-expand");
    if (!btn || !root.contains(btn)) return;
    e.preventDefault();
    const box = btn.closest(".zen-table-tools")?.nextElementSibling as HTMLElement | null;
    const table = box ? (isTable(box) ? box : box.querySelector("table")) : null;
    if (table?.isConnected) open(table);
  };
  root.addEventListener("click", onClick);

  refresh();

  return {
    open,
    close: () => view?.close(),
    get isOpen() { return view !== null; },
    refresh,
    destroy() {
      if (destroyed) return;
      view?.close();
      destroyed = true;
      mo.disconnect();
      ro?.disconnect();
      watched.clear();
      win.removeEventListener("resize", schedule);
      root.removeEventListener("click", onClick);
      for (const row of Array.from(root.querySelectorAll(".zen-table-tools"))) row.remove();
    },
  };
}
