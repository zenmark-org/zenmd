// The built-in "glossary" layout (builtins/layouts/glossary/glossary.js): the filter and the letter bar.
import { afterEach, beforeEach, expect, test } from "vitest";
import { createFakeApi } from "../../apps/zenmark/frontend/src/api/fake";
import { installRuntime } from "../src/plugins";

const scripts = import.meta.glob<unknown>("../../builtins/layouts/glossary/glossary.js");
let rt: ReturnType<typeof installRuntime>;
let mod: unknown;
let root: HTMLElement;

beforeEach(async () => {
  document.body.replaceChildren();
  rt = installRuntime({ importModule: async () => mod, api: createFakeApi(), navigate: () => {}, page: () => ({ path: "g/index.md", title: "Glossary" }) });
  root = document.body.appendChild(document.createElement("article"));
  mod = await Object.values(scripts)[0]();
  await rt.load({ js: ["/_layouts/glossary/glossary.js?v=test"], css: [] });
});
afterEach(() => rt.destroy());

const entry = (term: string, def: string, aliases = "") =>
  `<li class="zl-gloss-entry"><a data-zen-link="internal" class="zl-gloss-term">${term}</a>${aliases ? `<span class="zl-gloss-aliases">also ${aliases}</span>` : ""}<p class="zl-gloss-def">${def}</p></li>`;
const group = (id: string, letter: string, ...entries: string[]) =>
  `<section class="zl-gloss-group"><h2 id="${id}">${letter}</h2><ul class="zl-gloss-list">${entries.join("")}</ul></section>`;
const letters = (on: string) =>
  "ABCDEFGHIJKLMNOPQRSTUVWXYZ#".split("").map((l) => on.includes(l) ? `<a href="#letter-${l.toLowerCase()}" class="zl-gloss-letter zen-btn">${l}</a>` : `<span class="zl-gloss-letter zl-gloss-off zen-btn">${l}</span>`).join("");
const PAGE = `<div class="zl-gloss"><header><h1>Glossary</h1><p class="zl-gloss-count">3 terms</p></header>
<div class="zl-gloss-tools"><nav class="zl-gloss-bar" aria-label="Letters">${letters("AEZ")}</nav></div>
${group("letter-a", "A", entry("Apple", "A fruit.", "malus"))}${group("letter-e", "E", entry("Éclair", "A cake."))}${group("letter-z", "Z", entry("Zebra", "A striped animal."))}</div>`;

async function render(html = PAGE): Promise<HTMLElement> {
  const w = document.createElement("div");
  w.setAttribute("data-zen-layout", "glossary");
  w.setAttribute("data-zen-layout-source", "builtin");
  w.innerHTML = html; // test fixture only
  root.replaceChildren(w);
  rt.mountAll(root);
  for (let i = 0; i < 5; i++) await new Promise((r) => setTimeout(r, 0));
  return w;
}
const filter = (w: HTMLElement) => w.querySelector<HTMLInputElement>("input.zl-gloss-filter")!;
const type = (w: HTMLElement, v: string) => { const f = filter(w); f.value = v; f.dispatchEvent(new Event("input", { bubbles: true })); };
const shown = (w: HTMLElement) => Array.from(w.querySelectorAll<HTMLElement>(".zl-gloss-term")).filter((a) => !a.closest("[hidden]")).map((a) => a.textContent);
const groupsShown = (w: HTMLElement) => Array.from(w.querySelectorAll<HTMLElement>(".zl-gloss-group")).filter((g) => !g.hidden).map((g) => g.querySelector("h2")!.textContent);
const count = (w: HTMLElement) => w.querySelector(".zl-gloss-count")!.textContent;
const key = (el: Element, k: string) => el.dispatchEvent(new KeyboardEvent("keydown", { key: k, bubbles: true, cancelable: true }));

test("a filter field is added with the plugin kit field style", async () => {
  const w = await render();
  const f = filter(w);
  expect(f).toBeTruthy();
  expect(f.classList.contains("zen-field")).toBe(true);
  expect(f.type).toBe("search");
  expect(f.getAttribute("aria-label")).toBe("Filter terms");
  expect(shown(w)).toEqual(["Apple", "Éclair", "Zebra"]);
});

test("typing narrows by term, alias and definition; case and accents are ignored", async () => {
  const w = await render();
  type(w, "ZEB");
  expect(shown(w)).toEqual(["Zebra"]);
  expect(groupsShown(w)).toEqual(["Z"]);
  expect(count(w)).toBe("1 of 3 terms");
  type(w, "malus"); // an alias
  expect(shown(w)).toEqual(["Apple"]);
  type(w, "cake"); // a definition
  expect(shown(w)).toEqual(["Éclair"]);
  type(w, "eclair"); // without the accent
  expect(shown(w)).toEqual(["Éclair"]);
});

test("the filter is plain text, never a regular expression", async () => {
  const w = await render();
  type(w, ".*");
  expect(shown(w)).toEqual([]);
  expect(w.querySelector<HTMLElement>(".zl-gloss-none")!.hidden).toBe(false);
  type(w, "");
  expect(shown(w)).toEqual(["Apple", "Éclair", "Zebra"]);
  expect(w.querySelector<HTMLElement>(".zl-gloss-none")!.hidden).toBe(true);
  expect(count(w)).toBe("3 terms");
});

test("Esc clears the filter and keeps the focus in it", async () => {
  const w = await render();
  filter(w).focus();
  type(w, "zeb");
  key(filter(w), "Escape");
  expect(filter(w).value).toBe("");
  expect(shown(w)).toHaveLength(3);
  expect(document.activeElement).toBe(filter(w));
});

test("the letter bar: arrows move along the letters that have terms, Home and End jump to the ends", async () => {
  const w = await render();
  const links = Array.from(w.querySelectorAll<HTMLAnchorElement>(".zl-gloss-bar a"));
  expect(links.map((a) => a.textContent)).toEqual(["A", "E", "Z"]);
  // One Tab stop: only the first letter is in the tab order.
  expect(links.map((a) => a.tabIndex)).toEqual([0, -1, -1]);
  links[0].focus();
  key(links[0], "ArrowRight");
  expect(document.activeElement).toBe(links[1]);
  expect(links.map((a) => a.tabIndex)).toEqual([-1, 0, -1]);
  key(links[1], "ArrowRight");
  expect(document.activeElement).toBe(links[2]);
  key(links[2], "ArrowRight"); // no wrap past the end
  expect(document.activeElement).toBe(links[2]);
  key(links[2], "ArrowLeft");
  expect(document.activeElement).toBe(links[1]);
  key(links[1], "Home");
  expect(document.activeElement).toBe(links[0]);
  key(links[0], "End");
  expect(document.activeElement).toBe(links[2]);
});

test("a page without terms gets no filter", async () => {
  const w = await render(`<div class="zl-gloss"><header><h1>Glossary</h1><p class="zl-gloss-count">0 terms</p></header><p class="zl-gloss-empty">No terms yet.</p></div>`);
  expect(filter(w)).toBeNull();
});

test("mounting twice does not add a second filter", async () => {
  const w = await render();
  rt.mountAll(root);
  expect(w.querySelectorAll("input.zl-gloss-filter")).toHaveLength(1);
});
