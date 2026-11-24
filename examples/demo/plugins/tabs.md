# Tabs

Tabs show one panel at a time. A `tab-set` holds `tab-item` blocks, each with a
title. They are written the same way in MyST, Jupyter Book and Sphinx
(sphinx-design), so pages from those tools read here as they are.

## Syntax

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

In the MyST form the title comes after the name:

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

Options on a `tab-item`:

| Option | Meaning |
|---|---|
| `title` | the tab's title (the argument in the MyST form) |
| `selected` | open this tab first |
| `sync` | a key; tabs with the same key change together across the page |

Options on a `tab-set`: `sync-group` (default `tab`), `class` and `name`.

## Try it

Press a tab, or use the Left and Right arrow keys, Home and End.

:::tab-set
:::tab-item{title="macOS"}
Install with Homebrew.
:::
:::tab-item{title="Linux"}
Install with your package manager.
:::
:::tab-item{title="Windows"}
Use the installer.
:::
:::

## The MyST form

::::{tab-set}
:::{tab-item} One
The first panel, written as MyST.
:::
:::{tab-item} Two
The second panel, with **markdown** inside.
:::
::::

## Tabs that change together

Pick Go in one set and the other follows. Zenmark remembers the choice for this
folder.

:::tab-set
:::tab-item{title=Python sync=py}
`print("hello")`
:::
:::tab-item{title=Go sync=go}
`fmt.Println("hello")`
:::
:::

:::tab-set
:::tab-item{title=Python sync=py}
Run it with `python hello.py`.
:::
:::tab-item{title=Go sync=go}
Run it with `go run hello.go`.
:::
:::

Without scripts, for example in print, every panel shows in order under its
title.
