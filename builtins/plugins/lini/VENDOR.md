# Vendored files

The browser build (`web/`) of the `lini-wasm` npm package: the Lini compiler
as WebAssembly, with its wasm-bindgen glue.

| File | Package | Version | SHA-256 | License |
|---|---|---|---|---|
| `assets/lini_wasm.js` | lini-wasm | 1.1.1 | `0dd52206dd4bbb398560b522b4e0650ee1f3c5a94682a846102f33ded3938d46` | MIT |
| `assets/lini_wasm.d.ts` | lini-wasm | 1.1.1 | `addba7b6497e8e887f7a15aa842cdad3544f664512dbdc334398600cc6d0feda` | MIT |
| `assets/lini_wasm_bg.wasm` | lini-wasm | 1.1.1 | `6646d75d0147096c4f5f28f8411a727eb7c5172779e62e846530c57ba6312197` | MIT |
| `assets/lini_wasm_bg.wasm.d.ts` | lini-wasm | 1.1.1 | `c42575eec63df7f6deb6e5fb33ba37437dbea61bb71915344ec76cf4063b2c1e` | MIT |
| `assets/LICENSE-lini-wasm` | lini-wasm | 1.1.1 | `271c78d507725e309182c4fa3b8af42c1d3bf2822b227dce268bf71cc2f0c6a5` | MIT |

- **Source:** `https://registry.npmjs.org/lini-wasm/-/lini-wasm-1.1.1.tgz`.
  The tarball matched npm's integrity value
  `sha512-JHGEA62Ab1IzUJYoP8O9bNdGlhRb07BIwtIVFlu9wZDovBDtrX2LzypPoF1dcVw1rGumoHqyxCz+5v3hnNilhw==`.
  Only `package/web/*` and `package/LICENSE` are kept; the Node build is not.
- **Size:** 2.0 MB, almost all of it the `.wasm`.
- **Loading:** `lini_wasm.js` is an ES module. Its default export
  `init({ module_or_path })` fetches and instantiates the `.wasm`; pass the
  plugin's own asset URL explicitly, never relying on `import.meta.url`. Its
  only `fetch` is that one. `initSync({ module })` takes the bytes, which is
  how tests load it under Node.
- **API:**
  - `compile(src, assets?)` returns the SVG text and throws the compiler's
    message (`play.lini:3:5: error: …`) on an error;
  - `diagnostics(src)` never throws;
  - `version()` gives the engine version.

To update: download the new tarball from the same URL pattern, check its
`dist.integrity` against `npm view lini-wasm dist.integrity`, replace the
`web/` files and the licence, update this table (`shasum -a 256`), and run
`go test ./builtins/plugins/lini`.
