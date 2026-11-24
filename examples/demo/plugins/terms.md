# Glossary terms

A glossary is a definition list in which every term is a link target. A term
link jumps to it from the same page.

## Syntax

````markdown
:::glossary
API
  Application programming interface.

REST : protocol
  A style of API built on HTTP.
:::

The :term[API] of a :term[REST service <REST>].
````

Each term is a line of its own; its definition is indented under it and is
markdown. The id of a term is `term-` and its text in lower case, so `API` is
`term-api`.

## Try it

Here is the glossary, in the MyST form (`sorted` sorts the terms):

```{glossary}
:sorted:

REST : protocol
  A style of :term[API] built on HTTP.

API
  Application programming interface. A set of calls one program offers to
  another.

Idempotent
  A call you can repeat with the same result.
```

Now links: the :term[API] of a :term[REST service <REST>] and an
:term[idempotent] call. In a folder that uses MyST the role form is
`` {term}`API` ``.

A link to a term that is not in a glossary on the page, such as `:term[Missing]`,
is a warning.
