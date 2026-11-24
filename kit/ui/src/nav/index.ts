// The nav feature: sidebar tree in the `nav` slot, breadcrumb in `breadcrumb`.
import { hasFeature, type NavNode, type Workspace } from "../host";
import { copyText } from "../shell/clipboard";
import { el } from "../shell/dom";
import type { AppState, Feature } from "../shell/types";
import { renderBreadcrumb } from "./breadcrumb";
import { menuEntries, type EntryEnv } from "./entries";
import { decorateFiles, folderChain, imageSrc, toNavNodes, type FileTree } from "./files";
import { confirmDialog, openMenu, type MenuHandle } from "./menu";
import { findNode, movedOrder, siblingGroup } from "./siblings";
import { focusPath, renderTree } from "./tree";
import { loadView, renderViewSwitch, saveView, type NavView } from "./viewswitch";

export { renderBreadcrumb } from "./breadcrumb";
export { renderTree } from "./tree";

const storageKey = (root: string) => `zen.nav.expanded:${root}`;
const filesKey = (root: string) => `zen.files.expanded:${root}`;
const STATUS_MS = 6000;

/** The folders of the Files view the person opened, or null while there is no choice (all closed but the open page's). */
function loadFilesExpanded(root: string): Set<string> | null {
  try {
    const raw = localStorage.getItem(filesKey(root));
    if (raw) {
      const list: unknown = JSON.parse(raw);
      if (Array.isArray(list)) return new Set(list.filter((x): x is string => typeof x === "string"));
    }
  } catch { /* fall through to defaults */ }
  return null;
}

function loadExpanded(w: Workspace): Set<string> {
  try {
    const raw = localStorage.getItem(storageKey(w.root));
    if (raw) {
      const list: unknown = JSON.parse(raw);
      if (Array.isArray(list)) return new Set(list.filter((x): x is string => typeof x === "string"));
    }
  } catch { /* fall through to defaults */ }
  return new Set(); // collapsed by default: only the folders of the open page open (renderTree)
}

function saveExpanded(root: string, set: Set<string>): void {
  try { localStorage.setItem(storageKey(root), JSON.stringify(Array.from(set))); } catch { /* storage unavailable */ }
}

export const navFeature: Feature = {
  id: "nav",
  mount(slots, ctx) {
    const doc = slots.nav.ownerDocument;
    const treeHost = el("div", "zen-nav-tree");
    // The Pages | Files switch, and the Files view: one tree host per view, one shown at a time.
    const switchHost = el("div", "zen-nav-switch");
    const filesHost = el("div", "zen-nav-files");
    const filesTree = el("div", "zen-nav-filestree");
    const filesNote = el("p", "zen-nav-empty");
    filesNote.hidden = true;
    filesHost.append(filesTree, filesNote);
    filesHost.hidden = true;
    // What a menu action did, or why it failed (an alert), under the tree.
    const status = el("div", "zen-nav-status");
    status.setAttribute("role", "status");
    status.setAttribute("aria-live", "polite");
    status.hidden = true;
    const crumbHost = el("nav", "zen-nav-crumbs");
    crumbHost.setAttribute("aria-label", "Breadcrumb");
    slots.nav.append(switchHost, filesHost, treeHost, status);
    slots.breadcrumb.append(crumbHost);

    let root: string | null = null;
    let expanded = new Set<string>();
    let last: Partial<AppState> = {};
    let menu: MenuHandle | null = null;
    let statusTimer: ReturnType<typeof setTimeout> | undefined;
    let focusTimer: ReturnType<typeof setTimeout> | undefined;
    let view: NavView = "pages";
    let filesData: FileTree | null = null;
    let filesLoading = false;
    let filesExpanded: Set<string> | null = null;
    let filesSeq = 0;
    let filesMarks: Record<string, string> = {};
    let filesNav: NavNode[] = [];
    let filesImages = new Set<string>();
    /** The page that was just moved: it gets the focus when the new tree is drawn. */
    let pendingFocus = "";
    /** The folders that have a glossary index (asked once per folder and tree), for New Term… in the menu. */
    let glossaries = new Set<string>();
    const loadGlossaries = () => {
      if (!editable || !api.glossaryFolders) return;
      api.glossaryFolders().then((f) => { glossaries = new Set(f); }, () => { glossaries = new Set(); });
    };

    const say = (text: string, failed = false) => {
      clearTimeout(statusTimer);
      status.textContent = text;
      status.hidden = false;
      status.classList.toggle("zen-nav-status-error", failed);
      status.setAttribute("role", failed ? "alert" : "status");
      statusTimer = setTimeout(() => { status.hidden = true; status.textContent = ""; }, STATUS_MS);
    };
    const fail = (err: unknown) => say(err instanceof Error ? err.message : String(err), true);

    const api = ctx.api;
    // Without the editor feature (a read-only app) the sidebar only reads: no editing items, no reordering.
    const editable = hasFeature(api, "editor");
    const titleOf = (path: string) => findNode(ctx.store.get().workspace?.tree ?? [], path)?.title ?? path;

    const reorder = (parent: string, order: string[], moved: string) => {
      if (!editable || !api.reorderNav) return;
      pendingFocus = moved;
      clearTimeout(focusTimer);
      focusTimer = setTimeout(() => { pendingFocus = ""; }, 3000);
      api.reorderNav(parent, order).then(
        () => say(`Moved ${titleOf(moved)} to position ${order.indexOf(moved) + 1} of ${order.length}`),
        (err) => { pendingFocus = ""; fail(err); },
      );
    };
    const move = (path: string, delta: -1 | 1) => {
      const group = siblingGroup(ctx.store.get().workspace?.tree ?? [], path);
      const order = group && movedOrder(group.pages, path, delta);
      if (group && order) reorder(group.parent, order, path);
    };

    const showMenu = (path: string, at: { x: number; y: number }, row: HTMLElement, files = false) => {
      const { workspace, path: current } = ctx.store.get();
      const tree = files ? filesNav : workspace?.tree ?? [];
      const node = workspace && findNode(tree, path);
      if (!workspace || !node) return;
      const env: EntryEnv = {
        tree,
        current,
        open: (p) => (filesImages.has(p) ? openImage(p) : ctx.navigate(p)),
        editor: editable && api.openEditor ? (p) => { api.openEditor(p).catch(fail); } : undefined,
        newPage: editable && ctx.newPage ? (folder) => ctx.newPage!(folder) : undefined,
        newTerm: editable && ctx.newTerm ? (folder) => ctx.newTerm!(folder) : undefined,
        glossary: (p) => /(^|\/)(index|readme)\.md$/i.test(p) && glossaries.has(p.includes("/") ? p.slice(0, p.lastIndexOf("/")) : ""),
        move: !files && editable && api.reorderNav ? move : undefined,
        view: files ? "files" : undefined,
        mark: files ? (p) => filesMarks[p] : undefined,
        show: files && editable && api.setNavHidden ? (p) => { api.setNavHidden!(p, false).catch(fail); } : undefined,
        copy: (text) => { void copyText(doc, text).then((ok) => (ok ? say("Copied") : say("Could not copy", true))); },
        reveal: api.revealInFinder ? (p) => { api.revealInFinder!(p).catch(fail); } : undefined,
        hide: editable && api.setNavHidden ? (n) => hide(n) : undefined,
      };
      const entries = menuEntries(node, env);
      if (!entries.length) return;
      menu = openMenu({ doc, entries, x: at.x, y: at.y, returnFocus: row, label: node.title, onClose: () => { menu = null; } });
    };

    const hide = (node: NavNode) => {
      const below = node.children.length ? " The pages under it leave the sidebar with it." : "";
      confirmDialog({
        doc,
        title: `Hide “${node.title}” from the sidebar?`,
        message: `The page stays in the folder and links to it still work.${below} To show it again, remove hidden: true from nav: in its frontmatter.`,
        confirm: "Hide",
        onConfirm: () => { api.setNavHidden!(node.path, true).catch(fail); },
      });
    };

    /** Opens an image of the Files view in the image view over the window; the row has the focus, so Esc gives it back. */
    const openImage = (path: string) => {
      focusPath(filesTree, path);
      ctx.zoomImage?.(imageSrc(path), path.slice(path.lastIndexOf("/") + 1));
    };

    const showViews = () => {
      const w = ctx.store.get().workspace;
      const switchable = !!w && !w.single && !!api.fileTree;
      switchHost.hidden = !switchable;
      const files = switchable && view === "files";
      filesHost.hidden = !files;
      treeHost.hidden = files;
    };
    const filesShown = () => !filesHost.hidden;

    const setNote = (text: string, omitted = false) => {
      filesNote.textContent = text;
      filesNote.hidden = !text;
      filesNote.classList.toggle("zen-nav-omitted", omitted);
    };
    const drawFiles = () => {
      const { workspace, path } = ctx.store.get();
      if (!workspace || !filesData) {
        filesTree.replaceChildren();
        setNote(filesLoading ? "Loading files…" : "");
        return;
      }
      const { tree, badges, marks } = toNavNodes(filesData);
      filesNav = tree;
      filesMarks = marks;
      filesImages = new Set<string>();
      const walk = (ns: NavNode[]) => ns.forEach((n) => { if ((n.kind as string) === "file") filesImages.add(n.path); walk(n.children); });
      walk(tree);
      if (!tree.length) {
        filesTree.replaceChildren();
        setNote("No pages or images in this folder");
        return;
      }
      if (!filesExpanded) {
        filesExpanded = new Set(folderChain(path)); // collapsed by default, but for the open page's folders
      }
      const expanded = filesExpanded;
      renderTree(filesTree, tree, {
        current: path,
        expanded,
        badges,
        onOpen: (p) => (filesImages.has(p) ? openImage(p) : ctx.navigate(p)),
        onToggle: (p, open) => {
          if (filesImages.has(p)) { openImage(p); return; } // an image row is not a folder
          if (open) expanded.add(p); else expanded.delete(p);
          try { localStorage.setItem(filesKey(workspace.root), JSON.stringify(Array.from(expanded))); } catch { /* storage unavailable */ }
          drawFiles();
        },
        onContextMenu: (p, at, row) => showMenu(p, at, row, true),
      });
      decorateFiles(filesTree, marks);
      setNote(filesData.omitted > 0 ? `${filesData.omitted} more files not shown` : "", true);
    };
    /** Asks for the listing; the last one stays on the screen when it fails. */
    const loadFiles = () => {
      if (!api.fileTree) return;
      const seq = ++filesSeq;
      const forRoot = root;
      filesLoading = !filesData;
      if (filesLoading) drawFiles();
      api.fileTree().then(
        (t) => {
          if (seq !== filesSeq || root !== forRoot) return;
          filesLoading = false;
          filesData = t;
          drawFiles();
        },
        (err) => {
          if (seq !== filesSeq || root !== forRoot) return;
          filesLoading = false;
          fail(err);
          drawFiles();
        },
      );
    };
    const pick = (v: NavView) => {
      view = v;
      if (root) saveView(root, v);
      showViews();
      if (v === "files") { loadFiles(); drawFiles(); }
    };
    const switcher = renderViewSwitch(switchHost, () => view, pick);

    const drawTree = () => {
      const { workspace, path } = ctx.store.get();
      if (!workspace) { treeHost.replaceChildren(); return; }
      renderTree(treeHost, workspace.tree, {
        current: path,
        expanded,
        badges: ctx.store.get().badges,
        onOpen: (p) => ctx.navigate(p),
        onToggle: (p, open) => {
          if (open) expanded.add(p); else expanded.delete(p);
          saveExpanded(workspace.root, expanded);
          drawTree();
        },
        onContextMenu: showMenu,
        onMove: editable && api.reorderNav ? move : undefined,
        onReorder: editable && api.reorderNav ? reorder : undefined,
      });
      treeHost.querySelector("[role=tree]")?.setAttribute("aria-label", "Pages");
      if (pendingFocus && focusPath(treeHost, pendingFocus)) pendingFocus = "";
    };
    const drawCrumbs = () => {
      const { page, path, mode, dirty } = ctx.store.get();
      const crumbs = page && page.path === path ? page.breadcrumb : [];
      renderBreadcrumb(crumbHost, crumbs, { onOpen: (p) => ctx.navigate(p), dirty: mode === "edit" && !!dirty });
    };

    const render = (s: AppState) => {
      if (s.workspace !== last.workspace || s.path !== last.path || s.badges !== last.badges) {
        const r = s.workspace?.root ?? null;
        if (r !== root) {
          root = r;
          expanded = s.workspace ? loadExpanded(s.workspace) : new Set();
          view = r ? loadView(r) : "pages";
          filesData = null;
          filesLoading = false;
          filesExpanded = r ? loadFilesExpanded(r) : null;
          filesSeq++;
          filesTree.replaceChildren();
          setNote("");
          switcher.sync();
        }
        if (s.workspace !== last.workspace) loadGlossaries();
        drawTree();
        showViews();
        if (filesShown()) {
          // The folder changed (save, new page, hide, the watcher, Reload): ask for the listing again, only while shown.
          if (s.workspace !== last.workspace) loadFiles();
          drawFiles();
        }
      }
      if (s.page !== last.page || s.path !== last.path || s.mode !== last.mode || s.dirty !== last.dirty) drawCrumbs();
      last = { ...s };
    };
    render(ctx.store.get());
    const off = ctx.store.subscribe(render);
    const ro = typeof ResizeObserver === "function" ? new ResizeObserver(() => drawCrumbs()) : null;
    ro?.observe(slots.breadcrumb);

    return () => {
      off();
      ro?.disconnect();
      menu?.close();
      clearTimeout(statusTimer);
      clearTimeout(focusTimer);
      treeHost.remove();
      switchHost.remove();
      filesHost.remove();
      status.remove();
      crumbHost.remove();
    };
  },
};
