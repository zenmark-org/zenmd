// A queue that draws figures one at a time, and only once they come near the
// screen. Each plugin ships its own identical copy (plugins cannot import each
// other); a Go test checks they match. See specs/plugins/diagram-performance.
export const NEAR = 1000; // px around the viewport at which a waiting figure is queued

/**
 * Pure ordering: the index of the item to draw next, or -1. Items are
 * { visible, top }: figures on the screen come first, then the rest from the
 * top of the page down. Ties keep their order of arrival.
 */
export function nextIndex(items) {
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
export function createQueue(opts = {}) {
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
