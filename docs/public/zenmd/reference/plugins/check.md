# check

A checklist. On a page with `layout: course`, ticking every box of a step
unlocks the next step. On any other page it is a plain checklist that shows
the boxes as written. On by default.

## Names

| Name | Written as |
|---|---|
| `check` | `:::check`, or in MyST ```` ```{check} ```` and `:::{check}` |

## Options

None. The body is a task list: `- [ ]` is an open box, `- [x]` a ticked one.

## Example

````markdown
:::check
- [ ] I opened a folder
- [ ] I can see its pages
:::
````
