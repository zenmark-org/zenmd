# Cross-references

The `xref` plugin links to a heading or figure of the same page, to another page
with its title, to a file, and lists pages. Each name works as a MyST role and as
a Zenmark inline.

## A link to a label

A `(label)=` line before a heading, or a `:name:` on a block, is a target.

````markdown
(where-to-go)=
## Where to go

See {ref}`where-to-go`, or {ref}`this section <where-to-go>`.
See :ref[where-to-go].
````

(where-to-go)=
## Where to go

See {ref}`where-to-go`, or {ref}`this section <where-to-go>`. The Zenmark form
is :ref[where-to-go].

## The number of a figure

```{figure} ../images/zen-mark.svg
:name: the-mark
:alt: The Zenmark mark

The mark of Zenmark.
```

Look at {numref}`the-mark`, or {numref}`Fig. %s <the-mark>`, or :numref[the-mark].

````markdown
Look at {numref}`the-mark`, or {numref}`Fig. %s <the-mark>`, or :numref[the-mark].
````

## A link to a page

{doc}`../features/myst` shows the page's title; {doc}`the MyST guide <../features/myst>`
shows yours. The Zenmark form is :doc[../features/myst].

````markdown
{doc}`../features/myst`  {doc}`the MyST guide <../features/myst>`  :doc[../features/myst]
````

## A link to a file

{download}`../images/zen-mark.svg` links to a file of the folder; :download[../images/zen-mark.svg] is the same.

## A list of pages

```{toctree}
:caption: More about MyST

../features/myst
The Markdown guide <../features/markdown>
```

````markdown
```{toctree}
:caption: More about MyST

../features/myst
The Markdown guide <../features/markdown>
```
````

Paths are relative to the page. A path that leaves the folder is refused.
