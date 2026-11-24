// @vitest-environment happy-dom
// The sidebar of a read-only app (kit/features): no editing items, no reordering.
import { afterEach, beforeEach, expect, test, vi } from "vitest";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { createFakeApi, type FakeApi } from "../../../../apps/zenmark/frontend/src/api/fake";
import type { NavNode, Workspace } from "../host";
import { createStore } from "../shell/store";
import type { Ctx, Slots } from "../shell/types";
import { item } from "./fixtures";
import { navFeature } from "./index";
import type { FileTree } from "./files";

const p = (title: string, path: string, children: NavNode[] = []): NavNode => ({ title, path, kind: "page", children });
const ws = (): Workspace => ({
  root: "/docs", name: "docs", home: "README.md",
  tree: [p("Home", "README.md", [p("Alpha", "guides/alpha.md"), p("Beta", "guides/beta.md")])],
});

let unmount: (() => void) | undefined;
afterEach(() => { unmount?.(); unmount = undefined; });
beforeEach(() => { localStorage.clear(); document.body.replaceChildren(); });

function mount(api: FakeApi) {
  const store = createStore({ workspace: ws(), path: "README.md" });
  const ctx: Ctx = { api, store, brand: "Zenmark", navigate: (path) => store.set({ path }), newPage: () => {} };
  const slot = () => document.body.appendChild(document.createElement("div"));
  const slots: Slots = { nav: slot(), breadcrumb: slot(), actions: slot(), main: slot() };
  unmount = navFeature.mount(slots, ctx);
  return slots.nav;
}
const row = (nav: HTMLElement, path: string) => item(nav, path).querySelector<HTMLElement>(".zen-nav-row")!;
const labels = () => Array.from(document.querySelectorAll("[role=menuitem]")).map((i) => i.textContent);

test("read-only sidebar", () => {
  const api = createFakeApi({ build: { features: [] } });
  const nav = mount(api);
  row(nav, "guides/beta.md").dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, cancelable: true, clientX: 30, clientY: 40 }));
  // only what reads: no Open in Editor, New Page Here, Move Up/Down, Hide
  expect(labels()).toEqual(["Open", "Copy Path", "Copy Link", "Reveal in Finder"]);
  document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));

  // Alt+↓ does not move a page
  const target = row(nav, "guides/alpha.md");
  target.focus();
  item(nav, "guides/alpha.md").dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", altKey: true, bubbles: true, cancelable: true }));
  // dragging does not reorder: press, move far, release
  target.dispatchEvent(new MouseEvent("pointerdown", { bubbles: true, cancelable: true, clientX: 50, clientY: 10, button: 0 }));
  document.dispatchEvent(new MouseEvent("pointermove", { bubbles: true, clientX: 50, clientY: 80 }));
  expect(nav.querySelector(".zen-nav-dragging, .zen-nav-drop")).toBeNull();
  document.dispatchEvent(new MouseEvent("pointerup", { bubbles: true, clientX: 50, clientY: 80 }));
  expect(api.callsTo("reorderNav")).toHaveLength(0);
  expect(api.callsTo("setNavHidden")).toHaveLength(0);
});

test("with the editor the menu has every item", () => {
  const api = createFakeApi();
  const nav = mount(api);
  row(nav, "guides/beta.md").dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, cancelable: true, clientX: 30, clientY: 40 }));
  expect(labels()).toEqual(["Open", "Open in Editor", "New Page Here…", "Move Up", "Move Down", "Copy Path", "Copy Link", "Reveal in Finder", "Hide from Sidebar"]);
});

test("one heading", () => {
  const api = createFakeApi();
  const nav = mount(api);
  // the sidebar header (layout) is the only heading: the old folder row is gone, and so is its CSS
  expect(nav.querySelector(".zen-nav-ws, .zen-nav-ws-name")).toBeNull();
  expect(nav.textContent).not.toContain("docs");
  expect(readFileSync(resolve(__dirname, "nav.css"), "utf8")).not.toContain("zen-nav-ws");
  // the status under the tree is still there, below the tree
  const status = nav.querySelector<HTMLElement>(".zen-nav-status")!;
  expect(status.getAttribute("role")).toBe("status");
  expect(nav.querySelector(".zen-nav-tree")!.nextElementSibling).toBe(status);
});

// ---- the Files view (ui/files-view)
// The shape of the repo's own specs/ folder: scope folders holding feature folders with a spec.md, a README at
// the top, a page named like a folder, and images in a subfolder.
const specsTree = (): FileTree => ({
  omitted: 0,
  nodes: [
    { name: "engine", title: "engine", path: "engine/", kind: "folder", children: [
      { name: "index", title: "index", path: "engine/index/", kind: "folder", children: [
        { name: "spec.md", title: "Index", path: "engine/index/spec.md", kind: "page", nav: "tree" },
      ] },
    ] },
    { name: "ui", title: "ui", path: "ui/", kind: "folder", children: [
      { name: "files-view", title: "files-view", path: "ui/files-view/", kind: "folder", children: [
        { name: "shots", title: "shots", path: "ui/files-view/shots/", kind: "folder", children: [
          { name: "a b#1.png", title: "a b#1.png", path: "ui/files-view/shots/a b#1.png", kind: "image" },
        ] },
        { name: "spec.md", title: "Files view", path: "ui/files-view/spec.md", kind: "page", nav: "tree" },
        { name: "notes.md", title: "notes.md", path: "ui/files-view/notes.md", kind: "page", nav: "hidden" },
      ] },
    ] },
    { name: "README.md", title: "Specs", path: "README.md", kind: "page", nav: "tree" },
    { name: "ui.md", title: "ui", path: "ui.md", kind: "page", nav: "other" },
  ],
});
type FilesApi = FakeApi & { fileTree: ReturnType<typeof vi.fn> };
function filesApi(data: FileTree | Error = specsTree(), features?: string[]): FilesApi {
  const api = createFakeApi(features ? { build: { features } } : {}) as FilesApi;
  api.fileTree = vi.fn(() => (data instanceof Error ? Promise.reject(data) : Promise.resolve(structuredClone(data))));
  return api;
}
type Mounted = { nav: HTMLElement; store: ReturnType<typeof createStore>; zoom: ReturnType<typeof vi.fn>; navigate: ReturnType<typeof vi.fn> };
function mountFiles(api: FilesApi, workspace: Workspace = ws(), path = "README.md"): Mounted {
  const store = createStore({ workspace, path });
  const zoom = vi.fn();
  const navigate = vi.fn((p: string) => store.set({ path: p }));
  const ctx = { api, store, brand: "Zenmark", navigate, newPage: vi.fn(), zoomImage: zoom } as unknown as Ctx;
  const slot = () => document.body.appendChild(document.createElement("div"));
  const slots: Slots = { nav: slot(), breadcrumb: slot(), actions: slot(), main: slot() };
  unmount = navFeature.mount(slots, ctx);
  return { nav: slots.nav, store, zoom, navigate };
}
const tick = async () => { for (let i = 0; i < 4; i++) await Promise.resolve(); };
const switchTo = (nav: HTMLElement, name: "Pages" | "Files") =>
  Array.from(nav.querySelectorAll<HTMLElement>("[role=radio]")).find((r) => r.textContent === name)!.click();
const filesHost = (nav: HTMLElement) => nav.querySelector<HTMLElement>(".zen-nav-files")!;
const frow = (nav: HTMLElement, path: string) => item(filesHost(nav), path).querySelector<HTMLElement>(".zen-nav-row")!;
const fpaths = (nav: HTMLElement) => Array.from(filesHost(nav).querySelectorAll<HTMLElement>("[role=treeitem]")).map((i) => i.dataset.path);
const fvisible = (nav: HTMLElement) => Array.from(filesHost(nav).querySelectorAll<HTMLElement>("[role=treeitem]"))
  .filter((i) => !i.parentElement?.closest("[role=treeitem][aria-expanded=false]")).map((i) => i.dataset.path);

test("switch hidden in single mode", async () => {
  const api = filesApi();
  const single = mountFiles(api, { ...ws(), single: true, file: "README.md" });
  expect(single.nav.querySelector("[role=radiogroup]")!.closest("[hidden]")).not.toBeNull();
  await tick();
  expect(api.fileTree).not.toHaveBeenCalled();
  unmount?.(); document.body.replaceChildren();
  const many = mountFiles(filesApi());
  const group = many.nav.querySelector("[role=radiogroup]")!;
  expect(group.closest("[hidden]")).toBeNull();
  // before the tree in the tab order, Pages is chosen
  expect(group.compareDocumentPosition(many.nav.querySelector(".zen-nav-tree")!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  expect(group.querySelector("[aria-checked=true]")!.textContent).toBe("Pages");
});

test("the choice is remembered per folder", async () => {
  const a = mountFiles(filesApi());
  switchTo(a.nav, "Files");
  expect(localStorage.getItem("zen.nav.view:/docs")).toBe("files");
  unmount?.(); document.body.replaceChildren();
  const b = mountFiles(filesApi());
  await tick();
  expect(b.nav.querySelector("[role=radio][aria-checked=true]")!.textContent).toBe("Files");
  expect(filesHost(b.nav).hidden).toBe(false);
  unmount?.(); document.body.replaceChildren();
  const other = mountFiles(filesApi(), { ...ws(), root: "/elsewhere" });
  expect(other.nav.querySelector("[role=radio][aria-checked=true]")!.textContent).toBe("Pages");
});

test("files view uses the same tree: rows, folders collapsed except the top level, tree label", async () => {
  const m = mountFiles(filesApi());
  switchTo(m.nav, "Files");
  await tick();
  const host = filesHost(m.nav);
  expect(host.hidden).toBe(false);
  expect(m.nav.querySelector<HTMLElement>(".zen-nav-tree")!.hidden).toBe(true);
  expect(host.querySelector("[role=tree]")).not.toBeNull();
  expect(host.querySelector("[role=tree]")!.getAttribute("aria-label")).toBe("Files");
  // top-level folders are open, nested ones closed
  expect(item(host, "engine/").getAttribute("aria-expanded")).toBe("true");
  expect(item(host, "engine/index/").getAttribute("aria-expanded")).toBe("false");
  expect(item(host, "ui/files-view/").getAttribute("aria-expanded")).toBe("false");
  expect(fvisible(m.nav)).toEqual(["engine/", "engine/index/", "ui/", "ui/files-view/", "README.md", "ui.md"]);
  // images are announced as images
  expect(item(host, "ui/files-view/shots/a b#1.png").getAttribute("aria-label")).toBe("a b#1.png, image");
});

test("files view expands the open page's folders", async () => {
  const m = mountFiles(filesApi(), ws(), "ui/files-view/spec.md");
  switchTo(m.nav, "Files");
  await tick();
  expect(fvisible(m.nav)).toContain("ui/files-view/spec.md");
  expect(fvisible(m.nav)).not.toContain("engine/index/spec.md");
  expect(item(filesHost(m.nav), "ui/files-view/spec.md").getAttribute("aria-current")).toBe("page");
});

test("files expanded state is its own", async () => {
  const m = mountFiles(filesApi());
  switchTo(m.nav, "Files");
  await tick();
  frow(m.nav, "engine/index/").click();
  expect(JSON.parse(localStorage.getItem("zen.files.expanded:/docs")!)).toContain("engine/index/");
  expect(localStorage.getItem("zen.nav.expanded:/docs")).toBeNull();
  expect(item(filesHost(m.nav), "engine/index/").getAttribute("aria-expanded")).toBe("true");
  // folding the Pages tree does not touch the Files folders
  switchTo(m.nav, "Pages");
  row(m.nav, "README.md").click();
  expect(JSON.parse(localStorage.getItem("zen.files.expanded:/docs")!)).toContain("engine/index/");
  switchTo(m.nav, "Files");
  expect(item(filesHost(m.nav), "engine/index/").getAttribute("aria-expanded")).toBe("true");
});

test("badge tooltip", async () => {
  const m = mountFiles(filesApi(), ws(), "ui/files-view/spec.md");
  switchTo(m.nav, "Files");
  await tick();
  const badge = (p: string) => item(filesHost(m.nav), p).querySelector<HTMLElement>(".zen-nav-badge")!;
  expect(badge("ui.md").textContent).toBe("other");
  expect(badge("ui.md").title).toBe("Under Other pages in the Pages view");
  expect(badge("ui/files-view/notes.md").textContent).toBe("hidden");
  expect(badge("ui/files-view/notes.md").title).toBe("Hidden from the Pages view");
  expect(item(filesHost(m.nav), "README.md").querySelector(".zen-nav-badge")).toBeNull();
});

test("files view opens pages", async () => {
  const m = mountFiles(filesApi());
  switchTo(m.nav, "Files");
  await tick();
  frow(m.nav, "ui.md").click();
  expect(m.navigate).toHaveBeenCalledWith("ui.md");
  expect(item(filesHost(m.nav), "ui.md").getAttribute("aria-current")).toBe("page");
  expect(m.zoom).not.toHaveBeenCalled();
});

test("opens images in the zoom", async () => {
  const m = mountFiles(filesApi(), ws(), "ui/files-view/spec.md");
  switchTo(m.nav, "Files");
  await tick();
  const path = "ui/files-view/shots/a b#1.png";
  // a click on the folder first: it opens, then the image row opens the zoom, not the page navigation
  frow(m.nav, "ui/files-view/shots/").click();
  frow(m.nav, path).click();
  expect(m.zoom).toHaveBeenCalledTimes(1);
  expect(m.zoom.mock.calls[0][0]).toBe("/_ws/ui/files-view/shots/a%20b%231.png");
  expect(m.zoom.mock.calls[0][1]).toBe("a b#1.png");
  expect(m.navigate).not.toHaveBeenCalled();
  // the row has the focus, so Esc in the zoom gives it back there
  expect(document.activeElement).toBe(frow(m.nav, path));
  // Enter on the row does the same
  frow(m.nav, path).dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }));
  expect(m.zoom).toHaveBeenCalledTimes(2);
});

test("folder row toggles", async () => {
  const m = mountFiles(filesApi());
  switchTo(m.nav, "Files");
  await tick();
  expect(item(filesHost(m.nav), "ui/").getAttribute("aria-expanded")).toBe("true");
  frow(m.nav, "ui/").click();
  expect(item(filesHost(m.nav), "ui/").getAttribute("aria-expanded")).toBe("false");
  frow(m.nav, "ui/").dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true }));
  expect(item(filesHost(m.nav), "ui/").getAttribute("aria-expanded")).toBe("true");
  expect(m.navigate).not.toHaveBeenCalled();
});

test("not fetched while Pages is shown", async () => {
  const api = filesApi();
  const m = mountFiles(api);
  await tick();
  m.store.set({ workspace: { ...ws() } });
  await tick();
  expect(api.fileTree).not.toHaveBeenCalled();
  switchTo(m.nav, "Files");
  expect(api.fileTree).toHaveBeenCalledTimes(1);
});

test("refetches when shown", async () => {
  const api = filesApi();
  const m = mountFiles(api);
  switchTo(m.nav, "Files");
  await tick();
  expect(api.fileTree).toHaveBeenCalledTimes(1);
  const next: FileTree = { omitted: 0, nodes: [{ name: "new.md", title: "New", path: "new.md", kind: "page", nav: "tree" }] };
  api.fileTree.mockImplementation(() => Promise.resolve(next));
  m.store.set({ workspace: { ...ws() } }); // what workspace:changed does
  await tick();
  expect(api.fileTree).toHaveBeenCalledTimes(2);
  expect(fpaths(m.nav)).toEqual(["new.md"]);
});

test("keeps focus", async () => {
  const api = filesApi();
  const m = mountFiles(api);
  switchTo(m.nav, "Files");
  await tick();
  frow(m.nav, "ui.md").focus();
  frow(m.nav, "engine/index/").click();
  frow(m.nav, "ui.md").focus();
  m.store.set({ workspace: { ...ws() } });
  await tick();
  expect(document.activeElement).toBe(frow(m.nav, "ui.md"));
  expect(item(filesHost(m.nav), "engine/index/").getAttribute("aria-expanded")).toBe("true");
});

test("fetch error shows in status", async () => {
  const api = filesApi();
  const m = mountFiles(api);
  switchTo(m.nav, "Files");
  await tick();
  api.fileTree.mockImplementation(() => Promise.reject(new Error("disk went away")));
  m.store.set({ workspace: { ...ws() } });
  await tick();
  const status = m.nav.querySelector<HTMLElement>(".zen-nav-status")!;
  expect(status.getAttribute("role")).toBe("alert");
  expect(status.textContent).toBe("disk went away");
  expect(status.hidden).toBe(false);
  // the last listing is kept
  expect(fpaths(m.nav)).toContain("ui.md");
});

test("files loading", async () => {
  const api = filesApi();
  let resolve!: (t: FileTree) => void;
  api.fileTree.mockImplementation(() => new Promise<FileTree>((r) => { resolve = r; }));
  const m = mountFiles(api);
  switchTo(m.nav, "Files");
  const note = () => filesHost(m.nav).querySelector<HTMLElement>(".zen-nav-empty")!;
  expect(note().textContent).toBe("Loading files…");
  resolve(specsTree());
  await tick();
  expect(filesHost(m.nav).querySelector(".zen-nav-empty")).toBeNull();
});

test("files empty", async () => {
  const m = mountFiles(filesApi({ nodes: [], omitted: 0 }));
  switchTo(m.nav, "Files");
  await tick();
  expect(filesHost(m.nav).querySelector(".zen-nav-empty")!.textContent).toBe("No pages or images in this folder");
  expect(filesHost(m.nav).querySelector("[role=tree]")).toBeNull();
});

test("files omitted line", async () => {
  const data = specsTree();
  data.omitted = 12;
  const m = mountFiles(filesApi(data));
  switchTo(m.nav, "Files");
  await tick();
  const last = filesHost(m.nav).lastElementChild as HTMLElement;
  expect(last.textContent).toBe("12 more files not shown");
  expect(last.classList.contains("zen-nav-empty")).toBe(true);
  expect(last.getAttribute("role")).toBeNull();
  expect(last.tabIndex).toBe(-1);
  expect(filesHost(m.nav).querySelector("[role=tree]")).not.toBeNull();
});

test("files context menu: page, hidden page, no Move; Show in Sidebar", async () => {
  const api = filesApi();
  const m = mountFiles(api);
  switchTo(m.nav, "Files");
  await tick();
  frow(m.nav, "ui/files-view/").click();
  const ctxmenu = (p: string) => frow(m.nav, p).dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, cancelable: true, clientX: 30, clientY: 40 }));
  ctxmenu("ui/files-view/spec.md");
  expect(labels()).toEqual(["Open", "Open in Editor", "New Page Here…", "Copy Path", "Copy Link", "Reveal in Finder", "Hide from Sidebar"]);
  document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
  ctxmenu("ui/files-view/notes.md");
  expect(labels()).toContain("Show in Sidebar");
  (Array.from(document.querySelectorAll<HTMLElement>("[role=menuitem]")).find((i) => i.textContent === "Show in Sidebar"))!.click();
  expect(api.callsTo("setNavHidden")).toEqual([["ui/files-view/notes.md", false]]);
  // Alt+↓ does nothing
  frow(m.nav, "ui.md").focus();
  item(filesHost(m.nav), "ui.md").dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", altKey: true, bubbles: true, cancelable: true }));
  expect(api.callsTo("reorderNav")).toHaveLength(0);
});

test("read-only build: the Files view has no editing items", async () => {
  const api = filesApi(specsTree(), []);
  const m = mountFiles(api);
  switchTo(m.nav, "Files");
  await tick();
  frow(m.nav, "ui.md").dispatchEvent(new MouseEvent("contextmenu", { bubbles: true, cancelable: true, clientX: 30, clientY: 40 }));
  expect(labels()).toEqual(["Open", "Copy Path", "Copy Link", "Reveal in Finder"]);
});
