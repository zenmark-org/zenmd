// The items of the sidebar's context menu for one node. What the host cannot do is not in `env`,
// and the item is not in the menu.
import type { NavNode } from "../host";
import type { MenuEntry } from "./menu";
import { copyLink, folderOf, relativePath, siblingGroup } from "./siblings";

export type EntryEnv = {
  tree: NavNode[];
  /** The page open now ("" for none): Copy Link is relative to it. */
  current: string;
  open(path: string): void;
  editor?(path: string): void;
  newPage?(folder: string): void;
  /** New Term… on a glossary index; with `glossary`, which says whether a page is one. */
  newTerm?(folder: string): void;
  glossary?(path: string): boolean;
  /** Move the page one place among its siblings. */
  move?(path: string, delta: -1 | 1): void;
  copy?(text: string): void;
  reveal?(path: string): void;
  /** Ask, then hide the page from the sidebar. */
  hide?(node: NavNode): void;
  /** The Files view (ui/files-view): the rows are folders, pages and images as on disk, and there is no Move. */
  view?: "pages" | "files";
  /** Files: where the Pages tree puts a page: "tree", "other" or "hidden" (hidden pages get Show in Sidebar). */
  mark?(path: string): string | undefined;
  /** Files: put a hidden page back in the sidebar. */
  show?(path: string): void;
};

// Copy Link of an image: the path from the open page, with what would break the link written as %XX.
const encodeHref = (s: string) => s.replace(/[ ()<>%#?\\]/g, (c) => `%${c.charCodeAt(0).toString(16).toUpperCase().padStart(2, "0")}`);

/** The menu of a row of the Files view: a page, a folder (group) or an image ("file"). */
function filesEntries(node: NavNode, env: EntryEnv): MenuEntry[] {
  const at = node.path;
  const kind = node.kind as string;
  const groups: MenuEntry[][] = [];
  const add = (...items: (MenuEntry | false | undefined)[]) => {
    const g = items.filter((x): x is MenuEntry => !!x);
    if (g.length) groups.push(g);
  };
  const copyPath = !!env.copy && { label: "Copy Path", run: () => env.copy!(at) };
  const reveal = !!env.reveal && { label: "Reveal in Finder", run: () => env.reveal!(at) };
  if (kind === "group") {
    add(!!env.newPage && { label: "New Page Here…", run: () => env.newPage!(at) });
    add(copyPath, reveal);
  } else if (kind === "file") {
    add({ label: "Open", run: () => env.open(at) });
    add(
      copyPath,
      !!env.copy && { label: "Copy Link", run: () => env.copy!(`![](${encodeHref(relativePath(env.current, at))})`) },
      reveal,
    );
  } else {
    const hidden = env.mark?.(at) === "hidden";
    add(
      { label: "Open", run: () => env.open(at) },
      !!env.editor && { label: "Open in Editor", run: () => env.editor!(at) },
    );
    add(!!env.newPage && { label: "New Page Here…", run: () => env.newPage!(folderOf({ path: at, kind: "page" })) });
    add(copyPath, !!env.copy && { label: "Copy Link", run: () => env.copy!(copyLink(node.title, at, env.current)) }, reveal);
    add(
      !hidden && !!env.hide && { label: "Hide from Sidebar", run: () => env.hide!(node) },
      hidden && !!env.show && { label: "Show in Sidebar", run: () => env.show!(at) },
    );
  }
  return groups.flatMap((g, i) => (i ? ["separator" as const, ...g] : g));
}

export function menuEntries(node: NavNode, env: EntryEnv): MenuEntry[] {
  if (!node.path) return []; // the "Other pages" group
  if (env.view === "files") return filesEntries(node, env);
  const at = node.path;
  const groups: MenuEntry[][] = [];
  const add = (...items: (MenuEntry | false | undefined)[]) => {
    const g = items.filter((x): x is MenuEntry => !!x);
    if (g.length) groups.push(g);
  };
  if (node.kind === "page") {
    const sib = siblingGroup(env.tree, at);
    const i = sib ? sib.pages.indexOf(at) : -1;
    add(
      { label: "Open", run: () => env.open(at) },
      !!env.editor && { label: "Open in Editor", run: () => env.editor!(at) },
    );
    add(
      !!env.newPage && { label: "New Page Here…", run: () => env.newPage!(folderOf(node)) },
      !!env.newTerm && !!env.glossary?.(at) && { label: "New Term…", run: () => env.newTerm!(folderOf(node).replace(/\/$/, "")) },
      !!env.move && { label: "Move Up", run: () => env.move!(at, -1), disabled: !sib || i <= 0 },
      !!env.move && { label: "Move Down", run: () => env.move!(at, 1), disabled: !sib || i < 0 || i >= sib.pages.length - 1 },
    );
    add(
      !!env.copy && { label: "Copy Path", run: () => env.copy!(at) },
      !!env.copy && { label: "Copy Link", run: () => env.copy!(copyLink(node.title, at, env.current)) },
      !!env.reveal && { label: "Reveal in Finder", run: () => env.reveal!(at) },
    );
    add(!!env.hide && { label: "Hide from Sidebar", run: () => env.hide!(node) });
  } else {
    add(!!env.newPage && { label: "New Page Here…", run: () => env.newPage!(folderOf(node)) });
    add(
      !!env.copy && { label: "Copy Path", run: () => env.copy!(at) },
      !!env.reveal && { label: "Reveal in Finder", run: () => env.reveal!(at) },
    );
  }
  return groups.flatMap((g, i) => (i ? ["separator" as const, ...g] : g));
}
