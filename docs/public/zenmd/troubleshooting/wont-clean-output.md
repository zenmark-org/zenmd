---
layout: troubleshooting
error: "will not clean <folder>: it is not empty and has no .zen-build file from an earlier build"
versions: ">=0.4.0"
tags: [export]
---
# Export won't clean the output folder

This happens when you pass `--clean` and the output folder holds files that
zenmd didn't write.

## Symptoms

- `zenmd export site --clean` stops with the message above.
- Nothing is exported.

## Cause

`--clean` empties the output folder first. To be sure it never deletes your
own files, zenmd only empties a folder that holds the `.zen-build` file an
earlier export left there.

## Solution

1. Check that the output folder is the one you meant. `-o` and `export.out` in `.zenmd.yml` set it.
2. If it is, and nothing in it is yours, empty it yourself once.
3. Export again with `--clean`. From now on the folder holds a `.zen-build` file and is cleaned for you.

## Related

- [Export options](../reference/export-options.md)
