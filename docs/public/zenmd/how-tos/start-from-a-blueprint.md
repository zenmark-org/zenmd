# Start from a blueprint

Make a new docs folder with its sections, first pages and page templates
already in place.

## Steps

1. List the blueprints:
   ```sh
   zenmd blueprint list
   ```
2. Make the folder, answering the blueprint's questions with flags:
   ```sh
   zenmd init --blueprint techdocs ~/Docs/acme --title "Acme" --author "Docs team" --set product=Acme
   ```
3. Open it with `zenmd ~/Docs/acme`, or choose **File › New from Blueprint…** (⇧⌘N) in Zenmark to do steps 1 and 2 in a dialog.

Each section's new pages follow its page template: **New Page** in `how-tos/`
starts as a how-to. The rules are under `templates:` in the folder's
`.zenmd.yml`.

## Troubleshooting

- **"already exists":** the target folder must be empty or new. zenmd never overwrites.
- **"update available" in the list:** your copy of a built-in blueprint is older. `zenmd init --update techdocs` updates it, or writes `techdocs.new` beside it if you edited yours.
