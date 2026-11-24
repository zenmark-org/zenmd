# terms

The `terms` plugin draws a glossary and links to its terms. `glossary` is a
definition list in which every term is a link target, and `term` links to one
of them from the same page.

## What it owns

| Name | Draws | zen form | MyST form |
|---|---|---|---|
| `glossary` | a definition list; each term has the id `term-<slug>` | `:::glossary` | `` ```{glossary} `` and `:::{glossary}` |
| `term` | a link to a term on the same page | `:term[API]` | `` {term}`API` `` |

## glossary

Write each term on a line of its own and its definition indented under it. The
definition is markdown and may have several paragraphs. Several terms above one
definition share it. `term : classifier` adds a muted classifier after the term.

The slug of a term is its text in lower case, with a dash for anything that is
not a letter or digit: `Dead Letter Queue` is `term-dead-letter-queue`. A letter
that is not ASCII becomes `u` and its four hex digits. A term written twice
gets `-2` on the second, and the page has a warning.

| Option | Does |
|---|---|
| `sorted` | Sorts the terms, not case-sensitive. |
| `class` | Adds your own class names to the block. |
| `name` | Gives the block an `id`. |

Any other option is ignored, and the page has a warning that names it.

## term

`:term[API]` links to the term `API`. To show other text, give the term in angle
brackets: `:term[the API <API>]`. The zen form is not recognised right after a
letter or a digit; write a space before it. The MyST role form
`` {term}`API` `` has no such rule.

A link to a term that is not in a glossary on the same page is shown as a link
and the page has a warning. To link to a term on another page, use `{ref}` or
`{doc}`.

## Example

```markdown
:::glossary
API
  Application programming interface.

REST : protocol
  A style of API built on HTTP.
:::

The :term[API] of a :term[REST service <REST>].
```

The same in the MyST form:

````markdown
```{glossary}
:sorted:

API
  Application programming interface.

REST
  A style of API built on HTTP.
```

The {term}`API` and {term}`REST`.
````
