import { expect, test } from "vitest";
import { OVERLAY_EVENT, registerOverlay } from "./overlay";

test("registerOverlay: announces itself, closes on another owner's event, not on its own, and stops when removed", () => {
  let closed = 0;
  const seen: string[] = [];
  const on = (e: Event) => seen.push((e as CustomEvent).detail.owner);
  document.addEventListener(OVERLAY_EVENT, on);
  const off = registerOverlay(document, () => closed++);
  expect(seen).toHaveLength(1);
  document.dispatchEvent(new CustomEvent(OVERLAY_EVENT, { detail: { owner: seen[0] } }));
  expect(closed).toBe(0);
  document.dispatchEvent(new CustomEvent(OVERLAY_EVENT, { detail: { owner: "other" } }));
  expect(closed).toBe(1);
  const off2 = registerOverlay(document, () => {});
  expect(seen[1]).not.toBe(seen[0]); // a new owner every time
  off();
  off2();
  document.dispatchEvent(new CustomEvent(OVERLAY_EVENT, { detail: { owner: "other" } }));
  expect(closed).toBe(1);
  document.removeEventListener(OVERLAY_EVENT, on);
});
