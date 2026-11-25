// Draws .zen-lini placeholders with the vendored Lini compiler (WebAssembly,
// see ../VENDOR.md). The compiler loads once per page, the first time a block
// mounts, and only from the plugin's own asset URL.
export const LINI_MODULE = "/_plugins/lini/lini_wasm.js";
const LINI_WASM = "/_plugins/lini/lini_wasm_bg.wasm";
const MAX_SOURCE = 200 * 1024; // bytes of source compiled in the page

const loads = new Map(); // module url -> Promise<{ compile }>
let importer = (u) => import(/* @vite-ignore */ u);

/** A seam for tests: replaces the dynamic import of the compiler module. */
export function setImporter(fn) {
  importer = fn;
}

/**
 * Loads the compiler once per module url. The module's default export is its
 * init, given the explicit url of the .wasm file. Resolves with { compile(src) }
 * whatever the package's own call looks like.
 */
export function loadLini(moduleUrl = LINI_MODULE, wasmUrl = LINI_WASM) {
  const key = moduleUrl + "\n" + wasmUrl;
  let p = loads.get(key);
  if (p) return p;
  p = (async () => {
    const mod = await importer(moduleUrl);
    await mod.default({ module_or_path: wasmUrl });
    return { compile: (src) => String(mod.compile(src, undefined)) };
  })();
  loads.set(key, p);
  p.catch(() => { if (loads.get(key) === p) loads.delete(key); });
  return p;
}

const DROP = new Set(["script", "foreignobject", "iframe", "object", "embed"]);

function isLocalRef(v) {
  return typeof v === "string" && v.trim().startsWith("#");
}

/**
 * Parses SVG text as an XML document and removes everything that could run
 * or fetch: scripts, foreignObject, frames, event attributes, links that leave
 * the figure and animations of links. Returns the cleaned root <svg>, still in
 * its own document (insert it with importNode). Throws on a parse error.
 */
export function cleanSvg(svgText, doc = document) {
  const Parser = (doc.defaultView && doc.defaultView.DOMParser) || globalThis.DOMParser;
  const parsed = new Parser().parseFromString(String(svgText), "image/svg+xml");
  const root = parsed.documentElement;
  if (!root || root.localName.toLowerCase() === "parsererror" || parsed.getElementsByTagName("parsererror").length > 0) {
    throw new Error("the figure is not valid SVG");
  }
  if (root.localName !== "svg") throw new Error("the figure is not an SVG");
  const all = [root, ...root.querySelectorAll("*")];
  for (const el of all) {
    if (el !== root && !el.isConnected && !root.contains(el)) continue; // already removed with a parent
    const name = el.localName.toLowerCase();
    if (el !== root && DROP.has(name)) { el.remove(); continue; }
    if (el !== root && (name === "animate" || name === "set" || name === "animatetransform" || name === "animatemotion")) {
      const target = (el.getAttribute("attributeName") || "").toLowerCase();
      if (target === "href" || target === "xlink:href" || target.endsWith(":href")) { el.remove(); continue; }
    }
    for (const a of [...el.attributes]) {
      const an = a.name.toLowerCase();
      if (an.startsWith("on")) el.removeAttribute(a.name);
      else if (an === "href" || an === "xlink:href" || a.localName === "href") {
        if (!isLocalRef(a.value)) el.removeAttribute(a.name);
      }
    }
    if (el !== root && name === "use" && ![...el.attributes].some((a) => a.localName === "href")) el.remove();
  }
  // A figure's stylesheet scopes itself to the figure; its :root rule would style the page.
  for (const st of root.querySelectorAll("style")) st.textContent = (st.textContent || "").replace(/:root\b/g, ".zen-lini-svg");
  return root;
}

function appScheme(doc) {
  const t = doc.documentElement && doc.documentElement.dataset.theme;
  if (t === "dark" || t === "light") return t;
  const w = doc.defaultView;
  const dark = !!(w && w.matchMedia && w.matchMedia("(prefers-color-scheme: dark)").matches);
  return dark ? "dark" : "light";
}

/** Sets color-scheme on el's .zen-lini-svg (or on el itself) from the app's theme, else the system's. */
export function linkScheme(el, doc = el.ownerDocument || document) {
  const box = (el.querySelector && el.querySelector(".zen-lini-svg")) || el;
  box.style.colorScheme = appScheme(doc);
}

const watched = new WeakSet();
/** Keeps every figure of doc in step with the theme while the page is open. */
export function watchTheme(doc = document) {
  if (watched.has(doc)) return;
  watched.add(doc);
  const apply = () => doc.querySelectorAll(".zen-lini").forEach((el) => linkScheme(el, doc));
  const w = doc.defaultView;
  if (w && w.MutationObserver) new w.MutationObserver(apply).observe(doc.documentElement, { attributes: true, attributeFilter: ["data-theme"] });
  const mq = w && w.matchMedia && w.matchMedia("(prefers-color-scheme: dark)");
  if (mq && mq.addEventListener) mq.addEventListener("change", apply);
}

function showError(el, msg) {
  el.dataset.drawn = ""; // the source and the message show instead of the placeholder
  const box = (el.ownerDocument || document).createElement("div");
  box.className = "zen-plugin-error";
  const first = ("lini: " + msg).split("\n")[0];
  box.textContent = first;
  el.append(box);
  // The Console shows the page's console, so a broken figure leaves a line there too.
  console.warn(first.length > 200 ? first.slice(0, 199) + "…" : first);
}

function messageOf(err) {
  const m = err && err.message !== undefined ? err.message : String(err);
  return String(m).replace(/^Error:\s*/, "");
}

/** Draws one .zen-lini wrapper (once): the svg beside the source, or the error under it. */
export async function renderOne(el, lini) {
  if (el.dataset.rendered !== undefined) return;
  el.dataset.rendered = "";
  const doc = el.ownerDocument || document;
  const pre = el.querySelector("pre");
  if (!pre) { el.dataset.drawn = ""; return; }
  const src = pre.textContent || "";
  if (src.trim() === "") { el.dataset.drawn = ""; return; } // nothing to draw: the empty block shows as it is
  if (src.length > MAX_SOURCE) {
    showError(el, "figure too large");
    return;
  }
  let root;
  try {
    root = cleanSvg(lini.compile(src), doc);
  } catch (err) {
    showError(el, messageOf(err));
    return;
  }
  const box = doc.createElement("div");
  box.className = "zen-lini-svg";
  box.setAttribute("role", "img");
  const title = root.querySelector("title");
  box.setAttribute("aria-label", (title && title.textContent.trim()) || "Lini figure");
  box.append(doc.importNode(root, true));
  const details = doc.createElement("details");
  details.className = "zen-lini-source";
  const summary = doc.createElement("summary");
  summary.textContent = "Source";
  const copy = doc.createElement("pre");
  copy.textContent = src;
  details.append(summary, copy);
  pre.remove();
  el.append(box, details);
  linkScheme(el, doc);
  box.setAttribute("data-zen-zoomable", ""); // the page runtime adds zoom and pan to a drawn figure (plugins/diagrams)
  el.dataset.drawn = ""; // the placeholder of page.css gives way to the figure
}

/** Draws every not-yet-drawn figure under root. */
export async function renderAll(root, lini) {
  for (const el of root.querySelectorAll(".zen-lini:not([data-rendered])")) await renderOne(el, lini);
}

// The page runtime mounts this on every [data-zen-plugin="lini"] wrapper.
export default {
  async mount(wrapper, ctx) {
    const el = (String(wrapper.className || "").split(/\s+/).includes("zen-lini") ? wrapper : wrapper.querySelector(".zen-lini")) || wrapper;
    const doc = el.ownerDocument || document;
    // ctx.assetUrl finds the files in the app and in a site made by `zenmd build`.
    const has = ctx && typeof ctx.assetUrl === "function";
    const moduleUrl = has ? ctx.assetUrl("lini_wasm.js") : LINI_MODULE;
    const wasmUrl = has ? ctx.assetUrl("lini_wasm_bg.wasm") : LINI_WASM;
    let lini;
    try {
      lini = await loadLini(moduleUrl, wasmUrl);
    } catch (err) {
      if (el.dataset.rendered === undefined) {
        el.dataset.rendered = "";
        showError(el, "could not load the Lini compiler");
      }
      return;
    }
    watchTheme(doc);
    await renderOne(el, lini);
  },
};
