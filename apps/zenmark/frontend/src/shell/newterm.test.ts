// @vitest-environment happy-dom
import { beforeEach, expect, test, vi } from "vitest";
import { createFakeApi, type FakeApi } from "../api/fake";
import { openNewTerm } from "./newterm";

const SOURCES = {
  "README.md": "# Home\n",
  "glossary/index.md": "---\ntitle: Glossary\nlayout: glossary\n---\nWords.\n",
  "glossary/slug.md": "---\nlayout: term\n---\n# Slug\n\nA short name.\n",
};
let api: FakeApi;
beforeEach(() => {
  document.body.replaceChildren();
  api = createFakeApi({ sources: { ...SOURCES } });
});
const settle = async () => { for (let i = 0; i < 5; i++) await new Promise((r) => setTimeout(r, 0)); };

function open(folder = "glossary") {
  const onCreated = vi.fn();
  const onClose = vi.fn();
  const d = openNewTerm({ api, doc: document, folder, onCreated, onClose });
  const q = <T extends Element = HTMLElement>(s: string) => d.el.querySelector<T>(s)!;
  const submit = async () => { q<HTMLFormElement>("form").requestSubmit(); await settle(); };
  return { d, q, submit, onCreated, onClose };
}

test("asks for the term, as a dialog like New Page", () => {
  const { q } = open();
  expect(q("[role=dialog]").getAttribute("aria-modal")).toBe("true");
  expect(q("h2").textContent).toBe("New term");
  const input = q<HTMLInputElement>("input.zen-newpage-name");
  expect(document.activeElement).toBe(input);
  expect(q(".zen-newpage-hint").textContent).toContain("glossary");
});

test("creates <folder>/<slug>.md through the api and reports the path", async () => {
  const { q, submit, onCreated, onClose } = open();
  q<HTMLInputElement>("input.zen-newpage-name").value = "Éclair au café";
  await submit();
  expect(api.callsTo("newTerm")).toEqual([["glossary", "Éclair au café"]]);
  expect(onCreated).toHaveBeenCalledWith("glossary/eclair-au-cafe.md");
  expect(onClose).toHaveBeenCalled();
  expect(document.querySelector(".zen-newpage-backdrop")).toBeNull();
  expect(await api.source("glossary/eclair-au-cafe.md")).toContain("# Éclair au café");
});

test("an empty term says so and creates nothing", async () => {
  const { q, submit, onCreated } = open();
  q<HTMLInputElement>("input.zen-newpage-name").value = "   ";
  await submit();
  expect(q(".zen-newpage-error").textContent).not.toBe("");
  expect(api.callsTo("newTerm")).toHaveLength(0);
  expect(onCreated).not.toHaveBeenCalled();
});

test("a term that exists is said so and offered to open", async () => {
  const { q, submit, onCreated, onClose } = open();
  q<HTMLInputElement>("input.zen-newpage-name").value = "Slug";
  await submit();
  expect(onCreated).not.toHaveBeenCalled();
  expect(q(".zen-newpage-error").textContent).toContain("glossary/slug.md");
  expect(q(".zen-newpage-error").textContent).toContain("already exists");
  const create = q<HTMLButtonElement>("button[type=submit]");
  expect(create.textContent).toBe("Open it");
  expect(await api.source("glossary/slug.md")).toContain("A short name.");
  create.click();
  await settle();
  expect(onCreated).toHaveBeenCalledWith("glossary/slug.md");
  expect(onClose).toHaveBeenCalled();
});

test("editing the term after the notice goes back to Create", async () => {
  const { q, submit } = open();
  const input = q<HTMLInputElement>("input.zen-newpage-name");
  input.value = "Slug";
  await submit();
  input.value = "Slugs";
  input.dispatchEvent(new Event("input", { bubbles: true }));
  expect(q<HTMLButtonElement>("button[type=submit]").textContent).toBe("Create");
  expect(q(".zen-newpage-error").textContent).toBe("");
});

test("Esc and Cancel close it", () => {
  const { onClose } = open();
  document.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true }));
  expect(onClose).toHaveBeenCalledTimes(1);
  expect(document.querySelector(".zen-newpage-backdrop")).toBeNull();
});
