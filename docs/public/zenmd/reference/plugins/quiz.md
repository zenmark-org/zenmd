# quiz

Quizzes written in markdown. The answers stay out of the page: Zenmark checks
them when you press Check, and keeps your scores on this Mac. On by default.

## Names

| Name | What it draws | Written as |
|---|---|---|
| `quiz` | A quiz of numbered questions. | `:::quiz`, or ```` ```{quiz} ```` in MyST |
| `quiz-results` | The scores of every quiz in the folder. | `:::quiz-results` |

## Questions

| Write | For |
|---|---|
| `( )` and `(x)` | one right option |
| `[ ]` and `[x]` | several right options |
| `question = answer` | a free-text answer |
| a line starting with `>` | the explanation shown after checking |

## Options of `quiz`

| Option | Default | What it does |
|---|---|---|
| `id` | none, required | Names the stored results. One to 59 letters, digits, `.`, `_` or `-`, and unique in the page. |
| `title` | none | The heading. |
| `pass` | `70` | The percent that passes, 0 to 100. |
| `attempts` | no limit | The most tries. |
| `shuffle` | off | Mix the options. |
| `reveal` | all at the end | `each` checks one question at a time. |

A wrong `id` shows a message in place of the quiz, and the page reports it. An
option the plugin does not know is ignored.

## Example

````markdown
:::quiz{id=http-basics title="HTTP basics" pass=70}
1. Which method is idempotent?
   - ( ) POST
   - (x) PUT
   > PUT replaces the resource, so repeating it changes nothing.
2. What does TLS stand for? = Transport Layer Security
:::
````
