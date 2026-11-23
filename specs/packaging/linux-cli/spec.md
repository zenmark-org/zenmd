# packaging/linux-cli

| | |
|---|---|
| Status | specified |
| Target version | 0.8.0 |
| Branch | `feature/packaging-linux-cli` |
| Wave | 57 |
| Depends on | none (engine/session, kit/assembly and engine/export-cli-parity are done). Runs beside app/single-file in wave 57: that spec owns `internal/cli/cli.go`, this one only adds `internal/cli/build.go` |
| Owns | `cmd/zenmd/**` (new), `apps/zenmark/cliset/**` (new), `apps/zenmark/build.go`, `apps/zenmark/sitetemplates.go`, `apps/zenmark/service.go` (the three `builtin…` lines only), `apps/zenmark/assembly_gen.go` (moves), `kit/assembly/cmd/gen/**`, `internal/cli/build.go` (new: `cli.Build`), `engine/scriptplugins/host.go` and new `engine/scriptplugins/dup_*.go`, `docs/public/zenmd/getting-started/installation.md`, `docs/public/zenmd/reference/cli.md` |

## Purpose

`zenmd` runs on Linux (amd64 and arm64) as a plain command-line tool:
`zenmd export site`, `zenmd init`, `zenmd templates` and the rest of the
engine commands. Every release attaches it as a download next to the macOS
DMG. The Zenmark app and PDF export stay macOS only.

## Behaviour

1. **A build without the app.** `go build ./cmd/zenmd` makes a `zenmd` that
   doesn't import Wails and needs no cgo: `CGO_ENABLED=0` builds it for
   linux/amd64, linux/arm64, darwin/amd64 and darwin/arm64.
2. **Same engine, same output.** For the same folder and flags, the
   command-line-only `zenmd export site` writes byte-for-byte the files the
   app's `zenmd` writes, and prints the same lines. Both get their built-in
   plugins, layouts, dialects and blueprints from the one generated assembly
   (`apps/zenmark/app-assembly.yml`), so the two never drift.
3. **Every engine command works.** This covers:
   - `export site`;
   - `init`, including `--list`, `--update` and `--blueprint`;
   - `templates list`, `templates show` and `templates update`;
   - `help`, `version` and the folder config warnings.

   They have the same flags, exit codes (0 ok, 1 failure, 2 usage) and
   messages as the app's `zenmd`.
4. **App-only commands fail in one line.**
   - **Opening a folder or page** (`zenmd`, `zenmd <folder>`, `zenmd <page.md>`)
     prints `zenmd: the Zenmark app isn't part of this build; zenmd export site
     <folder> makes a website` and exits 1. It never hangs and never tries a
     GUI.
   - **`zenmd export pdf`** prints `zenmd: export pdf needs the Zenmark app on
     macOS` and exits 1.
5. **Commands that would show a folder print its path.** `zenmd templates
   --dir` and `zenmd init --dir` print the path on stdout and exit 0. They
   don't open a file manager.
6. **`zenmd version` says which build it is.** It prints `zenmd 0.8.0 (command
   line)`. The app's `zenmd` keeps printing `zenmd 0.8.0`. Both take the version
   from `-ldflags -X …/internal/cli.Version`.
7. **The config folder follows the platform.** On Linux the global config is
   `$XDG_CONFIG_HOME/zenmark/config.yml`, or `~/.config/zenmark/config.yml` when
   `XDG_CONFIG_HOME` is unset, through `os.UserConfigDir` as today. The
   templates folder sits next to it. `zenmd templates --dir` prints it.
8. **Script plugins work on linux/arm64.** The script helper
   (`zenmd __script-host`, `scriptplugins.HostMain`) builds on every
   linux and darwin architecture. On Linux it uses `syscall.Dup3(old, new, 0)`
   where darwin uses `syscall.Dup2`.
9. **The app's `zenmd` is unchanged.** The root `main.go` still builds
   the app with the CLI inside. `task build`, `task package`, the DMG and the
   app's `zenmd` link keep working as now.
10. **Releases attach the Linux CLI.** A `v*` tag adds the following to the
    GitHub release, next to the DMG:
    - `zenmd_<version>_linux_amd64.tar.gz` and `zenmd_<version>_linux_arm64.tar.gz`,
      each holding `zenmd`, `LICENSE` and `README.md`;
    - a `SHA256SUMS` file.

    They're built in CI from the tagged commit, after the tests pass.
11. **CI checks the build on every run.** The `ci.yml` test job:
    - cross-builds `./cmd/zenmd` with `CGO_ENABLED=0` for linux/amd64 and
      linux/arm64;
    - runs the amd64 binary on `examples/demo` (`zenmd export site examples/demo -o <tmp>`);
    - fails if it exits non-zero or writes no `index.html`.
12. **The docs say how to install it.** The Installation page in
    `docs/public/zenmd` gains a Linux section: download, check the
    checksum, put `zenmd` on the `PATH`, and what isn't there (the app,
    PDF). The CLI reference marks `export pdf` and opening a folder as
    "macOS app only".

## Architecture

The app's `zenmd` and the new one share every line except the four hooks
`cli.Run` takes (`Launcher`, `Builder`, `Initer`, `PDFExporter`):

```
main.go (app, Wails, cgo)          cmd/zenmd/main.go (no Wails, no cgo)
  │ app.Launch, app.ExportPDF        │ cliset.NoApp, cliset.NoPDF
  │ app.BuildSite → cliset.BuildSite │ cliset.BuildSite
  │ app.InitSite  → cliset.InitSite  │ cliset.InitSite
  └──────────── internal/cli.Run ────┘
                     │
     apps/zenmark/cliset: the generated assembly + the site/init hooks
                     │
     engine/session, engine/config, builtins/*  (already Wails-free)
```

- **`apps/zenmark/cliset`** is a new package that must never import Wails,
  `kit/desktop` or `apps/zenmark`. It holds:
  - the generated assembly (`assembly_gen.go`, moved here from `apps/zenmark`
    and now exporting its names);
  - `BuildSite` (from `apps/zenmark/build.go`);
  - `InitSite` and `initSite` (from `apps/zenmark/sitetemplates.go`), with the
    folder reveal passed in as a `Reveal func(dir string) error` (nil: print the
    path).
- **`apps/zenmark`** keeps the app. It calls `cliset` for the CLI hooks and
  passes `desktop.Reveal` as the reveal. `service.go`'s `builtinPlugins`,
  `builtinLayouts` and `builtinDialects` point at `cliset.Plugins`,
  `cliset.Layouts` and `cliset.Dialects`; `assemblyInfo` becomes `cliset.Info`.
  The test seams (`siteBuiltins`, the `builtin…` vars) stay where tests
  replace them.
- **`kit/assembly/cmd/gen`** gains `-out <dir>`: the folder the generated
  file is written to, which defaults to the app folder, so existing uses keep
  working. Its exported names are `Plugins`, `Layouts`, `Templates`,
  `Dialects` and `Info`. `TestAssemblyGenerated` checks the file at its new
  place.
- **`cmd/zenmd/main.go`** is a copy of the root `main.go` with the `cliset`
  hooks. It keeps the script-helper mode first and the alias notice.
- **`engine/scriptplugins`** moves the `Dup2` call into `dup_darwin.go` and
  `dup_linux.go` (`//go:build`), with one function `redirectStderr(stderr,
  stdout *os.File) error`.

## Interfaces

```go
// package cliset — apps/zenmark/cliset (no Wails, no cgo)
func Plugins() *plugins.Registry
func Layouts(version string) *enginelayouts.Registry
func Templates() fs.FS
func Dialects() *dialects.Registry
var Info session.BuildInfo

func BuildSite(b cli.Build, stdout, stderr io.Writer) error
func InitSite(i cli.Init, stdout, stderr io.Writer) error           // reveal = print the path
func InitSiteWith(reveal func(dir string) error) cli.Initer        // the app passes desktop.Reveal
func NoApp(l cli.Launch) error                                     // "the Zenmark app isn't part of this build; …"
func NoPDF(e cli.ExportPDF, stdout, stderr io.Writer) error        // "export pdf needs the Zenmark app on macOS"

// package cli — internal/cli
var Build = ""   // "" for the app, "command line" for cmd/zenmd; set by -ldflags or cmd/zenmd's init

// package scriptplugins — per OS
func redirectStderr(stderr, stdout *os.File) error
```

**Commands and flags:** no new `zenmd` flags and no new config keys. The new
artifacts are listed below; the Taskfile and workflow lines are in Notes.

| Artifact | Made by |
|---|---|
| `bin/zenmd-cli` | `task build:cli` (host OS and arch, `CGO_ENABLED=0`) |
| `dist/zenmd_<v>_linux_<arch>.tar.gz`, `dist/SHA256SUMS` | `task release:cli` (CI only) |

## Tests

Written first and committed failing.

| # | Behaviour | Test |
|---|---|---|
| 1 | No Wails, no cgo, four targets | `cmd/zenmd/deps_test.go: TestNoWailsNoCgo` (`go list -deps` for each GOOS/GOARCH with `CGO_ENABLED=0`: no `wailsapp`, no `kit/desktop`, no `apps/zenmark` import; the list succeeds) |
| 1 | `cliset` stays Wails-free | `apps/zenmark/cliset/deps_test.go: TestClisetImports` |
| 2 | Same output as the app's zenmd | `apps/zenmark/cliset_parity_test.go: TestCLIBuildMatchesApp` (exports `examples/demo` through `app.BuildSite` and `cliset.BuildSite`, compares every file and stdout) |
| 2 | One assembly | `apps/zenmark/assembly_test.go: TestAssemblyGenerated` (new path), `kit/assembly/cmd/gen/main_test.go: TestOutFlag` |
| 3 | Engine commands, exit codes | `cmd/zenmd/main_test.go: TestCommands` (builds the binary into `t.TempDir()`, runs export site, init --list, templates list, help, version, a usage error; checks exit codes and first lines) |
| 4 | App-only commands, one line, exit 1 | `cmd/zenmd/main_test.go: TestAppOnly` (no folder, a folder, a page, export pdf; each returns in under 2 s) |
| 5 | `--dir` prints the path | `apps/zenmark/cliset/init_test.go: TestDirPrintsPath` |
| 6 | Version line | `internal/cli/cli_test.go: TestVersionBuild`, `cmd/zenmd/main_test.go: TestCommands` |
| 7 | XDG config path | `engine/config/config_test.go: TestGlobalPathXDG` (linux only: with and without `XDG_CONFIG_HOME`) |
| 8 | Script helper on linux/arm64 | `engine/scriptplugins/dup_test.go: TestRedirectStderr`, covered for arm64 by `TestNoWailsNoCgo` building the package |
| 9 | App unchanged | the existing `apps/zenmark` and `build/darwin` tests, `task build` |
| 10–11 | Release and CI | reviewed by the lead (workflow YAML); `task release:cli` run locally into a temp folder: two archives, `SHA256SUMS` verifies with `shasum -a 256 -c` |
| 12 | Docs | `TestNoLegacyBuildKeyOutsideNotices` and the docs link checks pass; read by the lead |

**Edge cases:**
- a folder path with spaces;
- `XDG_CONFIG_HOME` set to a relative path (Go ignores it; the test pins that);
- `zenmd --version` and `zenmd version` both work;
- an unknown command on the CLI-only build gives exit 2 with the hint, as on the app;
- the old `zen` alias name prints the notice on both builds.

## UI

None. It's command-line only. The output reuses the app's `zenmd` messages
word for word, apart from the two app-only lines in Behaviour 4.

## Security

| Risk | Mitigation | Test |
|---|---|---|
| A download that isn't what CI built | Archives are built in CI from the tag after the tests pass, uploaded by the release job, with `SHA256SUMS`. The Installation page shows how to check it. | the lead checks `release.yml`; `task release:cli` output verifies |
| The CLI-only build quietly drops a check the app does (plugin trust, folder limits, the `.zenmd.yml` output-outside-folder rule) | Both builds go through the same `session.WriteSite` and `config.Load`; nothing is reimplemented in `cliset` | `TestCLIBuildMatchesApp`; the existing export-outside-folder test also runs through `cliset.BuildSite` (`cliset/build_test.go: TestOutputStaysInFolder`) |
| Script and external plugins on Linux escape their process group or outlive the export | The existing Setpgid and kill-group code already runs on Linux; CI runs `engine/extplugins` hardening tests on Linux | `engine/extplugins/hardening_test.go` (unchanged, runs in CI on Linux) |
| `--dir` on the CLI-only build runs a file manager or a shell | It only prints the path; no `xdg-open`, no exec | `TestDirPrintsPath` checks that the reveal isn't called |
| The CLI-only build reads the wrong config, or a shared one in tests | `os.UserConfigDir`, as today. Tests set `HOME` and `XDG_CONFIG_HOME` to a temp folder (see 8f14f21). | `TestGlobalPathXDG` |

## Notes

- **Out of scope:**
  - PDF export on Linux. It would need a browser engine: a download or a new
    dependency, so it needs a spec of its own.
  - The app on Linux, and Windows (`engine/extplugins` uses `syscall.Kill` and
    `Setpgid`).
  - Homebrew, apt or rpm packages, and an install script.
  - Signing or attesting the archives.
  - A darwin CLI-only archive (the app's `zenmd` covers macOS).
- **Integration (done by the lead at merge):**
  - `Taskfile.yml`:
    - add `build:cli`: `CGO_ENABLED=0 go build -ldflags "{{.VERSION_LDFLAGS}}" -o bin/zenmd-cli ./cmd/zenmd`;
    - add `release:cli`: for `linux/amd64` and `linux/arm64`, build into
      `dist/zenmd_{{.VERSION}}_linux_<arch>/`, copy `LICENSE` and `README.md`,
      `tar -czf`, then `shasum -a 256 *.tar.gz > SHA256SUMS`.
  - `.github/workflows/ci.yml`, test job, after `wails3 task test`:
    ```yaml
    - name: command-line zenmd builds without cgo and exports the demo
      run: |
        GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o /dev/null ./cmd/zenmd
        CGO_ENABLED=0 go build -o "$RUNNER_TEMP/zenmd" ./cmd/zenmd
        "$RUNNER_TEMP/zenmd" export site examples/demo -o "$RUNNER_TEMP/site"
        test -f "$RUNNER_TEMP/site/index.html"
    ```
  - `.github/workflows/release.yml`:
    - add a `cli` job (`needs: test`, `runs-on: ubuntu-latest`) that checks the
      tag matches `VERSION`, runs `task release:cli`, and uploads `dist/` as the
      artifact `zenmd-linux`;
    - the job that creates the GitHub release gets `needs: [test, cli]`,
      downloads `zenmd-linux`, and attaches the two archives and `SHA256SUMS`;
    - a `workflow_dispatch` run keeps them as artifacts only.
  - `CLAUDE.md`, Checks: the gen command becomes
    `go run ./kit/assembly/cmd/gen -out apps/zenmark/cliset apps/zenmark`
    (if the default isn't changed to that). Also add the line
    `CGO_ENABLED=0 go build ./cmd/zenmd   # the command-line-only zenmd`.
  - `docs/wiki/BUILD.md`: one paragraph on the two `zenmd` builds and
    which hooks differ.
  - `.gitignore`: `dist/`.
- **App Store / sandbox:** none. The App Store build doesn't use `cmd/zenmd`.
- **Changelog:** `Added: zenmd for Linux (amd64 and arm64), a command-line-only build attached to every release with checksums; it exports websites and runs init and templates, while the app and PDF export stay macOS only.`

## Prompt

> Implement `specs/packaging/linux-cli/spec.md` following the rules in
> `specs/README.md`, in the worktree `../zen-worktrees/packaging-linux-cli`
> on branch `feature/packaging-linux-cli`.
>
> **Context:**
> - `zenmd` today is the app's own binary. The root `main.go` passes
>   `app.Launch`, `app.BuildSite`, `app.InitSite` and `app.ExportPDF` to
>   `internal/cli.Run`, and package `apps/zenmark` imports Wails. So even the
>   CLI needs cgo and GTK on Linux.
> - The engine (`engine/...`, `builtins/...`, `internal/cli`, `kit/assembly`)
>   already builds with `CGO_ENABLED=0` on linux/amd64.
> - On linux/arm64 only `engine/scriptplugins/host.go:293` (`syscall.Dup2`)
>   fails.
> - `apps/zenmark/sitetemplates.go` uses `kit/desktop` (which imports Wails)
>   only for `revealInFinder`.
>
> **Steps:**
> 1. Write the tests in the Tests table and commit them failing.
> 2. Fix the `Dup2` call with per-OS files (`Dup3(old, new, 0)` on Linux).
> 3. Add `-out` to `kit/assembly/cmd/gen` with exported names, and generate
>    into `apps/zenmark/cliset`.
> 4. Move `BuildSite`, `InitSite` and `initSite` into `cliset`, with the reveal
>    passed in, and make `apps/zenmark` call them.
> 5. Add `cmd/zenmd/main.go` with `cliset.NoApp` and `cliset.NoPDF`, and the
>    `cli.Build` value "command line".
> 6. Update the two docs pages in `docs/public/zenmd`.
>
> **Rules:**
> - Only the Owns paths. Don't edit `Taskfile.yml`, the workflows,
>   `CLAUDE.md`, `.gitignore` or `CHANGELOG.md`; put what they need in your
>   report.
> - No new dependencies, no downloads, no `go.mod` changes.
> - Don't launch the app, run `task install`, `task package` or
>   `task release`, start servers, push or touch `main`.
> - After moving the generated file, run
>   `go run ./kit/assembly/cmd/gen -out apps/zenmark/cliset apps/zenmark` and
>   regenerate the Wails bindings as CLAUDE.md says.
>
> **Done when:**
> - `go vet ./... && go test ./...` pass on macOS;
> - `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./cmd/zenmd` and the same
>   for amd64 succeed;
> - `go list -deps ./cmd/zenmd` has no `wailsapp`;
> - `task build` still makes the app;
> - the frontend checks pass if bindings changed.
>
> **Report:** the files changed, the test output, the Integration lines you
> need, and the changelog line.
