# tabs

Tabs show one panel at a time. A `tab-set` holds `tab-item` blocks, each with a
title. The names and options follow sphinx-design, so a page written for MyST,
Jupyter Book or Sphinx reads the same here.

Without scripts (a printed page, or a site viewed with scripts off) every panel
shows in order under its title.

## Names

| Name | What it draws |
|---|---|
| `tab-set` | The frame that holds the tabs. |
| `tab-item` | One tab: its title and its panel. The title is the argument. |

## Written as

Zenmark's container form:

````markdown
:::tab-set
:::tab-item{title="macOS"}
Install with Homebrew.
:::
:::tab-item{title="Linux"}
Install with your package manager.
:::
:::
````

In the container form every block ends with `:::`, so a `tab-item` closes first
and then the `tab-set`. The MyST form, with the title after the name, uses more
colons outside:

````markdown
::::{tab-set}
:::{tab-item} macOS
Install with Homebrew.
:::
:::{tab-item} Linux
Install with your package manager.
:::
::::
````

The fenced form works too, with a longer fence outside:

`````markdown
````{tab-set}
```{tab-item} macOS
Install with Homebrew.
```
```{tab-item} Linux
Install with your package manager.
```
````
`````

## Options

On `tab-set`:

| Option | Meaning |
|---|---|
| `sync-group` | The group its synced tabs belong to (default `tab`). |
| `class` | Extra class names. |
| `name` | An id, so a link can point at the set. |

On `tab-item`:

| Option | Meaning |
|---|---|
| `title` | The tab's title (the argument in the MyST form). Without one it shows as "Tab" and a warning. |
| `sync` | A key. Tabs with the same key, in the same group, change together across the page. |
| `selected` | Open this tab first. |
| `class`, `name` | As above. |

An option not listed here (for example `class-label`) is ignored and shown as a
warning in Problems; the block still renders.

## Keyboard

Tab moves to the tab row. Left and Right arrows move between tabs (and wrap),
Home and End jump to the first and last. The tab under focus opens.

## Remembered choice

When tabs share a `sync` key, Zenmark remembers the last choice for the folder
and opens it the next time. Tabs without a key start on the first tab, or on
the one marked `selected`.

## Example

````markdown
:::tab-set
:::tab-item{title=Python sync=py}
`pip install zenmd`
:::
:::tab-item{title=Go sync=go}
`go install zenmd`
:::
:::
````
