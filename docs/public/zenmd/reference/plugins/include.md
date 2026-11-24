# include

The `include` plugin brings a file of the folder into a page. `include` shows
the file's markdown where the block stands, and `literalinclude` shows the
file's text as code.

## What it owns

| Name | Draws | zen form | MyST form |
|---|---|---|---|
| `include` | the markdown of a file, as part of the page | `:::include{file=path}` | `` ```{include} path `` and `:::{include} path` |
| `literalinclude` | the text of a file as a code block | `:::literalinclude{file=path}` | `` ```{literalinclude} path `` and `:::{literalinclude} path` |

## Which file

The path is relative to the page's folder, then to the folder's root. A file is
refused when:

- the path has `..` or is absolute;
- it is a symbolic link that leads out of the folder, or one whose target can't be checked;
- it is larger than 1 MB;
- it is not a regular file.

A refusal shows an error in the block and a page warning; neither names the path.
A file that includes itself, directly or through other files, stops with a warning.
The frontmatter of an included markdown file is not shown.

## Options

| Option | On | Does |
|---|---|---|
| `file` | both | The path, in the zen form. In the MyST form it is the text after the name. |
| `start-line` | both | The first line shown, counted from 0. |
| `end-line` | both | The first line left out, counted from 0. |
| `lines` | both | The lines shown, counted from 1, such as `1-3,5` or `4-`. It wins over `start-line` and `end-line`. |
| `start-after` | both | Show only what comes after the first line that has this text. |
| `end-before` | both | Show only what comes before the first line that has this text. |
| `language` | `literalinclude` | The code language. Without it, the file's extension decides. |
| `linenos` | `literalinclude` | Show line numbers. |
| `lineno-start` | `literalinclude` | The number of the first line (turns numbers on). |
| `lineno-match` | `literalinclude` | Numbers match the lines of the file. |
| `caption` | `literalinclude` | A title over the code. |
| `class`, `name` | both | Your own class names, and an `id` for links. |

Any other option, such as `emphasize-lines`, is ignored, and the page has a
warning that names it. Line numbers are left out of a block longer than 5000
lines. An included code block is not coloured by language.

## Example

````markdown
```{literalinclude} code/hello.go
:lines: 1-3,5
:linenos:
```
````

The same in the zen form:

```markdown
:::literalinclude{file=code/hello.go lines="1-3,5" linenos}
:::
```
