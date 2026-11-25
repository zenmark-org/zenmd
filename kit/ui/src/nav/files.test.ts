// @vitest-environment happy-dom
import { beforeEach, expect, test, vi } from "vitest";
import { folderChain, imageSrc, toNavNodes, type FileTree } from "./files";
import { renderTree } from "./tree";

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

test("toNavNodes: folders become groups, pages pages, images the file kind; rows show file names", () => {
  const { tree, titles } = toNavNodes(specsTree());
  expect(tree.map((n) => [n.kind, n.path, n.title])).toEqual([
    ["group", "engine/", "engine"], ["group", "ui/", "ui"], ["page", "README.md", "README.md"], ["page", "ui.md", "ui.md"],
  ]);
  const shots = tree[1].children[0].children[0];
  expect(shots.kind).toBe("group");
  expect(shots.children[0]).toEqual({ title: "a b#1.png", path: "ui/files-view/shots/a b#1.png", kind: "file", children: [] });
  expect(tree[2].children).toEqual([]);
  expect(titles["ui/files-view/spec.md"]).toBe("Files view");
  expect(titles["ui/files-view/"]).toBe("files-view");
});

test("badges: other and hidden only", () => {
  const { badges } = toNavNodes(specsTree());
  expect(badges).toEqual({ "ui/files-view/notes.md": "hidden", "ui.md": "other" });
});

test("folderChain lists the folders above a path", () => {
  expect(folderChain("a/b/c.md")).toEqual(["a/", "a/b/"]);
  expect(folderChain("c.md")).toEqual([]);
  expect(folderChain("a/")).toEqual([]);
  expect(folderChain("a/b/")).toEqual(["a/"]);
});

test("image src is /_ws/ and encoded", () => {
  expect(imageSrc("ui/files-view/shots/a b#1.png")).toBe("/_ws/ui/files-view/shots/a%20b%231.png");
  expect(imageSrc("café/niño.png")).toBe("/_ws/caf%C3%A9/ni%C3%B1o.png");
  expect(imageSrc("a/b?.png")).toBe("/_ws/a/b%3F.png");
});

let host: HTMLElement;
beforeEach(() => { document.body.replaceChildren(); host = document.createElement("div"); document.body.append(host); });

test("names are text", () => {
  const evil = `<img src=x onerror="alert(1)">`;
  const t: FileTree = { omitted: 0, nodes: [
    { name: evil, title: evil, path: `${evil}/`, kind: "folder", children: [{ name: "p.md", title: evil, path: `${evil}/p.md`, kind: "page", nav: "tree" }] },
    { name: `${evil}.png`, title: `${evil}.png`, path: `${evil}.png`, kind: "image" },
  ] };
  const { tree, badges } = toNavNodes(t);
  renderTree(host, tree, { current: "", onOpen: vi.fn(), onToggle: vi.fn(), expanded: new Set([`${evil}/`]), badges });
  expect(host.querySelector("img")).toBeNull();
  expect(host.textContent).toContain(evil);
});
