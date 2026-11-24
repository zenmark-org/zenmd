# Includes

Bring a file of the folder into a page. `include` shows a markdown file where
the block stands, and `literalinclude` shows a file's text as code. Both read
files inside this folder only.

## Syntax

````markdown
```{include} include/snippet.txt
```
````

| Name | Shows |
|---|---|
| `include` | the markdown of a file, as part of the page |
| `literalinclude` | the text of a file, as code |

The path is relative to the page, then to the folder. No `..`, no absolute
path, no link that leads out of the folder, and at most 1 MB.

## Include markdown

Here is `include/snippet.txt`, included:

```{include} include/snippet.txt
```

The zen form writes the path as `file=`. With `start-line` the first lines of
the file are left out (counted from 0), and `end-line` is the first line left
out at the end:

:::include{file=include/snippet.txt start-line=2 end-line=4}
:::

## Include code

```{literalinclude} include/hello.py
:caption: hello.py
```

Choose lines, with `lines` (from 1) or `start-line` and `end-line`, and show
line numbers with `linenos`:

:::literalinclude{file=include/hello.py lines="4-5,9" linenos}
:::

The language comes from the extension; `language` overrides it.

```{literalinclude} include/hello.py
:language: text
:lines: 1-2
```
