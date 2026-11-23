# Vendored files

| File | Package | Version | Source | SHA-256 | License |
|---|---|---|---|---|---|
| `assets/katex.min.js` | katex | 0.19.1 | https://registry.npmjs.org/katex/-/katex-0.19.1.tgz (`dist/katex.min.js`) | `fa140498b8cfc8302cbe5e7cb1bdbc1f952bc38b3fa258ede09e73edbf9a74bc` | MIT |
| `assets/katex.min.css` | katex | 0.19.1 | the same tarball (`dist/katex.min.css`) | `00e0b094171b07f58422075f469b4feac2b2f7bce1aeec943c08da4d697f4dcf` | MIT |
| `assets/fonts/*.woff2` | katex | 0.19.1 | the same tarball (`dist/fonts/`, the 20 `.woff2` files only) | see the tarball | MIT |
| `assets/KATEX-LICENSE` | katex | 0.19.1 | the same tarball (`LICENSE`) | | MIT |

The tarball's npm integrity is
`sha512-jG8uKntjSEcm2EpQOh0XturpPuuC9r3Sq5HLHdsFdWZT27E672SF19QEi+of0/YeOy15lDgRhKvr5614uRrEfA==`,
checked on download (2026-11-23).

`katex.min.js` is the UMD build: loading it as a classic script defines
`globalThis.katex`. Only the `.woff2` fonts are kept: WebKit loads the first
`url()` of each `@font-face`, which is the `.woff2`, so the `.woff` and `.ttf`
fallbacks in `katex.min.css` are never fetched.

To update: `npm pack katex@<version>` into an empty folder, check its integrity
against `npm view katex@<version> dist.integrity`, copy the same files, update
this table, and run `go test ./builtins/plugins/math`.
