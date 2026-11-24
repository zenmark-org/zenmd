# text

The `text` plugin draws small marks inside a line: a key on the keyboard, an
abbreviation with its explanation, a subscript and a superscript. They are
inlines, so they sit in the middle of a sentence. Nothing here runs a script.

## What it owns

| Name | Draws | zen form | MyST role form |
|---|---|---|---|
| `kbd` | one key cap per key, split on `+` and `-` | `:kbd[Ctrl+C]` | `` {kbd}`Ctrl+C` `` |
| `abbr` | an abbreviation with its explanation as a tooltip | `:abbr[HTML (HyperText Markup Language)]` | `` {abbr}`HTML (HyperText Markup Language)` `` |
| `sub` | subscript | `:sub[2]` | `` {sub}`2` `` |
| `sup` | superscript | `:sup[2]` | `` {sup}`2` `` |

The zen form is `:name[text]`. It is not recognised right after a letter or a
digit, so write a space before it (`E = mc :sup[2]`). The MyST role form works
in a folder that turns the `myst` dialect on, and has no such rule: `` mc{sup}`2` ``.

## kbd

Each key of a combination gets its own key cap, and the `+` or `-` between
keys is shown between the caps: `:kbd[⌘+K]` is a ⌘ cap, a plus and a K cap. A
`+` or `-` with no key before or after it is a key: `:kbd[Ctrl+-]` is two caps,
`Ctrl` and `-`.

## abbr

Write the abbreviation, then its explanation in brackets: `HTML (HyperText Markup
Language)`. The explanation shows as a tooltip and the abbreviation is
underlined with dots. The `title` option gives the explanation instead:
`:abbr[CSS]{title="Cascading Style Sheets"}`. An abbreviation without an
explanation is shown as it is, and the page has a warning.

## Options

| Option | On | Does |
|---|---|---|
| `title` | `abbr` | The explanation, instead of the one in brackets. |

Any other option is ignored, and the page has a warning that names the option.
Text longer than 200 characters is shown as plain text.

## Example

```markdown
Press :kbd[Ctrl+C] to copy. Pages are written in :abbr[HTML (HyperText Markup Language)].
Water is H :sub[2]O and the area is 4 m :sup[2].
```

In a folder that uses MyST:

```markdown
Press {kbd}`Ctrl+C`. Water is H{sub}`2`O.
```
