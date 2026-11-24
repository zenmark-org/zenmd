# Math

The `math` plugin draws TeX math: a centred equation, or a formula inside a line.
It is drawn with KaTeX, which loads only on pages that have math.

## An equation

````markdown
```{math}
:label: euler

e^{i\pi} + 1 = 0
```
````

```{math}
:label: euler

e^{i\pi} + 1 = 0
```

`:::math{label=euler}` is the same block without MyST. A label numbers the
equation and makes it a link target. Between two `$$` lines works too:

$$
\int_0^1 x^2\,dx = \frac13
$$ (eq:integral)

## Inside a line

The energy is {math}`E = mc^2`, and the area of a circle is :math[\pi r^2].

````markdown
The energy is {math}`E = mc^2`.
The area is :math[\pi r^2].
````

`$E = mc^2$` does the same in a folder that turns on MyST's `dollarmath`
extension (`myst_enable_extensions: [dollarmath]` in `_config.yml`). This guide
does not, so a `$` here is a dollar sign.

## A link to an equation

See {eq}`euler` for the identity, and :eq[eq:integral] for the integral.

````markdown
See {eq}`euler` for the identity, and :eq[eq:integral] for the integral.
````

The options are listed in the plugin's reference page.
