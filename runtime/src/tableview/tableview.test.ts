import { afterAll, afterEach, beforeAll, beforeEach, expect, test } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { OVERLAY_EVENT } from "./overlay";
import { expandableTable, fitSize, installTableView, overflows, TABLE_SIZE_MAX, TABLE_SIZE_MIN, type TableView } from "./tableview";

// happy-dom has no layout: scrollWidth and clientWidth come from data-test-sw / data-test-cw on the element,
// so a re-rendered table keeps its (fake) size.
const proto = HTMLElement.prototype;
const saved = { sw: Object.getOwnPropertyDescriptor(proto, "scrollWidth"), cw: Object.getOwnPropertyDescriptor(proto, "clientWidth") };
beforeAll(() => {
  const attr = (name: string) => function (this: HTMLElement) { return Number(this.getAttribute(name) ?? 0); };
  Object.defineProperty(proto, "scrollWidth", { get: attr("data-test-sw"), configurable: true });
  Object.defineProperty(proto, "clientWidth", { get: attr("data-test-cw"), configurable: true });
});
afterAll(() => {
  for (const [k, d] of [["scrollWidth", saved.sw], ["clientWidth", saved.cw]] as const) {
    if (d) Object.defineProperty(proto, k, d); else delete (proto as unknown as Record<string, unknown>)[k];
  }
});

let root: HTMLElement;
let tv: TableView;

const WIDE = 'data-test-sw="1200" data-test-cw="600"';
const FITS = 'data-test-sw="400" data-test-cw="400"';
const md = (attrs = WIDE, extra = "") =>
  `<table ${attrs}${extra}><thead><tr><th>Name</th><th>Qty</th></tr></thead><tbody><tr><td><a href="b.md" id="x">Ink</a></td><td>3</td></tr></tbody></table>`;
const zt = (box = WIDE, caption = "<caption><span class=\"zt-label\">Table 1</span>: Stock levels</caption>") =>
  `<div class="zen-block zt"><div class="zen-block-body"><div class="zt-scroll" ${box}><table id="t1">${caption}` +
  `<thead><tr><th scope="col">Item</th><th scope="col">Qty</th></tr></thead>` +
  `<tbody><tr><th scope="row">Ink</th><td>3</td></tr></tbody></table></div></div></div>`;
const set = (html: string) => { root.innerHTML = html; tv.refresh(); };
const click = (n: Element, init: MouseEventInit = {}) =>
  n.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, button: 0, ...init }));
const key = (k: string, init: KeyboardEventInit = {}) => {
  const e = new KeyboardEvent("keydown", { key: k, bubbles: true, cancelable: true, ...init });
  (document.activeElement ?? document).dispatchEvent(e);
  return e;
};
const expand = (i = 0) => root.querySelectorAll<HTMLButtonElement>(".zen-table-expand")[i];
const view = () => document.querySelector<HTMLElement>(".zen-zoom");
const scale = () => view()!.querySelector(".zen-zoom-scale")!.textContent;
const bar = (label: string) => view()!.querySelector<HTMLButtonElement>(`button[aria-label^="${label}"]`)!;
const frame = () => new Promise<void>((r) => setTimeout(r, 40));

beforeEach(() => {
  document.body.replaceChildren();
  Object.defineProperty(window, "innerWidth", { value: 1064, configurable: true });
  Object.defineProperty(window, "innerHeight", { value: 864, configurable: true });
  root = document.body.appendChild(document.createElement("article"));
  root.className = "zen-page";
  tv = installTableView(root);
});
afterEach(() => tv.destroy());

test("expandableTable: which tables", () => {
  root.innerHTML = md() + zt() +
    `<div data-zen-plugin="quiz">${md()}</div>` +
    `<div data-zen-plugin="table">${md()}</div>` +
    `<a href="x.md">${md()}</a><button>${md()}</button>` +
    `<table id="outer"><tbody><tr><td><table id="inner"><tbody><tr><td>n</td></tr></tbody></table></td></tr></tbody></table>` +
    md(WIDE, " data-zen-nozoom");
  const tables = Array.from(root.querySelectorAll("table"));
  const ok = tables.filter((t) => expandableTable(t, root));
  expect(ok.map((t) => t.id || t.closest("[data-zen-plugin]")?.getAttribute("data-zen-plugin") || "md")).toEqual(["md", "t1", "table", "outer"]);
  expect(expandableTable(root.querySelector("#inner")!, root)).toBeNull();
  expect(expandableTable(document.createElement("table"), root)).toBeNull(); // not in root
  expect(expandableTable(root.querySelector("tr")!, root)).toBeNull(); // not a table
});

test("overflows: the scroll box is the table, or .zt-scroll for the built-in", () => {
  root.innerHTML = md() + zt(FITS);
  const [a, b] = Array.from(root.querySelectorAll("table"));
  expect(overflows(a)).toBe(true);
  expect(overflows(b)).toBe(false);
  a.setAttribute("data-test-sw", "601"); // one pixel of slack is not overflow
  expect(overflows(a)).toBe(false);
});

test("the Expand button", () => {
  set(zt());
  const btn = expand();
  expect(btn.className).toContain("zen-btn");
  expect(btn.className).toContain("zen-btn-quiet");
  expect(btn.type).toBe("button");
  expect(btn.textContent).toContain("Expand");
  expect(btn.querySelector("svg")).not.toBeNull();
  expect(btn.title).toBe("Expand table");
  expect(btn.getAttribute("aria-label")).toBe("Expand table: Table 1: Stock levels");
  const row = btn.parentElement!;
  expect(row.className).toBe("zen-table-tools");
  expect(row.hasAttribute("data-zen-ui")).toBe(true);
  expect(row.nextElementSibling).toBe(root.querySelector(".zt-scroll")); // right before the scroll box
  set(md());
  expect(expand().getAttribute("aria-label")).toBe("Expand table");
  expect(expand().parentElement!.nextElementSibling).toBe(root.querySelector("table"));
  expect(expand().tabIndex).toBe(0);
});

test("the button follows overflow", () => {
  set(md(FITS));
  expect(expand()).toBeUndefined();
  const t = root.querySelector("table")!;
  t.setAttribute("data-test-sw", "1200");
  tv.refresh();
  expect(root.querySelectorAll(".zen-table-expand")).toHaveLength(1);
  tv.refresh(); // no second button
  expect(root.querySelectorAll(".zen-table-expand")).toHaveLength(1);
  t.setAttribute("data-test-sw", "400");
  tv.refresh();
  expect(root.querySelector(".zen-table-tools")).toBeNull();
});

test("the button follows overflow: the column narrows when the sidebar opens", () => {
  set(md('data-test-sw="700" data-test-cw="900"'));
  expect(expand()).toBeUndefined();
  root.querySelector("table")!.setAttribute("data-test-cw", "600");
  tv.refresh();
  expect(expand()).toBeDefined();
});

test("a re-render keeps the buttons", async () => {
  set(md() + zt());
  expect(root.querySelectorAll(".zen-table-expand")).toHaveLength(2);
  root.innerHTML = zt() + md(FITS) + md(); // the page feature replaces the article's HTML (no refresh call)
  await frame();
  expect(root.querySelectorAll(".zen-table-expand")).toHaveLength(2);
  expect(root.querySelectorAll(".zen-table-tools")).toHaveLength(2);
});

test("empty tables, no thead, colspan and rowspan, markdown in a caption", () => {
  set(`<table ${WIDE}></table>` +
    `<table ${WIDE}><tbody><tr><td>a</td><td>b</td></tr></tbody></table>` +
    `<table ${WIDE}><caption>A <em>fine</em> <code>table</code></caption><tbody><tr><th rowspan="2" scope="row">x</th><td colspan="2">y</td></tr><tr><td>1</td><td>2</td></tr></tbody></table>`);
  expect(root.querySelectorAll(".zen-table-expand")).toHaveLength(3);
  expect(expand(2).getAttribute("aria-label")).toBe("Expand table: A fine table");
  for (let i = 0; i < 3; i++) { tv.open(root.querySelectorAll("table")[i]); expect(tv.isOpen).toBe(true); tv.close(); }
});

test("opening shows a copy of the table", () => {
  set(zt());
  click(expand());
  const v = view()!;
  expect(tv.isOpen).toBe(true);
  expect(v.classList.contains("zen-tableview")).toBe(true);
  expect(v.getAttribute("role")).toBe("dialog");
  expect(v.getAttribute("aria-modal")).toBe("true");
  expect(v.getAttribute("aria-label")).toBe("Table: Table 1: Stock levels");
  expect(v.parentElement).toBe(document.body);
  const stage = v.querySelector(".zen-zoom-stage.zen-tableview-stage")!;
  const copy = stage.querySelector<HTMLTableElement>("table.zen-tableview-table")!;
  expect(copy).not.toBe(root.querySelector("table"));
  expect(copy.querySelector("caption")!.textContent).toContain("Stock levels");
  expect(copy.querySelectorAll("tbody td")).toHaveLength(1);
  expect(v.querySelectorAll("[id]")).toHaveLength(0); // ids are removed from the copy
  expect(root.querySelector("#t1")).not.toBeNull(); // and kept in the page
  expect(v.querySelector(".zen-zoom-bar")).not.toBeNull();
  expect(v.querySelector(".zen-table-tools")).toBeNull();
  tv.close();
  set(`<p>no table</p>`);
  tv.open(document.createElement("table")); // a table that is not in the page: nothing happens
  expect(tv.isOpen).toBe(false);
});

test("the module builds the view without parsing strings", () => {
  const src = readFileSync(resolve(__dirname, "tableview.ts"), "utf8");
  expect(src).not.toMatch(/innerHTML|insertAdjacentHTML|outerHTML|DOMParser|createContextualFragment/);
});

test("headers stay in view", () => {
  set(zt());
  click(expand());
  const copy = view()!.querySelector("table")!;
  for (const th of Array.from(copy.querySelectorAll("thead th"))) expect(th.classList.contains("zen-tv-head")).toBe(true);
  expect(copy.querySelector('tbody th[scope="row"]')!.classList.contains("zen-tv-stub")).toBe(true);
  expect(copy.querySelector("tbody td")!.className).toBe("");
  expect(copy.querySelector("thead th")!.closest(".zen-tableview")).not.toBeNull();
});

test("fitSize", () => {
  expect(fitSize(500, 1000)).toBe(1); // never above 100%
  expect(fitSize(2000, 1000)).toBe(0.5);
  expect(fitSize(1600, 1000)).toBeCloseTo(0.625);
  expect(fitSize(9000, 1000)).toBe(TABLE_SIZE_MIN); // never below 50%
  expect(fitSize(0, 1000)).toBe(1);
  expect(TABLE_SIZE_MIN).toBe(0.5);
  expect(TABLE_SIZE_MAX).toBe(2);
});

test("the bar sets the size", () => {
  set(md());
  click(expand());
  expect(scale()).toBe("100%"); // opens at 100%
  expect(view()!.querySelector(".zen-zoom-scale")!.getAttribute("aria-live")).toBe("polite");
  key("+");
  expect(scale()).toBe("125%");
  key("=");
  expect(scale()).toBe("156%");
  key("-");
  key("-");
  expect(scale()).toBe("100%");
  key("1");
  key("0"); // Fit: the table is 1200 wide in a stage of 1064 - 64
  expect(scale()).toBe("83%");
  for (let i = 0; i < 20; i++) key("+");
  expect(scale()).toBe(`${TABLE_SIZE_MAX * 100}%`);
  for (let i = 0; i < 30; i++) key("-");
  expect(scale()).toBe(`${TABLE_SIZE_MIN * 100}%`);
  key("1");
  click(bar("Larger"));
  expect(scale()).toBe("125%");
  click(bar("Smaller"));
  click(bar("Fit"));
  expect(scale()).toBe("83%");
  const pinch = new WheelEvent("wheel", { deltaY: -50, bubbles: true, cancelable: true });
  Object.defineProperty(pinch, "ctrlKey", { value: true });
  view()!.dispatchEvent(pinch);
  expect(pinch.defaultPrevented).toBe(true);
  expect(parseInt(scale()!)).toBeGreaterThan(83);
});

test("a narrow window starts at Fit", () => {
  Object.defineProperty(window, "innerWidth", { value: 375, configurable: true });
  set(md());
  click(expand());
  expect(scale()).toBe("50%"); // (375 - 64) / 1200 is below the minimum
  expect(view()!.querySelectorAll(".zen-zoom-bar button")).toHaveLength(4);
});

test("closing", () => {
  set(md());
  const btn = expand();
  const close = (how: () => void) => {
    btn.focus();
    click(btn);
    expect(document.activeElement?.getAttribute("aria-label")).toBe("Close (Esc)");
    how();
    expect(view()).toBeNull();
    expect(tv.isOpen).toBe(false);
    expect(document.activeElement).toBe(btn);
  };
  close(() => key("Escape"));
  close(() => click(bar("Close")));
  close(() => click(view()!.querySelector(".zen-tableview-stage")!)); // the backdrop outside the table
  btn.focus();
  click(btn);
  click(view()!.querySelector("td")!);
  click(view()!.querySelector("table")!);
  expect(tv.isOpen).toBe(true); // a click inside the table does not close it
  const e = key("x");
  expect(e.defaultPrevented).toBe(false);
  tv.close();
  const after = new KeyboardEvent("keydown", { key: "+", cancelable: true });
  document.dispatchEvent(after);
  expect(after.defaultPrevented).toBe(false);
});

test("links act like the page's", () => {
  set(md());
  const seen: string[] = [];
  root.addEventListener("click", (e) => { const a = (e.target as Element).closest("a"); if (a) { seen.push(a.getAttribute("href")!); e.preventDefault(); } });
  click(expand());
  const link = view()!.querySelector("a")!;
  const ev = new MouseEvent("click", { bubbles: true, cancelable: true, button: 0 });
  link.dispatchEvent(ev);
  expect(ev.defaultPrevented).toBe(true); // the copy's link does nothing itself
  expect(tv.isOpen).toBe(false);
  expect(seen).toEqual(["b.md"]); // the page's link got the click
});

test("focus stays in the view", () => {
  set(md());
  click(expand());
  const stop = Array.from(view()!.querySelectorAll<HTMLElement>("a[href], button, [tabindex='0']"));
  expect(stop.length).toBeGreaterThanOrEqual(6); // stage, a link, − + Fit ✕
  expect(document.activeElement).toBe(bar("Close"));
  const seen = new Set<Element | null>();
  for (let i = 0; i < stop.length + 2; i++) {
    const e = key("Tab");
    expect(e.defaultPrevented).toBe(true);
    expect(view()!.contains(document.activeElement)).toBe(true);
    seen.add(document.activeElement);
  }
  expect(seen.size).toBe(stop.length);
  for (let i = 0; i < stop.length + 2; i++) {
    key("Tab", { shiftKey: true });
    expect(view()!.contains(document.activeElement)).toBe(true);
  }
  expect(view()!.querySelector(".zen-tableview-stage")!.getAttribute("tabindex")).toBe("0");
});

test("arrow keys scroll the stage when a button has the focus", () => {
  set(md());
  click(expand());
  const stage = view()!.querySelector<HTMLElement>(".zen-tableview-stage")!;
  let moved = 0;
  stage.scrollBy = (() => { moved++; }) as typeof stage.scrollBy;
  expect(key("ArrowDown").defaultPrevented).toBe(true);
  expect(moved).toBe(1);
});

test("one overlay at a time", () => {
  set(md() + zt());
  click(expand(0));
  const first = view()!;
  click(expand(1)); // the second table opens, the first closes
  expect(document.querySelectorAll(".zen-zoom")).toHaveLength(1);
  expect(view()).not.toBe(first);
  // Another view over the window (the image zoom, a plugin's) announces itself with the shared event.
  const owners: string[] = [];
  const on = (e: Event) => owners.push((e as CustomEvent).detail.owner);
  document.addEventListener(OVERLAY_EVENT, on);
  tv.close();
  click(expand(0));
  document.removeEventListener(OVERLAY_EVENT, on);
  expect(owners).toHaveLength(1);
  const owner = owners[0];
  expect(owner).toMatch(/\S/);
  document.dispatchEvent(new CustomEvent(OVERLAY_EVENT, { detail: { owner: "someone-else" } }));
  expect(tv.isOpen).toBe(false);
  document.addEventListener(OVERLAY_EVENT, on);
  click(expand(0));
  document.removeEventListener(OVERLAY_EVENT, on);
  document.dispatchEvent(new CustomEvent(OVERLAY_EVENT, { detail: { owner: owners[owners.length - 1] } })); // its own event is ignored
  expect(tv.isOpen).toBe(true);
  // destroy closes and disconnects
  tv.destroy();
  expect(view()).toBeNull();
  root.innerHTML = md();
  expect(root.querySelector(".zen-table-expand")).toBeNull();
  click(root.querySelector("table")!);
  expect(view()).toBeNull();
  document.dispatchEvent(new CustomEvent(OVERLAY_EVENT, { detail: { owner: "x" } })); // no listener left to throw
});

test("the view closes when the page is re-rendered", async () => {
  set(md());
  click(expand());
  root.innerHTML = "<p>another page</p>";
  await frame();
  expect(tv.isOpen).toBe(false);
  expect(view()).toBeNull();
});

test("a spec-shaped table: many rows, long cells with code and links", () => {
  const rows = Array.from({ length: 40 }, (_, i) =>
    `<tr><td><code>runtime/src/file${i}.ts</code></td><td>Opens the table <a href="../spec${i}.md">spec ${i}</a> over the window, with a long explanation that has to wrap at forty characters.</td><td>x</td></tr>`).join("");
  set(`<table ${WIDE}><thead><tr><th>Owns</th><th>Behaviour</th><th>Test</th></tr></thead><tbody>${rows}</tbody></table>`);
  const clicked: string[] = [];
  root.addEventListener("click", (e) => { const a = (e.target as Element).closest("a"); if (a) { clicked.push(a.getAttribute("href")!); e.preventDefault(); } });
  click(expand());
  const copy = view()!.querySelector("table")!;
  expect(copy.querySelectorAll("tbody tr")).toHaveLength(40);
  expect(copy.querySelectorAll("thead th.zen-tv-head")).toHaveLength(3);
  click(copy.querySelectorAll("a")[7]);
  expect(clicked).toEqual(["../spec7.md"]);
  expect(tv.isOpen).toBe(false);
});

test("one overlay at a time with installImageZoom", async () => {
  const { installImageZoom } = await import("../zoom/zoom");
  set(md() + '<img src="a.png" alt="A" width="400" height="300">');
  const zoom = installImageZoom(root);
  click(expand());
  expect(tv.isOpen).toBe(true);
  zoom.open(root.querySelector("img")!);
  expect(tv.isOpen).toBe(false); // the image view closed the table view
  expect(zoom.isOpen).toBe(true);
  click(expand());
  expect(zoom.isOpen).toBe(false); // and the other way round
  expect(tv.isOpen).toBe(true);
  tv.close();
  zoom.destroy();
});
