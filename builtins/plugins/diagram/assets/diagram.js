// Renders .zen-diagram placeholders with the vendored mermaid bundle, which is
// loaded lazily as a classic script (it is an IIFE defining globalThis.mermaid).
export const MERMAID_URL = "/_plugins/diagram/mermaid.min.js";
// The queue below is a copy of queue.js, which has the tests (a test checks that they match):
// a built site in portable mode cannot load a script that imports another file.
// A queue that draws figures one at a time, and only once they come near the
// screen. Each plugin ships its own identical copy (plugins cannot import each
// other); a Go test checks they match. See specs/plugins/diagram-performance.
const NEAR = 1000; // px around the viewport at which a waiting figure is queued

/**
 * Pure ordering: the index of the item to draw next, or -1. Items are
 * { visible, top }: figures on the screen come first, then the rest from the
 * top of the page down. Ties keep their order of arrival.
 */
function nextIndex(items) {
  let best = -1;
  for (let i = 0; i < items.length; i++) {
    if (best < 0) { best = i; continue; }
    const a = items[i], b = items[best];
    if ((a.visible && !b.visible) || (a.visible === b.visible && a.top < b.top)) best = i;
  }
  return best;
}

const defaultYield = () => new Promise((resolve) => setTimeout(resolve, 0));

/**
 * createQueue({ doc, near, yield, IntersectionObserver }) returns
 * { add(el, draw), drawAll(), destroy(), sweep(), size }.
 * add() resolves when el was drawn, failed or was dropped, and sets
 * data-zen-pending on el until then, so a print waits for it.
 */
function createQueue(opts = {}) {
  const doc = opts.doc || document;
  const win = doc.defaultView || null;
  const near = opts.near === undefined ? NEAR : opts.near;
  const pause = opts.yield || defaultYield;
  const IO = opts.IntersectionObserver !== undefined ? opts.IntersectionObserver : (win && win.IntersectionObserver) || globalThis.IntersectionObserver;

  const items = new Map(); // el -> { el, draw, promise, resolve, seq }
  const ready = []; // items near the screen, waiting for their turn
  let seq = 0;
  let running = false;
  let dead = false;
  let observer = null;

  const gone = (el) => el.isConnected === false;
  const unmark = (it) => { try { delete it.el.dataset.zenPending; } catch { /* a frozen dataset */ } };

  function finish(it) {
    items.delete(it.el);
    unmark(it);
    it.resolve();
  }

  function enqueue(it) {
    if (dead || !items.has(it.el) || ready.includes(it)) return;
    if (observer) observer.unobserve(it.el);
    ready.push(it);
    run();
  }

  function measure(it) {
    if (typeof it.el.getBoundingClientRect !== "function") return { visible: false, top: it.seq };
    const r = it.el.getBoundingClientRect();
    const h = (win && win.innerHeight) || 0;
    return { visible: r.bottom > 0 && r.top < h, top: r.top };
  }

  async function run() {
    if (running) return;
    running = true;
    try {
      while (!dead && ready.length > 0) {
        const i = nextIndex(ready.map(measure));
        const it = ready.splice(i, 1)[0];
        if (gone(it.el)) { finish(it); continue; }
        try {
          await it.draw(it.el);
        } catch (err) {
          // the draw reports its own errors on its figure; the queue just goes on
        }
        finish(it);
        if (ready.length > 0) await pause();
      }
    } finally {
      running = false;
    }
    if (!dead && ready.length > 0) run();
  }

  function onEntries(entries) {
    const hit = entries.filter((e) => e.isIntersecting).map((e) => items.get(e.target)).filter(Boolean);
    hit.sort((a, b) => a.seq - b.seq);
    for (const it of hit) enqueue(it);
  }

  function drawAll() {
    if (dead) return;
    for (const it of [...items.values()]) enqueue(it);
  }

  if (typeof IO === "function") {
    try { observer = new IO(onEntries, { rootMargin: `${near}px 0px` }); } catch { observer = null; }
  }
  if (win && typeof win.addEventListener === "function") win.addEventListener("beforeprint", drawAll);

  return {
    add(el, draw) {
      const have = items.get(el);
      if (have) return have.promise;
      if (dead) return Promise.resolve();
      let resolve;
      const promise = new Promise((r) => (resolve = r));
      const it = { el, draw, promise, resolve, seq: seq++ };
      items.set(el, it);
      el.dataset.zenPending = "";
      const inPrint = typeof el.closest === "function" && el.closest(".zen-print-preview");
      if (observer && !inPrint) observer.observe(el);
      else enqueue(it);
      return promise;
    },
    drawAll,
    /** Drops the figures that left the page. */
    sweep() {
      for (const it of [...items.values()]) {
        if (!gone(it.el)) continue;
        const k = ready.indexOf(it);
        if (k >= 0) ready.splice(k, 1);
        if (observer) observer.unobserve(it.el);
        finish(it);
      }
    },
    destroy() {
      if (dead) return;
      dead = true;
      if (observer) observer.disconnect();
      if (win && typeof win.removeEventListener === "function") win.removeEventListener("beforeprint", drawAll);
      ready.length = 0;
      for (const it of [...items.values()]) finish(it);
    },
    get size() { return items.size; },
  };
}

const MAX_SOURCE = 200 * 1024; // characters of source drawn in the page, as for Lini
const THEMES = ["neutral", "dark", "default"];
let loading = null;
let seq = 0;

/** Loads mermaid once; resolves with globalThis.mermaid. */
export function loadMermaid(doc = document, g = globalThis, url = MERMAID_URL) {
  if (g.mermaid) return Promise.resolve(g.mermaid);
  if (loading && loading.doc === doc) return loading.p;
  doc.head.querySelectorAll("script").forEach((s) => s.dataset && s.dataset.zenDiagram !== undefined && s.remove());
  const p = new Promise((resolve, reject) => {
    const s = doc.createElement("script");
    s.src = url;
    s.dataset.zenDiagram = "";
    s.addEventListener("load", () => (g.mermaid ? resolve(g.mermaid) : reject(new Error("mermaid did not load"))));
    s.addEventListener("error", () => reject(new Error("could not load mermaid")));
    doc.head.append(s);
  });
  loading = { doc, p };
  p.catch(() => { if (loading && loading.p === p) loading = null; });
  return p;
}

function appIsDark(doc) {
  const t = doc.documentElement && doc.documentElement.dataset.theme;
  if (t === "dark") return true;
  if (t === "light") return false;
  const mm = doc.defaultView && doc.defaultView.matchMedia;
  return !!(mm && mm.call(doc.defaultView, "(prefers-color-scheme: dark)").matches);
}

/** The mermaid theme for el: its data-theme, forced to dark/neutral by the app theme. */
export function mermaidTheme(el, doc = document) {
  if (appIsDark(doc)) return "dark";
  const t = THEMES.includes(el.dataset.theme) ? el.dataset.theme : "neutral";
  return t === "dark" ? "neutral" : t;
}

/** Renders one diagram wrapper (once) into an svg, or its error under the source. */
export async function renderOne(el, mermaid) {
  if (el.dataset.rendered !== undefined) return;
  el.dataset.rendered = "";
  const doc = el.ownerDocument || document;
  const pre = el.querySelector("pre");
  if (!pre) { el.dataset.drawn = ""; return; }
  if ((pre.textContent || "").length > MAX_SOURCE) {
    showError(el, "the diagram is too large to draw");
    return;
  }
  const id = "zen-diagram-" + ++seq;
  mermaid.initialize({ startOnLoad: false, securityLevel: "strict", theme: mermaidTheme(el, doc) });
  try {
    const { svg } = await mermaid.render(id, pre.textContent);
    const box = doc.createElement("div");
    box.className = "zen-diagram-svg";
    box.innerHTML = svg; // mermaid output, sanitised by mermaid in strict mode
    pre.remove();
    el.append(box);
    el.dataset.drawn = ""; // the placeholder of page.css gives way to the figure
  } catch (err) {
    const stray = doc.getElementById && doc.getElementById("d" + id);
    if (stray) stray.remove();
    showError(el, err);
  }
}

function showError(el, err) {
  el.dataset.drawn = ""; // the source and the message show instead of the placeholder
  const box = (el.ownerDocument || document).createElement("div");
  box.className = "zen-plugin-error";
  const msg = err && err.message ? err.message : String(err);
  box.textContent = "diagram: " + msg;
  el.append(box);
  // The Console shows the page's console, so a broken diagram leaves a line there too.
  const first = ("diagram: " + msg).split("\n")[0];
  console.warn(first.length > 200 ? first.slice(0, 199) + "…" : first);
}

/** Renders every not-yet-rendered diagram under root. */
export async function renderAll(root, mermaid) {
  for (const el of root.querySelectorAll(".zen-diagram:not([data-rendered])")) await renderOne(el, mermaid);
}

const pages = new WeakMap(); // document -> the page's queue

function pageFor(doc) {
  let page = pages.get(doc);
  if (page) return page;
  page = {
    els: new Set(), // the diagrams mounted on this page and not released
    queue: createQueue({ doc }),
    release(el) {
      this.els.delete(el);
      this.queue.sweep();
      for (const e of this.els) if (e.isConnected === false) this.els.delete(e);
      if (this.els.size > 0) return;
      if (pages.get(doc) === this) pages.delete(doc);
      this.queue.destroy();
    },
  };
  pages.set(doc, page);
  return page;
}

// The page runtime mounts this on every [data-zen-plugin="diagram"] wrapper. The diagram is
// queued: it draws when it is near the screen and the ones before it are done, and mermaid
// loads when the first one draws. The returned clean-up leaves the page.
export default {
  mount(wrapper, ctx) {
    // The renderer's [data-zen-plugin] wrapper holds one .zen-diagram (with its theme).
    const el = (String(wrapper.className || "").split(/\s+/).includes("zen-diagram") ? wrapper : wrapper.querySelector(".zen-diagram")) || wrapper;
    const doc = el.ownerDocument || document;
    // ctx.assetUrl finds the bundle in the app and in a site made by `zenmd build`.
    const url = ctx && typeof ctx.assetUrl === "function" ? ctx.assetUrl("mermaid.min.js") : MERMAID_URL;
    const page = pageFor(doc);
    page.queue.sweep();
    page.els.add(el);
    const draw = async (e) => {
      let mermaid;
      try {
        mermaid = await loadMermaid(doc, globalThis, url);
      } catch (err) {
        showError(e, err);
        return;
      }
      await renderOne(e, mermaid);
    };
    return page.queue.add(el, draw).then(() => () => page.release(el));
  },
};
