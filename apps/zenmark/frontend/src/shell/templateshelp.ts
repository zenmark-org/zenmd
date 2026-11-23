// Settings › Blueprints and Settings › Templates (ui/settings-layout): the Settings frame, items on the left
// (the name, with a mark only for what is not built in, is invalid or has an update) and the selected one on the
// right. Blueprints are the ones of File › New from Blueprint… (templates/blueprints on this Mac); Templates are the
// page templates (templates/pages, the open folder's .zen/templates and the built-in ones). All text goes in
// through textContent; the only HTML mounted is what api.preview returned, through the page's own mountPage.
import type { Api, PageTemplate, SiteTemplate, TemplateUpdate } from "../api";
import { button, el, hasFeature, mountPage, reloadSettings, type SettingsItem } from "@zenmd/kit";

export type TemplatesSection = {
  items(): Promise<SettingsItem[]>;
  render(item: SettingsItem, right: HTMLElement): () => void;
  empty(): string;
  destroy(): void;
};

const message = (err: unknown) => (err instanceof Error ? err.message : String(err));
const base = (p: string) => p.split("/").pop() || p;

/**
 * The "Update available" row of a blueprint or a page template, with its Update button. An untouched copy is
 * replaced in place; an edited one is left as it is and the new version is written beside it (.new).
 */
function updateRow(right: HTMLElement, api: Api, kind: "blueprint" | "page", id: string, name: string, version: string, status: HTMLElement): void {
  const row = el("div", "zen-plugin-help-head zen-template-update");
  row.append(el("span", "zen-settings-plugin-state", `Update available: ${version}`));
  const go = button("Update", () => {
    status.textContent = "";
    go.disabled = true;
    api.updateTemplate(kind, id).then((r: TemplateUpdate) => {
      const text = r.inPlace
        ? `Updated ${name} to ${version}.`
        : `Your copy of ${name} has edits, so it was left as it is. The new version is in ${base(r.path)}: compare the two, keep what you like, then delete it.`;
      reloadSettings(go, () => { const s = right.querySelector("[role=status]"); if (s) s.textContent = text; });
    }, (err) => {
      go.disabled = false;
      status.textContent = `Could not update ${name}: ${message(err)}`;
    });
  });
  row.append(go);
  right.append(row);
}

function statusLine(): HTMLElement {
  const status = el("p", "zen-settings-status");
  status.setAttribute("role", "status");
  return status;
}

export function createTemplatesSection(opts: { api: Api; newsite?: boolean }): TemplatesSection {
  const { api } = opts;
  const inBuild = opts.newsite ?? true;
  let templates: SiteTemplate[] = [];
  let failure = "";

  async function items(): Promise<SettingsItem[]> {
    // Without New Site the build has no templates; the service would refuse, so it is not asked.
    if (!inBuild) return [];
    try {
      templates = await api.siteTemplates();
      failure = "";
    } catch (err) {
      templates = [];
      failure = `Could not list blueprints: ${message(err)}`;
      return [];
    }
    return templates.map((t): SettingsItem => ({
      id: t.id, label: t.name || t.id,
      note: [t.error ? "Invalid" : "", t.builtin ? "" : "Your blueprints", t.update ? "Update available" : ""].filter(Boolean).join(" · ") || undefined,
    }));
  }

  function render(item: SettingsItem, right: HTMLElement): () => void {
    const t = templates.find((x) => x.id === item.id);
    if (!t) return () => {};
    const status = statusLine();
    const head = el("div", "zen-plugin-help-head");
    if (t.version) head.append(el("span", "zen-settings-plugin-version", t.version));
    head.append(el("span", "zen-settings-plugin-kind", t.builtin ? "Built in" : "Your blueprints"));
    right.append(head);
    if (t.description) right.append(el("p", "zen-plugin-help-description", t.description));
    if (t.error) right.append(el("p", "zen-settings-plugin-problem", t.error));
    if (t.update) updateRow(right, api, "blueprint", t.id, t.name || t.id, t.update, status);
    if (t.ask.length > 0) {
      right.append(el("h4", "zen-plugin-section", "Asks when a site is made"));
      const list = el("ul", "zen-template-asks");
      for (const a of t.ask) list.append(el("li", "", a.default ? `${a.label} (default ${a.default})` : a.label));
      right.append(list);
    }
    const folder = button("Show Blueprints Folder", () => {
      status.textContent = "";
      api.revealTemplates().catch((err) => { status.textContent = `Could not show the blueprints folder: ${message(err)}`; });
    });
    const actions = el("div", "zen-settings-actions");
    actions.append(folder);
    right.append(actions, status);
    return () => {};
  }

  return { items, render, empty: () => failure || (inBuild ? "No blueprints." : "No blueprints in this build."), destroy() {} };
}

// ---- Settings › Templates: the page templates

const SOURCE_LABEL: Record<PageTemplate["source"], string> = { builtin: "Built-in", custom: "Custom", folder: "This folder" };
/** Built-in · Edited for an installed copy you changed; the other sources are one label. */
const sourceNote = (t: PageTemplate) => (t.edited ? `${SOURCE_LABEL[t.source]} · Edited` : SOURCE_LABEL[t.source]);
const SKELETON_PATH = "zen-template-skeleton.md";

export function createPageTemplatesSection(opts: { api: Api; hasWorkspace(): boolean }): TemplatesSection {
  const { api } = opts;
  let templates: PageTemplate[] = [];
  let failure = "";
  let seq = 0; // a preview that arrives for an older pane is dropped

  async function items(): Promise<SettingsItem[]> {
    try {
      templates = await api.pageTemplates();
      failure = "";
    } catch (err) {
      templates = [];
      failure = `Could not list page templates: ${message(err)}`;
      return [];
    }
    return templates.map((t): SettingsItem => ({
      id: t.id, label: t.name || t.id,
      note: [t.error ? "Invalid" : "", sourceNote(t), t.update ? "Update available" : ""].filter(Boolean).join(" · "),
    }));
  }

  function fieldsTable(t: PageTemplate): HTMLElement[] {
    if (t.fields.length === 0) return [];
    const table = el("div", "zen-plugin-options zen-template-fields");
    for (const f of t.fields) {
      const row = el("div", "zen-plugin-option zen-template-field");
      const line = el("div", "zen-plugin-option-line");
      line.append(el("code", "zen-plugin-option-name", f.name), el("span", "zen-plugin-option-type", f.type));
      if (f.required) line.append(el("span", "zen-plugin-option-default", "required"));
      if (f.default) line.append(el("span", "zen-plugin-option-default", `default ${f.default}`));
      row.append(line);
      table.append(row);
    }
    return [el("h4", "zen-plugin-section", "Fields"), table];
  }

  /** The structure: as nested lists, one line per element. */
  function outline(t: PageTemplate): HTMLElement[] {
    if (t.outline.length === 0) return [];
    const root = el("ul", "zen-template-outline");
    const stack: HTMLElement[] = [root];
    let last: HTMLElement | null = null;
    let depth = 0;
    for (const o of t.outline) {
      const d = Math.max(0, Math.min(o.depth, depth + (last ? 1 : 0)));
      while (depth > d) { stack.pop(); depth--; }
      if (d > depth && last) {
        const sub = el("ul");
        last.append(sub);
        stack.push(sub);
        depth = d;
      }
      last = el("li", "", o.text);
      stack[stack.length - 1].append(last);
    }
    return [el("h4", "zen-plugin-section", "Structure"), root];
  }

  function skeleton(t: PageTemplate, mySeq: number): HTMLElement[] {
    if (!t.skeleton.trim()) return [];
    const box = el("div", "zen-template-skeleton-box");
    const fallback = () => {
      if (mySeq !== seq || !box.isConnected) return;
      const pre = el("pre", "zen-settings-file-preview zen-template-skeleton");
      pre.textContent = t.skeleton;
      box.replaceChildren(pre);
    };
    // The skeleton is markdown: its preview is the page's own rendering, with the title filled in.
    api.preview(SKELETON_PATH, t.skeleton.replaceAll("{{title}}", "Page title")).then((page) => {
      if (mySeq !== seq || !box.isConnected) return;
      const shown = el("div", "zen-page zen-plugin-snippet-preview zen-template-skeleton");
      shown.setAttribute("aria-label", `Skeleton of ${t.name || t.id}`);
      mountPage(shown, page);
      box.replaceChildren(shown);
    }).catch(fallback);
    return [el("h4", "zen-plugin-section", "Skeleton"), box];
  }

  function render(item: SettingsItem, right: HTMLElement): () => void {
    const t = templates.find((x) => x.id === item.id);
    if (!t) return () => {};
    const mySeq = ++seq;
    const name = t.name || t.id;
    const status = statusLine();
    const head = el("div", "zen-plugin-help-head");
    if (t.version) head.append(el("span", "zen-settings-plugin-version", t.version));
    head.append(el("span", "zen-settings-plugin-kind", sourceNote(t)));
    right.append(head);
    if (t.description) right.append(el("p", "zen-plugin-help-description", t.description));
    if (t.error) right.append(el("p", "zen-settings-plugin-problem", t.error));
    if (t.layout) right.append(el("p", "zen-settings-hint", `Layout: ${t.layout}`));
    if (t.update) updateRow(right, api, "page", t.id, name, t.update, status);
    right.append(...fieldsTable(t), ...outline(t), ...skeleton(t, mySeq));
    if (t.rules.length > 0) {
      const list = el("ul", "zen-template-asks");
      for (const r of t.rules) list.append(el("li", "", r));
      right.append(el("h4", "zen-plugin-section", "Used by path rules in this folder"), list);
    }

    const run = (what: () => Promise<unknown>, done: (r: unknown) => string, fail: string, reload = false): void => {
      status.textContent = "";
      what().then((r) => {
        if (!reload) { status.textContent = done(r); return; }
        reloadSettings(status, () => { const s = right.querySelector("[role=status]"); if (s) s.textContent = done(r); });
      }, (err) => { status.textContent = `${fail}: ${message(err)}`; });
    };
    const editor = hasFeature(api, "editor");
    const actions = el("div", "zen-settings-actions");
    actions.append(button("Show in Finder", () => run(() => api.revealPageTemplates(), () => "", "Could not show the page templates folder")));
    if (editor && t.path) actions.append(button("Open in Editor", () => run(() => api.openPageTemplate(t.id), () => `${name} opened.`, `Could not open ${name}`)));
    if (editor && opts.hasWorkspace() && t.source !== "folder") {
      actions.append(button("Copy to This Folder", () => run(() => api.copyPageTemplate(t.id), (p) => `Copied to ${String(p)} in this folder. Edit it there.`, `Could not copy ${name}`, true)));
    }
    right.append(actions, status);
    return () => { seq++; };
  }

  return { items, render, empty: () => failure || "No page templates.", destroy() {} };
}
