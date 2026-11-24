// The Files view of the sidebar: the folder as it is on disk, drawn by the same renderTree as Pages.
// toNavNodes turns the engine's listing into NavNodes (folders are groups, images the "file" kind) and the badges.
import type { NavNode } from "../host";

// The listing's types (engine/session FileNode and FileTree). They belong in host.ts with `Api.fileTree`
// and `NavNode.kind` "file" (Integration); until then they are declared here.
export type FileNode = {
  name: string; title: string; path: string;
  kind: "folder" | "page" | "image";
  nav?: "tree" | "other" | "hidden";
  children?: FileNode[];
};
export type FileTree = { nodes: FileNode[]; omitted: number };

/** What the badge of a marked page says when the pointer rests on it. */
export const MARK_TIPS: Record<string, string> = {
  other: "Under Other pages in the Pages view",
  hidden: "Hidden from the Pages view",
};

type NavKind = NavNode["kind"] | "file";

/** The listing as tree rows: folders are groups (their path ends in "/"), pages pages, images "file" rows. */
export function toNavNodes(t: FileTree): { tree: NavNode[]; badges: Record<string, string>; titles: Record<string, string>; marks: Record<string, string> } {
  const badges: Record<string, string> = {};
  const titles: Record<string, string> = {};
  const marks: Record<string, string> = {};
  const conv = (nodes: FileNode[]): NavNode[] => nodes.map((n) => {
    const kind: NavKind = n.kind === "folder" ? "group" : n.kind === "image" ? "file" : "page";
    const title = n.kind === "page" ? n.title || n.name : n.name;
    titles[n.path] = title;
    if (n.nav) marks[n.path] = n.nav;
    if (n.nav === "other" || n.nav === "hidden") badges[n.path] = n.nav;
    return { title, path: n.path, kind: kind as NavNode["kind"], children: conv(n.children ?? []) };
  });
  return { tree: conv(t.nodes), badges, titles, marks };
}

/** The folders above a path, outermost first: "a/b/c.md" gives ["a/", "a/b/"]. */
export function folderChain(path: string): string[] {
  const parts = path.replace(/\/$/, "").split("/").slice(0, -1);
  return parts.map((_, i) => parts.slice(0, i + 1).join("/") + "/");
}

/** The folders that start open: the top level ones. */
export function topFolders(tree: NavNode[]): Set<string> {
  return new Set(tree.filter((n) => n.kind === "group").map((n) => n.path));
}

/** Where an image of the folder is served: only listed paths go in, each segment encoded. */
export function imageSrc(path: string): string {
  return "/_ws/" + path.split("/").map(encodeURIComponent).join("/");
}

/**
 * What renderTree has no place for: the badges' tooltips, and the images announced as images.
 * Text only goes in as properties.
 */
export function decorateFiles(host: HTMLElement, marks: Record<string, string>): void {
  host.querySelector("[role=tree]")?.setAttribute("aria-label", "Files");
  for (const li of Array.from(host.querySelectorAll<HTMLElement>("[role=treeitem]"))) {
    const path = li.dataset.path ?? "";
    const badge = li.querySelector<HTMLElement>(":scope > .zen-nav-row > .zen-nav-badge");
    if (badge && MARK_TIPS[marks[path]]) badge.title = MARK_TIPS[marks[path]];
    if (li.dataset.kind === "file") {
      const label = li.querySelector(":scope > .zen-nav-row > .zen-nav-label")?.textContent ?? path;
      li.setAttribute("aria-label", `${label}, image`);
    }
  }
}
