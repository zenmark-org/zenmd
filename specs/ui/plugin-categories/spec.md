# ui/plugin-categories

| | |
|---|---|
| Status | specified |
| Target version | 0.8.0 (MINOR: a manifest field and a grouped list; nothing changes for pages) |
| Branch | `feature/ui-plugin-categories` |
| Wave | 63 |
| Depends on | core/plugins, core/extplugins, ui/plugins (Settings › Plugins), ui/settings-layout |
| Owns | `engine/plugins/category.go` (new) + `category_test.go`, the `Category` field in `engine/plugins/plugins.go` (`Manifest`), the `category` key in `engine/extplugins/manifest.go` + its tests, the `Category` field and its sort in `engine/session/extplugins.go` + tests, `internal/cli/plugins.go` (new) + `plugins_test.go`, the `Category:` line in each `builtins/plugins/*/<id>.go` manifest (one line each), `kit/ui/src/shell/settings.ts` (`SettingsItem.group` and the list's group headings) + `settings.test.ts`, `apps/zenmark/frontend/src/shell/pluginhelp.ts` (the grouped items) + its tests, this spec |

## Purpose

Twenty built-in plugins, plus any a folder adds, are one flat list today:
`aside, callout, cards, check, codeblock…`. Each plugin now says what it is for,
in one of a few fixed categories. Settings › Plugins, `zenmd plugins`, the
reference docs and the demo group them by category, so a reader looking for
"something for diagrams" or "something interactive" finds it at a glance.

## Behaviour

### The categories (engine)

1. A plugin has one category, from a fixed list, in this order:

   | id | Shown as | Built-ins |
   |---|---|---|
   | `text` | Text and notes | callout, aside, quote, text |
   | `layout` | Layout | tabs, dropdown, cards |
   | `figures` | Figures and data | figure, table, math, diagram, lini |
   | `code` | Code | codeblock, sandbox |
   | `links` | Links and reuse | include, xref, terms |
   | `interactive` | Interactive | check, quiz, rating |
   | `other` | Other | (external plugins that name none) |

   The list lives in one place, `plugins.Categories`, with each category's
   id, its label and its one-line description.
2. `plugins.Manifest` gains `Category string`. Every built-in sets it.
   `plugins.CategoryOf(m)` returns it, or `other` when it is empty or not in the
   list.
3. An external plugin's `plugin.yml` may set `category: <id>`. An unknown id is
   not an error: the plugin loads, goes under Other, and Settings shows its
   problem line "unknown category "x"; it is listed under Other", as it shows
   other manifest notes.
4. A registry test checks every built-in sets a category from the list. A
   new built-in without one fails the build, so the list never silently grows
   an Other group.

### Settings › Plugins (app)

5. The left column keeps "All plugins" at the top, then the plugins grouped by
   category in the order of behaviour 1.
   - Each group has a small heading.
   - Within a group the plugins sort by id.
   - A group with no plugins has no heading.
6. The group headings use the frame's existing `.zen-settings-list-heading`,
   the style Reader's "Markdown" heading already uses.
   - `SettingsItem` gains `group?: string`, and the frame draws a heading
     before the first row of each new group.
   - Headings are `role="presentation"` and are skipped by ↑/↓, as Reader's
     is.
   - This is a change to the shared frame, so Layouts or Templates can group
     their rows later with no new code.
7. A plugin's right column shows its category under its title, as a line
   beside the version and the source ("1.0.0 · Built-in · Figures and data").
8. "All plugins" lists the plugins grouped the same way.
9. Folder and user plugins sit in their category with their existing source
   note ("Folder", "Yours"), not in a separate group.

### The CLI (CLI first)

10. `zenmd plugins [folder] [--json]` lists the plugins a folder renders with:
    built-ins, then the folder's and the user's, grouped by category. Each
    line has the id, on or off for this folder, the source, and the
    description:

    ```text
    Figures and data
      diagram   on   built-in   Mermaid diagrams
      lini      off  built-in   Lini figures: flows, mindmaps, sequences, charts
    ```

    - `--json` prints the list with `category` on each plugin, the same data
      Settings shows.
    - A folder that cannot be opened exits 1.
    - An unknown flag exits 2.
    - It is the CLI twin of Settings › Plugins: there is no config key,
      because a category is the plugin's own fact, not an option.

### Docs

11. The docs and the demo follow the same order with a heading per category:
    - `docs/public/zenmd/reference/builtins.md`, the Plugins section;
    - `examples/demo/plugins/index.md`.

    A test reads both and checks that every built-in appears under its
    category, so the docs cannot drift (Integration).
12. `docs/wiki/PLUGINS.md` documents `Category` and the `category:` key with
    the list (Integration).

## Architecture

- **`engine/plugins/category.go`** holds:
  - `type Category struct{ ID, Label, Description string }`;
  - `var Categories = []Category{…}` (ordered);
  - `func CategoryOf(m Manifest) string`;
  - `func CategoryLabel(id string) string`.

  It has no imports beyond the package.
- **`engine/extplugins/manifest.go`** reads `category`, keeps an unknown
  value as `other`, and adds the note to the plugin's problems as other
  soft manifest warnings are.
- **`engine/session/extplugins.go`**: `PluginInfo` gains `Category string
  json:"category"`. `Session.Plugins()` sorts by category order, then id, so
  the app and the CLI never sort twice.
- **`internal/cli/plugins.go`** follows `internal/cli/check.go`: it opens a
  headless session and prints `Session.Plugins()`.
- **`kit/ui/src/shell/settings.ts`**: in the left-column renderer, when
  `item.group` differs from the previous row's group, it appends a
  `.zen-settings-list-heading` (as `section.heading` does). Keyboard
  navigation already skips non-option rows.
- **`apps/zenmark/frontend/src/shell/pluginhelp.ts`**: `items()` sets `group`
  from `CategoryLabel`; the right column adds the category to its meta line.
  The label list comes from the API, `PluginInfo.categoryLabel`, so the
  frontend never keeps its own copy of the list.

## Interfaces

```go
// engine/plugins
type Category struct{ ID, Label, Description string }
var Categories []Category              // text, layout, figures, code, links, interactive, other
func CategoryOf(m Manifest) string     // m.Category if listed, else "other"
func CategoryLabel(id string) string   // "Figures and data"; "Other" for anything unknown
// Manifest gains:
Category string // one of Categories' ids; "" is "other"

// engine/session
// PluginInfo gains:
Category      string `json:"category"`
CategoryLabel string `json:"categoryLabel"`
```

```yaml
# plugin.yml (external plugins)
category: figures
```

```text
zenmd plugins [folder] [--json]
```

```ts
// kit/ui/src/shell/settings.ts
export type SettingsItem = { id: string; label: string; note?: string; disabled?: boolean; tip?: string; group?: string };
// kit/ui/src/host.ts and the app api (Integration): PluginInfo gains category: string; categoryLabel: string
```

## Tests

| # | Behaviour | Test |
|---|---|---|
| 1 | the fixed list, its order and labels | `engine/plugins/category_test.go: TestCategoriesOrder`, `TestCategoryLabel` |
| 2 | `CategoryOf` falls back to other | `TestCategoryOfUnknownAndEmpty` |
| 3 | `category:` in plugin.yml; unknown is a note, not an error | `engine/extplugins/manifest_test.go: TestManifestCategory`, `TestManifestUnknownCategory` |
| 4 | every built-in has a listed category | `builtins/builtins_test.go: TestEveryBuiltinHasACategory` (Integration: the file is shared) |
| 5, 9 | grouped and sorted: category order, then id; external plugins in their category | `engine/session/extplugins_test.go: TestPluginsSortedByCategory` |
| 5–6 | group headings in the frame, skipped by the arrow keys, none for an empty group | `kit/ui/src/shell/settings.test.ts: group headings`, `arrow keys skip group headings` |
| 7–8 | the category in the meta line; All plugins grouped | `apps/zenmark/frontend/src/shell/settings.test.ts: plugins grouped by category`, `category in the meta line` |
| 10 | `zenmd plugins` text and JSON, exit codes | `internal/cli/plugins_test.go: TestPluginsText`, `TestPluginsJSON`, `TestPluginsErrors` |
| 11 | docs and demo follow the categories | `builtins/docs_test.go: TestDocsGroupedByCategory` (Integration) |

Edge cases to cover:
- an external plugin with `category: Figures` (case), which is accepted as
  `figures`;
- a plugin with `category: ""`;
- a folder plugin with the same category as built-ins;
- a build that leaves out every plugin of a category (no heading);
- a single-file window (Settings › Plugins still lists them).

## UI

**UI consistency:**
- The Settings frame is unchanged: items on the left, details on the right,
  empty text on the left.
- The group headings reuse `.zen-settings-list-heading`, the existing style
  for a heading over the left column. The rows are unchanged.
- The category in the right column joins the existing meta line; it is not
  a new badge.

**States:** unchanged. Empty: "No plugins in this build". Loading and error
keep their status lines.

**Keyboard and accessibility:**
- ↑/↓ move over plugins only; headings are presentation.
- A screen reader hears the group heading as text before its first row.
- Tokens only; no motion.

## Security

None new. A category is a label. It never changes what a plugin may do, its
sandbox, or its trust prompt. An external plugin's `category` is read as data,
and an unknown value is clamped to `other`. The CLI prints the same data
Settings shows and opens the folder read-only.

## Notes

- **Out of scope:**
  - filtering or searching the list;
  - a category for layouts or dialects (the frame's `group` makes that easy
    later);
  - categories in the Inspector's Build tab, which stays a count.
- **Integration (done by the lead at merge):**
  - `kit/ui/src/host.ts` and `apps/zenmark/frontend/src/api/{index,fake,wails}.ts`:
    `category` and `categoryLabel` on `PluginInfo`, with the fake's plugins
    given categories.
  - `internal/cli/cli.go`: `case flags && first && arg == "plugins": return runPlugins(args[i+1:], cwd, stdout, stderr)`,
    and a help line.
  - `builtins/builtins_test.go`: `TestEveryBuiltinHasACategory`.
  - `builtins/docs_test.go`: `TestDocsGroupedByCategory`.
  - `docs/public/zenmd/reference/builtins.md` and
    `examples/demo/plugins/index.md`: one heading per category, the rows
    moved under it.
  - `docs/public/zenmd/reference/cli.md`: `zenmd plugins`.
  - `docs/wiki/PLUGINS.md`: the `Category` field, `category:` and the list.
  - `docs/wiki/UI.md`: one sentence that a Settings list may group its rows
    under headings.
  - Regenerate the Wails bindings, and the demo golden if the demo changes.
- **App Store / sandbox:** none.
- **Changelog (Added):** Plugins have a category: Text and notes, Layout,
  Figures and data, Code, Links and reuse, Interactive. Settings › Plugins
  and the new `zenmd plugins` list them grouped, and an external plugin sets
  its own with `category:` in `plugin.yml` (ui/plugin-categories).

## Prompt

> Implement `specs/ui/plugin-categories/spec.md` following the rules in
> `specs/README.md`. You work only in the worktree
> `/Users/dragosh/Developer/zen-worktrees/ui-plugin-categories` on branch
> `feature/ui-plugin-categories`, and only in the paths under *Owns*.
>
> **Context:**
> - The Settings frame is `kit/ui/src/shell/settings.ts`. `section.heading`
>   already draws one `.zen-settings-list-heading`; generalise it to
>   `item.group`.
> - Settings › Plugins is `apps/zenmark/frontend/src/shell/pluginhelp.ts`
>   (`createPluginsSection`, `items()`).
> - `PluginInfo` and `pluginInfo(m)` are in `engine/session/extplugins.go`.
> - The external manifest is `engine/extplugins/manifest.go` (`rawManifest`).
> - The CLI command follows `internal/cli/check.go`. Its dispatch line in
>   `cli.go` is Integration.
> - Each built-in's manifest is in `builtins/plugins/<id>/<id>.go`; add one
>   `Category:` line per the table in behaviour 1.
>
> **Rules:**
> - tests first, committed failing;
> - no new dependencies or downloads;
> - never launch the app, run `task install`, `task package` or
>   `task release`, start servers, push, or touch `main`;
> - run go commands with a throwaway `HOME` and `XDG_CONFIG_HOME`, keeping
>   the real `GOMODCACHE`, `GOPATH` and `GOCACHE`.
>
> **Done when:**
> - every test in the table passes;
> - `go vet ./... && go test ./...` passes;
> - `cd apps/zenmark/frontend && npx tsc --noEmit && npx vitest run` passes;
> - your report lists the Integration lines and anything unverified (how the
>   grouped list looks in the app).
