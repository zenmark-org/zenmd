# packaging/docs-ci

| | |
|---|---|
| Status | specified |
| Target version | 0.8.0 |
| Branch | `feature/packaging-docs-ci` |
| Wave | 58 |
| Depends on | packaging/linux-cli (the Linux `zenmd` and its release archives). `zenmd check` from engine/templates-blueprints phase 1 is used when it's there. |
| Owns | `action.yml` (new, repository root), `.github/actions/**` (new, the action's scripts), `examples/ci/**` (new: workflow examples), `docs/public/zenmd/how-tos/build-docs-in-ci.md` (new), `docs/public/zenmd/how-tos/index.md` (its link line), `build/action_test.go` (new) |

## Purpose

Build a documentation folder with `zenmd` in CI, on Linux, with no Mac
and no app. You can check it on every pull request and publish it as a
website, for example on GitHub Pages. This repository builds its own
`docs/public/zenmd` this way.

## Behaviour

1. **A GitHub Action.** The repository is usable as an action:

   ```yaml
   - uses: zenmark-org/zenmd@v0.8.0
     with:
       folder: docs
       output: _site
   ```

   It installs the Linux `zenmd` of the action's own version from that
   version's GitHub release, then runs `zenmd export site <folder> -o
   <output>`. It works on `ubuntu-*` runners (amd64 and arm64) and fails with
   one line on other runners ("zenmd's action runs on Linux runners").
2. **The download is checked.**
   - The action downloads `zenmd_<v>_linux_<arch>.tar.gz` and `SHA256SUMS`
     from `https://github.com/zenmark-org/zenmd/releases/download/v<v>/` only.
   - It checks the archive against `SHA256SUMS` and stops on a mismatch.
   - It puts `zenmd` in `$RUNNER_TEMP/zenmd/bin` and on `PATH` for later
     steps.
   - `version: <v>` picks another release, but only an exact version
     (`0.8.0`), never `latest`.
3. **Inputs mirror the CLI.** Every input is a `zenmd export site` flag
   from `apps/zenmark/export-options.json`:
   - `folder` (default `.`), `output` (default `_site`);
   - `hosted` (default `true` in CI: plain links for a web server);
   - `drafts`, `scheme`, `clean` (default `true`), `print`.

   A test checks the inputs against `export-options.json`, so a new site
   option is either an input or listed as left out on purpose.
4. **Optional check step.** With `check: true` (default `false` until
   `zenmd check` exists, then `true`), the action runs `zenmd check <folder>`
   first. It fails the job on errors and shows warnings as GitHub
   annotations (`::warning file=…,line=…::…`). Without a `zenmd check` in that
   version, `check: true` fails with one line naming the version that has it.
5. **Outputs.** `output` (the site's folder), `pages` (the page count) and
   `warnings` (the warning count). Build warnings also become annotations on
   the page and line they name.
6. **Example workflows** in `examples/ci/`:
   - `check-docs.yml` checks and builds on every pull request that touches
     the folder;
   - `publish-pages.yml` builds on `main` and deploys with
     `actions/upload-pages-artifact` and `actions/deploy-pages`.

   The how-to explains both and links them.
7. **This repository uses it.**
   - **Releases.** The release workflow builds `docs/public/zenmd` with the
     action at the tagged commit (`uses: ./`) after the Linux CLI job. It
     attaches `zenmd-docs-<v>.zip` to the release.
   - **Pull requests.** A pull request that touches `docs/public/**` runs the
     check build. CI stays off on `main` (since 0.7.0), so this runs only for
     pull requests and releases.
   - **Publishing to Pages** is listed as the next step; it isn't switched on
     here.
8. **Docs.** `docs/public/zenmd/how-tos/build-docs-in-ci.md` covers:
   - the action, its inputs and outputs, and both example workflows;
   - other CI systems (GitLab, plain shell): download, `shasum -c`, run, in
     five lines.

## Architecture

- **`action.yml`** is a composite action. Its steps call
  `.github/actions/install.sh` (download, checksum, `PATH`) and
  `.github/actions/run.sh` (check, export, then annotations from the warning
  lines).
- The scripts are POSIX `sh` using `curl`, `sha256sum` and `tar`, which
  Ubuntu runners have. They add no dependencies to the repository.
- The action's version comes from `github.action_ref` (`v0.8.0` → `0.8.0`).
  A branch or SHA ref (`uses: ./` or `@main`) builds `./cmd/zenmd` from the
  checked-out source with `go build` instead, which needs `setup-go` first.
  This repository's own workflows use it that way.
- `zenmd`'s warning lines (`<page>:<line>: <text>`) are parsed for
  annotations. The format is fixed by a test in `internal/cli`, so the action
  never guesses.

## Interfaces

```yaml
# action.yml
inputs:
  folder:  { default: "." }
  output:  { default: "_site" }
  version: { default: "" }        # "" = the action's ref; exact versions only
  hosted:  { default: "true" }
  drafts:  { default: "false" }
  scheme:  { default: "" }        # light | dark | auto
  clean:   { default: "true" }
  print:   { default: "true" }
  check:   { default: "false" }
outputs:
  output:   {}
  pages:    {}
  warnings: {}
```

**CLI:** no new `zenmd` flags or config keys. The action calls existing ones.
It needs the warning line format to stay stable, which
`internal/cli: TestWarningLineFormat` (Integration) fixes.

## Tests

| # | Behaviour | Test |
|---|---|---|
| 1 | Action metadata: runs on Linux only, version from ref | `build/action_test.go: TestActionMetadata` (parses `action.yml`) |
| 2 | Checksum mismatch stops; only the release URL; exact versions | `build/action_test.go: TestInstallScript` (runs `install.sh` against a local `file://` fixture folder via an env override used only in tests; a bad checksum exits 1; `latest` is refused) |
| 3 | Inputs match export-options.json | `build/action_test.go: TestInputsMatchExportOptions` |
| 4 | Check step, annotations, the missing-check message | `build/action_test.go: TestRunScript` (a fake `zenmd` on `PATH` that prints warning lines and exit codes) |
| 5 | Outputs and warning annotations | `build/action_test.go: TestRunScript` |
| 6 | Example workflows are valid and use the action | `build/action_test.go: TestExampleWorkflows` (parse YAML, pinned action versions) |
| 7 | This repo builds its docs | the release run, checked by the lead; locally `go build ./cmd/zenmd && sh .github/actions/run.sh` with `folder=docs/public/zenmd` |
| 8 | Docs | docs link checks; read by the lead |

**Edge cases:**
- a folder with spaces;
- `output` inside `folder` (zenmd's existing rule applies);
- an arm64 runner;
- a release without Linux archives (versions before 0.8.0): one line naming
  0.8.0 as the first.

## UI

None. It's a CI action. Its messages follow `zenmd`'s one-line style.

## Security

| Risk | Mitigation | Test |
|---|---|---|
| A tampered or substituted download runs in someone's CI | Only `github.com/zenmark-org/zenmd/releases/download/v<exact>/`; checked against `SHA256SUMS` before extracting; no `latest`, no redirects to other hosts (`curl --proto =https --fail`) | `TestInstallScript` |
| Inputs injected into shell commands | Inputs reach the scripts as environment variables, never `${{ }}` inside `run:` text; the scripts quote every use | `TestActionMetadata` (no `${{ inputs.` inside `run:`), `TestRunScript` with a folder named `a"; rm -rf ~ #` |
| A pull request from a fork publishing the site | The examples deploy only on `push` to `main`, with `pages: write` limited to the deploy job | `TestExampleWorkflows` |
| Annotations leaking file contents | Annotations carry zenmd's warning text only, never page text | `TestRunScript` |

## Notes

- **Out of scope:**
  - publishing to Netlify or S3 (the Publish add-on idea in docs/wiki/ADDONS.md);
  - a GitLab CI template;
  - PDF in CI (macOS only);
  - a Docker image.
- **Integration (done by the lead at merge):**
  - `.github/workflows/release.yml`: a `docs` job, `needs: [test, cli]`, on
    `ubuntu-latest`:
    - `actions/setup-go`;
    - `uses: ./` with `folder: docs/public/zenmd`, `output: dist/docs`;
    - `cd dist && zip -r zenmd-docs-${VERSION}.zip docs`;
    - upload; the release job attaches it.
  - `.github/workflows/docs.yml` (new): on `pull_request` with
    `paths: [docs/public/**]`, the same build with `check: true` once
    `zenmd check` exists.
  - `internal/cli`: a `TestWarningLineFormat` that fixes
    `<page>:<line>: <text>` on stderr.
  - `README.md`: one line under Install, "Build docs in CI: see …".
- **App Store / sandbox:** none.
- **Changelog:** `Added: a GitHub Action (uses: zenmark-org/zenmd@v0.8.0) that builds a docs folder into a website on Linux CI, with a checked download, an optional zenmd check, and warnings as annotations; this repository's docs are built with it.`

## Prompt

> Implement `specs/packaging/docs-ci/spec.md` following the rules in
> `specs/README.md`, in `../zen-worktrees/packaging-docs-ci` on
> `feature/packaging-docs-ci`.
>
> **Context:**
> - packaging/linux-cli is merged: `./cmd/zenmd` builds with
>   `CGO_ENABLED=0`, and releases attach `zenmd_<v>_linux_<arch>.tar.gz` and
>   `SHA256SUMS`.
> - The site flags are in `apps/zenmark/export-options.json`.
>
> **Steps:**
> 1. Write the Tests table's tests in `build/action_test.go` and commit them
>    failing.
> 2. Write `action.yml`, then `.github/actions/install.sh` and `run.sh`.
> 3. Add the two examples in `examples/ci/` and the how-to.
>
> **Rules:**
> - Only the Owns paths. Put the workflow and `internal/cli` changes in your
>   report.
> - No new Go dependencies, no downloads in tests (use `file://` fixtures).
> - Don't run the action against GitHub.
> - Don't launch the app, run `task install`, `task package` or
>   `task release`, start servers, push or touch `main`.
>
> **Done when:**
> - `go vet ./... && go test ./...` pass;
> - `sh -n` passes on both scripts;
> - your report has the Integration lines and the changelog line.
