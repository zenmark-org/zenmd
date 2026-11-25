// The Files view of the sidebar: the folder as it is on disk, drawn by the same renderTree as Pages.
// toNavNodes turns the engine's listing into NavNodes (folders are groups, images the "file" kind) and the badges.
import type { FileNode, FileTree, NavNode } from "../host";

export type { FileNode, FileTree } from "../host";

/** What the badge of a marked page says when the pointer rests on it. */
export const MARK_TIPS: Record<string, string> = {
  other: "Under Other pages in the Pages view",
  hidden: "Hidden from the Pages view",
};

/** The listing as tree rows: folders are groups (their path ends in "/"), pages pages, images "file" rows. Every
 *  row shows its file name, as on disk; titles holds each page's title (its heading), for the tooltip and Copy Link. */
export function toNavNodes(t: FileTree): { tree: NavNode[]; badges: Record<string, string>; titles: Record<string, string>; marks: Record<string, string> } {
  const badges: Record<string, string> = {};
  const titles: Record<string, string> = {};
  const marks: Record<string, string> = {};
  const conv = (nodes: FileNode[]): NavNode[] => nodes.map((n) => {
    const kind: NavNode["kind"] = n.kind === "folder" ? "group" : n.kind === "image" ? "file" : "page";
    titles[n.path] = n.kind === "page" ? n.title || n.name : n.name;
    if (n.nav) marks[n.path] = n.nav;
    if (n.nav === "other" || n.nav === "hidden") badges[n.path] = n.nav;
    return { title: n.name, path: n.path, kind, children: conv(n.children ?? []) };
  });
  return { tree: conv(t.nodes), badges, titles, marks };
}

/** The folders above a path, outermost first: "a/b/c.md" gives ["a/", "a/b/"]. */
export function folderChain(path: string): string[] {
  const parts = path.replace(/\/$/, "").split("/").slice(0, -1);
  return parts.map((_, i) => parts.slice(0, i + 1).join("/") + "/");
}

/** Where an image of the folder is served: only listed paths go in, each segment encoded. */
export function imageSrc(path: string): string {
  return "/_ws/" + path.split("/").map(encodeURIComponent).join("/");
}

/**
 * What renderTree has no place for: the badges' tooltips, and the images announced as images.
 * Text only goes in as properties.
 */
export function decorateFiles(host: HTMLElement, marks: Record<string, string>, titles: Record<string, string> = {}): void {
  host.querySelector("[role=tree]")?.setAttribute("aria-label", "Files");
  for (const li of Array.from(host.querySelectorAll<HTMLElement>("[role=treeitem]"))) {
    const path = li.dataset.path ?? "";
    const badge = li.querySelector<HTMLElement>(":scope > .zen-nav-row > .zen-nav-badge");
    if (badge && MARK_TIPS[marks[path]]) badge.title = MARK_TIPS[marks[path]];
    if (li.dataset.kind === "page" && titles[path]) { // the row shows the file name; the tooltip gives the page's title
      const row = li.querySelector<HTMLElement>(":scope > .zen-nav-row");
      if (row) row.title = titles[path];
    }
    if (li.dataset.kind === "file") {
      const label = li.querySelector(":scope > .zen-nav-row > .zen-nav-label")?.textContent ?? path;
      li.setAttribute("aria-label", `${label}, image`);
    }
  }
}
