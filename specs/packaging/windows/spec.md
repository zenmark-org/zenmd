# packaging/windows

| | |
|---|---|
| Status | specified |
| Target version | Later (milestone "Later"); each phase gets a version when it's scheduled |
| Branch | one per phase: `feature/windows-<phase>` |
| Wave | 70–72, one phase per wave (numbers kept free until scheduled) |
| Depends on | packaging/linux-cli (the Wails-free `cmd/zenmd` and `apps/zenmark/cliset`) |
| Owns | per phase (below) |

## Purpose

Zenmark and `zenmd` run on Windows 10 (22H2) and 11, on x64 and arm64. Phase
1 brings the command line. Phase 2 brings the app, and phase 3 an installer
that is signed and updates.

## Behaviour

### Phase 1: `zenmd.exe` (wave 70)

**Owns:**
- new `engine/extplugins/proc_windows.go` and `proc_unix.go` (moving the
  current code there);
- new `builtins/plugins/sandbox/proc_windows.go` and `proc_unix.go`;
- new `engine/scriptplugins/dup_windows.go`;
- `cmd/zenmd/**` (the Windows parts);
- path tests across `engine/**` (test files only);
- `docs/public/zenmd/getting-started/installation.md` (a Windows section).

1. **It builds.** `CGO_ENABLED=0 GOOS=windows go build ./cmd/zenmd` builds
   for amd64 and arm64.
2. **Plugin processes are contained like a process group.** External and
   script plugin processes, and the sandbox plugin's runner, start in a Job
   Object with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE`. A timeout, a cancel or an
   output cap ends the whole job, children included, the way `killGroup`
   ends the process group on Unix. The H2/L2 hardening tests have Windows
   versions: a grandchild that tries to leave still ends at the deadline.
3. **The script helper on Windows.** It moves stderr off the protocol
   handle with `SetStdHandle`. Same behaviour, different calls.
4. **Paths.**
   - Pages and links keep `/` in the engine.
   - Files on disk use `\`, via `filepath`.
   - Drive letters, UNC paths (`\\server\share`) and long paths (over 260
     characters, via `\\?\`) open.
   - Folder containment compares case-insensitively and rejects `..`,
     another drive, and junctions or symlinks that lead outside, as on Unix.
5. **Config.** The global config lives at `%AppData%\zen\config.yml`, via
   `os.UserConfigDir`, with the templates folder next to it. The project
   markers (`.zenmd.yml`, `.git`) are found the same way. The upward search
   stops at the user's profile folder.
6. **Same output.** `zenmd export site` on Windows writes the same files as on
   Linux and macOS for the same folder: `/` in URLs, LF line endings in the
   generated HTML, and the same file names. A pre-built copy of the demo site
   is compared in CI.
7. **Releases.** Each release attaches `zenmd_<v>_windows_<arch>.zip` (with
   `zenmd.exe`) and adds them to `SHA256SUMS`. The docs-ci action stays
   Linux-only; the how-to shows the PowerShell lines for a Windows runner.
8. **CI.** A `windows-latest` job runs `go test ./...` for the engine and CLI
   packages, and exports `examples/demo`.

### Phase 2: the app (wave 71)

**Owns:**
- `kit/desktop/*_windows.go` (new);
- `apps/zenmark/*_windows.go` (new);
- `build/windows/**` (new: icon, manifest, version info);
- `kit/ui/src/shell/platform.ts` (new);
- the menu labels in `apps/zenmark/menu.go`.

9. **The app runs on WebView2,** through Wails v3's Windows backend.
   WebView2 Runtime is part of Windows 11 and is installed on Windows 10 by
   the installer (phase 3).
10. **The keys follow the platform.**
    - Every ⌘ shortcut is Ctrl on Windows. Labels follow: "Ctrl+O", and
      the Settings shortcut becomes Ctrl+,.
    - The menu is in the window (File, Edit, View, Window, Help), with
      Settings under File and About under Help.
    - The shell reads the platform from `host.platform`; there are no
      user-agent checks.
11. **Native pieces.**
    - Open and save dialogs, Recents (the Jump List), and "Open with
      Zenmark" for `.md`, registered by the installer.
    - Single-instance: a second launch passes its path to the running app.
    - Dark mode follows Windows.
12. **PDF export and print.**
    - Export › PDF uses WebView2's `PrintToPdf`, with the same page size,
      margins and output as on macOS for the demo's print pages (compared by
      page count and text).
    - Print uses the system print dialog.
13. **UI consistency.** The same tokens and layout. Only the window frame,
    menu bar and keyboard labels differ, and none of the Settings or panel
    patterns do. Scrollbars use the overlay style the frontend already sets.

### Phase 3: installer, signing and updates (wave 72)

**Owns:**
- `build/windows/installer/**` (new);
- the Windows jobs in `.github/workflows/release.yml` (the lead applies
  them at merge);
- `docs/wiki/DISTRIBUTION.md` (a Windows section).

14. **The installer.**
    - An MSIX package for Windows 11 and 10 (19041+). Per-user install, no
      admin.
    - It registers `.md` and `.markdown` for "Open with", and puts `zenmd.exe`
      on the user's `PATH` via an App Execution Alias.
15. **Signing.** The MSIX and `zenmd.exe` are signed in CI with the
    project's code-signing certificate (Azure Trusted Signing or an OV/EV
    certificate, decided when scheduled). Secrets live in repository
    secrets, never in the repository. An unsigned build is never published.
16. **Updates.** The app checks the GitHub release feed the way the macOS app
    does, and the MSIX `AppInstaller` file handles the update. winget's
    manifest is submitted by hand for the first version, and by CI after.

## Architecture

- **Process handling.** It's the one real engine change. Today
  `engine/extplugins/exec.go` and `builtins/plugins/sandbox/sandbox.go` call
  `syscall.SysProcAttr{Setpgid: true}` and `syscall.Kill(-pid, …)` directly.
  Phase 1 moves them behind:

  ```go
  func startContained(cmd *exec.Cmd) (contained, error)
  type contained interface { Kill() error; Close() error }
  ```

  It has two implementations: `proc_unix.go` (`//go:build unix`: process
  group) and `proc_windows.go` (a Job Object via `golang.org/x/sys/windows`,
  already in the module graph through Wails; it's checked at phase 1 and,
  if missing, added by the lead first).
- **The app.** It reuses Wails v3's Windows support. Platform code goes in
  `*_windows.go` files next to the darwin ones in `kit/desktop` and
  `apps/zenmark`, with `host.platform` telling the shell. `build/windows`
  follows Wails' own layout.
- **Release.** It gains Windows jobs on `windows-latest`. Signing runs only
  on tags.

## Interfaces

```go
// engine/extplugins, builtins/plugins/sandbox (unexported, per OS)
func startContained(cmd *exec.Cmd) (contained, error)

// kit/ui host
platform: "darwin" | "windows" | "linux"
```

**CLI:** no new flags. `zenmd` behaves the same on every OS.

## Tests

| # | Behaviour | Test |
|---|---|---|
| 1 | Builds for windows/amd64 and arm64 | `cmd/zenmd/deps_test.go: TestNoWailsNoCgo` gains the two targets |
| 2 | Contained processes; escaped grandchild ends | `engine/extplugins/hardening_windows_test.go` (H2/L2 with a PowerShell or Go helper child), `proc_test.go` per OS |
| 3 | Script helper on Windows | `engine/scriptplugins/dup_windows_test.go` |
| 4 | Paths: drive, UNC, long, case, junction outside | `engine/workspace/paths_windows_test.go` |
| 5 | Config location, marker search stops at the profile | `engine/config/config_windows_test.go`, `internal/cli/open_windows_test.go` |
| 6 | Same site output | CI: export the demo on Windows and compare to the Linux output (`build/sitecompare_test.go`) |
| 7–8 | Release archives, CI job | reviewed by the lead; release dry run (`workflow_dispatch`) |
| 9–13 | App: shortcuts, menu, dialogs, PDF, look | `kit/ui/src/shell/platform.test.ts`, `apps/zenmark/menu_windows_test.go`; the rest checked by hand on Windows 11 x64 and arm64 |
| 14–16 | Installer, signing, update | release dry run; installing on a clean Windows 10 and 11 VM, by hand |

## UI

**UI consistency:** the app looks the same. Only these differ:
- the window frame;
- the in-window menu bar;
- Ctrl in shortcut labels.

They come from `host.platform`, in one place (`platform.ts`), and no view
gets its own Windows variant.

Accessibility: Narrator reads the sidebar, the reader and the dialogs as
VoiceOver does today (checked by hand in phase 2).

## Security

| Risk | Mitigation | Test |
|---|---|---|
| Plugin children outlive a timeout or cancel on Windows (no process groups) | Job Object with kill-on-close; every plugin process starts suspended, is assigned to the job, then resumed, so no child is born outside it | `hardening_windows_test.go` |
| Path tricks: case, short names (`PROGRA~1`), junctions, `\\?\`, alternate data streams (`a.md:secret`) | Containment on the final path (`GetFinalPathNameByHandle`), case-insensitive compare, `:` refused in page names | `paths_windows_test.go` |
| DLL planting next to `zenmd.exe` | Built with Go (no DLL search for its own code); `SetDefaultDllDirectories(LOAD_LIBRARY_SEARCH_SYSTEM32)` at start in the app | phase 2 test checks the call |
| Unsigned or tampered installer | Signed in CI only from tags; checksums published; the app updates only from the signed MSIX feed | release dry run |
| Signing secrets | Repository secrets only; never in logs (`::add-mask::`) | lead review of the workflow |

## Notes

- **Out of scope:** Windows on 32-bit, Windows 10 before 22H2, a portable
  `.exe` app without installer, the Microsoft Store listing (later, like the
  Mac App Store build), and Linux desktop app packaging.
- **Integration (done by the lead at merge):**
  - **Phase 1:**
    - `Taskfile.yml` `release:cli` adds the Windows targets;
    - the Windows CI job in `ci.yml`;
    - `CLAUDE.md`, Checks: one line for the Windows cross-build.
  - **Phase 2:** `docs/wiki/UI.md`, a Platforms section (frame, menu,
    keyboard labels).
  - **Phase 3:** the release workflow's Windows jobs, and the secrets listed
    in `docs/wiki/DISTRIBUTION.md`.
- **Decisions before scheduling:**
  - the signing route (Azure Trusted Signing needs an organisation
    identity);
  - whether phase 1 ships before the app (recommended: yes, it is
    self-contained).
- **Changelog:**
  - phase 1: `Added: zenmd for Windows (x64 and arm64), command line only.`
  - phase 2: `Added: Zenmark for Windows.`
  - phase 3: `Added: a signed Windows installer with updates.`

## Prompt

> Implement phase N of `specs/packaging/windows/spec.md` following the rules
> in `specs/README.md`, in `../zen-worktrees/windows-<phase>` on
> `feature/windows-<phase>`.
>
> **Context:**
> - packaging/linux-cli is merged: `cmd/zenmd` builds without Wails or cgo.
> - Today `engine/extplugins/exec.go` (`Setpgid`, `syscall.Kill`) and
>   `builtins/plugins/sandbox/sandbox.go` are the only Unix-only engine code.
>
> **Steps:**
> 1. Write the phase's tests and commit them failing; Windows-only tests use
>    `//go:build windows`.
> 2. Make them pass, with only the phase's Owns paths.
>
> **Rules:**
> - No new dependencies without the lead (`golang.org/x/sys/windows` is
>   checked first).
> - No downloads.
> - Don't launch the app, run `task install`, `task package` or
>   `task release`, start servers, push or touch `main`.
>
> **Done when:**
> - `go vet ./... && go test ./...` pass on macOS;
> - `GOOS=windows go vet ./engine/... ./internal/... ./cmd/...` passes;
> - your report says what must be checked on a real Windows machine.
