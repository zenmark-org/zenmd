# Text marks

Small marks inside a line: a key on the keyboard, an abbreviation with its
explanation, a subscript and a superscript. They are inlines, so they sit in
the middle of a sentence.

## Syntax

| Mark | Write |
|---|---|
| a key or a key combination | `:kbd[Ctrl+C]` |
| an abbreviation | `:abbr[HTML (HyperText Markup Language)]` |
| subscript | `:sub[2]` |
| superscript | `:sup[2]` |

Put a space before the mark: it is not recognised right after a letter or a digit.

## Try it

Press :kbd[Ctrl+C] to copy, :kbd[⌘+K] on a Mac, or :kbd[Ctrl+Shift+P] for the
command list. Each key gets its own cap.

Pages are written in :abbr[HTML (HyperText Markup Language)]; hover the
abbreviation to read what it stands for. The explanation can also be an option:
:abbr[CSS]{title="Cascading Style Sheets"}.

Water is H :sub[2]O, and the area of a square is the side :sup[2].

## The MyST form

A folder that uses MyST writes the same marks as roles:

````markdown
Press {kbd}`Ctrl+C`. Written in {abbr}`HTML (HyperText Markup Language)`.

H{sub}`2`O and the side{sup}`2`.
````
