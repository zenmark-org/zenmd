# aside

The `aside` plugin draws content that sits beside the text: a sidebar, a note
in the margin and a topic. Each is an `aside` in the plugin kit's frame. On a
wide page it floats to the right of the text; on a narrow page it takes the full
width, where it stands.

## What it owns

| Name | Draws | zen form | MyST form |
|---|---|---|---|
| `sidebar` | a framed aside with a title | `:::sidebar{title="…"}` | `` ```{sidebar} Title `` |
| `margin` | a narrow aside, usually with no title | `:::margin` | `` ```{margin} `` |
| `topic` | a framed passage on one subject, with a title | `:::topic{title="…"}` | `` ```{topic} Title `` |

Each also works as a MyST colon fence: `:::{sidebar} Title`. The body is
markdown.

## Options

| Option | On | Does |
|---|---|---|
| `title` | all | The title. In the MyST form it is the text after the name. |
| `subtitle` | `sidebar` | A muted line under the title. |
| `class` | all | Adds your own class names to the block. |
| `name` | all | Gives the block an `id`, so a link or `{ref}` can point to it. |

Any other option, such as `width`, is ignored, and the page has a warning that
names it.

## Example

````markdown
```{sidebar} Did you know?
:subtitle: A side note

The body is **markdown**.
```
````

The same in the zen form:

```markdown
:::sidebar{title="Did you know?" subtitle="A side note"}
The body is **markdown**.
:::
```
