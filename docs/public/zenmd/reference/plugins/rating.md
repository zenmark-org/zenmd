# rating

One question and 1 to 5 stars (or up to 10). The rating is kept on this Mac,
and a results block lists every rating in the folder. On by default.

## Names

| Name | What it draws | Written as |
|---|---|---|
| `rating` | One question with stars. | `:::rating`, or ```` ```{rating} ```` in MyST |
| `rating-results` | Every page you rated in the folder. | `:::rating-results` |

## Options of `rating`

| Option | Default | What it does |
|---|---|---|
| `id` | none, required | Names the stored rating. One to 57 letters, digits, `.`, `_` or `-`, and unique in the page. |
| `title` | `Rate this page` | The question. |
| `max` | `5` | The number of stars, 3 to 10. |
| `labels` | none | One label per star, written `Poor\|Fair\|Good`. |

Text inside the block shows under the question. A wrong `id`, `max` or
`labels` shows a message in place of the block, and the page reports it. An
option the plugin does not know is ignored. In MyST, write the options as
`:key: value` lines.

## Example

````markdown
:::rating{id=guide title="How useful was this page?" max=3 labels="Not much|Useful|Very useful"}
:::
````
