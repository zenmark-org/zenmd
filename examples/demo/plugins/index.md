---
nav:
---
# Plugins

Plugins add renderers to Zenmark. The built-in ones run when a page is
rendered, and their output is cleaned like everything else, so the content of a
page can never inject scripts into it.

| Plugin | Does | On by default |
|---|---|---|
| [codeblock](codeblock.md) | colours code blocks, adds Copy and Wrap buttons | yes |
| [diagram](diagram.md) | draws mermaid diagrams | yes |
| [check](check.md) | a checklist block | yes |
| [quiz](quiz.md) | questions with saved answers and scores | yes |
| [sandbox](sandbox.md) | runs a code block in a container | yes, after setup |
| [rating](rating.md) | star ratings kept on this Mac | yes |
| [callout](callout.md) | notes, tips, warnings and version notes | yes |
| [figure](figure.md) | a picture with a numbered caption | yes |
| [table](table.md) | a table with a numbered caption | yes |
| [math](math.md) | TeX math: equations, inline math, links to equations | yes |
| [tabs](tabs.md) | tab sets whose tabs can stay in step | yes |
| [dropdown](dropdown.md) | a title that opens and closes its text | yes |
| [cards](cards.md) | cards, grids, link buttons and badges | yes |
| [text](text.md) | key caps, abbreviations, subscript and superscript | yes |
| [quote](quote.md) | epigraph, pull quote and highlights | yes |
| [aside](aside.md) | sidebar, margin and topic beside the text | yes |
| [include](include.md) | include and literalinclude: files of the folder in a page | yes |
| [terms](terms.md) | glossary and links to its terms | yes |
| [xref](xref.md) | cross-references: ref, numref, doc, download, toctree | yes |

Settings has a **Plugins** section that lists them with help and snippets to
copy into a page; see [Settings](../configuration/settings.md).

Turn one off, or set its options, in frontmatter or `.zenmd.yml`:

```yaml
plugins:
  codeblock: true
  diagram: { theme: neutral }   # a mapping turns it on with options
  check: false                  # false turns it off
```

A plugin is a part of a page. The look of a whole page is a
[layout](../layouts/index.md).
