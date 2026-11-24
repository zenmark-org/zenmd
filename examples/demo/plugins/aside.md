# Asides

Content beside the text: a sidebar, a note in the margin and a topic. On a wide
page a sidebar and a margin note float to the right of the text; on a narrow
page they take the full width where they stand. Make the window narrower to see
it. A topic stays in the flow of the text, full width.

## Syntax

````markdown
:::sidebar{title="Did you know?"}
The body is markdown.
:::
````

| Name | For |
|---|---|
| `sidebar` | a framed aside with a title and an optional `subtitle`, floated right |
| `margin` | a narrow note, usually with no title, floated right |
| `topic` | a framed passage on one subject, with a title, in the flow |

## A sidebar

:::sidebar{title="Did you know?" subtitle="A side note"}
The body is **markdown**, so it can hold lists and links.
:::

A sidebar is written next to the paragraph it belongs to. The text flows
around it while there is room, and it drops below the paragraph's start when
the page is narrow. This paragraph is here to give the aside something to sit
beside: a few more lines of plain text, enough that you can see the text wrap
around the frame on the right.

## A note in the margin, MyST form

```{margin}
A short note in the margin.
```

Notes in the margin are narrower and have no title. The MyST form is a fence
named `margin`. As with the sidebar, the paragraph it is written before flows
around it, so write it right before the text it comments on.

## A topic

:::{topic} Background
A topic is a self-contained passage about one subject.
:::

The topic is a framed aside too, with its title in the head row, but it sits in
the flow of the text at full width.
