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

```{tip} A title of your own
The text after the name replaces the default title.
```
````

```{note}
A note keeps its default title, "Note".
```

```{tip} A title of your own
The text after the name replaces the default title.
```

`admonition`, the version notes and folding callouts, with an example of every
name, are on the [callout page](../plugins/callout.md). Every callout also works
without MyST, as `:::note`.

## Figures, images and tables

A **figure** is a picture with a caption, numbered per page. The picture is found
the way any page image is, relative to this page's folder. The first paragraph of
the body is the caption; any paragraphs after it are the legend:

````markdown
```{figure} ../images/zen-mark.svg
:alt: The Zenmark mark
:width: 120
:align: center
:name: zen-mark

The Zenmark mark.

The legend is the paragraphs after the caption.
```
````

```{figure} ../images/zen-mark.svg
:alt: The Zenmark mark
:width: 120
:align: center
:name: zen-mark

The Zenmark mark.

The legend is the paragraphs after the caption.
```

`:name:` gives the figure an id that a link can point to, so [the figure
above](#zen-mark) is a plain link. The `{image}` directive is the picture alone.
See the [figure page](../plugins/figure.md).

A `{table}` takes its caption as the argument and a markdown table as its body:

````markdown
```{table} Stock levels
:name: stock

| Item | In stock |
|------|----------|
| Ink  | 3        |
| Pens | 10       |
```
````

```{table} Stock levels
:name: stock

| Item | In stock |
|------|----------|
| Ink  | 3        |
| Pens | 10       |
```

`{list-table}` and `{csv-table}` build a table from a list or from CSV. See the
[table page](../plugins/table.md).

## Math

A display equation is the `math` block. In MyST it is a `{math}` directive, and
in a folder that uses MyST it can also sit between `$$` lines. Add a `:label:`
and the equation is numbered on its page and becomes a link target.

````markdown
```{math}
:label: eq:energy

E = mc^2
```

$$
\int_0^1 x^2\,dx = \frac13
$$ (eq:integral)
````

```{math}
:label: eq:energy

E = mc^2
```

$$
\int_0^1 x^2\,dx = \frac13
$$ (eq:integral)

Without MyST it is `:::math{label=euler}`:

:::math{label=euler}
e^{i\pi} + 1 = 0
:::

An equation with no label is not numbered. A wrong one keeps its TeX, with the
message under it.

## Code

`{code-block} lang` and `{code} lang` are plain fenced code
in `lang`, so the [code colouring](../plugins/codeblock.md) applies.

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

## Every MyST name and its built-in

Each name belongs to one built-in plugin, and each plugin has a page here.

- [`check`](../plugins/check.md): `{check}`
- [`diagram`](../plugins/diagram.md): `{mermaid}`
- [`rating`](../plugins/rating.md): `{rating}`, `{rating-results}`
- [`quiz`](../plugins/quiz.md): `{quiz}`, `{quiz-results}`
- [`sandbox`](../plugins/sandbox.md): `{sandbox}`
- [`callout`](../plugins/callout.md): `{note}`, `{tip}`, `{hint}`, `{important}`,
  `{seealso}`, `{warning}`, `{caution}`, `{attention}`, `{danger}`, `{error}`,
  `{admonition}`, `{versionadded}`, `{versionchanged}`, `{deprecated}`
- [`figure`](../plugins/figure.md): `{figure}`, `{image}`
- [`table`](../plugins/table.md): `{table}`, `{list-table}`, `{csv-table}`
- [`codeblock`](../plugins/codeblock.md): `{code-block}`, `{code}`
- [`math`](../plugins/math.md): `{math}` (the directive and the role), `{eq}`
- [`tabs`](../plugins/tabs.md): `{tab-set}`, `{tab-item}`
- [`dropdown`](../plugins/dropdown.md): `{dropdown}`
- [`cards`](../plugins/cards.md): `{card}`, `{grid}`, `{grid-item}`, `{grid-item-card}`,
  `{button-link}`, `{button-ref}`, the `{bdg-…}` roles
- [`text`](../plugins/text.md): `{kbd}`, `{abbr}`, `{sub}`, `{sup}`
- [`quote`](../plugins/quote.md): `{epigraph}`, `{pull-quote}`, `{highlights}`
- [`aside`](../plugins/aside.md): `{sidebar}`, `{margin}`, `{topic}`
- [`include`](../plugins/include.md): `{include}`, `{literalinclude}`
- [`terms`](../plugins/terms.md): `{glossary}`, `{term}`
- [`xref`](../plugins/xref.md): `{ref}`, `{numref}`, `{doc}`, `{download}`, `{toctree}`
