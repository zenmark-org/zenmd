// @vitest-environment happy-dom
// Tests for lini.js: the vendored compiler is real (loaded from the bytes, as
// Node has no fetch for file URLs), the DOM is happy-dom's.
import { beforeEach, describe, expect, test, vi } from "vitest";
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
// @ts-ignore plain JS module
import * as wasm from "./lini_wasm.js";
// @ts-ignore plain JS module
import lini, { LINI_MODULE, LINI_TIMEOUT, cleanSvg, linkScheme, loadLini, renderOne, setImporter, startLini, watchTheme } from "./lini.js";
// @ts-ignore plain JS module
import { NEAR, createQueue, nextIndex } from "./queue.js";
// @ts-ignore plain JS module
import { serve } from "./lini.worker.js";
import { whenReady } from "../../../../runtime/src/print/ready";

const here = dirname(fileURLToPath(import.meta.url));
const bytes = readFileSync(join(here, "lini_wasm_bg.wasm"));

let n = 0;
const url = () => `/_plugins/lini/test-${++n}.js`; // a fresh url per test: the loader caches by url

/** A module shaped like lini_wasm.js whose init takes the bytes, counting its calls. */
function fakeModule() {
  const calls: any[] = [];
  const mod = {
    default: async (o: any) => { calls.push(o); wasm.initSync({ module: bytes }); },
    compile: (src: string, a?: any) => wasm.compile(src, a),
  };
  return { mod, calls };
}
function install() {
  const f = fakeModule();
  const imports: string[] = [];
  setImporter(async (u: string) => { imports.push(u); return f.mod; });
  return { ...f, imports };
}

function block(src: string): HTMLElement {
  const el = document.createElement("div");
  el.className = "zen-lini";
  const pre = document.createElement("pre");
  pre.textContent = src;
  el.append(pre);
  document.body.append(el);
  return el;
}
const ctxFor = (u: string) => ({ assetUrl: vi.fn((f: string) => (f.endsWith(".wasm") ? "/_plugins/lini/" + f : u)) });

const SRC = '|box#api| "API"\n|box#db| "DB"\napi -> db "jobs"';

/** happy-dom's IntersectionObserver never fires: the plugin tests run without one unless a test sets its own. */
function setIO(v: any) {
  (document.defaultView as any).IntersectionObserver = v;
  (globalThis as any).IntersectionObserver = v;
}

beforeEach(() => {
  setIO(undefined);
  vi.unstubAllGlobals();
  document.body.innerHTML = "";
  delete document.documentElement.dataset.theme;
  vi.restoreAllMocks();
});

describe("loading", () => {
  test("loads once for many blocks", async () => {
    const { calls, imports } = install();
    const u = url();
    const ctx = ctxFor(u);
    const els = [block(SRC), block(SRC), block('|box| "x"')];
    await Promise.all(els.map((e) => lini.mount(e, ctx)));
    expect(imports).toEqual([u]);
    expect(calls).toHaveLength(1);
    expect(calls[0]).toEqual({ module_or_path: "/_plugins/lini/lini_wasm_bg.wasm" });
    for (const e of els) expect(e.querySelector(".zen-lini-svg svg")).not.toBeNull();
  });

  test("no block, no load", async () => {
    const { imports } = install();
    await watchTheme(document);
    expect(imports).toEqual([]);
  });

  test("uses ctx.assetUrl", async () => {
    install();
    const u = url();
    const ctx = ctxFor(u);
    await lini.mount(block(SRC), ctx);
    expect(ctx.assetUrl).toHaveBeenCalledWith("lini_wasm.js");
    expect(ctx.assetUrl).toHaveBeenCalledWith("lini_wasm_bg.wasm");
    const src = readFileSync(join(here, "lini.js"), "utf8");
    expect(src).not.toMatch(/https?:\/\//); // no network url in the loader
    expect(LINI_MODULE).toBe("/_plugins/lini/lini_wasm.js");
  });

  test("a compiler that cannot load", async () => {
    setImporter(async () => { throw new Error("blocked"); });
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const els = [block(SRC), block("|box| \"b\"")];
    const ctx = ctxFor(url());
    await Promise.all(els.map((e) => lini.mount(e, ctx)));
    for (const e of els) {
      expect(e.querySelector("pre")).not.toBeNull();
      const boxes = e.querySelectorAll(".zen-plugin-error");
      expect(boxes).toHaveLength(1);
      expect(boxes[0].textContent).toBe("lini: could not load the Lini compiler");
      expect(e.querySelector("svg")).toBeNull();
    }
    expect(warn).toHaveBeenCalled();
  });

  test("a wasm that fails to start is a load failure", async () => {
    setImporter(async () => ({ default: async () => { throw new Error("CompileError"); }, compile: () => "" }));
    const e = block(SRC);
    await lini.mount(e, ctxFor(url()));
    expect(e.querySelector(".zen-plugin-error")!.textContent).toBe("lini: could not load the Lini compiler");
  });

  test("loadLini hides the call behind { compile }", async () => {
    const { calls } = install();
    const l = await loadLini("/x/a.js", "/x/a.wasm");
    expect(calls[0]).toEqual({ module_or_path: "/x/a.wasm" });
    expect(l.compile(SRC)).toMatch(/^<svg/);
  });
});

describe("drawing", () => {
  test("data-drawn is set once the figure, an error or a failed load is shown (the placeholder in page.css goes)", async () => {
    install();
    const ok = block(SRC);
    expect("drawn" in ok.dataset).toBe(false);
    await lini.mount(ok, ctxFor(url()));
    expect("drawn" in ok.dataset).toBe(true);
    const bad = block("a => b");
    vi.spyOn(console, "warn").mockImplementation(() => {});
    await lini.mount(bad, ctxFor(url()));
    expect("drawn" in bad.dataset).toBe(true);
    const empty = block("   ");
    await lini.mount(empty, ctxFor(url()));
    expect("drawn" in empty.dataset).toBe(true);
    setImporter(async () => { throw new Error("blocked"); });
    const failed = block(SRC);
    await lini.mount(failed, ctxFor(url()));
    expect("drawn" in failed.dataset).toBe(true);
  });

  test("renders the svg in .zen-lini-svg", async () => {
    install();
    const e = block(SRC);
    await lini.mount(e, ctxFor(url()));
    const box = e.querySelector(".zen-lini-svg")!;
    expect(box).not.toBeNull();
    expect(box.querySelector("svg")).not.toBeNull();
    expect(box.querySelector("svg")!.textContent).toContain("API");
    expect(e.querySelector(":scope > pre")).toBeNull(); // the source moved into the details
    expect(box.getAttribute("role")).toBe("img");
    expect(box.getAttribute("aria-label")).toBe("Lini figure");
    const d = e.querySelector("details.zen-lini-source")!;
    expect(d.querySelector("summary")!.textContent).toBe("Source");
    expect(d.querySelector("pre")!.textContent).toBe(SRC);
    expect(e.querySelector(".zen-plugin-error")).toBeNull();
  });

  test("a title becomes the label", async () => {
    setImporter(async () => ({ default: async () => {}, compile: () => `<svg xmlns="http://www.w3.org/2000/svg"><title>Job flow</title><rect width="1" height="1"/></svg>` }));
    const e = block(SRC);
    await lini.mount(e, ctxFor(url()));
    expect(e.querySelector(".zen-lini-svg")!.getAttribute("aria-label")).toBe("Job flow");
  });

  test("rendering twice draws once", async () => {
    install();
    const e = block(SRC);
    const l = await loadLini(url(), "/w.wasm");
    await renderOne(e, l);
    await renderOne(e, l);
    expect(e.querySelectorAll(".zen-lini-svg")).toHaveLength(1);
  });

  test("a wrong figure keeps its source", async () => {
    install();
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const e = block("garbage ((\nsecond line");
    await lini.mount(e, ctxFor(url()));
    expect(e.querySelector("pre")!.textContent).toBe("garbage ((\nsecond line");
    const err = e.querySelector(".zen-plugin-error")!;
    expect(err.textContent!.startsWith("lini: ")).toBe(true);
    expect(err.textContent).not.toContain("\n");
    expect(e.querySelector("svg")).toBeNull();
    expect(warn).toHaveBeenCalledTimes(1);
    expect(String(warn.mock.calls[0][0]).startsWith("lini: ")).toBe(true);
  });

  test("an empty block is left alone", async () => {
    install();
    const e = block("   \n");
    await lini.mount(e, ctxFor(url()));
    expect(e.querySelector("svg")).toBeNull();
    expect(e.querySelector(".zen-plugin-error")).toBeNull();
  });

  test("a figure over 200 KB is not compiled", async () => {
    const f = fakeModule();
    const compile = vi.fn(f.mod.compile);
    setImporter(async () => ({ ...f.mod, compile }));
    vi.spyOn(console, "warn").mockImplementation(() => {});
    const e = block("|box| \"x\"\n" + "# ".padEnd(201 * 1024, "x"));
    await lini.mount(e, ctxFor(url()));
    expect(compile).not.toHaveBeenCalled();
    expect(e.querySelector(".zen-plugin-error")!.textContent).toBe("lini: figure too large");
    expect(e.querySelector("pre")).not.toBeNull();
  });

  test("a label with <script>, quotes and ampersands is text", async () => {
    install();
    const e = block('|box#a| "<script>alert(1)</script> & \\"q\\""');
    await lini.mount(e, ctxFor(url()));
    const svg = e.querySelector(".zen-lini-svg svg")!;
    expect(svg).not.toBeNull();
    expect(svg.querySelector("script")).toBeNull();
    expect(svg.textContent).toContain("<script>alert(1)</script>");
    expect(svg.textContent).toContain("&");
  });

  test("a 2,000-line figure draws", async () => {
    install();
    let s = "";
    for (let i = 0; i < 2000; i++) s += `|box#n${i}| "${i}"\n`;
    const e = block(s);
    await lini.mount(e, ctxFor(url()));
    expect(e.querySelector(".zen-lini-svg svg")).not.toBeNull();
  }, 60000);

  test("two blocks with the same #id do not share ids in the page", async () => {
    install();
    const a = block('|box#x| "A"\n|box#y| "B"\nx -> y "one"');
    const b = block('|box#x| "C"\n|box#y| "D"\nx -> y "two"');
    const ctx = ctxFor(url());
    await Promise.all([lini.mount(a, ctx), lini.mount(b, ctx)]);
    const ids = (el: Element) => [...el.querySelectorAll("[id]")].map((x) => x.id);
    const ia = ids(a), ib = ids(b);
    expect(ia.length).toBeGreaterThan(0);
    expect(ia.filter((i) => ib.includes(i))).toEqual([]);
    expect(new Set([...ia, ...ib]).size).toBe(ia.length + ib.length);
  });

  test("a block in a hidden container is drawn", async () => {
    install();
    const e = block(SRC);
    const tab = document.createElement("div");
    tab.hidden = true;
    tab.style.display = "none";
    tab.append(e);
    document.body.append(tab);
    await lini.mount(e, ctxFor(url()));
    expect(e.querySelector(".zen-lini-svg svg")).not.toBeNull();
  });

  test("mount accepts the wrapper around .zen-lini", async () => {
    install();
    const w = document.createElement("div");
    w.dataset.zenPlugin = "lini";
    const e = block(SRC);
    w.append(e);
    document.body.append(w);
    await lini.mount(w, ctxFor(url()));
    expect(e.querySelector(".zen-lini-svg svg")).not.toBeNull();
  });
});

describe("theme", () => {
  test("color-scheme follows data-theme and the system", async () => {
    install();
    const e = block(SRC);
    await lini.mount(e, ctxFor(url()));
    const box = e.querySelector<HTMLElement>(".zen-lini-svg")!;
    const mm = (dark: boolean) => vi.spyOn(window, "matchMedia").mockImplementation(((q: string) => ({ matches: dark && q.includes("dark"), media: q, addEventListener() {}, removeEventListener() {} })) as any);
    document.documentElement.dataset.theme = "dark";
    linkScheme(e, document);
    expect(box.style.colorScheme).toBe("dark");
    document.documentElement.dataset.theme = "light";
    linkScheme(e, document);
    expect(box.style.colorScheme).toBe("light");
    delete document.documentElement.dataset.theme;
    mm(true);
    linkScheme(e, document);
    expect(box.style.colorScheme).toBe("dark");
    mm(false);
    linkScheme(e, document);
    expect(box.style.colorScheme).toBe("light");
  });

  test("dark mode switched while the page is open", async () => {
    install();
    const e = block(SRC);
    await lini.mount(e, ctxFor(url()));
    const box = e.querySelector<HTMLElement>(".zen-lini-svg")!;
    document.documentElement.dataset.theme = "light";
    document.documentElement.dataset.theme = "dark";
    await new Promise((r) => setTimeout(r, 20));
    expect(box.style.colorScheme).toBe("dark");
    document.documentElement.dataset.theme = "light";
    await new Promise((r) => setTimeout(r, 20));
    expect(box.style.colorScheme).toBe("light");
  });

  test("the page's :root is not styled by a figure", async () => {
    install();
    const e = block(SRC);
    await lini.mount(e, ctxFor(url()));
    const css = [...e.querySelectorAll("style")].map((s) => s.textContent).join("\n");
    expect(css.length).toBeGreaterThan(0);
    expect(css).not.toMatch(/:root/);
  });

  test("lini.css: overflow-x auto, tokens only", () => {
    const css = readFileSync(join(here, "lini.css"), "utf8");
    expect(css).toMatch(/\.zen-lini-svg\s*\{[^}]*overflow-x:\s*auto/);
    expect(css).not.toMatch(/#[0-9a-fA-F]{3,8}\b/);
    expect(css).not.toMatch(/\b(rgb|hsl)a?\(/);
    expect(css).not.toMatch(/@keyframes|transition|animation/);
  });
});

describe("cleanSvg", () => {
  const wrap = (inner: string, attrs = "") => `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" ${attrs}>${inner}</svg>`;

  test("cleanSvg drops script, foreignObject, on* and outside hrefs", () => {
    const svg = cleanSvg(
      wrap(
        `<script>alert(1)</script><foreignObject><div>x</div></foreignObject><iframe/><object/><embed/>` +
          `<rect id="r" onclick="x()" onload="y()" width="1" height="1"/>` +
          `<a href="https://evil.test/"><text>l</text></a><a xlink:href="javascript:alert(1)"><text>m</text></a>` +
          `<use href="https://evil.test/a.svg#x"/><use xlink:href="data:image/svg+xml,x"/>` +
          `<rect width="2" height="2"><animate attributeName="href" to="javascript:alert(1)"/><set attributeName="xlink:href" to="x"/><animate attributeName="width" to="3"/></rect>`,
        'onload="z()"',
      ),
    );
    const html = svg.outerHTML;
    expect(html).not.toMatch(/script|foreignObject|iframe|<object|<embed/i);
    expect(html).not.toMatch(/\son[a-z]+=/i);
    expect(html).not.toMatch(/evil\.test|javascript:|data:image/);
    expect(svg.querySelector("use")).toBeNull();
    expect(svg.querySelector("set")).toBeNull();
    const anims = [...svg.querySelectorAll("animate")];
    expect(anims.map((a) => a.getAttribute("attributeName"))).toEqual(["width"]);
    expect(svg.querySelector("#r")).not.toBeNull();
    expect(svg.textContent).toContain("l");
  });

  test("keeps #refs", () => {
    const svg = cleanSvg(wrap(`<defs><rect id="a" width="1" height="1"/></defs><use href="#a"/><use xlink:href="#a"/><a href="#top"><text>t</text></a>`));
    expect(svg.querySelectorAll("use")).toHaveLength(2);
    expect(svg.querySelector("use")!.getAttribute("href")).toBe("#a");
    expect(svg.querySelector("a")!.getAttribute("href")).toBe("#top");
  });

  test("keeps the style element", () => {
    const svg = cleanSvg(wrap(`<style>.a { fill: red; }</style><rect class="a"/>`));
    expect(svg.querySelector("style")!.textContent).toContain("fill");
  });

  test("a parse error is a compile error", () => {
    expect(() => cleanSvg("<svg><g></svg>")).toThrow();
    expect(() => cleanSvg("not svg at all")).toThrow();
    expect(() => cleanSvg('<html xmlns="http://www.w3.org/1999/xhtml"><body/></html>')).toThrow();
  });

  test("a non-svg result shows the error box", async () => {
    setImporter(async () => ({ default: async () => {}, compile: () => "<svg><g></svg>" }));
    vi.spyOn(console, "warn").mockImplementation(() => {});
    const e = block(SRC);
    await lini.mount(e, ctxFor(url()));
    expect(e.querySelector(".zen-plugin-error")).not.toBeNull();
    expect(e.querySelector("pre")).not.toBeNull();
    expect(e.querySelector("svg")).toBeNull();
  });

  test("a hostile compiler result is cleaned in the page", async () => {
    setImporter(async () => ({ default: async () => {}, compile: () => `<svg xmlns="http://www.w3.org/2000/svg"><script>window.pwned=1</script><rect onclick="window.pwned=2" width="1" height="1"/></svg>` }));
    const e = block(SRC);
    await lini.mount(e, ctxFor(url()));
    expect(e.querySelector("script")).toBeNull();
    expect(e.querySelector("rect")!.hasAttribute("onclick")).toBe(false);
    expect((window as any).pwned).toBeUndefined();
  });

  test("no innerHTML", () => {
    const src = readFileSync(join(here, "lini.js"), "utf8");
    expect(src).not.toMatch(/innerHTML|outerHTML|insertAdjacentHTML|document\.write/);
    expect(src).toMatch(/importNode/);
  });
});

describe("help snippets", () => {
  test("the help snippets compile", () => {
    wasm.initSync({ module: bytes });
    const go = readFileSync(join(here, "..", "lini.go"), "utf8");
    const found = [...go.matchAll(/~~~lini\n([\s\S]*?)~~~/g)].map((m) => m[1]);
    expect(found).toHaveLength(3);
    for (const s of found) {
      const svg = wasm.compile(s);
      expect(svg).toMatch(/^<svg/);
      expect(() => cleanSvg(svg)).not.toThrow();
    }
  });
});


// ---- diagram-performance: the queue, the worker, the page ----

class FakeIO {
  static all: FakeIO[] = [];
  els = new Set<Element>();
  gone = false;
  constructor(public cb: any, public opts: any) { FakeIO.all.push(this); }
  observe(e: Element) { this.els.add(e); }
  unobserve(e: Element) { this.els.delete(e); }
  disconnect() { this.gone = true; this.els.clear(); }
  fire(e: Element, isIntersecting = true) { if (this.els.has(e)) this.cb([{ target: e, isIntersecting }], this); }
}
const lastIO = () => FakeIO.all[FakeIO.all.length - 1];
const tick = () => new Promise<void>((r) => setTimeout(r, 0));
const noYield = () => Promise.resolve();
const holder = (name = "x") => { const e = document.createElement("div"); e.id = name; document.body.append(e); return e; };
const rect = (e: Element, top: number, h = 100) => { (e as any).getBoundingClientRect = () => ({ top, bottom: top + h, height: h, left: 0, right: 10, width: 10 }); };

describe("nextIndex", () => {
  test("visible first, then top to bottom", () => {
    expect(nextIndex([])).toBe(-1);
    expect(nextIndex([{ visible: false, top: 5 }])).toBe(0);
    expect(nextIndex([{ visible: false, top: 5 }, { visible: true, top: 900 }, { visible: true, top: 100 }])).toBe(2);
    expect(nextIndex([{ visible: false, top: 900 }, { visible: false, top: -300 }, { visible: false, top: 100 }])).toBe(1);
    expect(nextIndex([{ visible: true, top: 10 }, { visible: true, top: 10 }])).toBe(0);
    expect(NEAR).toBe(1000);
  });
});

describe("queue", () => {
  test("queue: one at a time", async () => {
    const q = createQueue({ doc: document, IntersectionObserver: null, yield: noYield });
    let live = 0, max = 0;
    const order: number[] = [];
    const draw = (i: number) => async () => { live++; max = Math.max(max, live); await tick(); order.push(i); live--; };
    const els = [0, 1, 2, 3].map((i) => holder("q" + i));
    await Promise.all(els.map((e, i) => q.add(e, draw(i))));
    expect(max).toBe(1);
    expect(order).toEqual([0, 1, 2, 3]);
    expect(q.size).toBe(0);
    q.destroy();
  });

  test("queue: yields between figures", async () => {
    const log: string[] = [];
    const q = createQueue({ doc: document, IntersectionObserver: null, yield: async () => { log.push("yield"); } });
    await Promise.all([0, 1, 2].map((i) => q.add(holder("y" + i), async () => { log.push("draw" + i); })));
    expect(log.slice(0, 5)).toEqual(["draw0", "yield", "draw1", "yield", "draw2"]);
    q.destroy();
  });

  test("the default yield gives the window a turn", async () => {
    const q = createQueue({ doc: document, IntersectionObserver: null });
    const log: string[] = [];
    const p = Promise.all([q.add(holder("d0"), async () => { log.push("draw0"); }), q.add(holder("d1"), async () => { log.push("draw1"); })]);
    setTimeout(() => log.push("timer"), 0);
    await p;
    await tick();
    expect(log).toEqual(["draw0", "timer", "draw1"]);
    q.destroy();
  });

  test("queue: waits for the screen", async () => {
    const q = createQueue({ doc: document, IntersectionObserver: FakeIO, yield: noYield });
    const io = lastIO();
    expect(io.opts.rootMargin).toBe(`${NEAR}px 0px`);
    const a = holder("a"), b = holder("b");
    const da = vi.fn(async () => {}), db = vi.fn(async () => {});
    const pa = q.add(a, da), pb = q.add(b, db);
    await tick();
    expect(da).not.toHaveBeenCalled();
    expect(db).not.toHaveBeenCalled();
    expect(q.size).toBe(2);
    io.fire(b, false); // leaving does not queue
    await tick();
    expect(db).not.toHaveBeenCalled();
    io.fire(b);
    await pb;
    expect(db).toHaveBeenCalledTimes(1);
    expect(da).not.toHaveBeenCalled();
    expect(io.els.has(b)).toBe(false);
    expect(q.size).toBe(1);
    q.destroy();
    await pa; // dropped, not drawn
    expect(da).not.toHaveBeenCalled();
  });

  test("queue: no IntersectionObserver", async () => {
    const q = createQueue({ doc: document, IntersectionObserver: null, yield: noYield });
    const draws = [0, 1, 2].map(() => vi.fn(async () => {}));
    await Promise.all(draws.map((d, i) => q.add(holder("n" + i), d)));
    for (const d of draws) expect(d).toHaveBeenCalledTimes(1);
    q.destroy();
  });

  test("pending until drawn", async () => {
    const q = createQueue({ doc: document, IntersectionObserver: FakeIO, yield: noYield });
    const io = lastIO();
    const a = holder("pa"), b = holder("pb"), bad = holder("pc");
    expect(a.hasAttribute("data-zen-pending")).toBe(false);
    let release!: () => void;
    const gate = new Promise<void>((r) => (release = r));
    const pa = q.add(a, () => gate);
    const pb = q.add(b, async () => {});
    const pc = q.add(bad, async () => { throw new Error("boom"); });
    expect(a.hasAttribute("data-zen-pending")).toBe(true);
    expect(b.hasAttribute("data-zen-pending")).toBe(true);
    io.fire(a);
    await tick();
    expect(a.hasAttribute("data-zen-pending")).toBe(true); // drawing
    release();
    await pa;
    expect(a.hasAttribute("data-zen-pending")).toBe(false);
    expect(b.hasAttribute("data-zen-pending")).toBe(true);
    io.fire(bad);
    await pc; // a figure that throws still leaves the queue
    expect(bad.hasAttribute("data-zen-pending")).toBe(false);
    q.destroy();
    await pb;
    expect(b.hasAttribute("data-zen-pending")).toBe(false);
  });

  test("print draws everything: the print preview, and beforeprint", async () => {
    const q = createQueue({ doc: document, IntersectionObserver: FakeIO, yield: noYield });
    const frame = document.createElement("div");
    frame.className = "zen-print-preview";
    document.body.append(frame);
    const inPreview = document.createElement("div");
    frame.append(inPreview);
    const d1 = vi.fn(async () => {});
    await q.add(inPreview, d1); // no scrolling needed in the preview
    expect(d1).toHaveBeenCalledTimes(1);

    const a = holder("pr1"), b = holder("pr2");
    const root = document.createElement("div");
    root.append(a, b);
    document.body.append(root);
    const da = vi.fn(async () => {}), db = vi.fn(async () => {});
    const pa = q.add(a, da), pb = q.add(b, db);
    const ready = whenReady(root, { timeout: 3000, quiet: 20 });
    let done = false;
    ready.then(() => (done = true));
    await new Promise((r) => setTimeout(r, 120));
    expect(done).toBe(false); // waiting figures hold the print back
    document.defaultView!.dispatchEvent(new Event("beforeprint"));
    await Promise.all([pa, pb]);
    expect(da).toHaveBeenCalledTimes(1);
    expect(db).toHaveBeenCalledTimes(1);
    const r = await ready;
    expect(r.late).toEqual([]);
    q.destroy();
  });

  test("drawAll queues every waiting figure", async () => {
    const q = createQueue({ doc: document, IntersectionObserver: FakeIO, yield: noYield });
    const ds = [0, 1, 2].map(() => vi.fn(async () => {}));
    const ps = ds.map((d, i) => q.add(holder("da" + i), d));
    q.drawAll();
    await Promise.all(ps);
    for (const d of ds) expect(d).toHaveBeenCalledTimes(1);
    q.destroy();
  });

  test("a figure scrolled into view jumps ahead; the rest go top to bottom", async () => {
    Object.defineProperty(window, "innerHeight", { value: 800, configurable: true });
    const q = createQueue({ doc: document, IntersectionObserver: FakeIO, yield: noYield });
    const io = lastIO();
    const first = holder("o0"), far = holder("o1"), nearer = holder("o2"), onScreen = holder("o3");
    rect(first, 0); rect(far, 3000); rect(nearer, 1500); rect(onScreen, 300);
    const order: string[] = [];
    let release!: () => void;
    const gate = new Promise<void>((r) => (release = r));
    const ps = [
      q.add(first, async () => { await gate; order.push("first"); }),
      q.add(far, async () => { order.push("far"); }),
      q.add(nearer, async () => { order.push("nearer"); }),
      q.add(onScreen, async () => { order.push("onScreen"); }),
    ];
    io.fire(first);
    await tick();
    for (const e of [far, nearer, onScreen]) io.fire(e); // all near the screen, one is on it
    release();
    await Promise.all(ps);
    expect(order).toEqual(["first", "onScreen", "nearer", "far"]);
    q.destroy();
  });

  test("scrolling fast past twenty figures draws each once", async () => {
    const q = createQueue({ doc: document, IntersectionObserver: FakeIO, yield: noYield });
    const io = lastIO();
    const draws = Array.from({ length: 20 }, () => vi.fn(async () => {}));
    const els = draws.map((_, i) => holder("f" + i));
    const ps = els.map((e, i) => q.add(e, draws[i]));
    for (const e of els) { rect(e, els.indexOf(e) * 200); io.fire(e); io.fire(e); }
    await Promise.all(ps);
    for (const d of draws) expect(d).toHaveBeenCalledTimes(1);
    expect(q.size).toBe(0);
    q.destroy();
  });

  test("a figure that fails does not stop the others; adding twice draws once", async () => {
    const q = createQueue({ doc: document, IntersectionObserver: null, yield: noYield });
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const order: string[] = [];
    const a = holder("e0"), b = holder("e1"), c = holder("e2");
    const d = async (n: string) => { order.push(n); if (n === "b") throw new Error("boom"); };
    await Promise.all([q.add(a, () => d("a")), q.add(b, () => d("b")), q.add(c, () => d("c")), q.add(c, () => d("c2"))]);
    expect(order).toEqual(["a", "b", "c"]);
    expect(warn).not.toHaveBeenCalled(); // the draw reports its own errors
    q.destroy();
  });

  test("unmount mid-queue: destroy drops waiting figures and disconnects", async () => {
    const q = createQueue({ doc: document, IntersectionObserver: FakeIO, yield: noYield });
    const io = lastIO();
    const a = holder("u0"), b = holder("u1");
    const da = vi.fn(async () => {});
    const pa = q.add(a, da), pb = q.add(b, da);
    q.destroy();
    await Promise.all([pa, pb]);
    expect(io.gone).toBe(true);
    expect(da).not.toHaveBeenCalled();
    expect(a.hasAttribute("data-zen-pending")).toBe(false);
    io.fire(a);
    await tick();
    expect(da).not.toHaveBeenCalled();
    document.defaultView!.dispatchEvent(new Event("beforeprint"));
    await tick();
    expect(da).not.toHaveBeenCalled();
  });

  test("a detached figure is dropped without drawing", async () => {
    const q = createQueue({ doc: document, IntersectionObserver: null, yield: noYield });
    let release!: () => void;
    const gate = new Promise<void>((r) => (release = r));
    const a = holder("g0"), gone = holder("g1");
    const dg = vi.fn(async () => {});
    const pa = q.add(a, () => gate);
    const pg = q.add(gone, dg);
    gone.remove();
    release();
    await Promise.all([pa, pg]);
    expect(dg).not.toHaveBeenCalled();
    q.destroy();
  });

  test("the queue copies of both plugins are the same source", () => {
    const a = readFileSync(join(here, "queue.js"), "utf8");
    const b = readFileSync(join(here, "..", "..", "diagram", "assets", "queue.js"), "utf8");
    expect(a).toBe(b);
  });
});

/** A Worker class that runs the real lini.worker.js logic, with the real compiler, in this process. */
function fakeWorkers(o: { answer?: boolean; badInit?: boolean; throwNew?: boolean; delay?: number } = {}) {
  const made: any[] = [];
  const answer = o.answer !== false;
  class FW {
    onmessage: any = null;
    onerror: any = null;
    posted: any[] = [];
    terminated = false;
    scope: any;
    constructor(public url: string, public options: any) {
      if (o.throwNew) throw new Error("refused");
      made.push(this);
      this.scope = {
        location: { href: "http://localhost/_plugins/lini/lini.worker.js", origin: "http://localhost" },
        postMessage: (m: any) => { if (!this.terminated) Promise.resolve().then(() => this.onmessage && this.onmessage({ data: m })); },
        onmessage: null,
      };
      const mod = o.badInit
        ? { default: async () => { throw new Error("CompileError: wasm"); }, compile: () => "" }
        : fakeModule().mod;
      serve(this.scope, async () => mod);
    }
    postMessage(m: any) {
      this.posted.push(m);
      if (this.terminated) return;
      if (!answer && m.type === "compile") return;
      const run = () => Promise.resolve().then(() => this.scope.onmessage({ data: m }));
      if (o.delay && m.type === "compile") setTimeout(run, o.delay); else run();
    }
    terminate() { this.terminated = true; }
  }
  return { FW, made };
}

describe("worker", () => {
  const MOD = "/_plugins/lini/lini_wasm.js", WASM = "/_plugins/lini/lini_wasm_bg.wasm", WORKER = "/_plugins/lini/lini.worker.js";

  test("compiles in a worker", async () => {
    const { FW, made } = fakeWorkers();
    const l = startLini(MOD, WASM, { Worker: FW, workerUrl: WORKER });
    expect(l.mode).toBe("worker");
    const [a, b] = await Promise.all([l.compile('|box#one| "ONE-LABEL"'), l.compile('|box#two| "TWO-LABEL"')]);
    expect(a).toContain("ONE-LABEL");
    expect(a).not.toContain("TWO-LABEL");
    expect(b).toContain("TWO-LABEL");
    expect(await l.compile(SRC)).toMatch(/^<svg/);
    expect(made).toHaveLength(1); // one worker per page
    expect(made[0].url).toBe(WORKER);
    expect(made[0].options).toEqual({ type: "module" });
    expect(made[0].posted[0]).toEqual({ type: "init", module: MOD, wasm: WASM });
    const ids = made[0].posted.filter((m: any) => m.type === "compile").map((m: any) => m.id);
    expect(new Set(ids).size).toBe(3);
    l.end();
    expect(made[0].terminated).toBe(true);
  });

  test("a compile error from the worker is the figure's error", async () => {
    const { FW } = fakeWorkers();
    const l = startLini(MOD, WASM, { Worker: FW, workerUrl: WORKER });
    await expect(l.compile("garbage ((")).rejects.toThrow();
    expect(await l.compile(SRC)).toMatch(/^<svg/); // the worker lives on
    expect(l.mode).toBe("worker");
    l.end();
  });

  test("mount draws through the worker when there is one", async () => {
    const { FW, made } = fakeWorkers();
    vi.stubGlobal("Worker", FW);
    setImporter(async () => { throw new Error("the page must not load the compiler"); });
    const ctx = ctxFor(url());
    const els = [block(SRC), block('|box| "second"')];
    await Promise.all(els.map((e) => lini.mount(e, ctx)));
    for (const e of els) expect(e.querySelector(".zen-lini-svg svg")).not.toBeNull();
    expect(made).toHaveLength(1);
    expect(ctx.assetUrl).toHaveBeenCalledWith("lini.worker.js");
  });

  test("a figure past the time limit", async () => {
    vi.useFakeTimers();
    try {
      const { FW, made } = fakeWorkers({ answer: false });
      const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
      const l = startLini(MOD, WASM, { Worker: FW, workerUrl: WORKER });
      const e = block(SRC);
      const p = renderOne(e, l);
      await vi.advanceTimersByTimeAsync(LINI_TIMEOUT - 1);
      expect(e.querySelector(".zen-plugin-error")).toBeNull();
      await vi.advanceTimersByTimeAsync(2);
      await p;
      expect(e.querySelector(".zen-plugin-error")!.textContent).toBe("lini: the figure took too long to draw (over 10 s)");
      expect(e.querySelector("pre")).not.toBeNull(); // the source stays
      expect("drawn" in e.dataset).toBe(true);
      expect(warn).toHaveBeenCalledTimes(1);
      expect(made[0].terminated).toBe(true);
      // the next figure gets a fresh worker
      const e2 = block(SRC);
      const p2 = renderOne(e2, l);
      await vi.advanceTimersByTimeAsync(LINI_TIMEOUT + 1);
      await p2;
      expect(made).toHaveLength(2);
      expect(made[1].terminated).toBe(true);
      l.end();
    } finally {
      vi.useRealTimers();
    }
  });

  test("falls back to the page: no Worker", async () => {
    const info = vi.spyOn(console, "info").mockImplementation(() => {});
    install();
    const l = startLini(url(), "/w.wasm", { Worker: undefined, workerUrl: WORKER });
    expect(l.mode).toBe("page");
    expect(await l.compile(SRC)).toMatch(/^<svg/);
    expect(await l.compile(SRC)).toMatch(/^<svg/);
    expect(info).toHaveBeenCalledTimes(1);
    expect(info).toHaveBeenCalledWith("lini: drawing on the page (no worker)");
    // and through mount, the global Worker being absent
    install();
    const e = block(SRC);
    await lini.mount(e, ctxFor(url()));
    expect(e.querySelector(".zen-lini-svg svg")).not.toBeNull();
  });

  test("falls back to the page: the WebView refuses the worker", async () => {
    const info = vi.spyOn(console, "info").mockImplementation(() => {});
    const { FW } = fakeWorkers({ throwNew: true });
    install();
    const l = startLini(url(), "/w.wasm", { Worker: FW, workerUrl: WORKER });
    expect(await l.compile(SRC)).toMatch(/^<svg/);
    expect(l.mode).toBe("page");
    expect(info).toHaveBeenCalledTimes(1);
  });

  test("falls back to the page: the worker cannot start the module", async () => {
    const info = vi.spyOn(console, "info").mockImplementation(() => {});
    const { FW, made } = fakeWorkers({ badInit: true });
    install();
    const l = startLini(url(), "/w.wasm", { Worker: FW, workerUrl: WORKER });
    const [a, b] = await Promise.all([l.compile(SRC), l.compile('|box| "b"')]);
    expect(a).toMatch(/^<svg/);
    expect(b).toMatch(/^<svg/);
    expect(l.mode).toBe("page");
    expect(made[0].terminated).toBe(true);
    expect(info).toHaveBeenCalledTimes(1);
    const e = block(SRC);
    await renderOne(e, l);
    expect(e.querySelector(".zen-lini-svg svg")).not.toBeNull();
  });

  test("a compiler that cannot load anywhere is a load failure", async () => {
    vi.spyOn(console, "info").mockImplementation(() => {});
    vi.spyOn(console, "warn").mockImplementation(() => {});
    const { FW } = fakeWorkers({ badInit: true });
    setImporter(async () => { throw new Error("blocked"); });
    const l = startLini(url(), "/w.wasm", { Worker: FW, workerUrl: WORKER });
    const e = block(SRC);
    await renderOne(e, l);
    expect(e.querySelector(".zen-plugin-error")!.textContent).toBe("lini: could not load the Lini compiler");
  });

  test("the worker never gets the page", () => {
    const src = readFileSync(join(here, "lini.worker.js"), "utf8");
    expect(src).not.toMatch(/document|window|localStorage|innerHTML|XMLHttpRequest|importScripts|eval\s*\(|new Function/);
    expect(src).not.toMatch(/https?:\/\//);
    expect(src.split("\n").length).toBeLessThan(80);
  });

  test("the worker only loads plugin files", async () => {
    const importer = vi.fn(async () => fakeModule().mod);
    const sent: any[] = [];
    const scope: any = { location: { href: "http://localhost/_plugins/lini/lini.worker.js", origin: "http://localhost" }, postMessage: (m: any) => sent.push(m), onmessage: null };
    serve(scope, importer);
    await scope.onmessage({ data: { type: "init", module: "https://evil.test/x.js", wasm: "/w.wasm" } });
    await scope.onmessage({ data: { type: "init", module: "/_plugins/lini/lini_wasm.js", wasm: "http://evil.test/w.wasm" } });
    await scope.onmessage({ data: { type: "init", module: "data:text/javascript,1", wasm: "/w.wasm" } });
    expect(importer).not.toHaveBeenCalled();
    expect(sent.map((m) => m.type)).toEqual(["init-error", "init-error", "init-error"]);
    await scope.onmessage({ data: { type: "init", module: "/_plugins/lini/lini_wasm.js", wasm: "/_plugins/lini/lini_wasm_bg.wasm" } });
    expect(sent[3]).toEqual({ type: "ready" });
    expect(importer).toHaveBeenCalledWith("/_plugins/lini/lini_wasm.js");
    await scope.onmessage({ data: { type: "compile", id: 7, src: SRC } });
    expect(sent[4].id).toBe(7);
    expect(sent[4].svg).toMatch(/^<svg/);
    await scope.onmessage({ data: { type: "compile", id: 8, src: "garbage ((" } });
    expect(sent[5].id).toBe(8);
    expect(typeof sent[5].error).toBe("string");
  });

  test("the worker loads its own files under the app's scheme, and nothing from another host or a data: url", async () => {
    const run = async (module: string, wasm: string) => {
      const sent: any[] = [];
      const scope: any = { location: { href: "wails://wails/_plugins/lini/lini.worker.js", origin: "null" }, postMessage: (m: any) => sent.push(m), onmessage: null };
      serve(scope, async () => fakeModule().mod);
      await scope.onmessage({ data: { type: "init", module, wasm } });
      return sent[0].type;
    };
    expect(await run("wails://wails/_plugins/lini/lini_wasm.js", "/_plugins/lini/lini_wasm_bg.wasm")).toBe("ready");
    expect(await run("wails://evil/x.js", "/_plugins/lini/lini_wasm_bg.wasm")).toBe("init-error");
    expect(await run("data:text/javascript,1", "/_plugins/lini/lini_wasm_bg.wasm")).toBe("init-error");
    expect(await run("https://example.com/x.js", "/_plugins/lini/lini_wasm_bg.wasm")).toBe("init-error");
  });

  test("the worker answers a compile before init with an error", async () => {
    const sent: any[] = [];
    const scope: any = { location: { href: "http://localhost/w.js", origin: "http://localhost" }, postMessage: (m: any) => sent.push(m), onmessage: null };
    serve(scope, async () => fakeModule().mod);
    await scope.onmessage({ data: { type: "compile", id: 1, src: SRC } });
    expect(sent[0].id).toBe(1);
    expect(sent[0].error).toBeTruthy();
  });

  test("unmount while the worker is initialising", async () => {
    const { FW, made } = fakeWorkers();
    const l = startLini(MOD, WASM, { Worker: FW, workerUrl: WORKER });
    const p = l.compile(SRC);
    const caught = p.then(() => "ok", (e: any) => String(e && e.message));
    l.end();
    expect(await caught).toMatch(/ended|unmounted|closed/);
    expect(made[0].terminated).toBe(true);
    await expect(l.compile(SRC)).rejects.toThrow();
    expect(made).toHaveLength(1); // nothing starts after end
  });

  test("a compile in flight when the figure is gone is ignored", async () => {
    let answer!: (s: string) => void;
    const slow = { compile: () => new Promise<string>((r) => (answer = r)) };
    const e = block(SRC);
    const p = renderOne(e, slow);
    e.remove(); // the page was replaced
    answer('<svg xmlns="http://www.w3.org/2000/svg"><rect width="1" height="1"/></svg>');
    await p;
    expect(e.querySelector("svg")).toBeNull();
    expect(e.querySelector(".zen-plugin-error")).toBeNull();
  });
});

describe("the page", () => {
  test("unmount mid-queue: the last release ends the worker and drops what waits", async () => {
    vi.stubGlobal("IntersectionObserver", FakeIO);
    setIO(FakeIO);
    const { FW, made } = fakeWorkers();
    vi.stubGlobal("Worker", FW);
    const ctx = ctxFor(url());
    const a = block(SRC), b = block('|box| "b"'), c = block('|box| "c"');
    const io = () => lastIO();
    const pa = lini.mount(a, ctx);
    const pb = lini.mount(b, ctx);
    const pc = lini.mount(c, ctx);
    await tick();
    io().fire(a);
    const release = await pa;
    expect(typeof release).toBe("function");
    expect(a.querySelector("svg")).not.toBeNull();
    expect(b.hasAttribute("data-zen-pending")).toBe(true); // waits behind the placeholder
    expect(b.querySelector("svg")).toBeNull();
    // the page is replaced: b and c leave with it
    b.remove(); c.remove();
    release();
    await Promise.all([pb, pc]);
    expect(made[0].terminated).toBe(true);
    expect(io().gone).toBe(true);
    expect(b.querySelector("svg")).toBeNull();
  });

  test("a page with 50 small figures draws them all, in order, one at a time", async () => {
    install();
    const ctx = ctxFor(url());
    const els = Array.from({ length: 50 }, (_, i) => block(`|box#b| "fig-${i}"`));
    const seen: number[] = [];
    const obs = new MutationObserver(() => {
      els.forEach((e, i) => { if (e.querySelector("svg") && !seen.includes(i)) seen.push(i); });
    });
    obs.observe(document.body, { childList: true, subtree: true });
    await Promise.all(els.map((e) => lini.mount(e, ctx)));
    await tick();
    obs.disconnect();
    els.forEach((e, i) => expect(e.querySelector(".zen-lini-svg")!.textContent).toContain(`fig-${i}`));
    expect(seen).toEqual([...seen].sort((x, y) => x - y));
  });

  test("two figures with the same source draw twice", async () => {
    install();
    const ctx = ctxFor(url());
    const a = block(SRC), b = block(SRC);
    await Promise.all([lini.mount(a, ctx), lini.mount(b, ctx)]);
    expect(a.querySelector(".zen-lini-svg svg")).not.toBeNull();
    expect(b.querySelector(".zen-lini-svg svg")).not.toBeNull();
  });

  test("a figure that errors in the middle does not stop the rest", async () => {
    install();
    vi.spyOn(console, "warn").mockImplementation(() => {});
    const ctx = ctxFor(url());
    const els = [block(SRC), block("garbage (("), block('|box| "after"')];
    await Promise.all(els.map((e) => lini.mount(e, ctx)));
    expect(els[0].querySelector("svg")).not.toBeNull();
    expect(els[1].querySelector(".zen-plugin-error")).not.toBeNull();
    expect(els[2].querySelector("svg")).not.toBeNull();
  });

  test("the theme changed while a figure waits", async () => {
    vi.stubGlobal("IntersectionObserver", FakeIO);
    setIO(FakeIO);
    install();
    const e = block(SRC);
    const p = lini.mount(e, ctxFor(url()));
    await tick();
    document.documentElement.dataset.theme = "dark";
    await tick();
    lastIO().fire(e);
    await p;
    expect(e.querySelector<HTMLElement>(".zen-lini-svg")!.style.colorScheme).toBe("dark");
  });

  test("a figure far below stays a placeholder until scrolled near", async () => {
    vi.stubGlobal("IntersectionObserver", FakeIO);
    setIO(FakeIO);
    install();
    const ctx = ctxFor(url());
    const e = block(SRC);
    const p = lini.mount(e, ctx);
    await tick();
    expect("drawn" in e.dataset).toBe(false);
    expect(e.hasAttribute("data-zen-pending")).toBe(true);
    lastIO().fire(e);
    await p;
    expect("drawn" in e.dataset).toBe(true);
    expect(e.hasAttribute("data-zen-pending")).toBe(false);
  });
});
