// New Term… (the glossary page's button, and the Glossary page's menu in the sidebar): a small
// dialog, the same frame as New Page, that asks for the term. api.newTerm writes <folder>/<slug>.md
// from the term template; the dialog then reports the path (the shell shows it and opens the editor).
// A term that exists is never overwritten: the dialog says so and offers to open it.
import type { Api } from "../api";
import { button, el } from "@zenmd/kit";

export type NewTermOptions = {
  api: Api;
  doc: Document;
  /** The glossary folder ("" for the top level, no trailing slash). */
  folder: string;
  /** The term's page: new, or the existing one the user chose to open. */
  onCreated(path: string): void;
  onClose(): void;
};

export type NewTermDialog = { el: HTMLElement; close(): void };

let seq = 0;

export function openNewTerm(opts: NewTermOptions): NewTermDialog {
  const { api, doc } = opts;
  const id = ++seq;
  const backdrop = el("div", "zen-newpage-backdrop");
  const form = el("form", "zen-newpage");
  form.setAttribute("role", "dialog");
  form.setAttribute("aria-modal", "true");
  form.noValidate = true;
  const title = el("h2", "zen-newpage-title", "New term");
  title.id = `zen-newterm-title-${id}`;
  form.setAttribute("aria-labelledby", title.id);

  const label = el("label", "zen-newpage-label", "Term");
  const name = el("input", "zen-newpage-name");
  name.type = "text";
  name.id = `zen-newterm-name-${id}`;
  name.autocomplete = "off";
  name.spellcheck = true;
  label.htmlFor = name.id;
  const where = opts.folder === "" ? "the top folder" : opts.folder;
  const hint = el("p", "zen-newpage-hint", `A page named after the term is made in ${where}, and opens in the editor.`);
  const error = el("p", "zen-newpage-error");
  error.setAttribute("role", "alert");

  const actions = el("div", "zen-newpage-actions");
  const cancel = button("Cancel", () => close());
  const create = el("button", "primary", "Create");
  create.type = "submit";
  actions.append(cancel, create);

  form.append(title, label, name, hint, error, actions);
  backdrop.append(form);
  doc.body.append(backdrop);

  const before = doc.activeElement as HTMLElement | null;
  name.focus();

  let busy = false;
  /** The page of a term that exists: the next submit opens it instead of making it. */
  let existing = "";
  const reset = () => {
    existing = "";
    error.textContent = "";
    create.textContent = "Create";
  };
  name.addEventListener("input", () => { if (existing) reset(); });

  form.addEventListener("submit", async (e) => {
    e.preventDefault();
    if (busy) return;
    if (existing) {
      const path = existing;
      close();
      opts.onCreated(path);
      return;
    }
    if (name.value.trim() === "") {
      error.textContent = "Type the term.";
      name.focus();
      return;
    }
    busy = true;
    create.disabled = cancel.disabled = true;
    error.textContent = "";
    try {
      const r = await api.newTerm(opts.folder, name.value);
      if (r.existed) {
        existing = r.path;
        error.textContent = `${r.path} already exists.`;
        create.textContent = "Open it";
        busy = false;
        create.disabled = cancel.disabled = false;
        create.focus();
        return;
      }
      close();
      opts.onCreated(r.path);
    } catch (err) {
      error.textContent = err instanceof Error ? err.message : String(err);
      busy = false;
      create.disabled = cancel.disabled = false;
      name.focus();
    }
  });

  // Esc closes, Tab stays inside, a click outside closes (as in New Page).
  const onKey = (e: KeyboardEvent) => {
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      if (!busy) close();
    } else if (e.key === "Tab") {
      const items = Array.from(form.querySelectorAll<HTMLElement>("input, button:not([disabled])"));
      const first = items[0];
      const last = items[items.length - 1];
      if (e.shiftKey && doc.activeElement === first) { e.preventDefault(); last.focus(); }
      else if (!e.shiftKey && doc.activeElement === last) { e.preventDefault(); first.focus(); }
    }
  };
  doc.addEventListener("keydown", onKey, true);
  backdrop.addEventListener("mousedown", (e) => { if (e.target === backdrop && !busy) close(); });

  let closed = false;
  function close(): void {
    if (closed) return;
    closed = true;
    doc.removeEventListener("keydown", onKey, true);
    backdrop.remove();
    if (before?.isConnected) before.focus();
    opts.onClose();
  }

  return { el: backdrop, close };
}
