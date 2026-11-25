import { afterAll, afterEach, beforeAll, beforeEach, expect, test } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { installImageZoom } from "../zoom/zoom";
import {
  clampScale, DRAG_SLOP, fit, installDiagramView, limitPan, PAN_STEP, SCALE_MAX, SCALE_MIN, STEP, zoomAt, type DiagramView,
} from "./diagramview";

const css = readFileSync(resolve(__dirname, "../css/page.css"), "utf8");
const source = readFileSync(resolve(__dirname, "diagramview.ts"), "utf8");
/** Our rules in page.css: from the "Figure view" comment to the next blank line. */
const block = css.match(/\/\* Figure view[\s\S]*?\n\n/)?.[0] ?? "";

// happy-dom has no layout. The viewport is VW wide (its height is what the view sets); an svg reports
// data-test-w / data-test-h times its current scale() (as a browser reports a transformed box); the expanded stage is 1000×800.
let VW = 600;
const proto = Element.prototype;
const savedRect = proto.getBoundingClientRect;
beforeAll(() => {
  proto.getBoundingClientRect = function (this: Element) {
    const r = (width: number, height: number) => ({ x: 0, y: 0, left: 0, top: 0, width, height, right: width, bottom: height, toJSON() {} }) as DOMRect;
    if (this.classList.contains("zen-figure-view")) return r(VW, parseFloat((this as HTMLElement).style.height) || 0);
    if (this.classList.contains("zen-zoom-stage")) return r(1000, 800);
    const w = Number(this.getAttribute("data-test-w") ?? 0), h = Number(this.getAttribute("data-test-h") ?? 0);
    if (w && h) {
      const k = parseFloat(/scale\(([\d.e+-]+)\)/.exec((this as HTMLElement).style.transform)?.[1] ?? "1");
      return r(w * k, h * k);
    }
    return r(0, 0);
  };
});
afterAll(() => { proto.getBoundingClientRect = savedRect; });

let root: HTMLElement;
let dv: DiagramView;

const SIZE = 'data-test-w="600" data-test-h="300"';
const svgOf = (id = "a", attrs = SIZE, vb = 'viewBox="0 0 1200 600"') =>
  `<svg id="${id}" ${attrs} ${vb}><title>Flow</title><g id="${id}-g"/></svg>`;
const mermaid = (id = "a", attrs = SIZE, vb?: string) =>
  `<div class="zen-diagram" data-drawn><div class="zen-diagram-svg" data-zen-zoomable>${svgOf(id, attrs, vb)}</div></div>`;
const lini = (id = "l", attrs = SIZE) =>
  `<div class="zen-lini" data-drawn><div class="zen-lini-svg" data-zen-zoomable role="img" aria-label="Lini figure">${svgOf(id, attrs, 'viewBox="0 0 2400 1200"')}</div></div>`;
const set = (html: string) => { root.innerHTML = html; dv.refresh(); };

const viewport = (i = 0) => root.querySelectorAll<HTMLElement>(".zen-figure-view")[i];
const svg = (id = "a") => document.getElementById(id) as unknown as SVGElement;
const btn = (label: string, scope: ParentNode = root) => scope.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)!;
const state = (el: SVGElement = svg()) => {
  const t = el.style.transform;
  const m = /translate\(([-\d.e]+)px,\s*([-\d.e]+)px\)\s*scale\(([\d.e+-]+)\)/.exec(t);
  return m ? { x: parseFloat(m[1]), y: parseFloat(m[2]), scale: parseFloat(m[3]) } : null;
};
const click = (n: Element, init: MouseEventInit = {}) =>
  n.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, button: 0, ...init }));
const ptr = (n: Element, type: string, x: number, y: number) =>
  n.dispatchEvent(new MouseEvent(type, { bubbles: true, cancelable: true, button: 0, clientX: x, clientY: y }));
const drag = (n: Element, dx: number, dy: number, from = [300, 150]) => {
  ptr(n, "pointerdown", from[0], from[1]);
  ptr(n, "pointermove", from[0] + dx / 2, from[1] + dy / 2);
  ptr(n, "pointermove", from[0] + dx, from[1] + dy);
  ptr(n, "pointerup", from[0] + dx, from[1] + dy);
};
const key = (target: EventTarget, k: string, init: KeyboardEventInit = {}) => {
  const e = new KeyboardEvent("keydown", { key: k, bubbles: true, cancelable: true, ...init });
  target.dispatchEvent(e);
  return e;
};
const wheel = (n: Element, init: WheelEventInit) => {
  const e = new WheelEvent("wheel", { bubbles: true, cancelable: true, ...init });
  // happy-dom's WheelEvent has no modifier keys or position
  for (const k of ["ctrlKey", "metaKey", "clientX", "clientY", "deltaX", "deltaY"] as const) {
    Object.defineProperty(e, k, { value: init[k] ?? (k.endsWith("Key") ? false : 0) });
  }
  n.dispatchEvent(e);
  return e;
};
const zoomView = () => document.querySelector<HTMLElement>(".zen-zoom");
const pct = () => zoomView()!.querySelector(".zen-zoom-scale")!.textContent;
const frame = () => new Promise<void>((r) => setTimeout(r, 40));

beforeEach(() => {
  VW = 600;
  document.body.replaceChildren();
  Object.defineProperty(window, "innerWidth", { value: 1064, configurable: true });
  Object.defineProperty(window, "innerHeight", { value: 864, configurable: true });
  root = document.body.appendChild(document.createElement("article"));
  root.className = "zen-page";
  dv = installDiagramView(root);
});
afterEach(() => dv.destroy());

// ---- the maths ----

test("clampScale keeps 10% to 800%", () => {
  expect([SCALE_MIN, SCALE_MAX, STEP, PAN_STEP, DRAG_SLOP]).toEqual([0.1, 8, 1.25, 0.1, 4]);
  expect(clampScale(0.01)).toBe(0.1);
  expect(clampScale(50)).toBe(8);
  expect(clampScale(2)).toBe(2);
});

test("zoomAt keeps the point under the pointer where it was", () => {
  const s = zoomAt({ scale: 1, x: 10, y: 20 }, 2, 100, 50);
  expect(s.scale).toBe(2);
  // the figure point under (100, 50) before: ((100-10)/1, (50-20)/1) = (90, 30); after: (100 - x)/2 and (50 - y)/2
  expect((100 - s.x) / 2).toBeCloseTo(90);
  expect((50 - s.y) / 2).toBeCloseTo(30);
  expect(zoomAt({ scale: 7, x: 0, y: 0 }, 2, 0, 0).scale).toBe(8);
  expect(zoomAt({ scale: 0.1, x: 0, y: 0 }, 0.5, 0, 0).scale).toBe(0.1);
});

test("limitPan always leaves a quarter of the viewport on the figure", () => {
  const s = { scale: 2, x: 0, y: 0 };
  expect(limitPan({ ...s, x: 100000, y: 100000 }, 600, 300, 600, 300)).toEqual({ scale: 2, x: 450, y: 225 });
  expect(limitPan({ ...s, x: -100000, y: -100000 }, 600, 300, 600, 300)).toEqual({ scale: 2, x: 150 - 1200, y: 75 - 600 });
  expect(limitPan({ scale: 2, x: -10, y: -10 }, 600, 300, 600, 300)).toEqual({ scale: 2, x: -10, y: -10 }); // inside: untouched
  const small = limitPan({ scale: 0.1, x: 5000, y: 0 }, 600, 300, 600, 300); // a figure smaller than the quarter stays whole
  expect(small.x).toBe(600 - 60);
});

test("fit: centred, never past maxScale", () => {
  expect(fit(600, 300, 600, 300, 1)).toEqual({ scale: 1, x: 0, y: 0 });
  expect(fit(300, 150, 600, 300, 1)).toEqual({ scale: 1, x: 150, y: 75 });
  const f = fit(1200, 600, 1000, 800, 1, 32);
  expect(f.scale).toBeCloseTo(936 / 1200);
  expect(f.x).toBeCloseTo((1000 - 1200 * f.scale) / 2);
  expect(fit(100, 50, 1000, 800, 1, 32).scale).toBe(1);
  expect(fit(1000, 4000, 1000, 800, 1).scale).toBeCloseTo(0.2); // a tall figure fits the height
  expect(fit(0, 0, 100, 100, 1).scale).toBe(1);
});

// ---- the viewer ----

test("only zoomable figures", () => {
  set(
    mermaid("a") +
      `<div class="zen-diagram" data-drawn><div class="zen-diagram-svg">${svgOf("plain")}</div></div>` + // not marked
      `<div class="zen-diagram" data-drawn><div class="zen-diagram-svg" data-zen-zoomable>${svgOf("two")}${svgOf("two2")}</div></div>` +
      `<div class="zen-diagram" data-drawn><div class="zen-diagram-svg" data-zen-zoomable><p>no svg</p></div></div>` +
      `<div class="zen-diagram" data-drawn><div class="zen-diagram-svg" data-zen-zoomable>${svgOf("empty", "")}</div></div>` + // zero size
      `<div class="zen-diagram" data-drawn><pre>graph TD</pre><div class="zen-plugin-error">diagram: bad</div></div>`, // failed
  );
  expect(root.querySelectorAll(".zen-figure-view")).toHaveLength(1);
  expect(viewport().contains(svg("a"))).toBe(true);
  for (const id of ["plain", "two", "two2", "empty"]) expect(svg(id).closest(".zen-figure-view")).toBeNull();
  expect(root.querySelectorAll(".zen-figure-tools")).toHaveLength(1);
});

test("the viewport starts fitted and keeps the height", () => {
  set(mermaid());
  const v = viewport();
  expect(v.tabIndex).toBe(0);
  expect(v.getAttribute("role")).toBe("img");
  expect(v.getAttribute("aria-label")).toBe("Flow");
  expect(v.style.height).toBe("300px");
  expect(v.querySelector("[data-zen-zoomable]")!.parentElement).toBe(v);
  expect(state()).toEqual({ x: 0, y: 0, scale: 1 });
  expect(svg().style.width).toBe("600px");
  expect(svg().style.height).toBe("300px");
  expect(svg().style.transformOrigin).toMatch(/^0(px)?\s+0(px)?$/);
  dv.refresh(); // idempotent
  expect(root.querySelectorAll(".zen-figure-view")).toHaveLength(1);
});

test("a narrower figure is centred, and with no title the label is Diagram", () => {
  set(`<div class="zen-diagram"><div class="zen-diagram-svg" data-zen-zoomable><svg id="a" data-test-w="300" data-test-h="150"><g/></svg></div></div>`);
  expect(state()).toEqual({ x: 150, y: 0, scale: 1 });
  expect(viewport().getAttribute("aria-label")).toBe("Diagram");
});

test("a wide figure fits the column", () => {
  // the figure is 2400 wide by its viewBox; the page's CSS (max-width: 100%) makes it 600 in the column
  set(lini());
  expect(state(svg("l"))).toEqual({ x: 0, y: 0, scale: 1 });
  expect(svg("l").style.width).toBe("600px");
  expect(viewport().style.height).toBe("300px");
});

test("the corner bar", () => {
  set(mermaid());
  const bar = viewport().querySelector<HTMLElement>(".zen-figure-tools")!;
  expect(bar.classList.contains("zen-zoom-bar")).toBe(true);
  const buttons = Array.from(bar.querySelectorAll("button"));
  expect(buttons.map((b) => b.getAttribute("aria-label"))).toEqual([
    "Zoom in", "Zoom out", "Reset", "Expand", "Pan left", "Pan up", "Pan right", "Pan down",
  ]);
  for (const b of buttons) {
    expect(b.className).toContain("zen-zoom-btn");
    expect(b.type).toBe("button");
    expect(b.title).toBe(b.getAttribute("aria-label"));
  }
  expect(buttons.map((b) => b.textContent)).toEqual(["+", "−", "⟲", "⤢", "←", "↑", "→", "↓"]);
});

test("zoom in place", () => {
  set(mermaid());
  click(btn("Zoom in"));
  let s = state()!;
  expect(s.scale).toBeCloseTo(1.25);
  expect((300 - s.x) / s.scale).toBeCloseTo(300); // around the centre of the 600×300 viewport
  expect((150 - s.y) / s.scale).toBeCloseTo(150);
  click(btn("Zoom out"));
  expect(state()!.scale).toBeCloseTo(1);
  click(btn("Zoom in")); click(btn("Zoom in"));
  click(btn("Reset"));
  expect(state()).toEqual({ x: 0, y: 0, scale: 1 });
  for (let i = 0; i < 30; i++) click(btn("Zoom in"));
  expect(state()!.scale).toBe(8);
  for (let i = 0; i < 60; i++) click(btn("Zoom out"));
  expect(state()!.scale).toBeCloseTo(0.1);
  expect(viewport().hasAttribute("data-zoomed")).toBe(false);
});

test("a pinch zooms around the pointer; a plain wheel scrolls the page", () => {
  set(mermaid());
  const e = wheel(svg(), { ctrlKey: true, deltaY: -100, clientX: 100, clientY: 50 });
  expect(e.defaultPrevented).toBe(true);
  const s = state()!;
  expect(s.scale).toBeGreaterThan(1);
  expect((100 - s.x) / s.scale).toBeCloseTo(100, 1); // the figure point under the pointer did not move
  expect((50 - s.y) / s.scale).toBeCloseTo(50, 1);
  const meta = wheel(svg(), { metaKey: true, deltaY: 100, clientX: 100, clientY: 50 });
  expect(meta.defaultPrevented).toBe(true);
  const before = svg().style.transform;
  const plain = wheel(svg(), { deltaY: 120, deltaX: 5, clientX: 100, clientY: 50 });
  expect(plain.defaultPrevented).toBe(false);
  expect(svg().style.transform).toBe(before);
});

test("drag pans when zoomed", () => {
  set(mermaid());
  click(btn("Zoom in")); click(btn("Zoom in"));
  const before = state()!;
  expect(viewport().hasAttribute("data-zoomed")).toBe(true);
  drag(viewport(), 40, 20);
  const after = state()!;
  expect(after.scale).toBeCloseTo(before.scale);
  expect(after.x).toBeCloseTo(before.x + 40);
  expect(after.y).toBeCloseTo(before.y + 20);
  drag(viewport(), 100000, 100000); // stops at the edge
  expect(state()!.x).toBeLessThanOrEqual(600 - 150);
  expect(state()!.y).toBeLessThanOrEqual(300 - 75);
});

test("drag at fit does nothing", () => {
  set(mermaid());
  drag(viewport(), 80, 40);
  expect(state()).toEqual({ x: 0, y: 0, scale: 1 });
  expect(viewport().hasAttribute("data-zoomed")).toBe(false);
});

test("the arrow buttons pan by 10% of the viewport", () => {
  set(mermaid());
  click(btn("Zoom in")); click(btn("Zoom in"));
  const a = state()!;
  click(btn("Pan left"));
  expect(state()!.x).toBeCloseTo(a.x + 60); // the view moves left, the figure right
  click(btn("Pan up"));
  expect(state()!.y).toBeCloseTo(a.y + 30);
  click(btn("Pan right")); click(btn("Pan down"));
  expect(state()!.x).toBeCloseTo(a.x);
  expect(state()!.y).toBeCloseTo(a.y);
});

test("keyboard in place", () => {
  set(mermaid() + `<button id="other">x</button>`);
  const v = viewport();
  // not focused: the page keeps its keys
  const idle = key(root.querySelector("#other")!, "+");
  expect(idle.defaultPrevented).toBe(false);
  expect(state()).toEqual({ x: 0, y: 0, scale: 1 });
  v.focus();
  expect(key(v, "+").defaultPrevented).toBe(true);
  expect(state()!.scale).toBeCloseTo(1.25);
  key(v, "="); expect(state()!.scale).toBeCloseTo(1.5625);
  key(v, "-"); key(v, "-");
  expect(state()!.scale).toBeCloseTo(1);
  key(v, "+"); key(v, "+");
  const a = state()!;
  key(v, "ArrowLeft"); expect(state()!.x).toBeCloseTo(a.x + 60);
  key(v, "ArrowDown"); expect(state()!.y).toBeCloseTo(a.y - 30);
  key(v, "ArrowRight", { shiftKey: true }); expect(state()!.x).toBeCloseTo(a.x + 60 - 300);
  key(v, "0");
  expect(state()).toEqual({ x: 0, y: 0, scale: 1 });
  // a key on a bar button is not the viewport's
  click(btn("Zoom in"));
  const s = state()!.scale;
  key(btn("Reset"), "+");
  expect(state()!.scale).toBe(s);
  key(v, "Enter");
  expect(zoomView()).not.toBeNull();
});

// ---- expanded ----

test("expand moves the svg and back", () => {
  set(mermaid());
  const holder = root.querySelector(".zen-diagram-svg")!;
  const el = svg();
  click(btn("Expand"));
  const z = zoomView()!;
  expect(z.classList.contains("zen-figure-zoom")).toBe(true);
  expect(z.getAttribute("role")).toBe("dialog");
  expect(z.getAttribute("aria-modal")).toBe("true");
  expect(z.getAttribute("aria-label")).toBe("Diagram: Flow");
  expect(el.parentElement!.classList.contains("zen-zoom-stage")).toBe(true);
  expect(z.contains(el)).toBe(true);
  expect(document.querySelectorAll("#a")).toHaveLength(1); // moved, not copied
  expect(svg()).toBe(el);
  const ph = holder.querySelector<HTMLElement>(".zen-figure-placeholder")!;
  expect(ph).not.toBeNull();
  expect(ph.style.height).toBe("300px");
  expect(dv.openCount).toBe(1);
  key(document, "Escape");
  expect(zoomView()).toBeNull();
  expect(holder.querySelector(".zen-figure-placeholder")).toBeNull();
  expect(el.parentElement).toBe(holder);
  expect(dv.openCount).toBe(0);
  expect(state()).toEqual({ x: 0, y: 0, scale: 1 });
});

test("the expanded view", () => {
  set(mermaid());
  btn("Expand").focus();
  click(btn("Expand"));
  const z = zoomView()!;
  expect(Array.from(z.querySelectorAll(".zen-zoom-bar button")).map((b) => b.getAttribute("aria-label")))
    .toEqual(["Zoom out (−)", "Zoom in (+)", "Fit (0)", "Close (Esc)"]);
  const labels = Array.from(z.querySelectorAll(".zen-zoom-bar button")).map((b) => b.textContent);
  expect(labels).toEqual(["−", "+", "Fit", "✕"]);
  expect(z.querySelector(".zen-zoom-scale")!.getAttribute("aria-live")).toBe("polite");
  expect(document.activeElement).toBe(btn("Close (Esc)", z));
  expect(pct()).toBe("78%"); // 1200×600 fitted into 1000×800 less 32px each side
  key(document, "+"); expect(pct()).toBe("98%");
  key(document, "1"); expect(pct()).toBe("100%");
  key(document, "-"); expect(pct()).toBe("80%");
  key(document, "0"); expect(pct()).toBe("78%");
  key(document, "="); key(document, "_");
  expect(pct()).toBe("78%");
  click(btn("Zoom in (+)", z)); expect(pct()).toBe("98%");
  click(btn("Zoom out (−)", z)); click(btn("Fit (0)", z)); expect(pct()).toBe("78%");
});

test("the expanded view pans and zooms with the wheel and by dragging", () => {
  set(mermaid());
  click(btn("Expand"));
  key(document, "1"); // 100%: 1200×600 in a 1000×800 stage
  const el = svg();
  const a = state(el)!;
  expect(a.scale).toBe(1);
  const stage = zoomView()!.querySelector(".zen-zoom-stage")!;
  const plain = wheel(stage, { deltaX: 10, deltaY: 20 });
  expect(plain.defaultPrevented).toBe(true);
  expect(state(el)!.x).toBeCloseTo(a.x - 10);
  expect(state(el)!.y).toBeCloseTo(a.y - 20);
  const pinch = wheel(stage, { ctrlKey: true, deltaY: -50, clientX: 500, clientY: 400 });
  expect(pinch.defaultPrevented).toBe(true);
  expect(Number(pct()!.replace("%", ""))).toBeGreaterThan(100);
  key(document, "1");
  const b = state(el)!;
  drag(stage, 30, 10, [500, 400]); // at any scale
  expect(state(el)!.x).toBeCloseTo(b.x + 30);
  expect(state(el)!.y).toBeCloseTo(b.y + 10);
  key(document, "ArrowLeft"); expect(state(el)!.x).toBeCloseTo(b.x + 30 + 100);
  key(document, "ArrowUp", { shiftKey: true }); expect(state(el)!.y).toBeCloseTo(b.y + 10 + 400);
});

test("a figure with no viewBox opens at its fitted size", () => {
  set(`<div class="zen-diagram"><div class="zen-diagram-svg" data-zen-zoomable><svg id="a" ${SIZE}><g/></svg></div></div>`);
  click(btn("Expand"));
  expect(pct()).toBe("100%");
});

test("closing the expanded view", () => {
  set(mermaid());
  const holderBtn = btn("Expand");
  holderBtn.focus();
  click(holderBtn);
  click(btn("Zoom in")); // (the bar in place is under the dialog; the call keeps the in-place state out of it)
  click(btn("Close (Esc)", zoomView()!));
  expect(zoomView()).toBeNull();
  expect(document.activeElement).toBe(holderBtn);
  // the stage outside the figure closes it
  click(btn("Expand"));
  click(zoomView()!.querySelector(".zen-zoom-stage")!);
  expect(zoomView()).toBeNull();
  // a click on the figure itself does not
  click(btn("Expand"));
  click(svg());
  expect(zoomView()).not.toBeNull();
  key(document, "Escape");
  // opened from the keyboard, focus returns to the viewport
  viewport().focus();
  key(viewport(), "Enter");
  key(document, "Escape");
  expect(document.activeElement).toBe(viewport());
});

test("the in-place zoom is kept across an expanded view", () => {
  set(mermaid());
  click(btn("Zoom in")); click(btn("Zoom in"));
  const before = state()!;
  click(btn("Expand"));
  key(document, "Escape");
  expect(state()).toEqual(before);
});

test("focus stays inside the expanded view", () => {
  set(mermaid());
  click(btn("Expand"));
  const z = zoomView()!;
  const list = Array.from(z.querySelectorAll<HTMLButtonElement>("button"));
  list[list.length - 1].focus();
  const e = key(document.activeElement!, "Tab");
  expect(e.defaultPrevented).toBe(true);
  expect(document.activeElement).toBe(list[0]);
  const back = key(document.activeElement!, "Tab", { shiftKey: true });
  expect(back.defaultPrevented).toBe(true);
  expect(document.activeElement).toBe(list[list.length - 1]);
});

test("one view at a time", () => {
  set(mermaid("a") + mermaid("b") + `<p><img id="pic" src="x.png" alt="pic"></p>`);
  const zoom = installImageZoom(root);
  const img = root.querySelector("img")!;
  Object.defineProperty(img, "naturalWidth", { value: 2000 });
  Object.defineProperty(img, "naturalHeight", { value: 1000 });
  Object.defineProperty(img, "complete", { value: true });
  click(btn("Expand"));
  expect(dv.openCount).toBe(1);
  click(img); // the image zoom closes the figure view
  expect(dv.openCount).toBe(0);
  expect(zoom.isOpen).toBe(true);
  expect(document.querySelectorAll(".zen-zoom")).toHaveLength(1);
  expect(svg("a").closest(".zen-figure-view")).not.toBeNull(); // back in the page
  click(root.querySelectorAll<HTMLButtonElement>('button[aria-label="Expand"]')[0]); // and back the other way
  expect(zoom.isOpen).toBe(false);
  expect(document.querySelectorAll(".zen-zoom")).toHaveLength(1);
  // two figures: expanding the second closes the first
  key(document, "Escape");
  click(root.querySelectorAll<HTMLButtonElement>('button[aria-label="Expand"]')[0]);
  click(root.querySelectorAll<HTMLButtonElement>('button[aria-label="Expand"]')[1]);
  expect(document.querySelectorAll(".zen-zoom")).toHaveLength(1);
  expect(svg("a").closest(".zen-figure-view")).not.toBeNull();
  expect(svg("b").closest(".zen-figure-view")).toBeNull();
  zoom.destroy();
});

// ---- the page changes ----

test("destroy cleans up", () => {
  set(mermaid());
  const holder = root.querySelector(".zen-diagram-svg")!;
  click(btn("Expand"));
  dv.destroy();
  expect(zoomView()).toBeNull();
  expect(svg().parentElement).toBe(holder);
  expect(root.querySelector(".zen-figure-view")).toBeNull();
  expect(root.querySelector(".zen-figure-tools")).toBeNull();
  expect(svg().style.transform).toBe("");
  expect(svg().style.position).toBe("");
  const e = key(document, "Escape"); // no listener left
  expect(e.defaultPrevented).toBe(false);
  dv.destroy(); // twice is fine
});

test("a page replaced while expanded closes the view", async () => {
  set(mermaid());
  click(btn("Expand"));
  root.innerHTML = "<p>another page</p>";
  await frame();
  expect(zoomView()).toBeNull();
  expect(dv.openCount).toBe(0);
  expect(key(document, "Escape").defaultPrevented).toBe(false);
});

test("a new drawing gets fresh controls", async () => {
  set(mermaid());
  click(btn("Zoom in"));
  const holder = root.querySelector(".zen-diagram-svg")!;
  holder.innerHTML = svgOf("a2"); // a theme change draws it again into the same holder
  await frame();
  expect(root.querySelectorAll(".zen-figure-view")).toHaveLength(1);
  expect(root.querySelectorAll(".zen-figure-tools")).toHaveLength(1);
  expect(state(svg("a2"))).toEqual({ x: 0, y: 0, scale: 1 });
  expect(viewport().contains(svg("a2"))).toBe(true);
  // the whole figure replaced: the stale viewer is dropped
  root.innerHTML = mermaid("a3");
  await frame();
  expect(root.querySelectorAll(".zen-figure-view")).toHaveLength(1);
  expect(state(svg("a3"))).toEqual({ x: 0, y: 0, scale: 1 });
});

test("a mermaid diagram and a Lini figure on one page", () => {
  set(mermaid("a") + lini("l"));
  expect(root.querySelectorAll(".zen-figure-view")).toHaveLength(2);
  expect(viewport(1).querySelector(".zen-lini-svg")).not.toBeNull();
  expect(viewport(1).getAttribute("aria-label")).toBe("Flow");
});

test("resizing the window while zoomed keeps the zoom and refits", () => {
  set(mermaid());
  click(btn("Zoom in"));
  VW = 400;
  svg().setAttribute("data-test-w", "400");
  svg().setAttribute("data-test-h", "200");
  dv.refresh();
  expect(viewport().style.height).toBe("200px");
  expect(svg().style.width).toBe("400px");
  expect(state()!.scale).toBeCloseTo(1.25);
});

test("drags never click", () => {
  set(mermaid());
  let clicks = 0;
  svg("a").querySelector("g")!.addEventListener("click", () => clicks++);
  const g = svg("a").querySelector("g")!;
  click(btn("Zoom in")); click(btn("Zoom in"));
  drag(viewport(), 30, 0);
  click(g);
  expect(clicks).toBe(0); // the click that ends a drag is swallowed
  ptr(viewport(), "pointerdown", 300, 150);
  ptr(viewport(), "pointermove", 302, 151); // under 4px
  ptr(viewport(), "pointerup", 302, 151);
  click(g);
  expect(clicks).toBe(1);
  click(btn("Reset"));
  drag(viewport(), 10, 0); // at fit a drag past 4px is a selection: no click either
  click(g);
  expect(clicks).toBe(1);
  click(g);
  expect(clicks).toBe(2);
});

test("no innerHTML in the view", () => {
  expect(source).not.toMatch(/innerHTML|insertAdjacentHTML|outerHTML|document\.write/);
});

// ---- page.css ----

test("page.css: the figure view block", () => {
  expect(block).toMatch(/\.zen-figure-view\s*\{[^}]*overflow:\s*hidden/);
  expect(block).toMatch(/\.zen-figure-view\s*\{[^}]*position:\s*relative/);
});

test("page.css: the bar shows on hover, focus and touch", () => {
  expect(block).toMatch(/\.zen-figure-tools\.zen-zoom-bar\s*\{[^}]*visibility:\s*hidden/);
  expect(block).toMatch(/\.zen-figure-view:hover \.zen-figure-tools/);
  expect(block).toMatch(/\.zen-figure-view:focus-within \.zen-figure-tools/);
  expect(block).toMatch(/@media \(hover:\s*none\)\s*\{[^@]*\.zen-figure-tools\.zen-zoom-bar\s*\{[^}]*visibility:\s*visible/);
  expect(block).toMatch(/\.zen-figure-tools\.zen-zoom-bar\s*\{[^}]*top:/);
});

test("page.css: a Lini figure fits the column inside the view; the cursors; narrow windows", () => {
  expect(block).toMatch(/\.zen-figure-view \.zen-lini-svg\s*\{[^}]*overflow:\s*visible/);
  expect(block).toMatch(/\[data-zoomed\][^{]*\{[^}]*cursor:\s*grab/);
  expect(block).toMatch(/\[data-panning\][^{]*\{[^}]*cursor:\s*grabbing/);
  expect(block).toMatch(/@media \(max-width:\s*\d+px\)\s*\{[^@]*\.zen-figure-pan\s*\{[^}]*display:\s*none/);
});

test("page.css: print", () => {
  const print = css.slice(css.lastIndexOf("@media print"));
  expect(print).toMatch(/\.zen-figure-tools\s*\{[^}]*display:\s*none\s*!important/);
  expect(print).toMatch(/\.zen-figure-view svg\s*\{[^}]*transform:\s*none\s*!important/);
  expect(print).toMatch(/\.zen-figure-view\s*\{[^}]*height:\s*auto\s*!important/);
});

test("page.css: no motion, tokens only", () => {
  expect(block.length).toBeGreaterThan(100);
  expect(block).not.toMatch(/transition|animation|@keyframes/);
  expect(block).not.toMatch(/#[0-9a-f]{3,8}\b|rgb\(|hsl\(/i);
});
