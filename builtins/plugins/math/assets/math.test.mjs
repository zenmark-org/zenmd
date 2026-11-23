// Tests for math.js with a stub KaTeX and a tiny fake DOM (no deps).
import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { KATEX_URL, loadKatex, renderAll, wrapTex } from "./math.js";

class El {
  constructor(tag, doc) { this.tagName = tag.toUpperCase(); this.ownerDocument = doc; this.children = []; this.dataset = {}; this.attrs = {}; this.className = ""; this.textContent = ""; this.innerHTML = ""; this.listeners = {}; this.parent = null; }
  append(...cs) { for (const c of cs) { c.parent = this; this.children.push(c); } }
  replaceWith(n) { const p = this.parent; if (!p) return; n.parent = p; p.children = p.children.map((c) => (c === this ? n : c)); this.parent = null; }
  remove() { if (this.parent) this.parent.children = this.parent.children.filter((c) => c !== this); }
  setAttribute(k, v) { this.attrs[k] = String(v); }
  getAttribute(k) { return this.attrs[k] ?? null; }
  addEventListener(t, f) { (this.listeners[t] ||= []).push(f); }
  fire(t) { for (const f of this.listeners[t] || []) f({ type: t }); }
  all() { return this.children.flatMap((c) => [c, ...c.all()]); }
  has(cls) { return this.className.split(" ").includes(cls); }
  querySelector(s) { return this.querySelectorAll(s)[0] || null; }
  querySelectorAll(s) {
    const m = {
      ".zm-tex": (e) => e.has("zm-tex"),
      ".zm-tex:not([data-rendered])": (e) => e.has("zm-tex") && !("rendered" in e.dataset),
      ".zm-eq": (e) => e.has("zm-eq"),
      ".zm-row": (e) => e.has("zm-row"),
      ".zen-plugin-error": (e) => e.has("zen-plugin-error"),
      ".katex-error": (e) => e.has("katex-error"),
    }[s];
    if (s.startsWith("script") || s.startsWith("link")) return this.all().filter((e) => e.tagName === s.split("[")[0].toUpperCase());
    assert.ok(m, "unsupported selector " + s);
    return this.all().filter(m);
  }
}
function makeDoc() {
  const doc = { listeners: {} };
  doc.createElement = (t) => new El(t, doc);
  doc.head = new El("head", doc);
  doc.body = new El("body", doc);
  doc.documentElement = new El("html", doc);
  return doc;
}
/** The page's markup: a row with the pre of the TeX (and a number). */
function block(doc, tex, cls = "zen-block zm") {
  const w = doc.createElement("div"); w.className = cls; w.dataset.zenPlugin = "math";
  const row = doc.createElement("div"); row.className = "zm-row";
  const pre = doc.createElement("pre"); pre.className = "zm-tex"; pre.textContent = tex;
  row.append(pre); w.append(row); doc.body.append(w);
  return { w, row, pre };
}
function stub(failOn) {
  const calls = [];
  return {
    calls,
    render(tex, el, opts) {
      calls.push({ tex, opts });
      if (failOn && tex.includes(failOn)) {
        const e = el.ownerDocument.createElement("span");
        e.className = "katex-error"; e.setAttribute("title", "ParseError: Undefined control sequence: \\bad at position 1");
        el.append(e);
        return;
      }
      const k = el.ownerDocument.createElement("span"); k.className = "katex-display"; k.textContent = "rendered:" + tex;
      el.append(k);
    },
  };
}

test("renders each pending equation in display mode with safe options and marks it", async () => {
  const doc = makeDoc();
  const a = block(doc, "E = mc^2");
  const b = block(doc, "x^2");
  const k = stub();
  await renderAll(doc.body, k);
  for (const [i, x] of [a, b].entries()) {
    assert.equal("rendered" in x.pre.dataset, true);
    const eq = x.row.querySelector(".zm-eq");
    assert.ok(eq, "the rendered equation is in .zm-eq");
    assert.equal(eq.querySelector(".katex-error"), null);
    assert.equal(x.row.querySelector(".zm-tex"), null, "the plain TeX is replaced");
    assert.equal(eq.children[0].textContent, "rendered:" + ["E = mc^2", "x^2"][i]);
  }
  const o = k.calls[0].opts;
  assert.equal(o.displayMode, true);
  assert.equal(o.throwOnError, false);
  assert.equal(o.trust, false);
  assert.equal(o.strict, "warn");
  assert.ok(o.maxExpand > 0 && o.maxExpand <= 10000, "macro expansion is bounded");
  await renderAll(doc.body, k);
  assert.equal(k.calls.length, 2, "rendered equations are skipped");
});

test("an error keeps the TeX and shows KaTeX's message under it", async () => {
  const doc = makeDoc();
  const x = block(doc, "\\bad{x}");
  await renderAll(doc.body, stub("\\bad"));
  assert.equal(x.row.querySelector(".zm-tex").textContent, "\\bad{x}", "the TeX stays as text");
  assert.equal(x.row.querySelector(".zm-eq"), null);
  const err = x.w.querySelector(".zen-plugin-error");
  assert.equal(err.textContent, "math: ParseError: Undefined control sequence: \\bad at position 1");
  assert.equal(err.innerHTML, "", "message set as text, not HTML");
  assert.equal("rendered" in x.pre.dataset, true);
});

test("a failing equation warns in the page's console, first line, 200 characters", async () => {
  const warns = [];
  const orig = console.warn;
  console.warn = (...a) => warns.push(a.join(" "));
  try {
    const doc = makeDoc();
    block(doc, "\\bad");
    const k = stub();
    k.render = (tex, el) => { throw new Error("y".repeat(300) + "\nSECOND"); };
    await renderAll(doc.body, k);
    assert.equal(warns.length, 1);
    assert.ok(warns[0].length <= 200);
    assert.match(warns[0], /^math: y+/);
  } finally {
    console.warn = orig;
  }
});

test("wrapTex puts a multi-line equation in aligned, unless it has its own environment or asks not to", () => {
  assert.equal(wrapTex("a &= b \\\\ c &= d", false), "\\begin{aligned}a &= b \\\\ c &= d\\end{aligned}");
  assert.equal(wrapTex("a = b", false), "a = b");
  assert.equal(wrapTex("\\begin{matrix} a \\\\ b \\end{matrix}", false), "\\begin{matrix} a \\\\ b \\end{matrix}");
  assert.equal(wrapTex("a \\\\ b", true), "a \\\\ b");
});

test("a block with the zm-nowrap class is not wrapped", async () => {
  const doc = makeDoc();
  block(doc, "a \\\\ b", "zen-block zm zm-nowrap");
  block(doc, "c \\\\ d");
  const k = stub();
  await renderAll(doc.body, k);
  assert.deepEqual(k.calls.map((c) => c.tex), ["a \\\\ b", "\\begin{aligned}c \\\\ d\\end{aligned}"]);
});

test("KaTeX is lazy-loaded once from /_plugins/math/ as a classic script", async () => {
  const doc = makeDoc();
  const g = {};
  const p1 = loadKatex(doc, g);
  const p2 = loadKatex(doc, g);
  const scripts = doc.head.querySelectorAll("script");
  assert.equal(scripts.length, 1);
  assert.equal(scripts[0].src, "/_plugins/math/katex.min.js");
  assert.equal(KATEX_URL, "/_plugins/math/katex.min.js");
  assert.notEqual(scripts[0].type, "module");
  const k = stub();
  g.katex = k;
  scripts[0].fire("load");
  assert.equal(await p1, k);
  assert.equal(await p2, k);
});

test("a failed load rejects and can be retried", async () => {
  const doc = makeDoc();
  const g = {};
  const p = loadKatex(doc, g);
  doc.head.querySelectorAll("script")[0].fire("error");
  await assert.rejects(p);
  loadKatex(doc, g);
  assert.equal(doc.head.querySelectorAll("script").length, 1, "old tag removed before retry");
});

test("no remote URLs, eval or innerHTML in math.js", () => {
  const src = readFileSync(new URL("./math.js", import.meta.url), "utf8");
  assert.doesNotMatch(src, /[a-z][a-z0-9+.-]*:\/\//i);
  assert.doesNotMatch(src, /\beval\s*\(|new Function|innerHTML/);
  assert.match(src, /trust:\s*false/);
  assert.match(src, /throwOnError:\s*false/);
  assert.match(src, /strict:\s*"warn"/);
});

test("mount draws the block once, loading KaTeX from ctx.assetUrl", async () => {
  const mod = await import("./math.js");
  assert.equal(typeof mod.default.mount, "function");
  const doc = makeDoc();
  const x = block(doc, "a^2");
  const p = mod.default.mount(x.w, { assetUrl: (f) => "../_zen/plugins/math/" + f });
  const s = doc.head.querySelectorAll("script")[0];
  assert.equal(s.src, "../_zen/plugins/math/katex.min.js");
  const k = stub();
  globalThis.katex = k;
  try {
    s.fire("load");
    await p;
    await mod.default.mount(x.w, { assetUrl: (f) => f }); // a second mount does nothing
    assert.equal(x.w.querySelectorAll(".zm-eq").length, 1);
    assert.equal(k.calls.length, 1);
  } finally {
    delete globalThis.katex;
  }
});

test("mount shows a load failure under the TeX", async () => {
  const mod = await import("./math.js");
  const doc = makeDoc();
  const x = block(doc, "a^2");
  const p = mod.default.mount(x.w, { assetUrl: (f) => f });
  doc.head.querySelectorAll("script")[0].fire("error");
  await p;
  assert.equal(x.w.querySelector(".zm-tex").textContent, "a^2");
  assert.match(x.w.querySelector(".zen-plugin-error").textContent, /^math: could not load KaTeX/);
});
