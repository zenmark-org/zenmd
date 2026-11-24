// :::tab-set — ARIA tabs built on the titles and panels Go rendered (one tab-item each). A set whose
// items carry a sync key (:sync:) follows the other sets of the page with the same sync group
// (:sync-group:, default "tab"); the last synced choice is kept in ctx.storage under
// "tabs-<group>": {version:1, key}. Everything is built with DOM APIs and textContent; nothing is parsed as HTML.
const PREFIX = "tabs-";
const DEFAULT_GROUP = "tab";
let uid = 0;

/** The sets of the page that follow a sync group: group -> Set of {apply(key), has(key)}. */
const groups = new Map();

function join(group, member) {
  let set = groups.get(group);
  if (!set) groups.set(group, (set = new Set()));
  set.add(member);
  return () => {
    set.delete(member);
    if (set.size === 0) groups.delete(group);
  };
}

function h(tag, props, ...kids) {
  const e = document.createElement(tag);
  for (const k of Object.keys(props || {})) {
    if (k === "class") e.className = props[k];
    else if (k === "text") e.textContent = props[k];
    else e.setAttribute(k, props[k]);
  }
  for (const c of kids) if (c) e.append(c);
  return e;
}

function attrsOf(el) {
  try {
    const v = JSON.parse(el.getAttribute("data-zen-attrs") || "{}");
    return v && typeof v === "object" && !Array.isArray(v) ? v : {};
  } catch {
    return {};
  }
}

/** A stored value that is not a version-1 choice is no choice (""). */
export function normalise(v) {
  return v && typeof v === "object" && v.version === 1 && typeof v.key === "string" ? v.key : "";
}

/** The storage key of a sync group: at most 64 characters, letters, digits . _ - only. */
export function storageKey(group) {
  return (PREFIX + group.replace(/[^A-Za-z0-9._-]/g, "_")).slice(0, 64);
}

async function mountSet(el, ctx) {
  const frame = el.querySelector(":scope > .zen-tabs");
  const body = frame && frame.querySelector(":scope > .zen-tabs-body");
  if (!frame || !body) return;
  const wrappers = Array.from(body.children).filter((c) => c.getAttribute("data-zen-name") === "tab-item");
  const items = wrappers
    .map((w) => {
      const item = w.querySelector(":scope > .zen-tab-item");
      const title = item && item.querySelector(":scope > .zen-tab-title");
      const panel = item && item.querySelector(":scope > .zen-tab-panel");
      return item && title && panel ? { item, panel, title: (title.textContent ?? "").trim(), sync: String(attrsOf(w).sync ?? ""), selected: item.classList.contains("zen-tab-selected") } : null;
    })
    .filter(Boolean);
  if (items.length === 0) return;

  const group = String(ctx.attrs["sync-group"] || DEFAULT_GROUP);
  const base = `zt${++uid}`;
  const list = h("div", { class: "zen-tab-list", role: "tablist" });
  const tabs = items.map((it, i) => {
    const t = h("button", { type: "button", class: "zen-btn zen-btn-quiet", role: "tab", id: `${base}-t${i}`, "aria-controls": `${base}-p${i}`, "aria-selected": "false", text: it.title });
    t.tabIndex = -1;
    list.append(t);
    it.panel.setAttribute("role", "tabpanel");
    it.panel.id = `${base}-p${i}`;
    it.panel.setAttribute("aria-labelledby", t.id);
    it.panel.hidden = true;
    return t;
  });
  frame.insertBefore(list, body);
  frame.classList.add("zen-tabs-js");

  let current = -1;
  function show(i) {
    current = i;
    tabs.forEach((t, j) => {
      const on = j === i;
      t.setAttribute("aria-selected", on ? "true" : "false");
      t.tabIndex = on ? 0 : -1;
      items[j].panel.hidden = !on;
    });
  }
  const storeKey = storageKey(group);
  let dead = false;
  const member = {
    has: (key) => items.some((it) => it.sync === key),
    apply: (key) => {
      const i = items.findIndex((it) => it.sync === key);
      if (i >= 0) show(i);
    },
  };
  function choose(i, focus) {
    show(i);
    if (focus) tabs[i].focus();
    const key = items[i].sync;
    if (!key) return;
    for (const m of groups.get(group) ?? []) if (m !== member) m.apply(key);
    Promise.resolve(ctx.storage.set(storeKey, { version: 1, key })).catch(() => {});
  }
  tabs.forEach((t, i) => t.addEventListener("click", () => choose(i, false)));
  list.addEventListener("keydown", (e) => {
    const at = tabs.indexOf(document.activeElement);
    if (at < 0 || e.ctrlKey || e.metaKey || e.altKey) return;
    let to = -1;
    if (e.key === "ArrowRight") to = (at + 1) % tabs.length;
    else if (e.key === "ArrowLeft") to = (at - 1 + tabs.length) % tabs.length;
    else if (e.key === "Home") to = 0;
    else if (e.key === "End") to = tabs.length - 1;
    if (to < 0) return;
    e.preventDefault();
    choose(to, true);
  });

  const first = items.findIndex((it) => it.selected);
  show(first >= 0 ? first : 0);
  const leave = items.some((it) => it.sync) ? join(group, member) : () => {};

  // the choice of the last visit, when this set has that key
  if (items.some((it) => it.sync)) {
    try {
      const key = normalise(await ctx.storage.get(storeKey));
      if (!dead && key && member.has(key)) member.apply(key);
    } catch {
      // storage is a convenience: the set works without it
    }
  }

  return () => {
    dead = true;
    leave();
    list.remove();
    frame.classList.remove("zen-tabs-js");
    for (const it of items) {
      it.panel.removeAttribute("role");
      it.panel.removeAttribute("id");
      it.panel.removeAttribute("aria-labelledby");
      it.panel.hidden = false;
      it.panel.removeAttribute("hidden");
    }
  };
}

export default {
  mount(el, ctx) {
    return ctx.name === "tab-set" ? mountSet(el, ctx) : undefined;
  },
};
