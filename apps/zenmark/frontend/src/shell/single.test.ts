// @vitest-environment happy-dom
// The window of one markdown file on its own (app/single-file): the shell's one-file state.
import { afterEach, beforeEach, expect, test } from "vitest";
import { navFeature, pageFeature, type Workspace } from "@zenmd/kit";
import { createFakeApi, sampleWorkspace, type FakeApi } from "../api/fake";
import { createShell, type Shell } from "./shell";

const SOURCES = {
  "notes.md": "# Notes\n\nSee [B](b.md) and [up](../x.md) and [missing](gone.md).\n\n## Part\n",
  "b.md": "# B\n\nBack to [notes](notes.md).\n",
};
const HEADINGS: Record<string, string> = {
  "notes.md": '<h1 id="notes">Notes</h1><p><a data-zen-link="internal" data-path="b.md" data-anchor="" href="#">B</a></p><h2 id="part">Part</h2>',
  "b.md": '<h1 id="b">B</h1><p><a data-zen-link="internal" data-path="notes.md" data-anchor="" href="#">notes</a></p>',
};
const oneFile = (): Workspace => ({
  root: "/home/me/Downloads", name: "notes.md", home: "notes.md", title: "Notes", logo: "", single: true, file: "notes.md",
  tree: [{ title: "Notes", path: "notes.md", kind: "page", children: [] }],
});

let root: HTMLElement;
let shell: Shell | undefined;
const tick = () => new Promise((r) => setTimeout(r, 0));

function api(workspace: Workspace | null = oneFile(), page = "notes.md"): FakeApi {
  const a = createFakeApi({ sources: SOURCES, startup: { workspace, page, mode: "read" } });
  a.decorate = (p) => ({ ...p, html: HEADINGS[p.path] ?? p.html });
  return a;
}
async function boot(a: FakeApi): Promise<Shell> {
  shell = createShell(root, { api: a, features: [navFeature, pageFeature], search: "" });
  await shell.start();
  await tick();
  await tick();
  return shell;
}
const rows = () => Array.from(root.querySelectorAll<HTMLElement>(".zen-single-row"));
const key = (k: string, mods: KeyboardEventInit = { metaKey: true }) =>
  document.dispatchEvent(new KeyboardEvent("keydown", { key: k, bubbles: true, cancelable: true, ...mods }));

beforeEach(() => {
  localStorage.clear();
  sessionStorage.clear();
  root = document.createElement("div");
  document.body.replaceChildren(root);
});
afterEach(() => {
  shell?.destroy();
  shell = undefined;
});

test("a file opened alone: no tree, the outline, the file's name and Open Enclosing Folder", async () => {
  await boot(api());
  expect(root.querySelector('.zen-sidebar [role="tree"]')).toBeNull();
  expect(rows().map((r) => r.textContent)).toEqual(["Notes", "Part"]);
  expect(root.querySelector(".zen-sidebar-title")?.textContent).toBe("notes.md");
  expect(root.querySelector(".zen-breadcrumb .zen-crumbs")).toBeNull();
  const open = root.querySelector<HTMLButtonElement>(".zen-sidebar .zen-sidebar-btn.zen-single-folder");
  expect(open?.textContent).toBe("Open Enclosing Folder");
  // The close button says what it closes.
  expect(root.querySelector<HTMLElement>(".zen-sidebar-close")?.title).toBe("Close File ⇧⌘W");
});

test("a folder keeps its tree", async () => {
  await boot(api(sampleWorkspace(), "README.md"));
  expect(root.querySelector(".zen-sidebar [role=tree]")).not.toBeNull();
  expect(rows()).toEqual([]);
  expect(root.querySelector<HTMLElement>(".zen-sidebar-close")?.title).toBe("Close Folder ⇧⌘W");
});

test("Open Enclosing Folder asks Go", async () => {
  const a = api();
  await boot(a);
  root.querySelector<HTMLButtonElement>(".zen-single-folder")!.click();
  expect(a.callsTo("openEnclosingFolder")).toEqual([[]]);
});

test("the window's title is the page's title, then the file's name; it follows a link", async () => {
  await boot(api());
  expect(document.title).toBe("Notes — notes.md");
  shell!.navigate("b.md");
  await tick();
  await tick();
  expect(document.title).toBe("B — b.md");
  expect(root.querySelector(".zen-sidebar-title")?.textContent).toBe("b.md");
  expect(rows().map((r) => r.textContent)).toEqual(["B"]);
});

test("a link to another file opens it in the same window; back and forward work across them", async () => {
  const a = api();
  await boot(a);
  root.querySelector<HTMLAnchorElement>('a[data-path="b.md"]')!.click();
  await tick();
  await tick();
  expect(shell!.store.get().path).toBe("b.md");
  expect(shell!.store.get().workspace?.single).toBe(true);
  expect(a.callsTo("startup").length).toBe(1); // no new window, no new open
  shell!.back();
  await tick();
  expect(shell!.store.get().path).toBe("notes.md");
  shell!.forward();
  await tick();
  expect(shell!.store.get().path).toBe("b.md");
  // A broken link (outside the folder, or missing) does nothing.
  shell!.navigate("notes.md");
  await tick();
  await tick();
  const before = shell!.store.get().path;
  const broken = document.createElement("a");
  broken.setAttribute("data-zen-link", "broken");
  root.querySelector(".zen-page")!.append(broken);
  broken.click();
  expect(shell!.store.get().path).toBe(before);
});

test("what needs a folder is off, with a tip: New Page, New from Blueprint, folder settings, Export › Website", async () => {
  const a = api();
  await boot(a);
  // ⌘N and ⇧⌘N do nothing in a one-file window.
  key("n");
  key("n", { metaKey: true, shiftKey: true });
  await tick();
  expect(document.querySelector(".zen-newpage, .zen-newsite, dialog")).toBeNull();
  // Settings: This folder's file and the Website export are greyed with the tip; the PDF export works.
  root.querySelector<HTMLButtonElement>(".zen-settings-button[aria-expanded]")!.click();
  await tick();
  await tick();
  const row = (id: string) => root.querySelector<HTMLElement>(`[role=option][data-item="${id}"]`);
  const config = row("workspace")!;
  expect(config.getAttribute("aria-disabled")).toBe("true");
  expect(config.title).toBe("Open a folder to use this");
  root.querySelector<HTMLButtonElement>('.zen-settings-section[data-section="export"]')!.click();
  await tick();
  await tick();
  expect(row("website")!.getAttribute("aria-disabled")).toBe("true");
  expect(row("website")!.title).toBe("Open a folder to use this");
  expect(row("pdf")!.getAttribute("aria-disabled")).toBeNull();
  expect(row("pdf")!.getAttribute("aria-selected")).toBe("true"); // the first item that works is the one shown
});

test("workspace:opened with a file switches a folder's window to the file alone, and back", async () => {
  const a = api(sampleWorkspace(), "README.md");
  await boot(a);
  expect(rows()).toEqual([]);
  a.emit("workspace:opened", { workspace: oneFile(), page: "notes.md", mode: "read" });
  await tick();
  await tick();
  expect(rows().map((r) => r.textContent)).toEqual(["Notes", "Part"]);
  expect(root.querySelector(".zen-sidebar [role=tree]")).toBeNull();
  a.emit("workspace:opened", { workspace: sampleWorkspace(), page: "README.md", mode: "read" });
  await tick();
  await tick();
  expect(rows()).toEqual([]);
  expect(root.querySelector(".zen-sidebar [role=tree]")).not.toBeNull();
});

test("an outside edit that changes the title keeps the window one-file", async () => {
  const a = api();
  await boot(a);
  a.emit("workspace:changed", { ...oneFile(), title: "Edited" });
  await tick();
  expect(shell!.store.get().workspace?.single).toBe(true);
  expect(root.querySelector(".zen-sidebar-title")?.textContent).toBe("notes.md");
});

test("the welcome hint names Open…", async () => {
  await boot(createFakeApi());
  const hint = root.querySelector(".zen-welcome-menuhint")!.textContent!;
  expect(hint).toContain("Open…");
  expect(hint).not.toContain("Open Folder");
});
