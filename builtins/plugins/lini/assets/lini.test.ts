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
import lini, { LINI_MODULE, cleanSvg, linkScheme, loadLini, renderOne, setImporter, watchTheme } from "./lini.js";

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

beforeEach(() => {
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
