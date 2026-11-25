# lini

Draws [Lini](https://lini.rs/) figures from a `lini` block: a flow, a tree or
mindmap, a sequence, a chart, a floor plan or a circuit, written as text. A
figure follows the light or dark theme. A wrong figure keeps its source
visible, with the compiler's message under it. On by default; the compiler
(about 2 MB) loads only on a page that has a `lini` block. Turn it off with
`plugins: { lini: false }` in `.zenmd.yml` or a page's frontmatter, or in
Settings › Plugins.

## Names

| Name | Written as |
|---|---|
| `lini` | a fence: ```` ```lini ````, or a container: `:::lini` |

## Options

The plugin has no options. A block takes one attribute:

| Attribute | What it does |
|---|---|
| `file` | Draws a figure from a file of the folder instead of the block's text: ```` ```lini {file=figures/a.lini} ````, ```` ```lini figures/a.lini ```` or a `:file:` line. A path that starts with `/` is from the folder's root; any other is from the page's folder. |

An attribute the plugin does not know gives a warning for the page.

## Example

````markdown
```lini
{ direction: row; gap: 40; }
|box#open| "Open folder"
|box#read| "Read a page"
|box#share| "Share it"
open -> read "pick"
read -> share "export"
```
````

More figures are in the demo, and the language is described in
[Lini's own docs](https://lini.rs/docs/).

## Good to know

- The figure is drawn in the page by a compiler that ships with Zenmark (the
  `lini-wasm` package, MIT). A page without scripts shows the source as a code
  block.
- A **portable** site (`zenmd export site --portable`, opened from `file://`)
  cannot load the compiler, because browsers refuse to fetch WebAssembly from
  a file. Its figures stay as source, with the message "lini: could not load
  the Lini compiler". Serve the site, or leave `--portable` off.
- A figure's source stays available to screen readers, and the figure is an
  image with a label.
- A figure of more than 200 KB of source is not drawn.
