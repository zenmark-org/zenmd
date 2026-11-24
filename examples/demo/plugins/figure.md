# Figures

A **figure** is a picture with a caption, numbered per page. An **image** is the
picture alone. The picture is found the way any page image is, relative to this
page's folder.

## The `:::` form

The first paragraph of the body is the caption; the paragraphs after it are the
legend.

````markdown
:::figure{src=../images/zen-mark.svg alt="The Zenmark mark" width=120 name=mark-colon}
The Zenmark mark.

The legend is the paragraphs after the caption.
:::
````

:::figure{src=../images/zen-mark.svg alt="The Zenmark mark" width=120 name=mark-colon}
The Zenmark mark.

The legend is the paragraphs after the caption.
:::

## The MyST form

The path is the text after the name, and the options are `:key: value` lines.

````markdown
```{figure} ../images/zen-mark.svg
:alt: The Zenmark mark
:width: 120
:align: right
:name: zen-mark

The Zenmark mark, set to the right.
```
````

```{figure} ../images/zen-mark.svg
:alt: The Zenmark mark
:width: 120
:align: right
:name: zen-mark

The Zenmark mark, set to the right.
```

`name` gives the figure an id that a link can point to, so [the figure
above](#zen-mark) is a plain link.

## Images

An image has the same options and no caption or number:

````markdown
:::image{src=../images/zen-mark.svg alt="The Zenmark mark" width=48}
:::
````

:::image{src=../images/zen-mark.svg alt="The Zenmark mark" width=48}
:::

```{image} ../images/zen-mark.svg
:alt: The Zenmark mark
:width: 48
:align: left
```
