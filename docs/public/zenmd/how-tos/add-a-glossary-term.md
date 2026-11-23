# Add a glossary term

Add one term to a glossary folder, where each term is its own file.

## Steps

1. Add it:
   ```sh
   zenmd glossary add "Page template" --definition "The structure of one kind of page."
   ```
   This writes `glossary/page-template.md` from the term template. With more than one glossary, add `--folder <folder>`.
2. Open the new file and write an example under **Examples**. Link related terms in the frontmatter: `see: [blueprint.md]`.
3. The glossary's index lists every term A to Z, rebuilt as soon as you save.

In Zenmark: **New Term…** beside the glossary's filter.

## Troubleshooting

- **The term isn't in the sidebar:** that's on purpose. Term pages are reached from the glossary index.
