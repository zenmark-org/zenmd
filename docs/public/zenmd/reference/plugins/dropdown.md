# dropdown

A dropdown is a title that opens and closes its text. It is a plain `details`
element in the same frame as every Zenmark block, so it opens by click or Enter,
and a page without scripts reads the same. The name and options follow
sphinx-design.

## Names

| Name | What it draws |
|---|---|
| `dropdown` | A title row over a body. The title is the argument. |

## Written as

Zenmark's container form:

````markdown
:::dropdown{title="More detail"}
The body is **markdown**.
:::
````

The MyST form, with the title after the name:

````markdown
:::{dropdown} More detail
The body is **markdown**.
:::
````

The fenced form:

````markdown
```{dropdown} More detail
The body is **markdown**.
```
````

## Options

| Option | Meaning |
|---|---|
| `title` | The title (the argument in the MyST form). Without one it shows "Details". |
| `open` | Start unfolded. |
| `class` | Extra class names. |
| `name` | An id, so a link can point at it. |
| `color`, `icon` | Accepted and ignored with a warning: every block is black and white. |

Any other option is ignored and shown as a warning in Problems; the block still
renders.

## Example

````markdown
:::dropdown{title="Open at first" open}
Shown unfolded; the reader can fold it.
:::
````

A callout can also fold: see the `callout` built-in's `dropdown` class.
