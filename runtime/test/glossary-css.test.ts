import { readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { expect, test } from "vitest";

// The glossary's letter bar is one row (the letters share the width and shrink), and the filter and New Term…
// share the row under it. Read without a DOM, so the file path resolves.
const css = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), "../../builtins/layouts/glossary/glossary.css"), "utf8");
const rule = (sel: string) => css.match(new RegExp(`\\.${sel}\\s*\\{[^}]*\\}`))?.[0] ?? "";

test("the letter bar is one row: no wrap, letters share the width", () => {
  expect(rule("zl-gloss-bar")).toMatch(/flex-wrap:\s*nowrap/);
  expect(rule("zl-gloss-letter")).toMatch(/flex:\s*1 1 0/);
  expect(rule("zl-gloss-letter")).toMatch(/min-width:\s*0/);
  expect(rule("zl-gloss-row")).toMatch(/display:\s*flex/);
});
