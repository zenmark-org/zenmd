# xref

Links inside a page and inside the folder: to a heading or another block of the
same page, to a figure's number, to another page with its title, to a file, and a
list of pages.

## Names it owns

| Name | Kind | MyST form | Zenmark form |
|---|---|---|---|
| `ref` | inline | `` {ref}`label` `` | `:ref[label]` |
| `numref` | inline | `` {numref}`label` `` | `:numref[label]` |
| `doc` | inline | `` {doc}`guide/start` `` | `:doc[guide/start]` |
| `download` | inline | `` {download}`images/chart.png` `` | `:download[images/chart.png]` |
| `toctree` | block | `` ```{toctree} `` or `:::{toctree}` | `:::toctree` |

Every role takes a title: `` {ref}`the setup section <setup>` `` shows "the setup
section".

### ref and numref

A label is a `(label)=` line before a heading or a block (MyST dialect), the
`:name:` of a figure, table or other block, or the `:label:` of an equation.

````markdown
(setup)=
## Setup

See {ref}`setup`, or {ref}`the setup section <setup>`.
See :ref[setup].
````

`ref` shows the heading's text, or the number when the target has no heading.
`numref` shows the number of a figure ("Figure 1"), a table ("Table 1") or an
equation ("(1)"); its text may use `%s` or `{number}` for the number and `{name}`
for the heading:

```markdown
{numref}`Fig. %s <sales>`   shows  Fig. 1
```

A label the page does not have shows a warning and leaves the text. A `numref`
to a target with no number links like `ref` and warns.

### doc

A link to a page of the folder. The path is relative to the page, with or
without `.md`. It shows the page's title: its first H1, else its `title:`, else
its file name. With a title written in the role, that is shown.

```markdown
{doc}`guide/start`   :doc[guide/start]   {doc}`the guide <guide/start>`
```

### download

A link to a file of the folder, with the same path rules as `doc`. It shows the
file's name unless a title is given. Zenmark links to markdown pages and
images; another kind of file shows a warning and stays text.

### toctree

A list of links to pages, with their titles.

````markdown
```{toctree}
:caption: Guide

guide/start
The basics <guide/basics>
https://example.com
```
````

```markdown
:::toctree{caption="Guide"}
guide/start
The basics <guide/basics>
:::
```

An entry is a path, or `Title <path>`, or an `http(s)` address. It does not
change the sidebar.

## Options

| Option | Does |
|---|---|
| `caption` | A title over the list. |
| `hidden` | Shows nothing (MyST keeps a hidden toctree out of the page). |
| `name` | Makes the list a link target. |
| `class` | Adds a class name. |
| `maxdepth`, `numbered`, `titlesonly`, `reversed`, `includehidden`, `glob` | Accepted so a Sphinx page reads; they change nothing. A glob pattern in an entry is not expanded and shows a warning. |

Any other option shows a page warning and is ignored.

## Paths stay in the folder

A path is relative to the page, and `..` may climb to a folder above the page's
but never out of the open folder. A path that leaves the folder, an absolute
path, a backslash and a scheme such as `file:` or `javascript:` are refused with
a warning, and the text stays text. A page or file that is not in the folder shows a warning too.

## Order

`xref` is registered after `figure`, `table` and `math`, because it reads the
numbers they give when the page is finished.
