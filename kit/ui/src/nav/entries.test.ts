import { expect, test, vi } from "vitest";
import type { NavNode } from "../host";
import { menuEntries, type EntryEnv } from "./entries";
import type { MenuEntry } from "./menu";

const p = (title: string, path: string, children: NavNode[] = []): NavNode => ({ title, path, kind: "page", children });
const g = (title: string, path: string, children: NavNode[]): NavNode => ({ title, path, kind: "group", children });
const tree: NavNode[] = [
  p("Home", "README.md", [p("Install", "guides/install.md"), p("Usage", "guides/usage.md"), p("Tips", "tips.md")]),
  g("Other pages", "", [g("notes", "notes", [p("Scratch", "notes/scratch.md")])]),
];

function env(o: Partial<EntryEnv> = {}): EntryEnv & Record<string, ReturnType<typeof vi.fn>> {
  const full = {
    tree, current: "", open: vi.fn(), editor: vi.fn(), newPage: vi.fn(), move: vi.fn(), copy: vi.fn(), reveal: vi.fn(), hide: vi.fn(),
  };
  return { ...full, ...o } as never;
}
const items = (es: MenuEntry[]) => es.filter((e): e is Exclude<MenuEntry, "separator"> => e !== "separator");
const labels = (es: MenuEntry[]) => es.map((e) => (e === "separator" ? "-" : e.label));
const find = (es: MenuEntry[], label: string) => items(es).find((e) => e.label === label)!;

test("a page has every item, in groups", () => {
  const es = menuEntries(tree[0].children[0], env());
  expect(labels(es)).toEqual([
    "Open", "Open in Editor", "-", "New Page Here…", "Move Up", "Move Down", "-", "Copy Path", "Copy Link", "Reveal in Finder", "-", "Hide from Sidebar",
  ]);
});

test("Open, Open in Editor and New Page Here use the page and its folder", () => {
  const e = env();
  const es = menuEntries(tree[0].children[0], e);
  find(es, "Open").run(); expect(e.open).toHaveBeenCalledWith("guides/install.md");
  find(es, "Open in Editor").run(); expect(e.editor).toHaveBeenCalledWith("guides/install.md");
  find(es, "New Page Here…").run(); expect(e.newPage).toHaveBeenCalledWith("guides/");
  find(es, "Reveal in Finder").run(); expect(e.reveal).toHaveBeenCalledWith("guides/install.md");
  find(es, "Hide from Sidebar").run(); expect(e.hide).toHaveBeenCalledWith(tree[0].children[0]);
});

test("Move Up is disabled for the first sibling and Move Down for the last; both run with a direction", () => {
  const e = env();
  const first = menuEntries(tree[0].children[0], e);
  expect(find(first, "Move Up").disabled).toBe(true);
  expect(find(first, "Move Down").disabled).toBeFalsy();
  find(first, "Move Down").run();
  expect(e.move).toHaveBeenCalledWith("guides/install.md", 1);
  const last = menuEntries(tree[0].children[2], e);
  expect(find(last, "Move Down").disabled).toBe(true);
  find(last, "Move Up").run();
  expect(e.move).toHaveBeenLastCalledWith("tips.md", -1);
  // An only child: both are disabled.
  const only = menuEntries(tree[0], e);
  expect(find(only, "Move Up").disabled).toBe(true);
  expect(find(only, "Move Down").disabled).toBe(true);
});

test("Copy Path is workspace-relative, Copy Link is a markdown link relative to the open page", () => {
  const e = env({ current: "guides/usage.md" });
  const es = menuEntries(tree[0].children[0], e);
  find(es, "Copy Path").run(); expect(e.copy).toHaveBeenLastCalledWith("guides/install.md");
  find(es, "Copy Link").run(); expect(e.copy).toHaveBeenLastCalledWith("[Install](install.md)");
  const none = env();
  find(menuEntries(tree[0].children[0], none), "Copy Link").run();
  expect(none.copy).toHaveBeenLastCalledWith("[Install](guides/install.md)");
});

test("items the host does not support are left out, and no separator is left at an end or doubled", () => {
  const es = menuEntries(tree[0].children[0], env({ editor: undefined, newPage: undefined, move: undefined, reveal: undefined, hide: undefined }));
  expect(labels(es)).toEqual(["Open", "-", "Copy Path", "Copy Link"]);
  const bare = menuEntries(tree[0].children[0], env({ editor: undefined, newPage: undefined, move: undefined, copy: undefined as never, reveal: undefined, hide: undefined }));
  expect(labels(bare)).toEqual(["Open"]);
  for (const drop of ["editor", "newPage", "move", "reveal", "hide"] as const) {
    const l = labels(menuEntries(tree[0].children[1], env({ [drop]: undefined })));
    expect(l[0]).not.toBe("-");
    expect(l[l.length - 1]).not.toBe("-");
    expect(l.join(",")).not.toContain("-,-");
  }
});

test("a folder group: New Page Here, Copy Path, Reveal; the Other pages group has no menu", () => {
  const notes = tree[1].children[0];
  const es = menuEntries(notes, env());
  expect(labels(es)).toEqual(["New Page Here…", "-", "Copy Path", "Reveal in Finder"]);
  const e = env();
  find(menuEntries(notes, e), "New Page Here…").run();
  expect(e.newPage).toHaveBeenCalledWith("notes/");
  expect(menuEntries(tree[1], env())).toEqual([]);
});

test("a glossary index has New Term…, which runs with its folder; other pages do not", () => {
  const gl = p("Glossary", "g/index.md", [p("Zebra", "g/zebra.md")]);
  const e = env({ newTerm: vi.fn(), glossary: (path: string) => path === "g/index.md" });
  const es = menuEntries(gl, e);
  expect(labels(es)).toContain("New Term…");
  find(es, "New Term…").run();
  expect(e.newTerm).toHaveBeenCalledWith("g");
  expect(labels(menuEntries(tree[0].children[0], e))).not.toContain("New Term…");
  // not offered by a host without it (a read-only app has no newTerm)
  expect(labels(menuEntries(gl, env({ newTerm: undefined, glossary: () => true })))).not.toContain("New Term…");
  expect(labels(menuEntries(gl, env()))).not.toContain("New Term…");
});

// ---- the Files view (ui/files-view): the same menuEntries, with the file kind and the page marks.
const f = (title: string, path: string, kind: "page" | "file" | "group" = "page"): NavNode => ({ title, path, kind: kind as NavNode["kind"], children: [] });
const filesTree: NavNode[] = [
  g("guides", "guides/", [f("Install", "guides/install.md"), f("notes.md", "guides/notes.md"), f("a b#1.png", "guides/img/a b#1.png", "file")]),
  f("Home", "README.md"),
];
const marks: Record<string, string> = { "guides/install.md": "tree", "guides/notes.md": "hidden", "README.md": "other" };
function fenv(o: Partial<EntryEnv> = {}) {
  return env({ tree: filesTree, view: "files", mark: (p: string) => marks[p], show: vi.fn(), ...o });
}

test("files page menu: no Move, New Page Here in its folder, Hide", () => {
  const e = fenv();
  const es = menuEntries(filesTree[0].children[0], e);
  expect(labels(es)).toEqual(["Open", "Open in Editor", "-", "New Page Here…", "-", "Copy Path", "Copy Link", "Reveal in Finder", "-", "Hide from Sidebar"]);
  find(es, "New Page Here…").run();
  expect(e.newPage).toHaveBeenCalledWith("guides/");
  // an "other" page can be hidden too
  expect(labels(menuEntries(filesTree[1], fenv()))).toContain("Hide from Sidebar");
});

test("hidden page has Show in Sidebar", () => {
  const e = fenv();
  const es = menuEntries(filesTree[0].children[1], e);
  expect(labels(es)).toContain("Show in Sidebar");
  expect(labels(es)).not.toContain("Hide from Sidebar");
  find(es, "Show in Sidebar").run();
  expect(e.show).toHaveBeenCalledWith("guides/notes.md");
  // in the Pages view the item is never there
  expect(labels(menuEntries(tree[0].children[0], env({ show: vi.fn() })))).not.toContain("Show in Sidebar");
});

test("folder menu", () => {
  const e = fenv();
  const es = menuEntries(filesTree[0], e);
  expect(labels(es)).toEqual(["New Page Here…", "-", "Copy Path", "Reveal in Finder"]);
  find(es, "New Page Here…").run();
  expect(e.newPage).toHaveBeenCalledWith("guides/");
  find(es, "Copy Path").run();
  expect(e.copy).toHaveBeenCalledWith("guides/");
});

test("image menu", () => {
  const e = fenv();
  const es = menuEntries(filesTree[0].children[2], e);
  expect(labels(es)).toEqual(["Open", "-", "Copy Path", "Copy Link", "Reveal in Finder"]);
  find(es, "Open").run();
  expect(e.open).toHaveBeenCalledWith("guides/img/a b#1.png");
});

test("image link is relative and encoded", () => {
  const img = filesTree[0].children[2];
  const e = fenv({ current: "guides/install.md" });
  find(menuEntries(img, e), "Copy Link").run();
  expect(e.copy).toHaveBeenLastCalledWith("![](img/a%20b%231.png)");
  const top = fenv({ current: "" });
  find(menuEntries(img, top), "Copy Link").run();
  expect(top.copy).toHaveBeenLastCalledWith("![](guides/img/a%20b%231.png)");
  const up = fenv({ current: "docs/deep/x.md" });
  find(menuEntries(f("p", "pics/(p).png", "file"), up), "Copy Link").run();
  expect(up.copy).toHaveBeenLastCalledWith("![](../../pics/%28p%29.png)");
});

test("read-only files menu: no editing items", () => {
  const ro = { editor: undefined, newPage: undefined, move: undefined, hide: undefined, show: undefined };
  expect(labels(menuEntries(filesTree[0].children[0], fenv(ro)))).toEqual(["Open", "-", "Copy Path", "Copy Link", "Reveal in Finder"]);
  expect(labels(menuEntries(filesTree[0].children[1], fenv(ro)))).toEqual(["Open", "-", "Copy Path", "Copy Link", "Reveal in Finder"]);
  expect(labels(menuEntries(filesTree[0], fenv(ro)))).toEqual(["Copy Path", "Reveal in Finder"]);
});
