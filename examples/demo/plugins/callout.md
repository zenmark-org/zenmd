# Callouts

A callout sets a note, a tip or a warning apart from the text. Each kind has its
own title and a coloured rule on the left. Write it as a `:::` container, or the
MyST way as a fence with the name in braces. Both draw the same thing.

## The `:::` form

````markdown
:::note
The body is markdown.
:::

:::warning{title="Heads up"}
A title replaces the default one.
:::
````

:::note
The body is markdown.
:::

:::warning{title="Heads up"}
A title replaces the default one.
:::

## The MyST form

The text after the name is the title.

````markdown
```{tip} A title of your own
The text after the name replaces the default title.
```
````

```{tip} A title of your own
The text after the name replaces the default title.
```

## Every name

The blue ones, `note`, `tip`, `hint`, `important` and `seealso`:

```{note}
A note keeps its default title, "Note".
```

```{hint}
A hint, like a tip, has a blue rule.
```

```{important}
An important callout has a blue rule too.
```

```{seealso}
Other pages worth a look: [Markdown](../features/markdown.md).
```

The amber ones, `warning`, `caution` and `attention`:

```{warning}
A warning.
```

```{caution}
A caution.
```

```{attention}
An attention callout.
```

The red ones, `danger` and `error`:

```{danger}
A danger callout.
```

```{error}
An error callout.
```

`admonition` takes any title, so it is required. It is blue unless you add
`:class:` with another kind, such as `warning`:

````markdown
```{admonition} Your own title
:class: warning
An admonition coloured like a warning.
```
````

```{admonition} Your own title
:class: warning
An admonition coloured like a warning.
```

## Fold it

Add `class=dropdown` (MyST: `:class: dropdown`) to fold a callout. It opens by
click or Enter, and `open` starts it open:

````markdown
:::tip{title="Fold me" class=dropdown}
Hidden until you open it.
:::
````

:::tip{title="Fold me" class=dropdown}
Hidden until you open it.
:::

```{tip} Open at first
:class: dropdown
:open:
This one starts open.
```

## Version notes

`versionadded`, `versionchanged` and `deprecated` take a version and make a
one-line note:

```{versionadded} 0.7
The MyST dialect.
```

```{versionchanged} 0.8
Callouts became a built-in.
```

:::deprecated{version=1.0}
This example is only here to show the note.
:::
