// The Lini compiler in a module worker (see ../VENDOR.md), so a big figure
// never blocks the page. It only receives text and answers text:
//   in:  { type: "init", module, wasm } | { type: "compile", id, src }
//   out: { type: "ready" } | { type: "init-error", message } | { id, svg } | { id, error }
// It loads nothing but the two URLs it is sent, and only from its own origin.

const message = (err) => String((err && err.message !== undefined ? err.message : err)).replace(/^Error:\s*/, "");

/** Installs the handler on scope; load(url) imports the compiler module (a test seam). */
export function serve(scope, load = (u) => import(/* @vite-ignore */ u)) {
  let compile = null;
  const here = scope.location && scope.location.href ? new URL(scope.location.href) : null;
  const own = (u) => {
    if (!here) return true;
    const url = new URL(String(u), here); // a relative url is the plugin's own
    return url.origin === here.origin && (url.protocol === "http:" || url.protocol === "https:");
  };
  scope.onmessage = async (ev) => {
    const m = ev.data || {};
    if (m.type === "init") {
      try {
        if (!own(m.module) || !own(m.wasm)) throw new Error("the worker loads only the plugin's own files");
        const mod = await load(m.module);
        await mod.default({ module_or_path: m.wasm });
        compile = (src) => String(mod.compile(src, undefined));
        scope.postMessage({ type: "ready" });
      } catch (err) {
        scope.postMessage({ type: "init-error", message: message(err) });
      }
    } else if (m.type === "compile") {
      if (!compile) { scope.postMessage({ id: m.id, error: "the compiler is not loaded" }); return; }
      try {
        scope.postMessage({ id: m.id, svg: compile(String(m.src)) });
      } catch (err) {
        scope.postMessage({ id: m.id, error: message(err) });
      }
    }
  };
}

if (typeof WorkerGlobalScope !== "undefined" && typeof self !== "undefined" && self instanceof WorkerGlobalScope) serve(self);
