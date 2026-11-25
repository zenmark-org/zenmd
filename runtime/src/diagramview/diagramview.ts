// Figure view (specs/plugins/diagrams): a drawn figure that opts in with data-zen-zoomable (a mermaid diagram, a
// Lini figure, any plugin's svg) gets a viewport with corner controls: zoom, pan, reset, and Expand, which moves the
// svg itself over the window in the image zoom's frame. Sibling of the table view: one install per page root, observers
// that find marked figures, the same overlay rule (one view over the window at a time). Plain DOM; the svg is moved,
// never copied or rebuilt from text.

import { registerOverlay } from "../tableview/overlay";

export const SCALE_MIN = 0.1; // of the fitted size in place; of the natural size expanded
export const SCALE_MAX = 8;
export const STEP = 1.25;
export const PAN_STEP = 0.1; // of the viewport, per arrow click or key
export const DRAG_SLOP = 4; // px before a press becomes a drag
const PAD = 32; // room around the figure in the expanded view
const EPS = 1e-6;

export type ViewState = { scale: number; x: number; y: number };

export interface DiagramView {
  /** Re-checks every marked figure now (the observers call it; tests too). */
  refresh(): void;
  destroy(): void;
  /** How many views are open over the window (0 or 1). */
  readonly openCount: number;
}

export const clampScale = (s: number): number => Math.min(SCALE_MAX, Math.max(SCALE_MIN, s));

/** Zooms by factor around the viewport point (px, py): the figure point under it stays there. */
export function zoomAt(s: ViewState, factor: number, px: number, py: number): ViewState {
  const scale = clampScale(s.scale * factor);
  const k = scale / s.scale;
  return { scale, x: px - (px - s.x) * k, y: py - (py - s.y) * k };
}

/** Keeps at least a quarter of the viewport (or the whole figure, when smaller) on the figure of natural size w×h. */
export function limitPan(s: ViewState, w: number, h: number, vw: number, vh: number): ViewState {
  const sw = w * s.scale, sh = h * s.scale;
  const qx = Math.min(vw / 4, sw), qy = Math.min(vh / 4, sh);
  const lim = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v));
  return { scale: s.scale, x: lim(s.x, qx - sw, vw - qx), y: lim(s.y, qy - sh, vh - qy) };
}

/** The state that fits a figure w×h in a viewport vw×vh (pad px around it), never past maxScale, centred. */
export function fit(w: number, h: number, vw: number, vh: number, maxScale: number, pad = 0): ViewState {
  const scale = w > 0 && h > 0 ? Math.min(maxScale, (vw - 2 * pad) / w, (vh - 2 * pad) / h) : 1;
  return { scale, x: (vw - w * scale) / 2, y: (vh - h * scale) / 2 };
}

const round = (n: number) => String(+n.toFixed(3));
const transformOf = (s: ViewState) => `translate(${round(s.x)}px, ${round(s.y)}px) scale(${round(s.scale)})`;
const isSvg = (n: Element) => n.localName === "svg";
const POSITIONED = ["position", "left", "top", "width", "height", "max-width", "transform", "transform-origin"] as const;

function button(doc: Document, label: string, text: string, onClick: () => void, className = ""): HTMLButtonElement {
  const b = doc.createElement("button");
  b.type = "button";
  b.className = `zen-zoom-btn ${className}`.trim();
  b.title = label;
  b.setAttribute("aria-label", label);
  b.textContent = text;
  b.addEventListener("click", (e) => { e.stopPropagation(); onClick(); });
  return b;
}

/** Drag to pan on el. A move past DRAG_SLOP is a drag, and the click that ends it is swallowed (links keep working). */
function dragger(el: HTMLElement, win: Window, opts: { canPan(): boolean; pan(dx: number, dy: number): void; panning(on: boolean): void }): void {
  let down: { x: number; y: number; id: number } | null = null;
  let dragging = false;
  let swallow = false;
  let timer = 0;
  let lastX = 0, lastY = 0;
  el.addEventListener("pointerdown", (e) => {
    const p = e as PointerEvent;
    if (p.button !== 0 || p.isPrimary === false) return;
    if ((e.target as Element | null)?.closest?.(".zen-figure-tools, .zen-zoom-bar")) return;
    swallow = false;
    down = { x: p.clientX, y: p.clientY, id: p.pointerId };
    lastX = p.clientX; lastY = p.clientY;
    dragging = false;
  });
  el.addEventListener("pointermove", (e) => {
    const p = e as PointerEvent;
    if (!down) return;
    if (!dragging) {
      if (Math.hypot(p.clientX - down.x, p.clientY - down.y) <= DRAG_SLOP) return;
      dragging = true;
      if (opts.canPan()) {
        try { if (down.id !== undefined) el.setPointerCapture?.(down.id); } catch { /* not capturable */ }
        opts.panning(true);
      }
    }
    if (opts.canPan()) opts.pan(p.clientX - lastX, p.clientY - lastY);
    lastX = p.clientX; lastY = p.clientY;
  });
  const end = () => {
    if (!down) return;
    if (dragging) {
      swallow = true;
      win.clearTimeout(timer);
      timer = win.setTimeout(() => { swallow = false; }, 100);
      opts.panning(false);
    }
    down = null;
    dragging = false;
  };
  el.addEventListener("pointerup", end);
  el.addEventListener("pointercancel", end);
  el.addEventListener("click", (e) => {
    if (!swallow) return;
    swallow = false;
    e.preventDefault();
    e.stopPropagation();
  }, true);
}

interface Viewer {
  holder: HTMLElement;
  svg: SVGElement;
  viewport: HTMLElement;
  saved: Record<string, string>;
  w: number; // the figure's fitted size in the column
  h: number;
  vw: number; // the viewport's width when measured
  state: ViewState;
  expanded: { close(): void } | null;
  destroy(): void;
  remeasure(): void;
}

export function installDiagramView(root: HTMLElement): DiagramView {
  const doc = root.ownerDocument;
  const win = doc.defaultView ?? window;
  const viewers = new Map<HTMLElement, Viewer>();
  let destroyed = false;
  let open: Viewer | null = null;

  const ro = typeof ResizeObserver !== "undefined" ? new ResizeObserver(() => schedule()) : null;
  let queued = 0;
  function schedule(): void {
    if (queued || destroyed) return;
    const raf = win.requestAnimationFrame ?? ((f: FrameRequestCallback) => win.setTimeout(f, 16));
    queued = raf.call(win, () => { queued = 0; refresh(); });
  }

  const labelOf = (svg: Element) =>
    (svg.querySelector("title")?.textContent ?? "").replace(/\s+/g, " ").trim() || (svg.getAttribute("aria-label") ?? "").trim();

  /** The figure's natural size: width and height in px, else its viewBox, else the fitted size. */
  function naturalSize(v: Viewer): { w: number; h: number } {
    const num = (a: string | null) => (a && !a.trim().endsWith("%") ? parseFloat(a) : NaN);
    const aw = num(v.svg.getAttribute("width")), ah = num(v.svg.getAttribute("height"));
    if (aw > 0 && ah > 0) return { w: aw, h: ah };
    const vb = (v.svg.getAttribute("viewBox") ?? "").trim().split(/[\s,]+/).map(parseFloat);
    if (vb.length === 4 && vb[2] > 0 && vb[3] > 0) return { w: vb[2], h: vb[3] };
    return { w: v.w, h: v.h };
  }

  // ---- in place ----
  function applyInPlace(v: Viewer): void {
    const s = v.svg.style;
    s.position = "absolute";
    s.left = "0";
    s.top = "0";
    s.width = `${v.w}px`;
    s.height = `${v.h}px`;
    s.maxWidth = "none";
    s.transformOrigin = "0 0";
    s.transform = transformOf(v.state);
    v.viewport.style.height = `${v.h}px`;
    v.viewport.toggleAttribute("data-zoomed", v.state.scale > 1 + EPS);
  }
  const fitInPlace = (v: Viewer) => fit(v.w, v.h, v.viewport.getBoundingClientRect().width || v.w, v.h, 1);
  function setInPlace(v: Viewer, next: ViewState): void {
    if (v.expanded) return;
    const vw = v.viewport.getBoundingClientRect().width || v.w;
    v.state = limitPan(next, v.w, v.h, vw, v.h);
    applyInPlace(v);
  }
  const zoomInPlace = (v: Viewer, factor: number, px?: number, py?: number) => {
    const vw = v.viewport.getBoundingClientRect().width || v.w;
    setInPlace(v, zoomAt(v.state, factor, px ?? vw / 2, py ?? v.h / 2));
  };
  const panInPlace = (v: Viewer, dx: number, dy: number) => {
    if (v.state.scale <= 1 + EPS) return; // at fit there is nothing to pan
    setInPlace(v, { ...v.state, x: v.state.x + dx, y: v.state.y + dy });
  };
  function restoreInline(v: Viewer): void {
    for (const p of POSITIONED) {
      if (v.saved[p]) v.svg.style.setProperty(p, v.saved[p]);
      else v.svg.style.removeProperty(p);
    }
  }
  function measure(v: Viewer): void {
    restoreInline(v); // as the figure looks without the view: the column's width
    v.viewport.style.height = "";
    const r = v.svg.getBoundingClientRect();
    v.w = r.width;
    v.h = r.height;
    v.vw = v.viewport.getBoundingClientRect().width || v.w;
  }

  // ---- the expanded view ----
  function expand(v: Viewer): void {
    if (destroyed || v.expanded || !v.holder.isConnected) return;
    open?.expanded?.close();
    const active = doc.activeElement as HTMLElement | null;
    const returnTo = active && v.viewport.contains(active) ? active : v.viewport;
    const title = labelOf(v.svg);

    const overlay = doc.createElement("div");
    overlay.className = "zen-zoom zen-figure-zoom";
    overlay.setAttribute("role", "dialog");
    overlay.setAttribute("aria-modal", "true");
    overlay.setAttribute("aria-label", title ? `Diagram: ${title}` : "Diagram");
    const stage = doc.createElement("div");
    stage.className = "zen-zoom-stage";

    const nat = naturalSize(v);
    const placeholder = doc.createElement("div");
    placeholder.className = "zen-figure-placeholder";
    placeholder.style.height = `${v.h}px`;
    v.svg.replaceWith(placeholder);
    const s = v.svg.style;
    s.position = "absolute";
    s.left = "0";
    s.top = "0";
    s.width = `${nat.w}px`;
    s.height = `${nat.h}px`;
    s.maxWidth = "none";
    s.transformOrigin = "0 0";
    stage.append(v.svg);

    const label = doc.createElement("span");
    label.className = "zen-zoom-scale";
    label.setAttribute("aria-live", "polite");
    const closeBtn = button(doc, "Close (Esc)", "✕", () => close());
    const bar = doc.createElement("div");
    bar.className = "zen-zoom-bar";
    bar.append(
      button(doc, "Zoom out (−)", "−", () => zoom(1 / STEP)),
      label,
      button(doc, "Zoom in (+)", "+", () => zoom(STEP)),
      button(doc, "Fit (0)", "Fit", () => set(fitState())),
      closeBtn,
    );
    overlay.append(stage, bar);

    const size = () => {
      const r = stage.getBoundingClientRect();
      return { vw: r.width || win.innerWidth, vh: r.height || win.innerHeight };
    };
    const fitState = () => { const { vw, vh } = size(); return fit(nat.w, nat.h, vw, vh, 1, PAD); };
    let st: ViewState = { scale: 1, x: 0, y: 0 };
    function set(next: ViewState): void {
      const { vw, vh } = size();
      st = limitPan(next, nat.w, nat.h, vw, vh);
      v.svg.style.transform = transformOf(st);
      label.textContent = `${Math.round(st.scale * 100)}%`;
    }
    function zoom(factor: number, px?: number, py?: number): void {
      const { vw, vh } = size();
      set(zoomAt(st, factor, px ?? vw / 2, py ?? vh / 2));
    }
    const pan = (dx: number, dy: number) => set({ ...st, x: st.x + dx, y: st.y + dy });

    dragger(stage, win, {
      canPan: () => true,
      pan,
      panning: (on) => overlay.toggleAttribute("data-panning", on),
    });
    stage.addEventListener("click", (e) => {
      if (e.target === stage) close(); // the backdrop around the figure
    });
    const onWheel = (e: WheelEvent) => {
      e.preventDefault();
      if (e.ctrlKey || e.metaKey) {
        const r = stage.getBoundingClientRect();
        zoom(Math.exp(-e.deltaY * 0.01), e.clientX - r.left, e.clientY - r.top);
      } else pan(-e.deltaX, -e.deltaY);
    };
    const buttons = () => Array.from(overlay.querySelectorAll<HTMLElement>("button:not([disabled])"));
    const onKey = (e: KeyboardEvent) => {
      if (e.metaKey || e.ctrlKey || e.altKey) return;
      const k = e.key;
      const { vw, vh } = size();
      const f = (e.shiftKey ? 0.5 : PAN_STEP);
      if (k === "Escape") close();
      else if (k === "+" || k === "=") zoom(STEP);
      else if (k === "-" || k === "_") zoom(1 / STEP);
      else if (k === "0") set(fitState());
      else if (k === "1") zoom(1 / st.scale);
      else if (k === "ArrowLeft") pan(f * vw, 0);
      else if (k === "ArrowRight") pan(-f * vw, 0);
      else if (k === "ArrowUp") pan(0, f * vh);
      else if (k === "ArrowDown") pan(0, -f * vh);
      else if (k === "Tab") {
        const list = buttons();
        const i = list.indexOf(doc.activeElement as HTMLElement);
        const n = list.length;
        list[e.shiftKey ? (i <= 0 ? n - 1 : i - 1) : (i < 0 || i === n - 1 ? 0 : i + 1)]?.focus();
      } else return;
      e.preventDefault();
      e.stopPropagation();
    };
    const onResize = () => { if (Math.abs(st.scale - fitAtOpen) < 1e-3) set(fitState()); else set(st); };

    overlay.addEventListener("wheel", onWheel, { passive: false });
    doc.addEventListener("keydown", onKey, true);
    win.addEventListener("resize", onResize);
    doc.body.append(overlay);
    const unregister = registerOverlay(doc, () => close()); // one view over the window at a time
    const fitAtOpen = fitState().scale;
    set(fitState());
    closeBtn.focus();

    function close(): void {
      if (v.expanded !== handle) return;
      v.expanded = null;
      open = null;
      unregister();
      doc.removeEventListener("keydown", onKey, true);
      win.removeEventListener("resize", onResize);
      placeholder.replaceWith(v.svg); // the same element, back where it was
      applyInPlace(v);
      overlay.remove();
      if (returnTo.isConnected) returnTo.focus?.();
    }
    const handle = { close };
    v.expanded = handle;
    open = v;
  }

  // ---- attaching ----
  function attach(holder: HTMLElement, svg: SVGElement): Viewer | null {
    const first = svg.getBoundingClientRect();
    if (!(first.width > 0 && first.height > 0)) return null; // not laid out (hidden): left alone
    const saved: Record<string, string> = {};
    for (const p of POSITIONED) saved[p] = svg.style.getPropertyValue(p);

    const viewport = doc.createElement("div");
    viewport.className = "zen-figure-view";
    viewport.tabIndex = 0;
    viewport.setAttribute("role", "img");
    viewport.setAttribute("aria-label", labelOf(svg) || "Diagram");
    const v: Viewer = {
      holder, svg, viewport, saved, w: first.width, h: first.height, vw: 0,
      state: { scale: 1, x: 0, y: 0 }, expanded: null,
      destroy() {
        v.expanded?.close();
        restoreInline(v);
        ro?.unobserve(viewport);
        if (viewport.parentNode) {
          viewport.before(holder);
          viewport.remove();
        }
      },
      remeasure() {
        const wasFit = Math.abs(v.state.scale - 1) < EPS;
        const keep = v.state.scale;
        measure(v);
        v.state = wasFit ? fitInPlace(v) : limitPan({ ...v.state, scale: keep }, v.w, v.h, v.vw, v.h);
        applyInPlace(v);
      },
    };

    const bar = doc.createElement("div");
    bar.className = "zen-figure-tools zen-zoom-bar";
    bar.setAttribute("data-zen-ui", "");
    const pan = (dx: number, dy: number) => () => panInPlace(v, dx * PAN_STEP * (v.vw || v.w), dy * PAN_STEP * v.h);
    bar.append(
      button(doc, "Zoom in", "+", () => zoomInPlace(v, STEP)),
      button(doc, "Zoom out", "−", () => zoomInPlace(v, 1 / STEP)),
      button(doc, "Reset", "⟲", () => setInPlace(v, fitInPlace(v))),
      button(doc, "Expand", "⤢", () => expand(v)),
      button(doc, "Pan left", "←", pan(1, 0), "zen-figure-pan"),
      button(doc, "Pan up", "↑", pan(0, 1), "zen-figure-pan"),
      button(doc, "Pan right", "→", pan(-1, 0), "zen-figure-pan"),
      button(doc, "Pan down", "↓", pan(0, -1), "zen-figure-pan"),
    );

    holder.before(viewport);
    viewport.append(holder, bar);
    measure(v);
    v.state = fitInPlace(v);
    applyInPlace(v);

    dragger(viewport, win, {
      canPan: () => v.state.scale > 1 + EPS && !v.expanded,
      pan: (dx, dy) => setInPlace(v, { ...v.state, x: v.state.x + dx, y: v.state.y + dy }),
      panning: (on) => viewport.toggleAttribute("data-panning", on),
    });
    viewport.addEventListener("wheel", (e) => {
      if (!(e.ctrlKey || e.metaKey)) return; // a plain wheel scrolls the page
      e.preventDefault();
      const r = viewport.getBoundingClientRect();
      zoomInPlace(v, Math.exp(-e.deltaY * 0.01), e.clientX - r.left, e.clientY - r.top);
    }, { passive: false });
    viewport.addEventListener("keydown", (e) => {
      if (e.target !== viewport || e.metaKey || e.ctrlKey || e.altKey) return;
      const f = e.shiftKey ? 0.5 : PAN_STEP;
      const k = e.key;
      const vw = v.vw || v.w;
      if (k === "+" || k === "=") zoomInPlace(v, STEP);
      else if (k === "-" || k === "_") zoomInPlace(v, 1 / STEP);
      else if (k === "0") setInPlace(v, fitInPlace(v));
      else if (k === "Enter") expand(v);
      else if (k.startsWith("Arrow") && v.state.scale > 1 + EPS) {
        if (k === "ArrowLeft") panInPlace(v, f * vw, 0);
        else if (k === "ArrowRight") panInPlace(v, -f * vw, 0);
        else if (k === "ArrowUp") panInPlace(v, 0, f * v.h);
        else panInPlace(v, 0, -f * v.h);
      } else return;
      e.preventDefault();
      e.stopPropagation();
    });
    ro?.observe(viewport);
    return v;
  }

  function refresh(): void {
    if (destroyed) return;
    const marked = new Set<HTMLElement>(Array.from(root.querySelectorAll<HTMLElement>("[data-zen-zoomable]")));
    for (const [holder, v] of Array.from(viewers)) {
      const gone = !marked.has(holder) || !holder.isConnected;
      const redrawn = !v.expanded && v.svg.parentElement !== holder;
      if (gone || redrawn) { v.destroy(); viewers.delete(holder); }
    }
    for (const holder of Array.from(marked)) {
      const v = viewers.get(holder);
      if (v) {
        if (!v.expanded && (v.viewport.getBoundingClientRect().width || v.vw) !== v.vw) v.remeasure(); // the column changed
        continue;
      }
      const svgs = (Array.from(holder.children) as Element[]).filter(isSvg);
      const nv = svgs.length === 1 ? attach(holder, svgs[0] as SVGElement) : null;
      if (nv) { viewers.set(holder, nv); ro?.unobserve(holder); } else ro?.observe(holder); // watch until it has a size
    }
    mo.takeRecords(); // our own changes
  }
  const mo = new MutationObserver(() => schedule());
  mo.observe(root, { childList: true, subtree: true });
  win.addEventListener("resize", schedule);
  refresh();

  return {
    refresh,
    get openCount() { return open ? 1 : 0; },
    destroy() {
      if (destroyed) return;
      for (const v of Array.from(viewers.values())) v.destroy();
      viewers.clear();
      destroyed = true;
      mo.disconnect();
      ro?.disconnect();
      win.removeEventListener("resize", schedule);
    },
  };
}
