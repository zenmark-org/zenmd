// @vitest-environment happy-dom
import { beforeEach, expect, test, vi } from "vitest";
import { loadView, renderViewSwitch, saveView, type NavView } from "./viewswitch";

let host: HTMLElement;
beforeEach(() => { localStorage.clear(); document.body.replaceChildren(); host = document.createElement("div"); document.body.append(host); });

const radios = () => Array.from(host.querySelectorAll<HTMLElement>("[role=radio]"));

test("radiogroup with Pages and Files, in the Settings segment classes", () => {
  let v: NavView = "pages";
  renderViewSwitch(host, () => v, (n) => { v = n; });
  const group = host.querySelector("[role=radiogroup]")!;
  expect(group.classList.contains("zen-settings-segment")).toBe(true);
  expect(group.getAttribute("aria-label")).toBeTruthy();
  expect(radios().map((r) => r.textContent)).toEqual(["Pages", "Files"]);
  expect(radios().every((r) => r.classList.contains("zen-settings-segment-option"))).toBe(true);
  expect(radios().map((r) => r.getAttribute("aria-checked"))).toEqual(["true", "false"]);
  expect(radios().map((r) => r.tabIndex)).toEqual([0, -1]);
});

test("a click chooses; the arrow keys move between the options; sync follows the state", () => {
  let v: NavView = "pages";
  const set = vi.fn((n: NavView) => { v = n; });
  const sw = renderViewSwitch(host, () => v, set);
  radios()[1].click();
  expect(set).toHaveBeenLastCalledWith("files");
  expect(radios().map((r) => r.getAttribute("aria-checked"))).toEqual(["false", "true"]);
  radios()[1].dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowLeft", bubbles: true, cancelable: true }));
  expect(set).toHaveBeenLastCalledWith("pages");
  expect(document.activeElement).toBe(radios()[0]);
  radios()[0].dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowRight", bubbles: true, cancelable: true }));
  expect(v).toBe("files");
  v = "pages";
  sw.sync();
  expect(radios().map((r) => r.getAttribute("aria-checked"))).toEqual(["true", "false"]);
});

test("Space and Enter on a focused option choose it", () => {
  let v: NavView = "pages";
  renderViewSwitch(host, () => v, (n) => { v = n; });
  // a button: its own click (Enter and Space in the browser) does it
  radios()[1].focus();
  radios()[1].click();
  expect(v).toBe("files");
});

test("persists per root", () => {
  expect(loadView("/a")).toBe("pages");
  saveView("/a", "files");
  expect(loadView("/a")).toBe("files");
  expect(loadView("/b")).toBe("pages");
  expect(localStorage.getItem("zen.nav.view:/a")).toBe("files");
  localStorage.setItem("zen.nav.view:/c", "nonsense");
  expect(loadView("/c")).toBe("pages");
});

test("storage that throws is harmless", () => {
  const get = vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => { throw new Error("no"); });
  const set = vi.spyOn(Storage.prototype, "setItem").mockImplementation(() => { throw new Error("no"); });
  expect(loadView("/a")).toBe("pages");
  expect(() => saveView("/a", "files")).not.toThrow();
  get.mockRestore(); set.mockRestore();
});
