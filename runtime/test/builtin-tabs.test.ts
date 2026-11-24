// The built-in tabs plugin's frontend (builtins/plugins/tabs/assets/tabs.js), mounted on the markup its Go
// block hooks emit, with the last synced choice kept through the fake storage api.
import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { afterEach, beforeEach, expect, test } from "vitest";
import { createFakeApi, type FakeApi } from "../../apps/zenmark/frontend/src/api/fake";
import { installRuntime } from "../src/plugins";

let api: FakeApi;
let rt: ReturnType<typeof installRuntime>;
let mod: unknown;
let root: HTMLElement;

beforeEach(async () => {
  document.body.replaceChildren();
  api = createFakeApi();
  rt = installRuntime({ importModule: async () => mod, api, navigate: () => {}, page: () => ({ path: "a.md", title: "A" }) });
  root = document.body.appendChild(document.createElement("article"));
  // @ts-ignore plain browser JavaScript outside src, no types
  mod = await import("../../builtins/plugins/tabs/assets/tabs.js");
  await rt.load({ js: ["/_plugins/tabs/tabs.js?v=test"], css: [] });
});
afterEach(() => rt.destroy());

const settle = async () => { for (let i = 0; i < 8; i++) await new Promise((r) => setTimeout(r, 0)); };

type Item = { title: string; body?: string; sync?: string; selected?: boolean };

/** The markup Go emits for a tab set: the wrapper of each plugin block around the framed content. */
function setHTML(items: Item[], attrs: Record<string, string> = {}): string {
  const esc = (s: string) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/"/g, "&quot;");
  const rows = items.map((it) => {
    const a: Record<string, string> = { title: it.title };
    if (it.sync) a.sync = it.sync;
    if (it.selected) a.selected = "true";
    return `<div data-zen-plugin="tabs" data-zen-name="tab-item" data-zen-attrs="${esc(JSON.stringify(a))}">` +
      `<div class="zen-tab-item${it.selected ? " zen-tab-selected" : ""}"><div class="zen-block-head zen-tab-title">${esc(it.title)}</div>` +
      `<div class="zen-block-body zen-tab-panel"><p>${esc(it.body ?? `Body of ${it.title}`)}</p></div></div></div>`;
  });
  return `<div data-zen-plugin="tabs" data-zen-name="tab-set" data-zen-attrs="${esc(JSON.stringify(attrs))}">` +
    `<div class="zen-block zen-tabs"><div class="zen-tabs-body">${rows.join("")}</div></div></div>`;
}
async function mount(...sets: string[]): Promise<HTMLElement[]> {
  root.innerHTML = sets.join("");
  rt.mountAll(root);
  await settle();
  return Array.from(root.querySelectorAll<HTMLElement>('[data-zen-name="tab-set"]'));
}
const tabs = (w: HTMLElement) => Array.from(w.querySelectorAll<HTMLElement>('[role="tab"]'));
const panels = (w: HTMLElement) => Array.from(w.querySelectorAll<HTMLElement>('[role="tabpanel"]'));
const key = (el: HTMLElement, k: string) => el.dispatchEvent(new KeyboardEvent("keydown", { key: k, bubbles: true, cancelable: true }));
const shown = (w: HTMLElement) => panels(w).map((p) => !p.hidden);
const selectedIdx = (w: HTMLElement) => tabs(w).findIndex((t) => t.getAttribute("aria-selected") === "true");
const writes = () => api.callsTo("pluginStorageSet").map((c) => ({ key: c[1] as string, value: JSON.parse(c[2] as string) }));

test("builds an ARIA tab list with one panel per item", async () => {
  const [w] = await mount(setHTML([{ title: "One" }, { title: "Two" }, { title: "Three" }]));
  const list = w.querySelector('[role="tablist"]')!;
  expect(list).not.toBeNull();
  expect(tabs(w).map((t) => t.textContent)).toEqual(["One", "Two", "Three"]);
  expect(tabs(w).every((t) => t.tagName === "BUTTON" && t.type === "button")).toBe(true);
  expect(panels(w)).toHaveLength(3);
  tabs(w).forEach((t, i) => {
    expect(t.getAttribute("aria-controls")).toBe(panels(w)[i].id);
    expect(panels(w)[i].getAttribute("aria-labelledby")).toBe(t.id);
    expect(t.id).not.toBe("");
  });
  expect(new Set([...tabs(w), ...panels(w)].map((e) => e.id)).size).toBe(6);
  expect(shown(w)).toEqual([true, false, false]);
  expect(selectedIdx(w)).toBe(0);
  expect(tabs(w).map((t) => t.tabIndex)).toEqual([0, -1, -1]);
  expect(w.querySelector(".zen-tabs")!.classList.contains("zen-tabs-js")).toBe(true);
});

test("the selected item starts open", async () => {
  const [w] = await mount(setHTML([{ title: "One" }, { title: "Two", selected: true }]));
  expect(selectedIdx(w)).toBe(1);
  expect(shown(w)).toEqual([false, true]);
});

test("a click opens a tab and hides the others", async () => {
  const [w] = await mount(setHTML([{ title: "One" }, { title: "Two" }]));
  tabs(w)[1].click();
  expect(shown(w)).toEqual([false, true]);
  expect(tabs(w).map((t) => t.tabIndex)).toEqual([-1, 0]);
  expect(selectedIdx(w)).toBe(1);
});

test("arrow keys move and wrap, Home and End jump, and focus follows", async () => {
  const [w] = await mount(setHTML([{ title: "One" }, { title: "Two" }, { title: "Three" }]));
  const t = tabs(w);
  t[0].focus();
  key(t[0], "ArrowRight");
  expect(selectedIdx(w)).toBe(1);
  expect(document.activeElement).toBe(t[1]);
  key(t[1], "ArrowRight");
  key(t[2], "ArrowRight");
  expect(selectedIdx(w)).toBe(0);
  key(t[0], "ArrowLeft");
  expect(selectedIdx(w)).toBe(2);
  key(t[2], "Home");
  expect(selectedIdx(w)).toBe(0);
  expect(document.activeElement).toBe(t[0]);
  key(t[0], "End");
  expect(selectedIdx(w)).toBe(2);
  expect(shown(w)).toEqual([false, false, true]);
  const down = new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true, cancelable: true });
  t[2].dispatchEvent(down);
  expect(down.defaultPrevented).toBe(false);
});

test("tabs with the same sync key move together across sets, and the choice is remembered", async () => {
  const [a, b, c] = await mount(
    setHTML([{ title: "Py", sync: "py" }, { title: "Go", sync: "go" }]),
    setHTML([{ title: "Python", sync: "py" }, { title: "Golang", sync: "go" }]),
    setHTML([{ title: "Other", sync: "x" }, { title: "Else", sync: "y" }]),
  );
  tabs(a)[1].click();
  await settle();
  expect(selectedIdx(a)).toBe(1);
  expect(selectedIdx(b)).toBe(1);
  expect(selectedIdx(c)).toBe(0);
  expect(writes().at(-1)!.value).toMatchObject({ version: 1, key: "go" });
  expect(writes().at(-1)!.key).toMatch(/^tabs-/);
  tabs(b)[0].click();
  expect(selectedIdx(a)).toBe(0);
});

test("sync groups keep their own choices", async () => {
  const [a, b] = await mount(
    setHTML([{ title: "A1", sync: "k" }, { title: "A2", sync: "j" }], { "sync-group": "one" }),
    setHTML([{ title: "B1", sync: "k" }, { title: "B2", sync: "j" }], { "sync-group": "two" }),
  );
  tabs(a)[1].click();
  expect(selectedIdx(a)).toBe(1);
  expect(selectedIdx(b)).toBe(0);
});

test("the last synced choice is read on mount; a key the set lacks is ignored", async () => {
  await mount(setHTML([{ title: "Py", sync: "py" }, { title: "Go", sync: "go" }]));
  const stored = writes();
  expect(stored).toHaveLength(0); // mounting alone writes nothing
  const [first] = await mount(setHTML([{ title: "Py", sync: "py" }, { title: "Go", sync: "go" }]));
  tabs(first)[1].click();
  await settle();
  const last = writes().at(-1)!;
  rt.unmountAll();
  const [w] = await mount(setHTML([{ title: "Py", sync: "py" }, { title: "Go", sync: "go" }]));
  await settle();
  expect(selectedIdx(w)).toBe(1);
  await api.pluginStorageSet("tabs", last.key, JSON.stringify({ version: 1, key: "missing" }));
  rt.unmountAll();
  const [z] = await mount(setHTML([{ title: "Py", sync: "py" }, { title: "Go", sync: "go" }]));
  await settle();
  expect(selectedIdx(z)).toBe(0);
});

test("a set without sync keys writes nothing", async () => {
  const [w] = await mount(setHTML([{ title: "One" }, { title: "Two" }]));
  tabs(w)[1].click();
  await settle();
  expect(writes()).toHaveLength(0);
});

test("titles are text, never markup", async () => {
  const [w] = await mount(setHTML([{ title: "<img src=x onerror=alert(1)>" }, { title: "Two" }]));
  expect(w.querySelector("img")).toBeNull();
  expect(tabs(w)[0].textContent).toBe("<img src=x onerror=alert(1)>");
});

test("unmounting puts the plain markup back", async () => {
  const [w] = await mount(setHTML([{ title: "One" }, { title: "Two" }]));
  rt.unmountAll();
  expect(w.querySelector('[role="tablist"]')).toBeNull();
  expect(panels(w)).toHaveLength(0);
  expect(w.querySelectorAll(".zen-tab-title")).toHaveLength(2);
  expect(w.querySelectorAll("[hidden]")).toHaveLength(0);
  expect(w.querySelector(".zen-tabs")!.classList.contains("zen-tabs-js")).toBe(false);
});

test("a set with no items is left alone", async () => {
  const [w] = await mount(setHTML([]));
  expect(w.querySelector('[role="tablist"]')).toBeNull();
});

test("an item alone is left alone", async () => {
  root.innerHTML = `<div data-zen-plugin="tabs" data-zen-name="tab-item" data-zen-attrs="{}"><div class="zen-tab-item"><div class="zen-tab-title">Solo</div><div class="zen-tab-panel">x</div></div></div>`;
  rt.mountAll(root);
  await settle();
  expect(root.querySelector('[role="tab"]')).toBeNull();
});

test("the stylesheet keeps the focus ring and uses tokens only", async () => {
  const css = readFileSync(join(dirname(fileURLToPath(import.meta.url)), "../../builtins/plugins/tabs/assets/tabs.css"), "utf8");
  expect(css).toMatch(/focus-visible/);
  expect(css).not.toMatch(/#[0-9a-f]{3,8}\b|rgba?\(|hsla?\(/i);
});
