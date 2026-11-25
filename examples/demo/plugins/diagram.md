---
plugins:
  diagram: { theme: neutral }
---
# Diagrams

Write a ` ```mermaid ` block and Zenmark draws it. Diagrams follow the light or
dark theme.

## Flowchart

```mermaid
graph LR
  A[Open folder] --> B{README.md?}
  B -- yes --> C[Show README]
  B -- no --> D[Pick a page]
  C --> E[Read]
  D --> E
  E -->|follow a link| C
```

## Sequence

```mermaid
sequenceDiagram
  participant You
  participant Sidebar
  participant Page
  You->>Sidebar: choose a page
  Sidebar->>Page: open it
  Page-->>You: the rendered page
```

## A diagram from a file

A diagram can live in its own file. A path that starts with `/` is from the
folder's root; any other path is from this page's folder.

```mermaid
:file: /plugins/diagrams/flow.mmd
```

## Zoom and pan

A large diagram is shrunk to the reading column. Point at it (or Tab to it) to
get the controls in its corner: zoom in and out, reset, pan, and expand, which
opens it over the whole window. A pinch (or ⌘ or ⌃ with the wheel) zooms
around the pointer, and dragging pans once it is zoomed. With the figure
focused, `+`, `-` and `0` zoom and reset, the arrows pan, and Enter expands.

```mermaid
graph LR
  A[Write a page] --> B[Save]
  B --> C{Valid Markdown?}
  C -- no --> D[Show the problem]
  D --> A
  C -- yes --> E[Render]
  E --> F[Parse frontmatter]
  F --> G[Resolve links]
  G --> H{Plugins on?}
  H -- yes --> I[Run plugins]
  H -- no --> J[Plain page]
  I --> K[Draw diagrams]
  I --> L[Draw tables]
  I --> M[Draw figures]
  K --> N[Sanitise]
  L --> N
  M --> N
  J --> N
  N --> O[Show the page]
  O --> P{Export?}
  P -- site --> Q[Build the site]
  P -- print --> R[Print or PDF]
  Q --> S[Publish]
  R --> S
  S --> T[Readers open it]
```

## When a diagram is wrong

The source stays visible with mermaid's error message under it:

```mermaid
graph TD
  A --> 
```

## Options

```yaml
plugins:
  diagram: { theme: neutral }   # neutral (default), default or dark
```
