---
layout: troubleshooting
error: "plugin changed; allow it again"
versions: ">=0.4.0"
tags: [plugins, trust]
---
# A plugin stopped working after a change

This happens to a folder's own plugin, in `.zen/plugins/`, after any of its
files changed.

## Symptoms

- The plugin's blocks show as plain markdown, with a note on the page.
- Settings › Plugins lists the plugin as undecided.

## Cause

You allowed the plugin for one exact set of files. When a file changes,
zenmd stops the plugin until you allow it again, so a changed plugin never
runs without you knowing. A plugin that writes into its own folder stops
itself this way; it should use its data folder.

## Solution

1. Open **Settings › Plugins** and select the plugin.
2. Read what it will run, then choose **Allow**.

## Check that it's fixed

The plugin's blocks render again.

## Related

- [Why plugins ask first](../explanation/plugin-trust.md)
