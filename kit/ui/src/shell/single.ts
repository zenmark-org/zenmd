// One markdown file opened on its own (app/single-file): the sidebar has no tree. It shows the page's outline,
// in the rows the tree uses (nav.css), and a quiet Open Enclosing Folder button. What needs a folder is greyed
// out in the places that already grey things (the Settings rows), with one tooltip.
import type { Workspace } from "../host";
import { button, el } from "./dom";
import type { AppState, Ctx } from "./types";
import "./single.css";

/** The tooltip of everything a one-file window leaves off: New Page, Export › Website, the folder's settings. */
export const FOLDER_ONLY_TIP = "Open a folder to use this";

/** Whether the workspace is one file opened on its own (Workspace.single). */
export const isSingle = (w: Pick<Workspace, "single"> | null | undefined): boolean => w?.single === true;

/** The file's own name: the last segment of a workspace path. */
export const fileName = (path: string): string => path.slice(path.lastIndexOf("/") + 1);

/** The window title of a one-file window: the page's title (its H1), then the file's name. */
export const windowTitle = (pageTitle: string, path: string): string => `${pageTitle} — ${fileName(path)}`;

/** One heading of a page. */
export type OutlineItem = { level: number; text: string; id: string };

/**
 * The headings of a rendered page, in order. The html is read into an inert document, so nothing in it runs or
 * loads. A heading without an id cannot be scrolled to and is left out.
 */
export function outlineOf(html: string): OutlineItem[] {
  if (!html) return [];
  const parsed = new DOMParser().parseFromString(html, "text/html");
  const out: OutlineItem[] = [];
  for (const h of Array.from(parsed.querySelectorAll<HTMLElement>("h1, h2, h3, h4, h5, h6"))) {
    const id = h.id;
    const text = (h.textContent ?? "").trim();
    if (id && text) out.push({ level: Number(h.tagName.slice(1)), text, id });
  }
  return out;
}

const INDENT = 14; // px per level, as the tree's

/**
 * Fills `host` (the sidebar's body) with the outline of the page shown and the Open Enclosing Folder button, and
 * keeps them in step with the store (the page, the anchor). Returns the clean-up.
 */
export function mountSingleSidebar(host: HTMLElement, o: { ctx: Ctx; openEnclosing(): void }): () => void {
  const { ctx } = o;
  const doc = host.ownerDocument;
  const body = el("div", "zen-nav-tree zen-single");
  const outline = el("div", "zen-single-outline");
  const foot = el("div", "zen-single-foot");
  const open = button("Open Enclosing Folder", () => o.openEnclosing(), { className: "ghost zen-sidebar-btn zen-single-folder" });
  open.title = "Open Enclosing Folder (⇧⌘O)";
  foot.append(open);
  body.append(outline, foot);
  host.append(body);

  const rowsOf = () => Array.from(outline.querySelectorAll<HTMLElement>(".zen-single-row"));

  function mark(anchor: string): void {
    for (const li of Array.from(outline.querySelectorAll<HTMLElement>(".zen-nav-item"))) {
      if (li.dataset.id === anchor && anchor !== "") li.setAttribute("aria-current", "location");
      else li.removeAttribute("aria-current");
    }
  }

  function draw(s: AppState): void {
    const page = s.page && s.page.path === s.path ? s.page : undefined;
    if (!page) { // not loaded yet: nothing to say, and never the previous page's outline
      outline.replaceChildren();
      return;
    }
    const items = outlineOf(page.html);
    if (!items.length) {
      outline.replaceChildren(el("p", "zen-nav-empty", "No headings"));
      return;
    }
    const min = Math.min(...items.map((i) => i.level));
    const ul = el("ul", "zen-nav-list");
    ul.setAttribute("aria-label", "Outline");
    items.forEach((it, i) => {
      const li = el("li", "zen-nav-item");
      li.dataset.id = it.id;
      const row = el("button", "zen-nav-row zen-single-row");
      row.type = "button";
      row.title = it.text;
      row.dataset.id = it.id;
      row.tabIndex = i === 0 ? 0 : -1;
      row.style.paddingLeft = `calc(var(--space-2) + ${(it.level - min) * INDENT}px)`;
      row.append(el("span", "zen-nav-label", it.text));
      row.addEventListener("click", () => {
        ctx.navigate(s.path, it.id);
        // The same heading again: the store does not change, so the page is told to scroll here.
        doc.getElementById(it.id)?.scrollIntoView?.({ block: "start" });
      });
      li.append(row);
      ul.append(li);
    });
    outline.replaceChildren(ul);
    mark(s.anchor);
  }

  outline.addEventListener("keydown", (e) => {
    const rows = rowsOf();
    const at = rows.indexOf(doc.activeElement as HTMLElement);
    if (at < 0) return;
    const to = ({ ArrowDown: at + 1, ArrowUp: at - 1, Home: 0, End: rows.length - 1 } as Record<string, number>)[e.key];
    if (to === undefined) return;
    e.preventDefault();
    const next = rows[Math.min(rows.length - 1, Math.max(0, to))];
    for (const r of rows) r.tabIndex = r === next ? 0 : -1;
    next.focus();
  });

  let last: Partial<AppState> = {};
  const render = (s: AppState): void => {
    if (s.page !== last.page || s.path !== last.path || s.workspace !== last.workspace) draw(s);
    else if (s.anchor !== last.anchor) mark(s.anchor);
    last = { ...s };
  };
  render(ctx.store.get());
  const off = ctx.store.subscribe(render);

  return () => {
    off();
    body.remove();
  };
}
