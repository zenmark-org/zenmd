# Lini figures

Write a ` ```lini ` block and Zenmark draws it as a figure. Lini is a small
language for flows, mindmaps, sequences, charts, floor plans and circuits; its
own docs are at <https://lini.rs/docs/>. Figures follow the light or dark theme.
The plugin is on by default, and its compiler loads only on a page that has a
lini block.

## A flow

```lini
{ direction: row; gap: 40; }
|box#open| "Open folder"
|box#read| "Read a page"
|box#share| "Share it"
open -> read "pick"
read -> share "export"
```

## A mindmap

```lini
|mindmap| "Zenmark" [
  |topic| "Read"
  |topic| "Write" [
    |topic| "Editor"
    |topic| "Preview"
  ]
  |topic| "Share"
]
```

## A sequence

```lini
{ layout: sequence; }
|box#you| "You"
|box#editor| "Editor"
you -> editor "type"
editor -> you "page reloads"
```

## A schematic, from a file

A circuit drawn with `{ layout: schematic; }`: parts with references such as
`|R#R1| "1k"`, an IC as a `|component|` with numbered pins, and wires between
pins (`U1.vout - R1 - D1 - |gnd|`). The settings block holds the layout and the
defines (the `5V` power label); the parts and wires come after it. This one lives
in its own file, `plugins/figures/regulator.lini`, and the block only names it:

```lini /plugins/figures/regulator.lini
```

## When a figure is wrong

The source stays visible with the compiler's message under it:

```lini
|box#a| "A"
|box#b| "B"
a => b
```

## Turn it off

```yaml
plugins:
  lini: false   # for a folder in .zenmd.yml, or one page in its frontmatter
```
