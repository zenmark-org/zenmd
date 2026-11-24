# codeblock

Colours fenced code by language and gives each block a Copy button and a Wrap
button. A block with no language, or one the colouring does not know, stays
plain. On by default.

## Names

`codeblock` owns no name of its own. It works on every fence that has a
language. In a folder that uses MyST,
`{code-block} lang` and `{code} lang` are plain fences in `lang`, so they are
coloured too.

## Options

Set them in a page's frontmatter, in `.zenmd.yml` or in Settings › Plugins.

| Option | Default | What it does |
|---|---|---|
| `style` | `zen` | The colours. Only `zen` is shipped. |
| `lineNumbers` | `false` | Show a line-number gutter. |
| `wrap` | `false` | Wrap long lines at first. The Wrap button changes it for one block. |

An option the plugin does not know is ignored.

## Example

````markdown
---
plugins:
  codeblock: { lineNumbers: true }
---
# A page

```go
func Hello(name string) string {
	return "hello, " + name
}
```
````
