# engine/blueprint-hierarchy

| | |
|---|---|
| Status | specified |
| Target version | 0.8.0 |
| Branch | `feature/engine-blueprint-hierarchy` (phase 1), then one branch per phase |
| Wave | 50–52 (before engine/template-catalog, whose packs take their `requires` from this) |
| Depends on | engine/templates-blueprints, core/layouts (a layout's `requires`), ui/settings-layout |
| Owns | `engine/sitetemplates/hierarchy.go` + test (new), `engine/sitetemplates/pages.go` + tests (the `plugins:` key), `builtins/pagetemplates/*.yml` + tests, `builtins/blueprints/**` + tests, `apps/zenmark/frontend/src/shell/templateshelp.ts` + test, `apps/zenmark/frontend/src/api/index.ts`, `apps/zenmark/frontend/src/api/fake.ts`, `apps/zenmark/frontend/src/api/wails.ts`, `docs/wiki/TEMPLATES.md`, `examples/demo/features/blueprints.md`, `examples/demo/templates/**` |

## Purpose

When you write pages in Zenmark, four things build on each other, from the
top down:

```text
Blueprint        a named collection of page structures: a folder's sections
  └ Page template  the structure of one kind of page, for one layout, with the plugins it uses
      └ Layout       how the page looks (visual rendering)
          └ Plugin     what a block does (functional)
```

Today the code skips levels. A blueprint gives layouts to its index and home
pages directly, and lists its layouts and plugins by hand. Page templates
don't say which plugins they use. Settings › Blueprints shows only a
blueprint's questions, not what it is made of. This spec makes every level
build only on the one below it, and shows the chain in Settings and the docs.

## Behaviour

### A. Page templates name their layout and plugins

1. **A `plugins:` key:** a page template may list the plugins its structure uses, beside its layout:
   ```yaml
   id: how-to
   layout: docs
   plugins: [codeblock]
   ```
   - The plugin ids follow the same rules as `requires.plugins`. An unknown key is still an error.
   - `layout:` stays optional; without it, the page uses the folder's default layout.
2. **What a template needs:** its `layout:`, plus that layout's own `requires` (core/layouts), plus its `plugins:`.
3. **Checking the built-in templates:** every directive or fence a built-in template's skeleton uses (`:::quiz`, ```` ```mermaid ````) must be in what it needs, or the test fails.

### B. Every page of a blueprint comes from a page template

4. **The index of a folder section** comes from a page template, never from a bare layout.
   - `index: { template: <id> }` is the way to say it. Without `index:`, it is the built-in **`section-index`** template (layout `landing`).
   - Its skeleton is:
     ```text
     # {{title}}

     {{tagline}}
     ```
     The tagline is filled from `index.tagline`; when the section gives none, the tagline line is dropped.
   - The links to the starting pages are added under it, as they are today.
5. **New built-in page templates** cover the pages blueprints made from bare layouts:
   | Template | Layout | For |
   |---|---|---|
   | `section-index` | `landing` | a section's index, with cards for its pages |
   | `glossary-index` | `glossary` | a glossary folder's index (`nav: { children: folder, sort: title }`) |
   | `releases-index` | `landing` | a releases folder's index, newest first (`nav: { children: folder, sort: newest-first }`) |
   | `home` | the folder's layout | a plain start page with links to each section, for a blueprint with no `home.template` |
6. **The old index keys still work, with a note:**
   - `index: { layout: glossary }` reads as `glossary-index`;
   - `index: { layout: landing }` reads as `section-index`.
   - Any other `index.layout` is still read as before, and `zenmd blueprint check` and Settings › Blueprints both say "Use a page template for the index: index: { template: … }".
7. **The home** follows `home.template`, and `home` when none is given. A page listed in `home.pages`, or in a section's `pages`, may name a `template:`. Its `layout:` still works, with the same note as behaviour 6 (it is how a slides page was made).
8. **What a blueprint needs** is worked out from its page templates:
   - the union of what each template it uses needs (behaviour 2), for every section, index, home and page;
   - plus the folder's `layout:`.
   - A `requires:` written in `blueprint.yml` is still read. When it lists less than the worked-out set, `zenmd blueprint check` and Settings say what's missing. When it lists more, they say nothing.
   - `kit/assembly` checks the worked-out set against the build (`TestAssemblyGenerated` and the assembly checks), so a built-in blueprint can't need something the build lacks.
9. **The built-in blueprints follow the model:**
   - **techdocs:** every index uses `index: { template: … }`; `requires:` is removed.
   - **learning and planner:** they become blueprints with sections, made of their own page templates, and no loose files to copy:
     - learning: `course` (layout `course`, plugins `quiz`, `check`) and `term`;
     - planner: `roadmap` (layout `wide`, plugin `diagram`), `week`, `decision` and `meeting`.
   - Each one's `version` goes up a minor step. A folder made with the new version has the same pages as before, and the tests compare them.

### C. Settings shows the chain

10. **Settings › Blueprints, the right column:**
    - After the version, description and questions, a **"Made of"** part lists, in sidebar order:
      - the home;
      - each section, with its index and its page template;
      - the root pages.
    - Each row reads `<section>  →  <page template>  →  <layout>`, with the plugins after it in the muted note style:
      ```text
      Home              overview       docs
      Getting started   tutorial       docs      codeblock
        index           section-index  landing
      Releases          release-notes  release
      Glossary          term           term
        index           glossary-index glossary
      ```
    - A page template's name is a link button that opens Settings › Templates on that template. A layout's name opens Settings › Layouts on that layout.
    - Under the rows, **"Needs"** lists the worked-out layouts and plugins. It also lists what this build lacks, marked "Not in this build", and the notes of behaviours 6 and 8.
11. **Settings › Templates, the right column:**
    - **"Layout"** names the template's layout as a link button to Settings › Layouts, followed by "and the plugins it needs: …" when the layout has `requires`.
    - **"Plugins"** lists the template's own `plugins:`, each a link button to Settings › Plugins.
    - **"Used by"** lists the blueprints that use the template, each a link button to Settings › Blueprints. The open folder's path rules stay under their own heading, as today.
12. **The order of the Settings sections** stays Blueprints, Templates, Layouts, Plugins, which matches the hierarchy top to bottom. A test now holds it.
13. **The same chain in the CLI:**
    - `zenmd blueprint show <id>` prints the "Made of" rows and "Needs";
    - `zenmd templates show <id>` prints Layout, Plugins and Used by.
    - Both take `--json`.

### D. The docs say it once

14. **One diagram and four one-line definitions:** `docs/wiki/TEMPLATES.md` opens with the diagram from Purpose and the four definitions:
    - **Blueprint:** a named collection of page structures;
    - **Page template:** the structure of one kind of page, for one layout, with the plugins it uses;
    - **Layout:** how a page looks;
    - **Plugin:** what a block does.
15. **The demo uses the same words, in the same order:**
    - Features › Blueprints says what a blueprint is made of;
    - Templates › Blueprints (the blueprint.yml reference) shows `index: { template: … }`, the `template:` key on pages, and `requires:` as worked out (writing it is optional);
    - Templates › Page templates gains `plugins:` beside `layout:`.

## Architecture

- **`engine/sitetemplates/hierarchy.go`** (new) holds the model.
  - `Resolve(t Template) Tree` walks a blueprint: home, sections, indexes and pages. For each, it finds its page template (the blueprint's own, then the copy on this Mac, then the built-in one, as `pageTemplate` does now).
  - Each node carries its page template, layout and plugins. `Tree.Needs` is their union, plus each layout's `requires`. Layout `requires` come from a `LayoutNeeds func(id string) []string` the caller passes (the session, from the layout registry; kit/assembly, from the catalog).
- **`generate`** in `sections.go` asks the tree which template each page uses. The index switch (template, glossary, landing) collapses into "render the page template's skeleton", and the old keys are mapped to template ids when parsing (behaviour 6).
- **`kit/assembly`** uses `Resolve` in place of `t.Requires` for blueprint needs. This is an Integration line: the lead edits `kit/assembly/catalog.go`.
- **The app** gains `Zen.BlueprintTree(id) (BlueprintTree, error)` and a `usedBy` field on page templates. The Settings sections render from those.

## Interfaces

```go
// engine/sitetemplates
type Node struct {
	Role     string   // "home" | "section" | "index" | "page"
	Title    string   // section or page title, with {{…}} left as written
	File     string   // the file it writes, relative to the new folder
	Template string   // page template id ("" only for a legacy layout-only page)
	Layout   string   // the template's layout, or the folder's
	Plugins  []string // the template's plugins
	Note     string   // "" or the advice of behaviours 6–8
	Children []Node   // a section's index and pages
}
type Tree struct {
	Nodes   []Node
	Needs   Needs    // layouts and plugins, worked out
	Written Needs    // what blueprint.yml's requires: says, if anything
	Missing Needs    // in Needs, not in Written (when Written is given)
	Notes   []string
}
func Resolve(t Template, layoutNeeds func(id string) []string) (Tree, error)

// PageTemplate gains:
//   Plugins []string
//   UsedBy  []string // blueprint ids, filled by the session
```

- **Service:** `BlueprintTree(id string) (BlueprintTree, error)`; `PageTemplates()` gains `plugins` and `usedBy`.
- **TS types:** `SiteTemplate.tree?: BlueprintTree`, `PageTemplate.plugins: string[]`, `PageTemplate.usedBy: string[]`.
- **CLI:** `zenmd blueprint show <id> [--json]` (new); `zenmd templates show <id>` gains Plugins and Used by.

## Tests

| # | Behaviour | Test |
|---|---|---|
| 1 | `plugins:` is read; a bad id or an unknown key is an error | `engine/sitetemplates/pages_test.go: TestPagePlugins` |
| 2 | A template needs its layout, the layout's requires and its plugins | `hierarchy_test.go: TestTemplateNeeds` |
| 3 | Every built-in template's skeleton directives and fences are in what it needs | `builtins/pagetemplates/pagetemplates_test.go: TestSkeletonPluginsDeclared` |
| 4 | A section with no `index:` gets `section-index`; tagline filled or the line dropped; links added | `hierarchy_test.go: TestDefaultIndex`, `sections_test.go` |
| 5 | The four new templates exist and parse | `pagetemplates_test.go: TestNewIndexTemplates` |
| 6 | `index.layout: glossary` and `landing` map to the templates; another layout gives the note | `TestLegacyIndexLayout` |
| 7 | The home uses `home.template` or `home`; a page's `template:` is used; a page `layout:` gives the note | `TestHomeAndPageTemplates` |
| 8 | Needs are the union; a written `requires` that lists less is reported as missing, one that lists more is silent | `TestNeedsWorkedOut` |
| 8 | kit/assembly uses the worked-out needs | `kit/assembly/assembly_test.go` (Integration) |
| 9 | techdocs, learning and planner make the same pages as before (file list and frontmatter), from page templates only, with no `requires:` and no notes | `builtins/blueprints/*_test.go: TestBuiltinsFollowTheModel` |
| 10 | Settings › Blueprints shows Made of rows in order, link buttons go to the right section and item, and Needs lists what this build lacks | `apps/zenmark/frontend/src/shell/templateshelp.test.ts` |
| 11 | Settings › Templates shows Layout (and its requires), Plugins and Used by, with link buttons | same file |
| 12 | Settings sections run Blueprints, Templates, Layouts, Plugins | `apps/zenmark/frontend/src/shell/settings.test.ts` |
| 13 | `zenmd blueprint show` and `templates show` print the chain, with `--json` | `internal/cli` and `apps/zenmark/sitetemplates_test.go` |
| 14–15 | The demo pages render with no warnings; the reference mentions `index: { template: … }` and `plugins:` | `engine/session/demo_test.go` |

**Edge cases:**
- a page template missing at every level, which is an error naming the section;
- a template whose layout isn't in this build;
- a blueprint with no sections (only files), which keeps working with a "Made of: files only" row;
- `{{…}}` in titles, shown as written;
- the same template used by several sections, listed once in Needs.

## UI

- **The pattern reused:** the Settings frame, with items on the left and details on the right.
  - **"Made of"** is a plain table in the right column, styled like the Fields table of Settings › Templates (`zen-plugin-section` heading, the same table class). The plugin names use the `--fg-muted` note style.
  - **The link buttons** are the quiet link style the Settings frame already uses to jump between sections (the MyST hint uses it to open Settings › Reader). They are keyboard reachable, and their names say where they go ("Open how-to in Templates").
- **States:**
  - **empty:** "Made of: files only";
  - **error:** the blueprint's error line, as today;
  - **loading:** the right column fills when the tree arrives, with no spinner (it is local and fast).
- **Look:** tokens only.

## Security

| Risk | Mitigation | Test |
|---|---|---|
| A page template found by id outside the template folders | id rules unchanged (`^[a-z][a-z0-9-]*$`), looked up only in the blueprint's `.zen/templates`, the pages folder and the built-ins | existing tests and `TestDefaultIndex` |
| Text from a blueprint or template shown in Settings | all of it goes in through `textContent`, as the section does now | `templateshelp.test.ts` |
| None new otherwise | the change is to what is derived and shown; nothing new runs | – |

## Phases

| Phase | Wave | What |
|---|---|---|
| 1 | 50 | engine: `plugins:` on page templates, `Resolve`, the four new templates, index and home from templates, legacy keys with notes, needs worked out; techdocs moved to the model; `zenmd blueprint show`, `templates show` additions |
| 2 | 51 | Settings › Blueprints "Made of" and "Needs", Settings › Templates Layout/Plugins/Used by, link buttons, the order test; the docs and demo pages |
| 3 | 52 | learning and planner rebuilt as section blueprints from their own page templates (`course`, `roadmap`, `week`, `decision`, `meeting`) |

## Notes

- **The words:** in the UI and the docs, a page template may be called a "page structure" in prose, but its name stays **page template** (Settings › Templates, `templates:` and `zenmd templates`), so nothing is renamed twice.
- **Integration (lead, at merge):**
  - `kit/assembly/catalog.go`: blueprint needs from `Resolve`;
  - `apps/zenmark/sitetemplates.go`: `BlueprintTree` and `usedBy`;
  - `apps/zenmark/bindings_test.go`: the new method;
  - `internal/cli/cli.go`: `blueprint show`;
  - `specs/engine/templates-blueprints/spec.md`: a pointer to this spec for the model;
  - `specs/engine/template-catalog/spec.md`: the packs' `requires` come from `Resolve`;
  - the demo golden files.
- **Changelog:**
  - phase 1: `Changed: blueprints are made of page templates all the way down: section indexes and the home come from page templates (section-index, glossary-index, releases-index, home), page templates list their plugins, and a blueprint's needs are worked out from them (zenmd blueprint show).`
  - phase 2: `Added: Settings › Blueprints shows what a blueprint is made of (sections, page templates, layouts, plugins); Settings › Templates shows each template's layout, plugins and the blueprints that use it.`
  - phase 3: `Changed: the Learning path and Planner blueprints are built from their own page templates.`

## Prompt

> Implement phase <N> of `specs/engine/blueprint-hierarchy/spec.md` following CLAUDE.md and docs/wiki/PLUGIN-UI.md, in the worktree `../zen-worktrees/<branch>` on branch `<branch>`.
> - Read `engine/sitetemplates` (`sections.go` `generate`, `pages.go`, `manifest.go`) first. Reuse the page template lookup; don't copy it.
> - Write the tests first and commit them failing.
> - Phase 1 must keep every page a fresh techdocs folder writes byte for byte, except where the spec changes it: index frontmatter now comes from templates. Show any difference in your report.
> - Only the Owns paths; anything in a shared file goes in your report as an Integration line.
> - No new dependencies or downloads. Don't launch the app, run task package/install/release, push or touch main. Never write to `~/Library/Application Support/zen`.
