# Templates

Templates say what a page, or a whole folder, starts with. They build on the
parts the earlier sections showed, from the top down:

```text
Blueprint        a collection of page templates: a folder's sections
  └ Page template  the structure of one kind of page, for one layout
      └ Layout       how the page looks
          └ Plugin     what a block does
```

They live in Zenmark's settings folder, in `templates/`, with one folder for
each kind. Settings › Templates and Settings › Blueprints show where, and
**Show in Finder** opens it.

```text
templates/
  pages/          page templates, one <id>.yml each
  blueprints/     blueprints, one folder each, with a blueprint.yml
```

A folder can keep its own page templates too, in `.zen/templates/<id>.yml`.
Those win over the ones in `templates/pages/`, which win over the built-in ones.

| Reference | What it covers |
|---|---|
| [Page templates](page-templates.md) | the fields, the starting text and the structure of one kind of page |
| [Blueprints](blueprints.md) | `blueprint.yml`: the questions, the home page and the sections of a new folder |
