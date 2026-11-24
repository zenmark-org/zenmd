// One view over the window at a time (plugins/table, shared with ui/image-zoom and plugin viewers).
// The contract is a document event, not a function: plugin JS can't import the runtime.

export const OVERLAY_EVENT = "zen:overlay-open";

let counter = 0;

/** Call when a view opens: sends OVERLAY_EVENT with a new owner, and calls close when another owner's event comes.
 *  Returns the remover; call it when the view closes. */
export function registerOverlay(doc: Document, close: () => void): () => void {
  const owner = `overlay-${++counter}-${Math.random().toString(36).slice(2, 8)}`;
  const on = (e: Event) => {
    if ((e as CustomEvent<{ owner?: string }>).detail?.owner !== owner) close();
  };
  doc.addEventListener(OVERLAY_EVENT, on);
  doc.dispatchEvent(new CustomEvent(OVERLAY_EVENT, { detail: { owner } }));
  return () => doc.removeEventListener(OVERLAY_EVENT, on);
}
