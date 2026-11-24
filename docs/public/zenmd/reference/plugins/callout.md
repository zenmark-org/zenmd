# callout

Sets a note apart from the text, as the admonitions of MyST, Jupyter Book and
Sphinx do. Each kind has a title and a coloured rule on the left. A callout can
fold. On by default.

## Names

Every name works as a container, `:::note`, and in MyST as a fence,
```` ```{note} ````, or a colon fence, `:::{note}`. The text after the MyST name
is the title.

| Name | Title | Rule |
|---|---|---|
| `note`, `tip`, `hint`, `important`, `seealso` | the kind's own: Note, Tip, Hint, Important, See also | blue |
| `warning`, `caution`, `attention` | Warning, Caution, Attention | amber |
| `danger`, `error` | Danger, Error | red |
| `admonition` | the title you give, required | blue, or the colour of the kind in `class` |
| `versionadded`, `versionchanged`, `deprecated` | a one-line note: "New in version", "Changed in version", "Deprecated since version" | none |

## Options

| Option | Default | What it does |
|---|---|---|
| `title` | the kind's title | Replaces the title. In MyST, the text after the name. |
| `class` | none | `dropdown` folds the callout. For `admonition`, a kind such as `warning` picks the colour. Other words are kept as classes. |
| `open` | off | With `dropdown`, starts it unfolded. |
| `name` | none | An id a link can point to. |
| `version` | none | For the three version names: the version. In MyST, the text after the name. |

An option not listed is ignored, with a page warning ("option :x: is not
supported and was ignored"). An `admonition` with no title is shown as Note,
and a version note with no version shows only its lead words; both warn too.

## Example

````markdown
:::warning{title="Heads up"}
A title replaces the default one.
:::

```{tip} Closed at first
:class: dropdown
Hidden until opened.
```
````
