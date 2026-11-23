// @vitest-environment happy-dom
// One markdown file on its own (app/single-file): the sidebar shows its outline in place of the tree.
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { createFakeApi } from "../../../../apps/zenmark/frontend/src/api/fake";
import type { Page, Workspace } from "../host";
import { createSettingsFrame } from "./settings";
import { FOLDER_ONLY_TIP, fileName, isSingle, mountSingleSidebar, outlineOf, windowTitle } from "./single";
import { createStore } from "./store";
import type { Ctx } from "./types";

const HTML = '<h1 id="notes">Notes</h1><p>x</p><h2 id="first">First part</h2><h3 id="deep">Deeper</h3><h2 id="second">Second &amp; last</h2>';
const single = (file = "notes.md"): Workspace => ({
  root: "/home/me/Downloads", name: file, home: file, single: true, file,
  tree: [{ title: "Notes", path: file, kind: "page", children: [] }],
});
const page = (path: string, html = HTML): Page => ({
  path, title: "Notes", html, frontmatter: {}, breadcrumb: [], assets: { css: [], js: [] }, warnings: [], layout: { name: "article", source: "" },
});

let host: HTMLElement;
let stop: (() => void) | undefined;
let navigated: [string, string][];
let enclosing: number;
beforeEach(() => {
  document.body.replaceChildren();
  host = document.body.appendChild(document.createElement("div"));
  navigated = [];
  enclosing = 0;
});
afterEach(() => { stop?.(); stop = undefined; });

function mount(html = HTML) {
  const store = createStore({ workspace: single(), path: "notes.md", anchor: "" });
  if (html !== null) store.set({ page: page("notes.md", html) });
  const ctx: Ctx = { api: createFakeApi(), store, brand: "Zenmark", navigate: (p, a = "") => { navigated.push([p, a]); store.set({ path: p, anchor: a }); } };
  stop = mountSingleSidebar(host, { ctx, openEnclosing: () => { enclosing++; } });
  return { store, ctx };
}
const rows = () => Array.from(host.querySelectorAll<HTMLElement>(".zen-single-row"));
const button = () => host.querySelector<HTMLButtonElement>("button.zen-sidebar-btn")!;

test("the one-file state is told by the workspace", () => {
  expect(isSingle(single())).toBe(true);
  expect(isSingle({ ...single(), single: false })).toBe(false);
  const folder: Workspace = { root: "/x", name: "x", home: "", tree: [] };
  expect(isSingle(folder)).toBe(false);
  expect(isSingle(null)).toBe(false);
});

test("the outline is the page's headings, in order, with their ids and levels", () => {
  expect(outlineOf(HTML)).toEqual([
    { level: 1, text: "Notes", id: "notes" },
    { level: 2, text: "First part", id: "first" },
    { level: 3, text: "Deeper", id: "deep" },
    { level: 2, text: "Second & last", id: "second" },
  ]);
  expect(outlineOf("<p>none</p>")).toEqual([]);
  expect(outlineOf("")).toEqual([]);
  // A heading with no id cannot be scrolled to; it is left out. Scripts in the html never run.
  expect(outlineOf('<h2>No id</h2><h2 id="a">A</h2><script>window.__ran = 1</script>')).toEqual([{ level: 2, text: "A", id: "a" }]);
  expect((window as unknown as { __ran?: number }).__ran).toBeUndefined();
});

test("the sidebar shows the outline in place of the tree, and Open Enclosing Folder", () => {
  mount();
  expect(host.querySelector('[role="tree"], [role="treeitem"]')).toBeNull();
  expect(rows().map((r) => r.textContent)).toEqual(["Notes", "First part", "Deeper", "Second & last"]);
  // The panel column's own list rows, the same style as the tree's.
  expect(rows().every((r) => r.classList.contains("zen-nav-row"))).toBe(true);
  expect(host.querySelector("ul.zen-nav-list")?.getAttribute("aria-label")).toBe("Outline");
  // Levels indent, from the shallowest heading.
  expect(rows().map((r) => r.style.paddingLeft)).toEqual([
    "calc(var(--space-2) + 0px)", "calc(var(--space-2) + 14px)", "calc(var(--space-2) + 28px)", "calc(var(--space-2) + 14px)",
  ]);
  expect(button().textContent).toBe("Open Enclosing Folder");
  expect(button().classList.contains("ghost")).toBe(true);
  button().click();
  expect(enclosing).toBe(1);
});

test("a page with no headings says No headings, in the column where the rows would be", () => {
  mount("<p>just text</p>");
  expect(rows()).toEqual([]);
  expect(host.querySelector(".zen-nav-empty")?.textContent).toBe("No headings");
  expect(button()).not.toBeNull();
  // Before the page has loaded there is nothing to say yet.
});

test("a heading scrolls the page to it, and the current one is marked", () => {
  const { store } = mount();
  const second = rows()[3];
  second.click();
  expect(navigated).toEqual([["notes.md", "second"]]);
  expect(store.get().anchor).toBe("second");
  expect(rows()[3].parentElement?.getAttribute("aria-current")).toBe("location");
  expect(rows().filter((r) => r.parentElement?.hasAttribute("aria-current"))).toHaveLength(1);
});

test("it follows the page: a link to a sibling file shows that page's outline", () => {
  const { store } = mount();
  store.set({ path: "b.md", anchor: "", page: page("b.md", '<h1 id="b">B</h1><h2 id="b2">B two</h2>') });
  expect(rows().map((r) => r.textContent)).toEqual(["B", "B two"]);
  // A page that is not the one shown yet (a load in flight) does not draw the old outline under the new name.
  store.set({ path: "c.md" });
  expect(rows()).toEqual([]);
  expect(host.querySelector(".zen-nav-empty")).toBeNull();
});

test("keyboard: Tab reaches the outline once, the arrows move through it", () => {
  mount();
  expect(rows().map((r) => r.tabIndex)).toEqual([0, -1, -1, -1]);
  rows()[0].focus();
  rows()[0].dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true, cancelable: true }));
  expect(document.activeElement).toBe(rows()[1]);
  expect(rows().map((r) => r.tabIndex)).toEqual([-1, 0, -1, -1]);
  rows()[1].dispatchEvent(new KeyboardEvent("keydown", { key: "End", bubbles: true, cancelable: true }));
  expect(document.activeElement).toBe(rows()[3]);
  rows()[3].dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowUp", bubbles: true, cancelable: true }));
  expect(document.activeElement).toBe(rows()[2]);
  rows()[2].dispatchEvent(new KeyboardEvent("keydown", { key: "Home", bubbles: true, cancelable: true }));
  expect(document.activeElement).toBe(rows()[0]);
  // Enter or Space on a row is a click on a button: it navigates.
  rows()[1].click();
  expect(navigated.at(-1)).toEqual(["notes.md", "first"]);
});

test("destroy leaves nothing behind", () => {
  mount();
  stop?.();
  stop = undefined;
  expect(host.children).toHaveLength(0);
});

test("the window title is the page's title, then the file's name", () => {
  expect(windowTitle("Notes", "notes.md")).toBe("Notes — notes.md");
  expect(windowTitle("B", "sub/b.md")).toBe("B — b.md");
  expect(fileName("a/b/c.md")).toBe("c.md");
  expect(fileName("c.md")).toBe("c.md");
});

test("a greyed item says why: the Settings frame takes a tip for a disabled row", async () => {
  const frame = createSettingsFrame([{
    id: "config", label: "Configuration",
    items: () => [{ id: "global", label: "config.yml" }, { id: "workspace", label: ".zenmd.yml", disabled: true, tip: FOLDER_ONLY_TIP }],
    render: () => {},
  }]);
  document.body.append(frame.el);
  frame.show();
  await new Promise((r) => setTimeout(r, 0));
  const row = (id: string) => frame.el.querySelector<HTMLElement>(`[role=option][data-item="${id}"]`)!;
  expect(FOLDER_ONLY_TIP).toBe("Open a folder to use this");
  expect(row("workspace").getAttribute("aria-disabled")).toBe("true");
  expect(row("workspace").title).toBe("Open a folder to use this");
  expect(row("global").title).toBe("");
  frame.destroy();
});

test("black and white tokens only", () => {
  const css = readFileSync(resolve(__dirname, "single.css"), "utf8");
  expect(css).not.toMatch(/#[0-9a-f]{3,8}\b|rgba?\(|hsla?\(/i);
  expect(css).toMatch(/var\(--/);
  expect(css).not.toMatch(/transition|animation/); // nothing moves, so prefers-reduced-motion has nothing to respect
});

vi.setConfig({ testTimeout: 10000 });
