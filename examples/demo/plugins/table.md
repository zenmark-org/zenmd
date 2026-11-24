# Tables

A table with a caption, numbered per page. There are three names: `table` takes a
markdown table, `list-table` a list of lists and `csv-table` CSV.

## `table`

The caption is the `title`, or in MyST the text after the name.

````markdown
:::table{title="Stock levels" name=stock}
| Item | In stock |
|------|----------|
| Ink  | 3        |
| Pens | 10       |
:::
````

:::table{title="Stock levels" name=stock}
| Item | In stock |
|------|----------|
| Ink  | 3        |
| Pens | 10       |
:::

````markdown
```{table} Stock levels, MyST form
:name: stock-myst
:align: center

| Item | In stock |
|------|----------|
| Ink  | 3        |
| Pens | 10       |
```
````

```{table} Stock levels, MyST form
:name: stock-myst
:align: center

| Item | In stock |
|------|----------|
| Ink  | 3        |
| Pens | 10       |
```

## `list-table`

One item for each row, and one inside it for each cell. `header-rows`,
`stub-columns`, `widths` and `align` work as in MyST:

````markdown
```{list-table} Supplies
:header-rows: 1
:stub-columns: 1
:widths: 30 70

* - Item
  - Notes
* - Ink
  - Black, **refillable**
* - Pens
  - Sold in packs of `10`
```
````

```{list-table} Supplies
:header-rows: 1
:stub-columns: 1
:widths: 30 70

* - Item
  - Notes
* - Ink
  - Black, **refillable**
* - Pens
  - Sold in packs of `10`
```

:::list-table{title="Supplies, container form" header-rows=1}
* - Item
  - Notes
* - Ink
  - Black
:::

## `csv-table`

CSV in the body. `header-rows`, `header` and `delim` are read too:

````markdown
```{csv-table} Prices
:header-rows: 1

Item, Price
Apple, 1.20
"Pear, green", 2.50
```
````

```{csv-table} Prices
:header-rows: 1

Item, Price
Apple, 1.20
"Pear, green", 2.50
```

:::csv-table{title="Prices, container form" header-rows=1}
Item, Price
Apple, 1.20
Plum, 0.80
:::

`:file:` reads a CSV file of the folder instead, at most 1 MB, and never one
outside the folder:

````markdown
```{csv-table} Prices
:file: data/prices.csv
:header-rows: 1
```
````
