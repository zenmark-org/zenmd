// glossary.js — the "glossary" page layout: a filter field over the list and
// keyboard movement along the letter bar. The page works without it: the
// list and the bar are in the page, and the letters are plain links. Text only
// (textContent); the filter is a plain-text match, never a regular expression.
const fold = (s) => String(s).normalize("NFD").replace(/\p{M}+/gu, "").toLowerCase();

function el(doc, name, cls, text) {
  const e = doc.createElement(name);
  if (cls) e.className = cls;
  if (text !== undefined) e.textContent = text;
  return e;
}

function letterBar(bar) {
  const links = Array.from(bar.querySelectorAll("a"));
  if (links.length === 0) return;
  // One Tab stop on the bar; the arrows move along it.
  const roving = (to) => links.forEach((a) => { a.tabIndex = a === to ? 0 : -1; });
  roving(links[0]);
  bar.addEventListener("keydown", (e) => {
    const i = links.indexOf(e.target);
    if (i < 0 || e.altKey || e.ctrlKey || e.metaKey) return;
    let to = null;
    if (e.key === "ArrowRight") to = links[Math.min(i + 1, links.length - 1)];
    else if (e.key === "ArrowLeft") to = links[Math.max(i - 1, 0)];
    else if (e.key === "Home") to = links[0];
    else if (e.key === "End") to = links[links.length - 1];
    if (!to) return; // Enter is the link's own: it jumps to the letter
    e.preventDefault();
    roving(to);
    to.focus();
  });
  bar.addEventListener("focusin", (e) => { if (links.includes(e.target)) roving(e.target); });
}

export default {
  mount(host) {
    if (host.querySelector("input.zl-gloss-filter")) return; // already mounted
    const doc = host.ownerDocument;
    const entries = Array.from(host.querySelectorAll(".zl-gloss-entry"));
    const tools = host.querySelector(".zl-gloss-tools");
    if (entries.length === 0 || !tools) return;
    const bar = host.querySelector(".zl-gloss-bar");
    if (bar) letterBar(bar);

    const input = el(doc, "input", "zl-gloss-filter zen-field");
    input.type = "search";
    input.placeholder = "Filter terms";
    input.setAttribute("aria-label", "Filter terms");
    input.autocomplete = "off";
    // In the row with New Term…, before it; a page without the row (an older template) gets it under the bar.
    const row = tools.querySelector(".zl-gloss-row");
    if (row) row.insertBefore(input, row.firstChild);
    else tools.appendChild(input);

    const countEl = host.querySelector(".zl-gloss-count");
    if (countEl) { countEl.setAttribute("role", "status"); countEl.setAttribute("aria-live", "polite"); }
    const total = entries.length;
    const totalText = countEl ? countEl.textContent : "";
    const none = el(doc, "p", "zl-gloss-none", "No terms match.");
    none.hidden = true;
    const groups = Array.from(host.querySelectorAll(".zl-gloss-group"));
    (groups[groups.length - 1] || tools).after(none);
    const texts = entries.map((e) => fold(e.textContent.replace(/\s+/g, " ")));

    const apply = () => {
      const q = fold(input.value).trim().replace(/\s+/g, " ");
      let n = 0;
      entries.forEach((e, i) => {
        const hit = q === "" || texts[i].includes(q);
        e.hidden = !hit;
        if (hit) n++;
      });
      groups.forEach((g) => { g.hidden = !g.querySelector(".zl-gloss-entry:not([hidden])"); });
      none.hidden = n > 0;
      if (countEl) countEl.textContent = q === "" ? totalText : `${n} of ${total} terms`;
    };
    input.addEventListener("input", apply);
    input.addEventListener("keydown", (e) => {
      if (e.key === "Escape" && input.value !== "") {
        e.preventDefault();
        e.stopPropagation();
        input.value = "";
        apply();
      }
    });
  },
};
