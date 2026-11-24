# Cards

Cards, grids, buttons that are links, and badges, written as in MyST, Jupyter
Book and Sphinx (sphinx-design). Each works as `:::name`, as `:::{name}` and as a
fenced `{name}` block. Zenmark draws all of them in its own black and white
style; the colour options of Sphinx are ignored with a warning.

## A card

````markdown
:::card{title="A card"}
Header text
^^^
The body is markdown.
+++
Footer text
:::
````

A line of `^^^` ends the header and a line of `+++` starts the footer; both are
optional. Add `link` to make the whole card a link.

:::card{title="A card"}
Header text
^^^
The body is **markdown**.
+++
Footer text
:::

The MyST form, as a link:

```{card} Read the guide
:link: https://example.com
The whole card is the link.
```

## A grid

The argument is the number of columns at each width: `1 2 2 3` means one column
when narrow, two, two, then three when wide.

````markdown
::::{grid} 1 2 2 3
:::{grid-item-card} One
First card.
:::
:::{grid-item-card} Two
Second card.
:::
::::
````

::::{grid} 1 2 2 3
:::{grid-item-card} One
First card.
:::
:::{grid-item-card} Two
Second card.
:::
:::{grid-item-card} Three
Third card.
:::
::::

A `grid-item` holds plain content:

:::grid{columns="1 2"}
:::grid-item
**Left.** Plain text in a grid cell.
:::
:::grid-item
**Right.** Another cell.
:::
:::

## Buttons

`button-link` opens a link; `button-ref` opens a label on the page or a page of
the folder.

````markdown
```{button-link} https://example.com
Open the site
```
````

```{button-link} https://example.com
Open the site
```

```{button-ref} badges
:ref-type: ref
Jump to the badges
```

:::button-link{href="https://example.com" align=center expand}
A full-width button
:::

## Badges

A badge is written `:bdg-primary[text]`. The colour names are accepted and every
badge is drawn alike: :bdg[plain] :bdg-primary[primary] :bdg-secondary[secondary]
:bdg-success[success] :bdg-info[info] :bdg-warning[warning] :bdg-danger[danger]
:bdg-light[light] :bdg-dark[dark]. The `-line` forms: :bdg-primary-line[primary]
:bdg-success-line[success].

## Options

| Block | Options |
|---|---|
| `card`, `grid-item-card` | `title`, `link`, `link-type` (`url`, `ref`, `doc`, `any`), `link-alt`, `text-align`, `class`, `name` |
| `grid` | `columns` (or the argument), `gutter` (0 to 5), `class`, `name` |
| `grid-item` | `class`, `name` |
| `button-link`, `button-ref` | `href` or `ref`, `ref-type`, `align`, `expand`, `class`, `name` |
