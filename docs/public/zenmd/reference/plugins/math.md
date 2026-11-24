# math

Draws TeX math with KaTeX: a centred equation, or a formula inside a line. KaTeX
is loaded only on pages that have math. Before it loads, and on a page read
without scripts, the TeX shows as text.

## Names it owns

| Name | Kind | MyST form | Zenmark form |
|---|---|---|---|
| `math` | block | `` ```{math} `` or `:::{math}` | `:::math` |
| `math` | block | `$$ … $$` (MyST dialect) | |
| `math` | inline | `` {math}`E = mc^2` `` | `:math[E = mc^2]` |
| `math` | inline | `$E = mc^2$` (MyST dialect, `dollarmath` on) | |
| `eq` | inline | `` {eq}`label` `` | `:eq[label]` |

### A display equation

````markdown
:::math{label=euler}
e^{i\pi} + 1 = 0
:::
````

````markdown
```{math}
:label: euler

e^{i\pi} + 1 = 0
```
````

Between two `$$` lines, with an optional label after the closing `$$`:

```markdown
$$
\int_0^1 x^2\,dx = \frac13
$$ (eq:integral)
```

### An equation inside a line

```markdown
The energy is :math[E = mc^2].
The energy is {math}`E = mc^2`.
```

`$E = mc^2$` does the same when the folder turns on MyST's `dollarmath`
extension: `myst_enable_extensions` in a Jupyter Book `_config.yml` or a
Sphinx `conf.py`, or `enable_extensions` in a `myst.yml`. Without it, a `$` is
a dollar sign. Money is safe even with it on: `$5 and $10` is not math, because
the TeX may not start or end with a space and the closing `$` may not be
followed by a digit.

### A link to an equation

```markdown
See :eq[euler] for the identity.
See {eq}`euler` for the identity.
```

It shows the equation's number, `(1)`, and links to it. A label that is not a
numbered equation of the page shows a warning, and the text stays.

## Options

| Option | Does |
|---|---|
| `label` | Numbers the equation on its page, (1), (2) and so on, and makes it a link target. |
| `name` | Makes a link target without a number. `label` wins when both are given. |
| `class` | Adds a class name to the block. |
| `nowrap` | Does not put an equation with `\\` in `aligned`. |

Options are written as `:label: value` lines or a `---` YAML block in the MyST
form, and as `{label=value}` in the Zenmark form. An option the plugin does not
support shows a page warning and is ignored.

## When something is wrong

A TeX error keeps the TeX visible, with KaTeX's message under it (for a formula
inside a line, as its tooltip). KaTeX never runs with `trust`, so `\href`,
`\url` and `\includegraphics` do nothing.
