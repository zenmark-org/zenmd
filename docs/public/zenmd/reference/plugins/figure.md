# figure

A picture with a caption and an optional legend, numbered per page as
Figure 1, Figure 2. An image is the picture alone. The file is found the way a
page image is, relative to the page's folder. On by default.

## Names

| Name | What it draws | Written as |
|---|---|---|
| `figure` | A picture, a caption and a legend. | `:::figure{src=…}`, or ```` ```{figure} path ```` in MyST |
| `image` | The picture alone, with no caption or number. | `:::image{src=…}`, or ```` ```{image} path ```` in MyST |

In a figure, the first paragraph of the body is the caption and the paragraphs
after it are the legend.

## Options

In MyST, write them as `:key: value` lines at the top of the body, or between
`---` lines. In the container form, put them in braces.

| Option | Default | What it does |
|---|---|---|
| `src` | none, required | The image path. In MyST it is the text after the name. |
| `alt` | none | The text for people who cannot see the picture. |
| `width`, `height` | the picture's own | A number (pixels) or a percent. |
| `align` | `center` for a figure, `left` for an image | `left`, `center` or `right`. |
| `name` | none | An id a link can point to. |
| `target` | none | A link the picture opens. |
| `class` | none | Extra classes on the figure. |
| `figclass` | none | Extra classes on the caption frame. |
| `nonumber` | off | Leave the figure unnumbered. `numbered=false` does the same. |

An option not listed is ignored, with a page warning ("option :x: is not
supported and was ignored"). A value that cannot be used, such as `align: top`
or a `width` that is not px or %, is ignored the same way. An image that is not
found warns and shows a note in its place.

## Example

````markdown
```{figure} ../images/zen-mark.svg
:alt: The Zenmark mark
:width: 120
:name: zen-mark

The Zenmark mark.
```
````
