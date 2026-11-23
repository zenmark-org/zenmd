# MyST

Markdown written for another tool can render in Zenmark without being
rewritten. A **dialect** is another way to write the blocks that plugins already
draw. **MyST** is the syntax of MyST, Jupyter Book and Sphinx. CommonMark and GFM
stay on; see [Markdown](markdown.md).

## Turn it on

In Settings › Reader, pick MyST and tick "Use MyST in this folder". The choice is
saved in the folder's `.zenmd.yml`, so the folder renders the same everywhere:

```yaml
markdown:
  dialects: [myst]
```

Zenmark also offers to turn it on when it finds a `myst.yml`, a Jupyter Book
`_config.yml` or a Sphinx `conf.py` in the folder. This guide has it on. Without
it, a fence like ```` ```{check} ```` stays a plain code block.

## A fence with a name

A fence whose first word is a name in braces is the block that owns that name.
The text after the braces is the title. This is a checklist, written the MyST
way:

````markdown
```{check}
- [ ] I turned MyST on
- [ ] I can read this page
```
````

```{check}
- [ ] I turned MyST on
- [ ] I can read this page
```

It renders exactly like `:::check`. The colon form of MyST works too:

````markdown
:::{check}
- [x] A colon fence in braces
- [ ] Another box
:::
````

:::{check}
- [x] A colon fence in braces
- [ ] Another box
:::

A block can take options. The ones this guide shows are in the
[ratings](../plugins/rating.md) page; in MyST you write them as `:key: value`
lines at the top of the body:

````markdown
```{rating}
:id: myst-rating
:title: How was MyST?
```
````

```{rating}
:id: myst-rating
:title: How was MyST?
```

A name that no plugin owns stays code, and nothing is reported.

## Callouts

A note, a tip or a warning sets a thought apart from the text. The text after the
name is the title, and each kind has its own title and a coloured rule on the left.

````markdown
```{note}
A note keeps its default title, "Note".
```
````

```{note}
A note keeps its default title, "Note".
```

```{tip} A title of your own
The text after the name replaces the default title.
```

```{hint}
A hint, like a tip, has a blue rule.
```

```{important}
An important callout has a blue rule too.
```

```{seealso}
Other pages worth a look: [Markdown](markdown.md).
```

`warning`, `caution` and `attention` have an amber rule:

```{warning}
A warning.
```

```{caution}
A caution.
```

```{attention}
An attention callout.
```

`danger` and `error` have a red rule:

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

Add `:class: dropdown` to fold a callout. It opens by click or Enter, and
`:open:` starts it open:

````markdown
```{note} Fold me
:class: dropdown
Hidden until you open it.
```
````

```{note} Fold me
:class: dropdown
Hidden until you open it.
```

```{tip} Open at first
:class: dropdown
:open:
This one starts open.
```

`versionadded`, `versionchanged` and `deprecated` take a version and make a
one-line note:

```{versionadded} 0.7
The MyST dialect.
```

```{versionchanged} 0.8
Callouts became a built-in.
```

```{deprecated} 1.0
This example is only here to show the note.
```

Every callout also works without MyST, as `:::note` with the title as an
attribute, such as `:::note{title="Remember" class=dropdown open}`.

## Code

`{code-block} lang`, `{code} lang` and `{code-cell} lang` are plain fenced code
in `lang`, so the [code colouring](../plugins/codeblock.md) applies. A code cell
is never run.

````markdown
```{code-block} python
def greet(name):
    return f"Hello, {name}"
```
````

```{code-block} python
def greet(name):
    return f"Hello, {name}"
```

Roles such as `` {ref}`x` ``, cross-references and `{eval-rst}` are not read.
