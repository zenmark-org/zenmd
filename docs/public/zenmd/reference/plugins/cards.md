# cards

Cards, grids, buttons that are links, and badges, as written for MyST, Jupyter
Book and Sphinx (sphinx-design). One built-in owns all of them.

Everything is drawn in Zenmark's own black and white style. Colour options are
accepted and ignored with a warning.

## Names

| Name | What it draws |
|---|---|
| `card` | A framed card with an optional header, body and footer. |
| `grid` | A responsive grid; the argument is the number of columns at each width. |
| `grid-item` | One cell of a grid, with plain content. |
| `grid-item-card` | One cell of a grid that is a card. |
| `button-link` | A button that opens a link. |
| `button-ref` | A button that opens a target in the folder: a label, or a page. |
| `bdg`, `bdg-primary`, `bdg-secondary`, `bdg-success`, `bdg-info`, `bdg-warning`, `bdg-danger`, `bdg-light`, `bdg-dark` | A small text badge. |
| the same with `-line`, such as `bdg-primary-line` | The same badge with no fill. |

Every block works as `:::name{...}`, as `:::{name}` and as a fenced
`` ```{name} `` block. Badges are written `:bdg-primary[text]`.

## Card

````markdown
:::card{title="A card"}
Header text
^^^
The body is markdown.
+++
Footer text
:::
````

MyST form:

````markdown
:::{card} A card
Header text
^^^
The body is markdown.
+++
Footer text
:::
````

A line of `^^^` ends the header; a line of `+++` starts the footer. Both are
optional. Inside a code fence they are code.

| Option | Meaning |
|---|---|
| `title` | The card's title (the argument in the MyST form). |
| `link` | Makes the whole card a link. |
| `link-type` | What `link` is: `url` (default), `ref` (a label on the page), `doc` (a page of the folder; `.md` is added when missing) or `any` (decided by its look). |
| `link-alt` | The link's text when the card has no title. |
| `text-align` | `left`, `center` or `right`. |
| `class`, `name` | Extra class names, and an id. |

Other options (`shadow`, `img-top`, `width`, `class-card`, ...) are ignored with
a warning.

## Grid

````markdown
:::grid{columns="1 2 2 3"}
:::grid-item-card{title="One"}
First card.
:::
:::grid-item-card{title="Two"}
Second card.
:::
:::
````

In the container form every block ends with `:::`. MyST form: `::::{grid} 1 2 2 3`
with `::::` to close it.

The columns are for the extra-small, small, medium and large widths of the
reading column: under 480 px, from 480 px, from 640 px and from 800 px. Give 1
to 4 numbers from 1 to 12; a missing number repeats the one before it. No
argument means one column.

| Option | Meaning |
|---|---|
| `columns` | The argument above, for the container form. |
| `gutter` | The gap between cells, 0 to 5, or four numbers for the four widths. |
| `class`, `name` | Extra class names, and an id. |

`grid-item` and `grid-item-card` take `class` and `name`; a card cell takes the
card's options. Options such as `columns` on an item, `reverse` and
`child-direction` are ignored with a warning.

## Buttons

````markdown
:::button-link{href="https://example.com"}
Open the site
:::
````

MyST form: the argument is the target and the body is the text.

````markdown
```{button-link} https://example.com
Open the site
```

```{button-ref} my-label
:ref-type: ref
Go to the section
```
````

| Option | Meaning |
|---|---|
| `href` or `ref` | The target, in the container form. |
| `ref-type` | For `button-ref`: `any` (default), `ref`, `doc` or `myst`. |
| `align` | `left`, `center` or `right`. |
| `expand` | A full-width button. |
| `class`, `name` | Extra class names, and an id. |

`color`, `outline`, `shadow` and `tooltip` are ignored with a warning. A link
with a scheme other than http, https or mailto is refused with a warning.

## Badges

````markdown
Status: :bdg-success[stable] :bdg-warning-line[beta] :bdg[plain]
````

The text is plain text. The colour names are accepted so Sphinx pages read the
same, but every badge is drawn alike.
