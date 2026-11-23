---
layout: troubleshooting
error: "plugin \"sandbox\": option \"images\" can only be set in the global config; ignored"
versions: "0.5.0 to 0.5.1"
fixed-in: "0.7.0"
tags: [export, plugins]
---
# Export warns about a plugin option on every page

This happens when your global `config.yml` sets an option that only it may
set, such as the sandbox plugin's `images`, and you export a website.

## Symptoms

- `zenmd export site` prints the message above once for every page.
- The summary line counts one warning per page, plus one more.

## Cause

An export leaves out options that belong to your Mac only, so a website
never carries them. Each page still saw the value from your global config and
reported it as if the folder had set it.

## Solution

1. Update to 0.7.0 or later: the export notes the option once.
2. Nothing is wrong with your settings, and the website is complete. You can ignore the warnings.
3. To silence them in an older version, remove the option from `config.yml` while you export, and put it back afterwards.

## Check that it's fixed

The export prints a single note about the option, not one per page.

## Related

- [Export a website](../how-tos/export-a-website.md)
- [Configuration](../reference/configuration.md)

