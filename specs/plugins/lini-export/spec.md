# plugins/lini-export

| | |
|---|---|
| Status | specified |
| Target version | 0.8.0 (MINOR: an add-on, a plugin option and an export flag; nothing changes without the add-on) |
| Branch | `feature/plugins-lini-export` |
| Wave | 66 |
| Depends on | plugins/lini, plugins/diagram-performance, kit/addons (the add-on contract, Vale's pattern), engine/export-cli-parity (`export-options.json`) |
| Owns | `addons/lini/` (new: `lini.go`, `lini_test.go`, build tag `addon_lini`), `apps/zenmark/addons_lini.go` (+ test, same tag), the static-figure hook in `builtins/plugins/lini/lini.go` (+ `lini_test.go`, `static.go`, `static_test.go` new), `builtins/plugins/lini/assets/lini.js` (a static figure is left alone) + `lini.test.ts`, `docs/public/zenmd/reference/plugins/lini.md` (a "Static figures" section), this spec |

## Purpose

Today a Lini figure is drawn by the reader's browser. An exported website
carries its source and the 2 MB compiler. A portable site, the default, can't
carry the compiler at all and shows the source.

With the `lini` add-on and the `lini` command-line tool installed:
- `zenmd export site` writes each figure as a finished SVG image, which works
  in portable sites, with scripts off, in feeds and in print;
- `zenmd check` and Problems list Lini's own errors and warnings, with their
  line numbers.

Without the add-on, or without the tool, nothing changes.

## Behaviour

### The add-on and the tool

1. `lini` is an add-on (kit/addons):
   - The Go package `addons/lini` carries the build tag `addon_lini`.
   - A build includes it with `ADDONS=lini`, alone or with Vale
     (`ADDONS=vale,lini`).
   - Its `init` registers `{ID: "lini", Name: "Lini figures", Summary: "Draws
     Lini figures as SVG when a site is exported, and checks them, with the
     lini tool"}`.
   - It installs the plugin's hooks of behaviours 4 and 8.
2. **Finding the tool.** The add-on looks for the `lini` binary in this order:
   - `$ZEN_LINI`;
   - a helper bundled in the app (`Contents/Helpers/lini`, a later
     `resources.yml` entry, out of scope here);
   - `PATH`.

   It runs `lini --version` once per process. It needs a 1.x version, the
   language the vendored compiler speaks: a missing or other major version
   counts as no tool. Settings › Add-ons shows the version found, or "lini
   not found: install it with `cargo install lini`".
3. **Never in the App Store build.** The App Store channel can't run outside
   programs. Its build never includes the add-on, and the build fails if
   `ADDONS` names it there.

### Static figures in exported sites

4. **The static hook.** With the add-on and the tool, the lini plugin's
   `RenderBlock` compiles the figure with the tool when the page is rendered
   for a website export (`Context.Output == "site"`, behaviour 7):
   - The command is `lini - --format svg`: the source on stdin, the SVG on
     stdout.
   - It runs with a 10 s timeout, in an empty working directory, with no
     environment beyond `PATH`.
   - The source is the block's own text or its file (`{file=…}`, read
     through `ctx.ReadFile` as now).
5. **The output** is an image, never inline SVG, because the page sanitiser
   forbids `<style>`, which every Lini SVG needs:

   ```html
   <div class="zen-lini zen-lini-static" data-drawn>
     <img src="data:image/svg+xml;base64,…" alt="…">
   </div>
   ```

   - `alt` is the figure's `<title>` if it has one, else "Lini figure".
   - The `data-drawn` mark means the "Drawing diagram…" placeholder never
     shows.
   - The figure's colours follow the reader's system light or dark mode,
     because `light-dark()` inside an image follows the system. A site
     exported with `--scheme light` or `dark` passes `--theme light` or
     `dark` to the tool, so the figure matches the site.
   - A click opens it in the image zoom, as any image does.
6. **The page loads nothing for static figures.** When every Lini figure of
   a page is static, the page lists no Lini script, and the compiler's files
   aren't copied into the site. `lini.js` leaves a `.zen-lini-static` alone if
   another figure on the page loads it.
7. **`Context.Output`** is a new field of `plugins.Context`: `"site"` while
   `zenmd export site` renders, `"print"` for print and PDF, and `""` in the
   app's reader and editor (Integration: engine/render, engine/site,
   docs/wiki/PLUGINS.md).
   - The app keeps drawing in the browser, so it keeps zoom and pan, and the
     theme switch from Settings.
   - The export option of behaviour 10 can turn static on in print too.

### Diagnostics in check and Problems

8. **The check hook.** With the add-on and the tool, every Lini figure the
   page renders, in the app and in `zenmd check`, is also checked with
   `lini - --check --json`. That is the same JSON as the compiler's
   `diagnostics()`: code, severity, message, line and column.
   - Each error or warning becomes one page warning through `ctx.Warn`:
     `lini: 11:18: expected '{' (P001)`.
   - Lines count from the figure's first line, and the file's name is added
     when it comes from a file: `figures/radio.lini:11:18: …`.
   - They show in Problems, in `zenmd check` (one line each, as a structure
     problem is shown), and in the warning lines of `zenmd export site`.
   - A check reuses the compile of behaviour 4 when both run.
9. **Results are cached** in memory by the SHA-256 of the source and the
   options, so an unchanged figure is checked and compiled once per process.

### The export option (CLI first)

10. `zenmd export site --lini auto|static|browser`, with the config key
    `export.lini` (in `export-options.json`, so the CLI and both Settings
    forms list it):
    - `auto`, the default, makes static figures when the add-on and the tool
      are there, else browser figures;
    - `static` makes static figures and fails the export with "the lini tool
      was not found" when they can't be made;
    - `browser` always draws in the reader's browser, as today.

    Settings › Export › Website gets the matching choice. It is shown only
    in a build with the add-on.
11. A figure the tool cannot compile is never static. The page keeps the
    source and the error box as now, the export prints the warning, and with
    `--lini static` it exits 1, as `zenmd check` does on errors.

## Architecture

- **`builtins/plugins/lini`** gets two hooks that are nil by default, so a
  build without the add-on links no exec code into the plugin:
  - `SetStatic(func(ctx context.Context, src string, opts StaticOptions) (svg []byte, diags []Diagnostic, err error))`;
  - `SetCheck(func(ctx context.Context, src string) ([]Diagnostic, error))`.

  `RenderBlock` calls them per behaviours 4 and 8. `static.go` builds the
  `<img>` and its `alt`, and maps diagnostics to `ctx.Warn` lines.
- **`addons/lini`** follows `addons/vale`:
  - it finds the binary (behaviour 2);
  - it runs it with `exec.CommandContext` with a timeout, stdin and capped
    output (4 MB);
  - it parses `--json`;
  - it keeps the cache (behaviour 9);
  - it registers the add-on and both hooks in `init`.
- **The site builder** sets `Context.Output = "site"` and passes the scheme,
  and the asset list drops `lini.js` for a page whose Lini figures are all
  static (Integration).
- **The sanitiser** must allow `data:image/svg+xml;base64,` in `img src`
  (Integration). An SVG shown as an image never runs scripts or loads other
  resources, which every browser guarantees, so this is safe from any page.
  It is a one-line change to `engine/render/sanitise.go` with a test.

## Interfaces

```go
// builtins/plugins/lini
type Diagnostic struct {
    Code, Severity, Message string // "P001", "error" | "warning", "expected '{'"
    Line, Col int                  // 1-based, in the figure's source
}
type StaticOptions struct{ Theme string } // "" (follow the system), "light", "dark"
func SetStatic(f func(ctx context.Context, src string, o StaticOptions) ([]byte, []Diagnostic, error))
func SetCheck(f func(ctx context.Context, src string) ([]Diagnostic, error))

// engine/plugins (Integration): a new Context field
Output string // "site" while zenmd export site renders, "print" for print and PDF, "" in the app

// addons/lini (build tag addon_lini)
func Find() (path, version string, err error)
```

```text
zenmd export site [--lini auto|static|browser]     config: export.lini (auto)
ADDONS=lini task build|package                     the add-on, as ADDONS=vale
ZEN_LINI=/path/to/lini                             where the tool is, first
```

## Tests

| # | Behaviour | Test |
|---|---|---|
| 1 | the add-on registers with its name and summary only under `addon_lini`; no exec code without it | `addons/lini/lini_test.go: TestRegistered`; `apps/zenmark/addons_default_test.go` (no lini in the default build) |
| 2 | finding the tool: `$ZEN_LINI`, then PATH; 1.x only | `TestFindOrder`, `TestFindVersion` (a fake `lini` script in a temp dir) |
| 3 | the App Store build refuses the add-on | `Taskfile` check test, as Vale's channel test |
| 4–5 | site render uses the tool; an `<img>` with a base64 SVG, alt from `<title>`, `data-drawn` | `builtins/plugins/lini/static_test.go: TestStaticImage`, `TestStaticAlt`, with a fake static hook |
| 4 | a 10 s timeout, empty env, no args from the page | `addons/lini: TestRunTimeout`, `TestRunEnv` (a fake tool that sleeps, or prints its env) |
| 5 | `--scheme light`/`dark` passes the theme | `TestStaticTheme` |
| 6 | an all-static page lists no lini.js; `lini.js` skips `.zen-lini-static` | `engine/site` test (Integration); `lini.test.ts: a static figure is left alone` |
| 7 | the app keeps browser figures (`Output == ""`) | `TestAppStaysBrowser` |
| 8 | diagnostics become page warnings with line and column, and the file name for a file | `TestCheckWarnings`, `TestCheckFileName` |
| 8 | `zenmd check` lists them; export prints them | `internal/cli` check test with the fake hook (Integration) |
| 9 | the cache: one run per unchanged figure | `TestCache` (counting fake tool) |
| 10 | the flag, the config key and the Settings choice agree | `export-options.json` tests (existing: CLI and Settings checked against the table), with the `lini` row |
| 10–11 | `--lini static` without the tool fails; a figure that doesn't compile keeps its source, and `static` exits 1 | `TestStaticRequired`, `TestStaticCompileError` |
| — | the sanitiser allows only `data:image/svg+xml;base64,` in `img src` | `engine/render/sanitise_test.go: TestDataSVGImage` (Integration): allowed in img, refused in `a href` and other data: types |

Edge cases to cover:
- a tool that prints more than 4 MB;
- a tool that exits 0 with no SVG;
- a figure from a file with CRLF;
- two identical figures on one page (one run);
- the radio-sized schematic (60+ parts) within the timeout;
- `--lini browser` in a build with the add-on.

## UI

- **Settings › Add-ons:** lists "Lini figures", with the version of the tool
  found or the install hint, in the existing add-ons list.
- **Settings › Export › Website:** gets a "Lini figures" choice: Automatic,
  As images, In the reader's browser. It reuses the export form's existing
  segment control, as `--scheme` does, and is shown only in a build with the
  add-on.
- **Problems:** gets Lini's diagnostics in the existing warnings list.

**UI consistency:** the Settings frame and the export form patterns,
unchanged.

## Security

| Risk | Mitigation | Test |
|---|---|---|
| The page runs a program | Only the add-on runs it, only the `lini` binary found by behaviour 2, with the source on stdin and fixed arguments. Nothing from the page reaches the command line or environment | `TestRunEnv`, `TestNoPageArgs` |
| A hostile figure hangs or floods | 10 s timeout, output capped at 4 MB, a process per figure, the 200 KB source limit of plugins/lini | `TestRunTimeout`, `TestOutputCap` |
| A static SVG runs script in the site | It is shown only through `<img>`, which never runs SVG scripts or loads its resources. The sanitiser allows `data:` for `img src` with the `image/svg+xml` type only | `TestDataSVGImage` |
| A wrong tool on PATH | 1.x only. `$ZEN_LINI` is chosen by the user. The version is shown in Settings | `TestFindVersion` |
| The App Store build runs programs | It never includes the add-on | the channel test |

## Notes

- **Out of scope:**
  - bundling the `lini` binary as a helper (a later `resources.yml` entry, as
    Vale's, with a download the user approves);
  - static mermaid (no command-line tool is part of mermaid);
  - static figures in the app's reader.
- **Integration (done by the lead at merge):**
  - `engine/plugins/plugins.go` and `docs/wiki/PLUGINS.md`:
    `Context.Output`.
  - `engine/render` passes it through; `engine/site` sets `"site"`, the print
    path `"print"`, and drops `lini.js` from all-static pages.
  - `engine/render/sanitise.go`: allow `data:image/svg+xml;base64,` in
    `img src`, with its test.
  - `apps/zenmark/export-options.json`: the `--lini` row. Then the
    CLI flag in `internal/cli` and the Settings › Export choice, driven by
    the table.
  - `Taskfile.yml`: `addon_lini` in the known add-ons, and the App Store
    refusal.
  - `internal/cli` check: page warnings from plugins already reach `zenmd
    check`; confirm it, and add the test.
  - `docs/public/zenmd/reference/cli.md`: `--lini`.
- **App Store / sandbox:** never included (behaviour 3).
- **Changelog (Added):** With the `lini` add-on and the `lini` tool
  installed, `zenmd export site` writes Lini figures as SVG images (portable
  sites, scripts off, print), and `zenmd check` and Problems show Lini's
  errors with line numbers. `--lini auto|static|browser` chooses
  (plugins/lini-export).

## Prompt

> Implement `specs/plugins/lini-export/spec.md` following the rules in
> `specs/README.md`. You work only in the worktree
> `/Users/dragosh/Developer/zen-worktrees/plugins-lini-export` on branch
> `feature/plugins-lini-export`, and only in the paths under *Owns*.
>
> **Context:**
> - `addons/vale/` and `apps/zenmark/addons_vale.go` are the add-on pattern:
>   build tag, `init`, `addons.Register`, finding the binary, running it.
> - `kit/addons/` holds the registry.
> - `builtins/plugins/lini/lini.go` `RenderBlock` is where the hooks are
>   called.
> - `engine/plugins/plugins.go` `Context` gains `Output`, which is
>   Integration. Until it exists, read it through a small accessor you can
>   switch at merge, and say so.
> - Test the add-on with a fake `lini` shell script in a temp dir that answers
>   `--version`, `--check --json` and `--format svg`.
> - Never call a real network or install anything.
>
> **Rules:**
> - tests first, committed failing;
> - no new dependencies or downloads (`GOPROXY=off`);
> - never launch the app, run `task install`, `task package` or
>   `task release`, start servers, push, or touch `main`;
> - run go commands with a throwaway `HOME` and `XDG_CONFIG_HOME`, keeping
>   the real `GOMODCACHE`, `GOPATH` and `GOCACHE`.
>
> **Done when:**
> - every test in the table passes, with and without `-tags addon_lini`;
> - `go vet ./... && go test ./...` passes, and so do `go test -tags addon_lini ./addons/... ./apps/zenmark/...`;
> - your report lists the Integration lines and anything unverified (a real
>   `lini` binary, which the user installs).
