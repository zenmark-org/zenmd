# Reader

The reader is the main view of Zenmark: it shows one page of the folder at a
time, as a clean article. You are using it now.

## Reading a page

Pick a page in the sidebar, or follow a link in the text. The page shows its
headings, lists, tables, images and code, and the plugins this build has draw
their blocks in place. Layouts change the look of a whole page; the
Layouts part of this guide shows each one.

- **Zoom** with <kbd>⌘+</kbd> and <kbd>⌘-</kbd>, and reset with <kbd>⌘0</kbd>.
- **Light or dark** follows the Mac, or the Appearance setting.
- **Text size, font, line spacing and width** are in Settings, under
  Accessibility.

## Markdown

Every folder reads CommonMark and GFM, and Settings › Reader lists them first,
as the default. Two pages go further:

- [Markdown](markdown.md) is one page with every kind of text Zenmark renders.
- [MyST](myst.md) turns on another syntax for the same blocks, for folders
  written for MyST, Jupyter Book or Sphinx.

## Live reload

Change a file in any editor and the open page updates by itself, keeping your
scroll position. New, renamed and deleted pages update the sidebar.

## Wide tables

A table wider than the reading column scrolls sideways. It also gets an
**Expand** button above it, which opens the table over the whole window: the
header row and the row headers stay in view as you scroll, + and − change the
text size, Fit picks the largest size that shows every column, and Esc closes
it. A table that fits has no button.

| Item | Region | Q1 | Q2 | Q3 | Q4 | Total | Share | Target | Notes |
|------|--------|----|----|----|----|-------|-------|--------|-------|
| Ink | North | 120 | 135 | 142 | 160 | 557 | 21% | 540 | Refillable cartridges sold best |
| Pens | North | 340 | 310 | 295 | 380 | 1325 | 50% | 1400 | The `fine` tip is back in stock |
| Paper | South | 90 | 95 | 110 | 130 | 425 | 16% | 400 | Recycled stock only |
| Tape | South | 40 | 45 | 38 | 52 | 175 | 7% | 200 | See the [Markdown](markdown.md) page for more tables |
| Glue | East | 22 | 18 | 25 | 31 | 96 | 4% | 100 | Discontinued in Q3 |
| Clips | East | 15 | 20 | 22 | 17 | 74 | 3% | 80 | Sold in boxes of 100 |

:::csv-table{title="Stock by quarter" header-rows=1 stub-columns=1}
Item, Region, Q1, Q2, Q3, Q4, Total, Target, Supplier, Lead time, Notes
Ink, North, 120, 135, 142, 160, 557, 540, Acme Ltd, 5 days, Refillable cartridges sold best
Pens, North, 340, 310, 295, 380, 1325, 1400, Penworks, 9 days, The fine tip is back in stock
Paper, South, 90, 95, 110, 130, 425, 400, Greenleaf, 3 days, Recycled stock only
Tape, South, 40, 45, 38, 52, 175, 200, Acme Ltd, 5 days, Sold in rolls of 12
:::
