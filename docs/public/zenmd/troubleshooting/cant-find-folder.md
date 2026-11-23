---
layout: troubleshooting
error: "zenmd: docs/public/zenmd is neither a directory nor a markdown file"
versions: "0.4.0 to 0.5.1"
tags: [cli]
---
# zenmd can't find my folder

This happens when you give `zenmd` a relative path from a terminal that is in
another folder.

## Symptoms

- `zenmd docs/site` prints the message above, then the whole help.
- The same path works from another terminal.

## Cause

A relative path is read from the folder your terminal is in, not from where
the folder lives. The message shows the path as you typed it, so it doesn't
say where zenmd looked.

## Solution

1. Give the full path: `zenmd ~/Docs/site`.
2. Or go to the folder's parent first: `cd ~/Docs`, then `zenmd site`.

## Check that it's fixed

Zenmark opens and the terminal logs `folder opened` with the page count.

## Related

- [The zenmd command](../reference/cli.md)

> **Missing:** the version that fixes the message. The next release names the
> full path and prints one hint line instead of the help (#29), so set
> `fixed-in:` when it is tagged.
