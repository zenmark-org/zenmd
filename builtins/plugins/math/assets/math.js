// Draws .zm-tex placeholders with the vendored KaTeX bundle, which is loaded
// lazily as a classic script (a UMD build that defines globalThis.katex). The
// stylesheet and its fonts are listed by the plugin, so they come with the page.
export const KATEX_URL = "/_plugins/math/katex.min.js";
let loading = null;

// What KaTeX may do with the page's text: no trust (no \href, \url, \includegraphics), warnings
// for what it cannot do, errors drawn by us under the TeX, and a bounded macro expansion.
const OPTIONS = { displayMode: true, throwOnError: false, trust: false, strict: "warn", maxExpand: 1000, output: "htmlAndMathml" };

/** Loads KaTeX once; resolves with globalThis.katex. */
export function loadKatex(doc = document, g = globalThis, url = KATEX_URL) {
  if (g.katex) return Promise.resolve(g.katex);
  if (loading && loading.doc === doc) return loading.p;
  doc.head.querySelectorAll("script").forEach((s) => s.dataset && s.dataset.zenMath !== undefined && s.remove());
  const p = new Promise((resolve, reject) => {
    const s = doc.createElement("script");
    s.src = url;
    s.dataset.zenMath = "";
    s.addEventListener("load", () => (g.katex ? resolve(g.katex) : reject(new Error("KaTeX did not load"))));
    s.addEventListener("error", () => reject(new Error("could not load KaTeX")));
    doc.head.append(s);
  });
  loading = { doc, p };
  p.catch(() => { if (loading && loading.p === p) loading = null; });
  return p;
}

/** A display equation of several lines (\\) goes in aligned, as MyST and Sphinx do, unless it has its own environment. */
export function wrapTex(tex, nowrap) {
  if (nowrap || /\\begin\s*\{/.test(tex) || !/\\\\/.test(tex)) return tex;
  return "\\begin{aligned}" + tex + "\\end{aligned}";
}

function nowrapped(el) {
  for (let n = el; n; n = n.parentNode) if (String(n.className || "").split(/\s+/).includes("zm-nowrap")) return true;
  return false;
}

function showError(host, doc, msg) {
  const box = doc.createElement("div");
  box.className = "zen-plugin-error";
  box.setAttribute("role", "status");
  box.textContent = "math: " + msg;
  host.append(box);
  // The Console shows the page's console, so a broken equation leaves a line there too.
  const first = ("math: " + msg).split("\n")[0];
  console.warn(first.length > 200 ? first.slice(0, 199) + "…" : first);
}

/** Draws one equation (once): the rendered math in place of the TeX, or the TeX with the message under it. */
export function renderOne(pre, katex) {
  if (pre.dataset.rendered !== undefined) return;
  pre.dataset.rendered = "";
  const doc = pre.ownerDocument || document;
  const row = pre.parentNode;
  const host = (row && row.parentNode) || row;
  const box = doc.createElement("div");
  box.className = "zm-eq";
  try {
    katex.render(wrapTex(pre.textContent, nowrapped(pre)), box, OPTIONS);
  } catch (err) {
    showError(host, doc, err && err.message ? err.message : String(err));
    return;
  }
  // With throwOnError false KaTeX draws a parse error as a .katex-error span whose title is the message.
  const bad = box.querySelector(".katex-error");
  if (bad) {
    showError(host, doc, bad.getAttribute("title") || "could not draw the equation");
    return;
  }
  pre.replaceWith(box);
}

/** Draws every not-yet-drawn equation under root. */
export function renderAll(root, katex) {
  for (const pre of root.querySelectorAll(".zm-tex:not([data-rendered])")) renderOne(pre, katex);
}

// The page runtime mounts this on every [data-zen-plugin="math"] wrapper.
export default {
  async mount(wrapper, ctx) {
    const doc = wrapper.ownerDocument || document;
    // ctx.assetUrl finds the bundle in the app and in a site made by `zenmd build`.
    const url = ctx && typeof ctx.assetUrl === "function" ? ctx.assetUrl("katex.min.js") : KATEX_URL;
    let katex;
    try {
      katex = await loadKatex(doc, globalThis, url);
    } catch (err) {
      for (const pre of wrapper.querySelectorAll(".zm-tex:not([data-rendered])")) {
        pre.dataset.rendered = "";
        showError((pre.parentNode && pre.parentNode.parentNode) || pre.parentNode, doc, err.message);
      }
      return;
    }
    renderAll(wrapper, katex);
  },
};
