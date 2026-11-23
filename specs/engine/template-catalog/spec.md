# engine/template-catalog

| | |
|---|---|
| Status | specified |
| Target version | 0.8.0 |
| Branch | `feature/engine-template-catalog` (phase 1), then one branch per phase |
| Wave | 53–56 (after engine/blueprint-hierarchy) |
| Depends on | engine/templates-blueprints (blueprints, page templates, the install record and fingerprints), core/config (user-scoped keys) |
| Owns | `engine/catalog/**` (new), `internal/cli/catalog*.go` + tests, `apps/zenmark/catalog.go` + test, `kit/ui/src/shell/catalog*.ts` + tests, `docs/CATALOG.md` (new); phase 4 adds `.github/workflows/templates-sync.yml` and `Taskfile.yml` task `templates:sync` (Integration) |

## Purpose

Blueprints and page templates are published and updated on their own, apart
from the app. A signed catalog in a separate repository lists them with
their versions. `zenmd` and Zenmark download a template from it, check it, and
install it. They also update the copies you haven't edited, and offer the new
version beside a copy you have edited. A template fix no longer waits for an
app release, and anyone can publish their own catalog.

## Why

Today the templates are embedded in the binary (`builtins/blueprints`,
`builtins/pagetemplates`). An install copies them to
`~/Library/Application Support/zenmark/templates/`, and only a new app version
brings a newer copy. Removing the slides from techdocs (2.4.0) reaches users
only with the next release. And an edited copy only gets an `<id>.new` beside
it, which you have to swap by hand.

## Words

- **Pack:** one blueprint or one page template, published as a versioned
  archive. It is data only, never code (see Security).
- **Catalog:** the signed `index.json` of a **source**, listing its packs.
- **Source:** where a catalog lives. The official one is
  `zenmark-org/templates`. You can add your own.
- **Bundled:** the snapshot of the official catalog embedded in the app, so
  that the first run and offline use work as they do today.

## Behaviour

### A. The pack and the catalog

1. **The pack's archive:** a `.tar.gz` that holds one folder, `<id>/`.
   - A blueprint has `blueprint.yml` at its root. A page template has `template.yml`.
   - Its manifest's `id` and `version` must match the catalog entry.
2. **The catalog:** `index.json` with this shape:
   ```json
   {
     "schema": 1,
     "source": "zenmark-org/templates",
     "generated": "2026-11-22T18:00:00Z",
     "packs": [
       {
         "id": "techdocs",
         "kind": "blueprint",
         "version": "2.4.0",
         "name": "Technical documentation",
         "description": "…",
         "requires": { "zen": ">=0.7.0", "layouts": ["docs", "landing", "term", "glossary", "release"], "plugins": ["codeblock", "diagram"], "templates": ["overview", "how-to"] },
         "url": "https://github.com/zenmark-org/templates/releases/download/techdocs-2.4.0/techdocs-2.4.0.tar.gz",
         "sha256": "…",
         "size": 18234,
         "changes": "No presentation.md slides; the home is the overview."
       }
     ]
   }
   ```
   - It is signed with Ed25519. The signature sits beside it as `index.json.sig` (base64).
   - A catalog may list several versions of one pack; the newest one that fits wins (behaviour 6).
3. **A blueprint's page templates:** a blueprint names the page templates it needs in `requires.templates`. Installing the blueprint also installs or updates each of them from the same source.

### B. Sources

4. **The source list** comes from the user's `config.yml` only, never from a folder's `.zenmd.yml`:
   ```yaml
   catalog:
     check: weekly            # daily | weekly | never; how often to look for updates
     sources:
       - name: official       # always there; cannot be removed, only turned off
         url: https://github.com/zenmark-org/templates/releases/latest/download/
       - name: acme
         url: https://templates.acme.example/
         key: "ed25519:BASE64…"   # optional; without a key the source is unsigned
   ```
   - The official source's key is built into the app; `config.yml` cannot change it.
   - Two keys are built in, the current one and the next, so the key can be rotated with no app update in between.
5. **An unsigned source** (no `key`) works only after you confirm it once, when you add it, in Zenmark's dialog or with `zenmd catalog sources add … --unsigned`. Its packs show "Unsigned source" wherever they are listed.

### C. Checking and choosing

6. **Choosing a version:** a pack fits when `requires.zen` matches this build, and every layout and plugin it needs is in the build (as `requires` is checked today). The newest fitting version wins.
   - A newer version that doesn't fit is listed as "Needs Zenmark 0.9" (or names the missing layout) and is never installed.
7. **Checking for updates** fetches only `index.json` and its signature, nothing else.
   - With `check: daily` or `weekly`, Zenmark checks at most that often, and only when a window is open.
   - `zenmd` checks only when asked.
   - The request carries no cookies and no user data, and its User-Agent is `zen/<version>`.
   - The catalog is cached in `<user cache dir>/zen/catalog/<source>/`, and a failed check uses the cache.
8. **Rollback protection:** a catalog whose `generated` is older than the last one accepted from that source is refused. So is any `version` lower than the one installed, except with `--version <v>` and `--force`, or Zenmark's "Install this version…".

### D. Installing and updating

9. **Installing** downloads the archive and checks it before anything is written:
   - its size matches and is at most 5 MB, packed and unpacked;
   - its SHA-256 matches;
   - it passes the archive rules (Security);
   - its manifest is valid (`zenmd blueprint check` / `templates check`).
   - It is then installed through the existing copy and record code (`engine/sitetemplates`: `copyTree`, `replaceTree`, the record and the fingerprints). The record gains `source` and `version` for each id.
10. **Updating:**
    - A copy you haven't edited (its fingerprint matches its install) is replaced in place, as the built-in update does today.
    - An edited copy is never touched. The new version goes to `<id>.new` beside it, and the update is listed as "You edited this template".
    - Zenmark then offers:
      - **Use the new version:** yours moves to `<id>.old`, and `.new` becomes the template;
      - **Keep mine:** `.new` is deleted, and this version is not offered again;
      - **Show the changes:** a per-file list of added, removed and changed files, with a line diff for the text files.
11. **Bundled and downloaded copies** are one set:
    - the bundled snapshot counts as the official source at its version;
    - whichever is newer wins, under the same rules;
    - the app's own start-up install (`sitetemplates.Install`) keeps working with no network.
12. **Folders made from a blueprint** keep their own copies (`.zen/templates/*.yml`, the pages written). Nothing here changes a folder.
13. **Deleting:** a template you deleted isn't installed again by a check or update. `zenmd catalog install <id>` brings it back.

### E. CLI first

14. **`zenmd catalog`**, the CLI for everything above:
    | Command | What it does |
    |---|---|
    | `zenmd catalog list [--available\|--installed\|--outdated] [--json]` | id, kind, installed version, newest fitting version, source, state (up to date / update / edited / needs Zenmark x / unsigned) |
    | `zenmd catalog search <words>` | matches id, name and description |
    | `zenmd catalog show <id>` | the entry, its requires, changes, and the files of the installed copy |
    | `zenmd catalog install <id>[@<version>] [--source <name>] [--force]` | download, check, install (and the blueprint's templates) |
    | `zenmd catalog install <path-or-https-url>` | a local folder or `.tar.gz` (for authors); unsigned, so it asks to confirm, or takes `--yes` |
    | `zenmd catalog update [<id>…\|--all] [--dry-run]` | behaviour 10; prints what was replaced and which `.new` files were written |
    | `zenmd catalog use-new <id>` / `keep-mine <id>` | the two answers of behaviour 10 |
    | `zenmd catalog check` | fetch the catalogs now; exit 0 when nothing is outdated, 10 when updates exist (for scripts) |
    | `zenmd catalog sources [add <name> <url> [--key …\|--unsigned]\|remove <name>\|off <name>\|on <name>]` | edit `catalog.sources` in `config.yml` |
    | `zenmd catalog pack <folder> -o <dir>` | for publishers: check the folder and write `<id>-<version>.tar.gz` plus its index entry (JSON) |
    | `zenmd catalog index <dir> -o index.json [--sign-key-file <path>]` | for publishers: build the catalog from the entries and sign it. The private key is read only from a file or `ZEN_CATALOG_KEY`, never from a flag value |

    `zenmd templates list|show|update` keeps working: `update` becomes `zenmd catalog update` for page templates.

### F. In Zenmark

15. **Settings › Templates** (the Settings frame, as today):
    - The left column has three groups:
      - **Installed**: an "Update" note on items with a newer fitting version, and "Edited" on items with a `.new`;
      - **Available**: catalog packs not installed;
      - **Sources**.
    - The top of the left column shows "Check for Updates", with the time of the last check.
    - The right column shows the selected item's details, its changes text, and one primary action: Install, Update, Use the New Version, or none when it is up to date. Keep Mine and Show the Changes are quiet buttons.
    - A source shows its URL, Signed or Unsigned, On/Off, and Remove (except the official one).
16. **New from Blueprint…** also lists the available blueprints that aren't installed, marked "Download". Choosing one installs it first (the click is the consent), then goes on as today.
17. **The update count:** when a check finds updates, Settings › Templates shows the count beside its name. There is no pop-up and no badge elsewhere.
18. **Offline:**
    - with no network, Settings shows "Couldn't reach <source>; showing what was found on <date>";
    - Install and Update are disabled for packs that aren't cached;
    - nothing else changes.

### G. Publishing (the templates repository)

19. **`zenmark-org/templates`** is a new repository:
    - one folder per pack, under `blueprints/<id>/` and `pages/<id>/`;
    - its CI runs `zenmd catalog pack` on each changed pack.
    - Raising a pack's `version` and merging cuts a release named `<id>-<version>` with the archive.
    - The CI then rebuilds `index.json` from every release, signs it with the key from a repository secret, and uploads it to the `latest` release.
20. **The app's bundled snapshot:**
    - `task templates:sync VERSION=<catalog generated date>` downloads the official catalog and checks its signature;
    - it puts the archives the app ships into `builtins/blueprints` and `builtins/pagetemplates`;
    - it writes `builtins/catalog.lock` with the catalog date and every pack's SHA-256.
    - `TestBundledMatchesLock` fails when the files and the lock disagree.
    - The sync is run by a person and its result reviewed like any change. CI never syncs on its own.
21. **The move:** the templates' source of truth moves from this repository to `zenmark-org/templates` (phase 4). After that, editing `builtins/blueprints` by hand fails `TestBundledMatchesLock`.

## Architecture

- **`engine/catalog`** (new, public engine package; it imports `engine/sitetemplates` and the standard library):
  - `index.go`: parse, verify (`crypto/ed25519`) and choose;
  - `fetch.go`: an HTTP client with limits;
  - `archive.go`: safe unpacking;
  - `state.go`: the cache, the last accepted `generated`, and the "keep mine" answers;
  - `install.go`: calls into `sitetemplates` for copy, record and fingerprint.
  - No new dependencies: `archive/tar`, `compress/gzip`, `crypto/ed25519`, `crypto/sha256` and `net/http` are in the standard library.
- **`engine/sitetemplates`** gains an exported install entry that takes an `fs.FS` plus a source and version: `InstallFrom(dir, fsys, id, Origin)`. It reuses today's `Update` logic. This is the one change to a file this spec doesn't own (Integration).
- **`internal/cli/catalog*.go`:** the `zenmd catalog` commands, all through `engine/catalog`.
- **`apps/zenmark/catalog.go`:** service methods that wrap the same calls, plus the scheduled check (a ticker in the app, gated by `catalog.check` and the last check time).
- **`kit/ui/src/shell/catalog*.ts`:** the Settings › Templates groups and actions.
- **Data flow:** `Check()` fetches each source's index, checks its signature, applies rollback protection and caches it. `Plan()` compares the catalog with the record and gives a list of states. `Install` and `Update` download, check, unpack into a temporary folder, then call `sitetemplates.InstallFrom`.

## Interfaces

```go
package catalog

type Source struct {
	Name, URL string
	Key       []ed25519.PublicKey // nil: unsigned (confirmed)
	Off       bool
	Official  bool
}

type Requires struct {
	Zen       string   `json:"zen"` // SemVer range, ">=0.7.0"
	Layouts   []string `json:"layouts,omitempty"`
	Plugins   []string `json:"plugins,omitempty"`
	Templates []string `json:"templates,omitempty"`
}

type Pack struct {
	ID, Kind, Version, Name, Description, Changes string
	Requires                                       Requires
	URL, SHA256                                    string
	Size                                           int64
}

type Index struct {
	Schema    int       `json:"schema"`
	Source    string    `json:"source"`
	Generated time.Time `json:"generated"`
	Packs     []Pack    `json:"packs"`
}

type State string // "up-to-date" | "update" | "edited" | "available" | "needs" | "deleted"

type Item struct {
	ID, Kind, Installed, Newest, Source, Needs string
	State                                      State
	Unsigned                                   bool
	Pack                                       *Pack
}

type Env struct {
	TemplatesDir, CacheDir, AppVersion string
	Layouts, Plugins                   []string
	Client                             *http.Client // tests pass an httptest client
	Now                                func() time.Time
}

func Check(ctx context.Context, env Env, sources []Source) (map[string]*Index, []error)
func Plan(env Env, idx map[string]*Index) ([]Item, error)
func Install(ctx context.Context, env Env, it Item, force bool) (sitetemplates.UpdateResult, error)
func Update(ctx context.Context, env Env, ids []string, dryRun bool) ([]sitetemplates.UpdateResult, error)
func UseNew(env Env, id string) error
func KeepMine(env Env, id string) error
func Diff(env Env, id string) ([]FileChange, error)
func PackFolder(dir, out string) (Pack, error)                                        // publishers
func BuildIndex(entries []Pack, source string, key ed25519.PrivateKey) (index, sig []byte, error)
```

- **Config, user scope only:** `catalog.check` (`daily|weekly|never`, default `weekly`) and `catalog.sources`.
- **Service methods:**
  - `CatalogCheck() ([]Item, error)`
  - `CatalogItems() ([]Item, error)`, from the cache with no network
  - `CatalogInstall(id string) (Result, error)`
  - `CatalogUpdate(ids []string) ([]Result, error)`
  - `CatalogUseNew(id)`, `CatalogKeepMine(id)`, `CatalogDiff(id)`
  - `CatalogSources() / CatalogSetSource(...)`
- **Event:** `catalog:changed` after any check or install.

## Tests

| # | Behaviour | Test |
|---|---|---|
| 1 | An archive with one `<id>/` folder whose manifest matches the entry installs; a mismatched id or version is refused | `engine/catalog/archive_test.go: TestPackArchiveShape` |
| 2 | A good signature is accepted; a changed byte, a wrong key or a missing `.sig` is refused for a signed source; a rotated (second) key is accepted | `index_test.go: TestIndexSignature` |
| 3 | Installing a blueprint installs the page templates it requires from the same source | `install_test.go: TestBlueprintBringsTemplates` |
| 4 | `.zenmd.yml` `catalog:` is ignored with a Problems note; `config.yml` sources are read; the official key can't be overridden | `engine/catalog/config_test.go`, `core/config` test |
| 5 | An unsigned source is used only after confirmation; its items are marked Unsigned | `TestUnsignedNeedsConfirm` |
| 6 | The newest fitting version wins; a newer one needing a later zen or a missing layout is `needs` with the reason | `index_test.go: TestChoose` |
| 7 | A check makes only index and sig requests, with no cookies and User-Agent `zen/<v>`; a failed check falls back to the cache; `check: never` makes no request | `fetch_test.go: TestCheckRequests` (httptest) |
| 8 | An older `generated` is refused; a lower version is refused without force | `TestRollback` |
| 9 | Size over the limit, wrong SHA-256, or a bad manifest: nothing is written | `install_test.go: TestInstallChecksFirst` |
| 10 | An untouched copy is replaced in place; an edited copy gets `.new`; use-new keeps `.old`; keep-mine removes `.new` and is remembered; the diff lists changed files | `install_test.go: TestUpdateEditedAndUntouched`, `TestUseNewKeepMine`, `TestDiff` |
| 11 | Bundled vs downloaded: the newer wins; start-up install works with no network | `TestBundledAndDownloaded` |
| 12 | A folder made from a blueprint is never touched by an update | `TestFoldersUntouched` |
| 13 | A deleted template isn't reinstalled by update; an explicit install brings it back | `TestDeletedStaysDeleted` |
| 14 | Every `zenmd catalog` command, the exit code 10 of `check`, and that `index` refuses a key passed as a flag value | `internal/cli/catalog_test.go` |
| 15–18 | Settings › Templates groups, actions, the update count and offline text; New from Blueprint lists Download items | `kit/ui/src/shell/catalog.test.ts`, `apps/zenmark/frontend/src/shell/newsite.test.ts` |
| 19 | `zenmd catalog pack` + `index` round-trip: the produced index verifies and installs | `TestPackIndexRoundTrip` |
| 20–21 | The bundled files match `builtins/catalog.lock` | `builtins/catalog_test.go: TestBundledMatchesLock` |

**Archive edge cases** (`TestArchiveRules`):
- `../` and absolute paths;
- symlinks and hard links;
- devices and FIFOs;
- a file type outside the list;
- more than 500 entries;
- a gzip bomb (unpacked size over the limit);
- duplicate paths;
- names differing only in case;
- `.DS_Store` (skipped);
- an empty archive;
- two top-level folders.

## UI

- **The pattern reused:** the Settings frame, with items on the left (grouped Installed, Available and Sources, as Settings › Reader groups its items under a heading), details on the right, and empty text in the left column ("No templates in this source"). Buttons use the shared Settings button styles: one primary action, the rest quiet.
- **States:**
  - **Loading:** "Checking…" in place of the last-checked time, and the list stays usable.
  - **Error:** one line per source that failed, with the cached date.
  - **Populated** and **empty**, as above.
- **Keyboard:** the list and actions are reachable as in the other Settings sections. "Check for Updates" can be reached with Tab and has no shortcut.
- **Look:** tokens only (docs/wiki/UI.md). Update and Edited notes use the existing `--fg-muted` small note style.

## Security

| Risk | Mitigation | Test |
|---|---|---|
| A tampered catalog or archive (compromised host, man-in-the-middle) | HTTPS only; Ed25519 signature on `index.json` with the key built into the app; the SHA-256 and size of each archive come from the signed index | 2, 9 |
| An old catalog that brings back a vulnerable or broken version (rollback or freeze) | the last accepted `generated` is kept per source; older is refused; no lower versions without force | 8 |
| An archive that writes outside the templates folder (zip slip, links) | unpack into a fresh temporary folder; refuse `..`, absolute paths, symlinks, hard links and special files; copy into place with `copyTree` | `TestArchiveRules` |
| Resource exhaustion (huge or many files, a gzip bomb) | 5 MB packed and unpacked, 500 entries, 1 MB per file; the download stops at the declared size plus 1 byte | `TestArchiveRules`, 9 |
| Code in a pack | packs are data only. Allowed files: `.yml .yaml .md .txt .png .jpg .jpeg .gif .webp .svg`. Anything else (`.js`, `.tengo`, `.sh`, executables) refuses the whole pack. Layouts and plugins aren't in the catalog; they keep their own trust flow (docs/wiki/PLUGINS.md, Trust) | `TestArchiveRules` |
| A folder adds a source, or swaps the official key (a downloaded project steering your installs) | `catalog:` is read only from the user's `config.yml`; in `.zenmd.yml` it's ignored with a Problems note; the official key is compiled in | 4 |
| An unsigned third-party source | added only by the user, confirmed once, and always shown as Unsigned | 5 |
| The user's edits lost by an update | fingerprints: edited copies are never replaced; use-new keeps `.old` | 10 |
| Privacy | only index requests on a schedule, no cookies, no identifiers, a fixed User-Agent; `check: never` turns it off | 7 |
| Leaking the signing key | `catalog index` reads the key only from a file or `ZEN_CATALOG_KEY`, refuses a flag value, and never prints it; in CI it's a repository secret | 14 |
| Redirects to other hosts | redirects are followed only over HTTPS, at most 5 | `fetch_test.go: TestRedirects` |
| A template that steers a folder to unsafe output (an export path, say) | unchanged: page templates and blueprints go through today's manifest checks and the `.zenmd.yml` export rules (no output outside the folder) | existing tests |

## Phases

| Phase | Wave | What | Branch |
|---|---|---|---|
| 1 | 53 | `engine/catalog`: index, signature, choose, archive rules, install and update from a **local** archive or folder; `sitetemplates.InstallFrom`; `zenmd catalog install <path>`, `pack`, `index`, `list --installed` | `feature/engine-template-catalog` |
| 2 | 54 | network: sources in `config.yml`, fetch, cache, rollback, `check`, `install <id>`, `update`, `use-new`, `keep-mine`, `diff`, `sources` | `feature/engine-catalog-remote` |
| 3 | 55 | Zenmark: service methods, the scheduled check, Settings › Templates groups and actions, Download in New from Blueprint | `feature/ui-catalog` |
| 4 | 56 | the `zenmark-org/templates` repository and its CI (created by the user, who adds the signing secret); `task templates:sync`; `builtins/catalog.lock`; templates leave this repo as the source of truth | `feature/templates-repo` |

Phases 1 and 2 are CLI only (CLI first). Phase 3 needs both. Phase 4 needs
phase 2, and the user to create the repository and the key.

## Notes

- **Decisions to confirm with the user:**
  1. the name of the repository (`zenmark-org/templates`);
  2. `check: weekly` on by default, or off until turned on;
  3. layouts and plugins left out of the catalog for now (they run code and have their own trust flow; a later spec can add them as signed packs that still go through Allow/Block).
- **Key handling:** the user makes the Ed25519 key pair on their own machine (`zenmd catalog keygen -o <file>`, phase 4), keeps the private key out of chat and out of git, and adds it as the templates repository's secret. Only the public key is committed, in `engine/catalog/keys.go`.
- **Not in this spec:**
  - updating the copies inside a folder made from a blueprint (`.zen/templates/`); a later `zenmd catalog sync-folder` could offer it, with the same `.new` rule;
  - ratings, accounts and uploads;
  - a web page listing the catalog.
- **Integration (lead, at merge):**
  - `engine/sitetemplates/install.go`: `InstallFrom`, the record's `source` and `version`;
  - `core/config`: the `catalog:` key with user scope;
  - `apps/zenmark/app-assembly.yml`: none (the catalog is part of the `newsite` feature);
  - `specs/README.md`: rows;
  - `docs/wiki/TEMPLATES.md`: "Getting more templates" points at docs/CATALOG.md;
  - `Taskfile.yml`: `templates:sync` (phase 4);
  - `apps/zenmark/bindings_test.go`: the new methods.
- **Changelog:**
  - phase 1: `Added: zenmd catalog install|pack|index: install a blueprint or page template from a checked archive, and publish your own.`
  - phase 2: `Added: template updates without an app update: zenmd catalog check|update|use-new|keep-mine, from the signed official catalog or your own sources.`
  - phase 3: `Added: Settings › Templates shows available templates and updates, with Install, Update and Use the New Version; New from Blueprint can download a blueprint.`
  - phase 4: `Changed: the built-in templates come from zenmark-org/templates; the app ships a checked snapshot.`

## Prompt

> Implement phase <N> of `specs/engine/template-catalog/spec.md` following CLAUDE.md, in the worktree `../zen-worktrees/<branch name>` on branch `<branch>`.
> - Read engine/sitetemplates (install.go: record, fingerprints, `copyTree`, `replaceTree`, `Update`) first and reuse it; don't copy it.
> - Write the tests first and commit them failing, including every archive edge case and every Security row of your phase.
> - Use only the standard library for archives, hashing, signatures and HTTP. Tests use `httptest` and a test key pair made in the test, never the network.
> - Only the Owns paths. List anything needed in a shared file as an Integration line in your report.
> - No new dependencies or downloads. Don't launch the app, run task package/install/release, push or touch main. Never write to `~/Library/Application Support/zenmark` or the user's cache folder; tests use temporary folders.
