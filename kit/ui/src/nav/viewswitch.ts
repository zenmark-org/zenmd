// The Pages | Files switch at the top of the sidebar: the Settings segment control (role radiogroup, arrow keys,
// the same classes), remembered per folder.
import { el } from "../shell/dom";

export type NavView = "pages" | "files";

const OPTIONS: { value: NavView; label: string }[] = [{ value: "pages", label: "Pages" }, { value: "files", label: "Files" }];
const key = (root: string) => `zen.nav.view:${root}`;

export function renderViewSwitch(host: HTMLElement, get: () => NavView, set: (v: NavView) => void): { sync(): void } {
  const group = el("div", "zen-settings-segment");
  group.setAttribute("role", "radiogroup");
  group.setAttribute("aria-label", "Sidebar view");
  const radios = OPTIONS.map((o) => {
    const b = el("button", "zen-settings-segment-option", o.label);
    b.type = "button";
    b.setAttribute("role", "radio");
    b.addEventListener("click", () => { set(o.value); sync(); });
    return b;
  });
  function sync(): void {
    OPTIONS.forEach((o, i) => {
      const on = o.value === get();
      radios[i].setAttribute("aria-checked", String(on));
      radios[i].tabIndex = on ? 0 : -1;
    });
  }
  group.addEventListener("keydown", (e) => {
    const step = ({ ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 } as Record<string, number>)[e.key];
    if (step === undefined) return;
    e.preventDefault();
    e.stopPropagation();
    const at = Math.max(0, OPTIONS.findIndex((o) => o.value === get()));
    const next = Math.min(OPTIONS.length - 1, Math.max(0, at + step));
    set(OPTIONS[next].value);
    sync();
    radios[next].focus();
  });
  group.append(...radios);
  host.replaceChildren(group);
  sync();
  return { sync };
}

/** The view chosen for a folder; Pages when none was or the storage is unavailable. */
export function loadView(root: string): NavView {
  try { if (localStorage.getItem(key(root)) === "files") return "files"; } catch { /* storage unavailable */ }
  return "pages";
}

export function saveView(root: string, v: NavView): void {
  try { localStorage.setItem(key(root), v); } catch { /* storage unavailable */ }
}
