# table

A table with an optional caption, numbered per page as Table 1, Table 2. On by
default.

## Names

| Name | Body | Written as |
|---|---|---|
| `table` | A markdown table. | `:::table{title=…}`, or ```` ```{table} Caption ```` in MyST |
| `list-table` | A bullet list of bullet lists: one item per row, one inside it per cell. | `:::list-table`, or ```` ```{list-table} Caption ```` |
| `csv-table` | CSV text, or a file of the folder. | `:::csv-table`, or ```` ```{csv-table} Caption ```` |

## Options

In MyST, write them as `:key: value` lines at the top of the body, or between
`---` lines. In the container form, put them in braces.

| Option | Applies to | Default | What it does |
|---|---|---|---|
| `title` | all | none | The caption. In MyST, the text after the name. |
| `name` | all | none | An id a link can point to. |
| `align` | all | none | `left`, `center` or `right`. |
| `width` | all | automatic | The width of the table: pixels or a percent. |
| `class` | all | none | Extra classes. |
| `nonumber` | all | off | Leave the table unnumbered. `numbered=false` does the same. |
| `header-rows` | `list-table`, `csv-table` | `0` | How many rows are the header. |
| `stub-columns` | `list-table`, `csv-table` | `0` | How many columns are row headers. |
| `widths` | `list-table`, `csv-table` | automatic | Relative column widths, one per column, such as `30 70`; `auto` leaves it to the browser. |
| `header` | `csv-table` | none | One header row, as CSV, put before the data. |
| `delim` | `csv-table` | `,` | The separator: one character, `tab` or `space`. |
| `file` | `csv-table` | none | A CSV file of the folder instead of the body: at most 1 MB, and never one outside the folder. |

An option not listed for that name is ignored, with a page warning ("option
:x: is not supported and was ignored"). A value that cannot be used, such as
`widths` that do not match the columns, is ignored the same way. A `file` that
is not found in the folder shows an error in place of the table.

## Example

````markdown
```{csv-table} Prices
:header-rows: 1

Item, Price
Apple, 1.20
"Pear, green", 2.50
```
````
