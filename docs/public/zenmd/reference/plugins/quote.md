# quote

The `quote` plugin draws quotes set apart from the text: an epigraph, a pull
quote and a list of highlights. Each is a blockquote in the plugin kit's frame.

## What it owns

| Name | Draws | zen form | MyST form |
|---|---|---|---|
| `epigraph` | a short quote that opens a page or a section | `:::epigraph` | `` ```{epigraph} `` and `:::{epigraph}` |
| `pull-quote` | a line repeated from the text, in larger type | `:::pull-quote` | `` ```{pull-quote} `` and `:::{pull-quote}` |
| `highlights` | the key points, as a quote | `:::highlights` | `` ```{highlights} `` and `:::{highlights}` |

The body is markdown. A last line that starts with `--` (or `---`, or a dash)
is the attribution and is shown under the quote, with its own markdown:
`-- *The Mythical Man-Month*`.

## Options

| Option | Does |
|---|---|
| `class` | Adds your own class names to the block. |
| `name` | Gives the block an `id`, so a link or `{ref}` can point to it. |

In the zen form, write them in braces: `:::epigraph{name=dijkstra}`. In the
MyST form, write `:name: dijkstra` at the top of the body. Any other option is
ignored, and the page has a warning that names it.

## Example

```markdown
:::epigraph
Simplicity is prerequisite for reliability.

-- Edsger W. Dijkstra
:::
```

The same in the MyST form:

````markdown
```{epigraph}
Simplicity is prerequisite for reliability.

-- Edsger W. Dijkstra
```
````
