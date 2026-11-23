# Build docs in CI

Build a docs folder with `zenmd` on a Linux CI runner, check it on every pull
request and publish it as a website. No Mac and no app are needed.

## Use the GitHub Action

```yaml
- uses: zenmark-org/zenmd@v0.8.0
  with:
    folder: docs
    output: _site
```

The action installs the Linux `zenmd` of its own version from that version's
GitHub release, then runs `zenmd export site docs -o _site`. It runs on
`ubuntu-*` runners (amd64 and arm64). The download comes from
`github.com/zenmark-org/zenmd/releases` only, and is checked against the
release's `SHA256SUMS` before it is used.

### Inputs

Each input is a `zenmd export site` flag (see the
[export options reference](../reference/export-options.md)).

| Input | Default | Flag |
|---|---|---|
| `folder` | `.` | the folder to build |
| `output` | `_site` | `-o` |
| `hosted` | `true` | `--hosted` (plain links for a web server); `false` is `--portable` |
| `drafts` | `false` | `--drafts` |
| `scheme` | empty | `--scheme` (`light`, `dark` or `auto`) |
| `clean` | `true` | `--clean` |
| `print` | `true` | `print.html`; `false` is `--no-print` |
| `version` | the action's version | an exact release such as `0.8.0`, never `latest` |
| `check` | `false` | run `zenmd check` first and fail on errors |

`check: true` needs a `zenmd` that has the `check` command; without one the
step fails with a line that names the version.

### Outputs

- `output`: the site's folder;
- `pages`: the number of pages built;
- `warnings`: the number of build warnings.

Each warning also shows on the job as an annotation on the page it names.

## Example workflows

- [check-docs.yml](https://github.com/zenmark-org/zenmd/blob/main/examples/ci/check-docs.yml) checks and builds on
  every pull request that touches the docs.
- [publish-pages.yml](https://github.com/zenmark-org/zenmd/blob/main/examples/ci/publish-pages.yml) builds on
  `main` and deploys with GitHub Pages. It never runs for pull requests, and
  only its deploy job may write to Pages.

Copy one into `.github/workflows/` and set the folder.

## Other CI systems

Download the release, check it, run it:

```sh
V=0.8.0 A=amd64   # or arm64
B=https://github.com/zenmark-org/zenmd/releases/download/v$V
curl -fsLO $B/zenmd_${V}_linux_$A.tar.gz -O $B/SHA256SUMS
grep " zenmd_${V}_linux_$A.tar.gz\$" SHA256SUMS | sha256sum -c - && tar -xzf zenmd_${V}_linux_$A.tar.gz
zenmd_${V}_linux_$A/zenmd export site docs -o _site --hosted --clean
```

## Troubleshooting

- **"zenmd's action runs on Linux runners":** use an `ubuntu-*` runner.
- **"checksum mismatch":** the download was not the release's file. Nothing was
  installed; run the job again.
