# Quotes

A quote set apart from the text: an epigraph, a pull quote and a list of
highlights. A last line that starts with `--` is the attribution.

## Syntax

````markdown
:::epigraph
Simplicity is prerequisite for reliability.

-- Edsger W. Dijkstra
:::
````

| Name | For |
|---|---|
| `epigraph` | a short quote that opens a page or a section |
| `pull-quote` | a line repeated from the text, in larger type |
| `highlights` | the key points |

## An epigraph

:::epigraph
Simplicity is prerequisite for reliability.

-- Edsger W. Dijkstra
:::

## A pull quote

The same, written the MyST way:

````markdown
```{pull-quote}
Make it work, make it right, make it fast.

-- Kent Beck
```
````

```{pull-quote}
Make it work, make it right, make it fast.

-- Kent Beck
```

## Highlights

:::{highlights}
- A quote is markdown, so it can have **bold** words and lists
- The attribution is optional
- `:name:` gives it an id that a link can point to
:::
