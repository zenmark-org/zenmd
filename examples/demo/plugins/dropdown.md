# Dropdowns

A dropdown is a title that opens and closes its text. It opens by click or
Enter, and a page without scripts reads the same. It is the `dropdown` of
sphinx-design, so a page written for MyST, Jupyter Book or Sphinx works as it is.

## Syntax

````markdown
:::dropdown{title="More detail"}
The body is **markdown**.
:::
````

In the MyST form the title comes after the name:

````markdown
:::{dropdown} More detail
The body is **markdown**.
:::
````

Options:

| Option | Meaning |
|---|---|
| `title` | the title (the argument in the MyST form); "Details" without one |
| `open` | start unfolded |
| `class`, `name` | extra class names, and an id |
| `color`, `icon` | ignored, with a warning: every block is black and white |

## Try it

:::dropdown{title="More detail"}
The body is **markdown**, with a list:

- one
- two
:::

```{dropdown} The MyST form
The text after the name is the title.
```

:::{dropdown} Open at first
:open:
This one starts unfolded. Press its title to fold it.
:::
