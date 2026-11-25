// Tests for diagram.js with a stub mermaid and a tiny fake DOM (no deps).
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { MERMAID_URL, loadMermaid, renderAll, mermaidTheme } from "./diagram.js";
import { NEAR, createQueue, nextIndex } from "./queue.js";

class El {
  constructor(tag, doc) { this.tagName = tag.toUpperCase(); this.ownerDocument = doc; this.children = []; this.dataset = {}; this.attrs = {}; this.className = ""; this.textContent = ""; this.innerHTML = ""; this.listeners = {}; this.parent = null; }
  append(...cs) { for (const c of cs) { c.parent = this; this.children.push(c); } }
  remove() { if (this.parent) this.parent.children = this.parent.children.filter((c) => c !== this); }
  setAttribute(k, v) { this.attrs[k] = String(v); }
  getAttribute(k) { return this.attrs[k] ?? null; }
  addEventListener(t, f) { (this.listeners[t] ||= []).push(f); }
  fire(t) { for (const f of this.listeners[t] || []) f({ type: t }); }
  all() { return this.children.flatMap((c) => [c, ...c.all()]); }
  get isConnected() { let e = this; while (e.parent) e = e.parent; return e.tagName === "BODY"; }
  closest(sel) { let e = this; while (e) { if (sel === ".zen-print-preview" && e.className.split(" ").includes("zen-print-preview")) return e; e = e.parent; } return null; }
  hasAttribute(k) { return k === "data-zen-pending" ? "zenPending" in this.dataset : k in this.attrs; }
  querySelector(s) { return this.querySelectorAll(s)[0] || null; }
  querySelectorAll(s) {
    const m = {
      ".zen-diagram:not([data-rendered])": (e) => e.className.split(" ").includes("zen-diagram") && !("rendered" in e.dataset),
      ".zen-diagram": (e) => e.className.split(" ").includes("zen-diagram"),
      "pre": (e) => e.tagName === "PRE",
      ".zen-plugin-error": (e) => e.className === "zen-plugin-error",
      ".zen-diagram-svg": (e) => e.className === "zen-diagram-svg",
    }[s];
    if (s.startsWith("script")) return this.all().filter((e) => e.tagName === "SCRIPT");
    assert.ok(m, "unsupported selector " + s);
    return this.all().filter(m);
  }
}
function makeDoc(appTheme) {
  const doc = { listeners: {} };
  doc.createElement = (t) => new El(t, doc);
  doc.head = new El("head", doc);
  doc.body = new El("body", doc);
  doc.documentElement = new El("html", doc);
  if (appTheme) doc.documentElement.dataset.theme = appTheme;
  doc.defaultView = { matchMedia: () => ({ matches: false }) };
  return doc;
}
function diagram(doc, src, theme = "neutral") {
  const d = doc.createElement("div");
  d.className = "zen-diagram"; d.dataset.theme = theme; d.dataset.zenPlugin = "diagram";
  const pre = doc.createElement("pre"); pre.textContent = src; d.append(pre);
  doc.body.append(d);
  return d;
}
function stub(fail) {
  const calls = { init: [], render: [] };
  return {
    calls,
    initialize(c) { calls.init.push(c); },
    async render(id, src) {
      calls.render.push({ id, src });
      if (fail && src.includes(fail)) throw new Error("Parse error on line 2");
      return { svg: `<svg id="${id}"><g/></svg>` };
    },
  };
}

test("renders each pending diagram into an svg and marks it", async () => {
  const doc = makeDoc();
  const a = diagram(doc, "graph TD\nA-->B");
  const b = diagram(doc, "graph LR\nX-->Y");
  const m = stub();
  await renderAll(doc.body, m);
  for (const d of [a, b]) {
    assert.equal("rendered" in d.dataset, true);
    assert.equal(d.querySelector("pre"), null);
    assert.match(d.querySelector(".zen-diagram-svg").innerHTML, /^<svg/);
  }
  assert.deepEqual(m.calls.render.map((c) => c.src), ["graph TD\nA-->B", "graph LR\nX-->Y"]);
  assert.equal(new Set(m.calls.render.map((c) => c.id)).size, 2);
  for (const c of m.calls.init) {
    assert.equal(c.startOnLoad, false);
    assert.equal(c.securityLevel, "strict");
  }
  await renderAll(doc.body, m);
  assert.equal(m.calls.render.length, 2, "rendered diagrams are skipped");
});

test("data-drawn is set once the svg or the error is shown, not before (the placeholder in page.css goes)", async () => {
  const doc = makeDoc();
  const ok = diagram(doc, "graph TD\nA-->B");
  const bad = diagram(doc, "graph TD\nBAD");
  let drawnDuringRender = null;
  const m = stub("BAD");
  const render = m.render;
  m.render = async (id, src) => { drawnDuringRender ??= "drawn" in ok.dataset; return render(id, src); };
  await renderAll(doc.body, m);
  assert.equal(drawnDuringRender, false, "still a placeholder while mermaid draws");
  assert.equal("drawn" in ok.dataset, true);
  assert.equal("drawn" in bad.dataset, true);
});

test("a failed load marks the diagram drawn, so its source and error show", async () => {
  const mod = await import("./diagram.js");
  const doc = makeDoc();
  const d = diagram(doc, "graph TD\n A-->B");
  const p = mod.default.mount(d, { assetUrl: (f) => "/x/" + f });
  doc.head.querySelectorAll("script")[0].fire("error");
  await p;
  assert.equal("drawn" in d.dataset, true);
  assert.equal(d.querySelector(".zen-plugin-error").textContent, "diagram: could not load mermaid");
});

test("syntax error keeps the source and shows the message", async () => {
  const doc = makeDoc();
  const d = diagram(doc, "graph TD\nBAD<b>");
  await renderAll(doc.body, stub("BAD"));
  assert.equal(d.querySelector("pre").textContent, "graph TD\nBAD<b>");
  const err = d.querySelector(".zen-plugin-error");
  assert.equal(err.textContent, "diagram: Parse error on line 2");
  assert.equal(err.innerHTML, "", "message set as text, not HTML");
  assert.equal("rendered" in d.dataset, true);
});

test("a diagram that fails to draw warns in the page's console, first line, 200 characters", async () => {
  const warns = [];
  const orig = console.warn;
  console.warn = (...a) => warns.push(a.join(" "));
  try {
    const doc = makeDoc();
    diagram(doc, "graph TD\nBAD");
    const m = stub("BAD");
    m.render = async () => { throw new Error("Parse error on line 2\n" + "x".repeat(300) + "\nSECOND"); };
    await renderAll(doc.body, m);
    assert.equal(warns.length, 1);
    assert.equal(warns[0], "diagram: Parse error on line 2");
    warns.length = 0;
    const doc2 = makeDoc();
    diagram(doc2, "graph TD\nBAD");
    const m2 = stub();
    m2.render = async () => { throw new Error("y".repeat(300)); };
    await renderAll(doc2.body, m2);
    assert.equal(warns.length, 1);
    assert.ok(warns[0].length <= 200, "cut to 200 characters: " + warns[0].length);
    assert.match(warns[0], /^diagram: y+/);
  } finally {
    console.warn = orig;
  }
});

test("theme follows data-theme, forced by the app theme", () => {
  const el = (t) => ({ dataset: { theme: t } });
  assert.equal(mermaidTheme(el("neutral"), makeDoc()), "neutral");
  assert.equal(mermaidTheme(el("default"), makeDoc()), "default");
  assert.equal(mermaidTheme(el("dark"), makeDoc("light")), "neutral");
  assert.equal(mermaidTheme(el("neutral"), makeDoc("dark")), "dark");
  assert.equal(mermaidTheme(el("bogus"), makeDoc()), "neutral");
  const sys = makeDoc(); sys.defaultView.matchMedia = () => ({ matches: true });
  assert.equal(mermaidTheme(el("neutral"), sys), "dark");
});

test("mermaid is lazy-loaded once from /_plugins/diagram/ as a classic script", async () => {
  const doc = makeDoc();
  const g = {};
  const p1 = loadMermaid(doc, g);
  const p2 = loadMermaid(doc, g);
  const scripts = doc.head.querySelectorAll("script");
  assert.equal(scripts.length, 1);
  assert.equal(scripts[0].src, "/_plugins/diagram/mermaid.min.js");
  assert.equal(MERMAID_URL, "/_plugins/diagram/mermaid.min.js");
  assert.notEqual(scripts[0].type, "module");
  const m = stub();
  g.mermaid = m;
  scripts[0].fire("load");
  assert.equal(await p1, m);
  assert.equal(await p2, m);
});

test("failed load rejects and can be retried", async () => {
  const doc = makeDoc();
  const g = {};
  const p = loadMermaid(doc, g);
  doc.head.querySelectorAll("script")[0].fire("error");
  await assert.rejects(p);
  loadMermaid(doc, g);
  assert.equal(doc.head.querySelectorAll("script").length, 1, "old tag removed before retry");
});

test("no remote URLs, eval or innerHTML from source in diagram.js", () => {
  const src = readFileSync(new URL("./diagram.js", import.meta.url), "utf8");
  assert.doesNotMatch(src, /[a-z][a-z0-9+.-]*:\/\//i);
  assert.doesNotMatch(src, /\beval\s*\(|new Function/);
  assert.match(src, /securityLevel:\s*"strict"/);
});

test("default export mounts one wrapper with the page runtime's shape", async () => {
  const mod = await import("./diagram.js");
  assert.equal(typeof mod.default.mount, "function");
  const doc = makeDoc();
  const g = { mermaid: stub() };
  globalThis.mermaid = g.mermaid;
  try {
    const d = diagram(doc, "graph TD\n A-->B");
    await mod.default.mount(d);
    await mod.default.mount(d); // a second mount does nothing
    assert.equal(d.querySelectorAll(".zen-diagram-svg").length, 1);
    assert.equal(d.querySelectorAll("pre").length, 0);
    assert.equal(g.mermaid.calls.render.length, 1);
  } finally {
    delete globalThis.mermaid;
  }
});

test("mount loads mermaid from ctx.assetUrl", async () => {
  const mod = await import("./diagram.js");
  const doc = makeDoc();
  const d = diagram(doc, "graph TD\n A-->B");
  const p = mod.default.mount(d, { assetUrl: (f) => "../_zen/plugins/diagram/" + f });
  const s = doc.head.querySelectorAll("script")[0];
  assert.equal(s.src, "../_zen/plugins/diagram/mermaid.min.js");
  globalThis.mermaid = stub();
  try {
    s.fire("load");
    await p;
    assert.equal(d.querySelectorAll(".zen-diagram-svg").length, 1);
  } finally {
    delete globalThis.mermaid;
  }
});

test("mount on the renderer's wrapper draws its one diagram once, with the diagram's theme", async () => {
  const mod = await import("./diagram.js");
  const doc = makeDoc();
  const wrapper = doc.createElement("div");
  wrapper.dataset.zenPlugin = "diagram";
  const d = doc.createElement("div");
  d.className = "zen-diagram"; d.dataset.theme = "default";
  const pre = doc.createElement("pre"); pre.textContent = "graph TD\n A-->B"; d.append(pre);
  wrapper.append(d);
  doc.body.append(wrapper);
  const m = stub();
  globalThis.mermaid = m;
  try {
    await mod.default.mount(wrapper);
    assert.equal(m.calls.render.length, 1);
    assert.equal(m.calls.init[0].theme, "default");
    assert.equal(d.querySelectorAll(".zen-diagram-svg").length, 1);
    assert.equal(wrapper.children.length, 1); // nothing appended outside the diagram
  } finally {
    delete globalThis.mermaid;
  }
});

// ---- diagram-performance ----

const tick = () => new Promise((r) => setTimeout(r, 0));
const noYield = () => Promise.resolve();
class FakeIO {
  static all = [];
  constructor(cb, opts) { this.cb = cb; this.opts = opts; this.els = new Set(); this.gone = false; FakeIO.all.push(this); }
  observe(e) { this.els.add(e); }
  unobserve(e) { this.els.delete(e); }
  disconnect() { this.gone = true; this.els.clear(); }
  fire(e) { if (this.els.has(e)) this.cb([{ target: e, isIntersecting: true }], this); }
}
const holder = (doc) => { const e = doc.createElement("div"); doc.body.append(e); return e; };

test("nextIndex: visible first, then top to bottom", () => {
  assert.equal(nextIndex([]), -1);
  assert.equal(nextIndex([{ visible: false, top: 5 }, { visible: true, top: 900 }, { visible: true, top: 100 }]), 2);
  assert.equal(nextIndex([{ visible: false, top: 900 }, { visible: false, top: -300 }, { visible: false, top: 100 }]), 1);
  assert.equal(nextIndex([{ visible: true, top: 10 }, { visible: true, top: 10 }]), 0);
  assert.equal(NEAR, 1000);
});

test("queue: one at a time, with a yield between", async () => {
  const doc = makeDoc();
  const log = [];
  let live = 0, max = 0;
  const q = createQueue({ doc, IntersectionObserver: null, yield: async () => { log.push("yield"); } });
  const draw = (i) => async () => { live++; max = Math.max(max, live); log.push("draw" + i); await tick(); live--; };
  await Promise.all([0, 1, 2].map((i) => q.add(holder(doc), draw(i))));
  assert.equal(max, 1);
  assert.deepEqual(log.slice(0, 5), ["draw0", "yield", "draw1", "yield", "draw2"]);
  q.destroy();
});

test("queue: waits for the screen, pending until drawn, drawAll", async () => {
  const doc = makeDoc();
  const q = createQueue({ doc, IntersectionObserver: FakeIO, yield: noYield });
  const io = FakeIO.all.at(-1);
  assert.equal(io.opts.rootMargin, "1000px 0px");
  const a = holder(doc), b = holder(doc), c = holder(doc);
  const calls = [];
  const pa = q.add(a, async () => calls.push("a"));
  const pb = q.add(b, async () => calls.push("b"));
  const pc = q.add(c, async () => { calls.push("c"); throw new Error("x"); });
  await tick();
  assert.deepEqual(calls, []);
  assert.equal(a.hasAttribute("data-zen-pending"), true);
  io.fire(b);
  await pb;
  assert.deepEqual(calls, ["b"]);
  assert.equal(b.hasAttribute("data-zen-pending"), false);
  q.drawAll();
  await Promise.all([pa, pc]);
  assert.deepEqual(calls.sort(), ["a", "b", "c"]);
  assert.equal(c.hasAttribute("data-zen-pending"), false, "a failing draw leaves the queue too");
  q.destroy();
});

test("queue: no IntersectionObserver draws all, the print preview draws at once, destroy drops", async () => {
  const doc = makeDoc();
  const q = createQueue({ doc, IntersectionObserver: null, yield: noYield });
  const done = [];
  await Promise.all([0, 1].map((i) => q.add(holder(doc), async () => done.push(i))));
  assert.deepEqual(done, [0, 1]);
  q.destroy();

  const q2 = createQueue({ doc, IntersectionObserver: FakeIO, yield: noYield });
  const frame = holder(doc);
  frame.className = "zen-print-preview";
  const inside = doc.createElement("div");
  frame.append(inside);
  let drawn = 0;
  await q2.add(inside, async () => drawn++);
  assert.equal(drawn, 1);
  const waiting = q2.add(holder(doc), async () => drawn++);
  q2.destroy();
  await waiting;
  assert.equal(drawn, 1);
  assert.equal(FakeIO.all.at(-1).gone, true);
});

test("a diagram over 200 KB is not drawn", async () => {
  const doc = makeDoc();
  const warns = [];
  const orig = console.warn;
  console.warn = (...a) => warns.push(a.join(" "));
  try {
    const d = diagram(doc, "graph TD\n" + "A-->B\n".repeat(40000));
    const m = stub();
    await renderAll(doc.body, m);
    assert.equal(m.calls.render.length, 0);
    assert.equal(d.querySelector(".zen-plugin-error").textContent, "diagram: the diagram is too large to draw");
    assert.equal("drawn" in d.dataset, true);
    assert.ok(d.querySelector("pre"));
  } finally {
    console.warn = orig;
  }
});

test("mount queues the diagram: drawn one at a time, pending until drawn, released on unmount", async () => {
  const mod = await import("./diagram.js");
  const doc = makeDoc();
  const m = stub();
  globalThis.mermaid = m;
  try {
    const els = [0, 1, 2].map((i) => diagram(doc, "graph TD\nA" + i + "-->B"));
    const releases = await Promise.all(els.map((d) => mod.default.mount(d)));
    for (const d of els) {
      assert.equal(d.querySelectorAll(".zen-diagram-svg").length, 1);
      assert.equal(d.hasAttribute("data-zen-pending"), false);
    }
    assert.deepEqual(m.calls.render.map((c) => c.src.slice(0, 10)), ["graph TD\nA0", "graph TD\nA1", "graph TD\nA2"]);
    assert.equal(typeof releases[0], "function");
    for (const r of releases) r();
  } finally {
    delete globalThis.mermaid;
  }
});

test("mount waits behind the placeholder until the figure is near", async () => {
  const mod = await import("./diagram.js");
  const doc = makeDoc();
  doc.defaultView.IntersectionObserver = FakeIO;
  const m = stub();
  globalThis.mermaid = m;
  try {
    const d = diagram(doc, "graph TD\nA-->B");
    const p = mod.default.mount(d);
    await tick();
    assert.equal("drawn" in d.dataset, false);
    assert.equal(d.hasAttribute("data-zen-pending"), true);
    FakeIO.all.at(-1).fire(d);
    const release = await p;
    assert.equal("drawn" in d.dataset, true);
    release();
    assert.equal(FakeIO.all.at(-1).gone, true, "the last release takes the observer away");
  } finally {
    delete globalThis.mermaid;
  }
});

test("the queue copy is the same file as lini's", () => {
  const a = readFileSync(new URL("./queue.js", import.meta.url), "utf8");
  const b = readFileSync(new URL("../../lini/assets/queue.js", import.meta.url), "utf8");
  assert.equal(a, b);
  assert.doesNotMatch(a, /[a-z][a-z0-9+.-]*:\/\//i);
  assert.doesNotMatch(a, /innerHTML/);
});
