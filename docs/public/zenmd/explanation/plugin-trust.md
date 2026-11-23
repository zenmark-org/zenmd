# Why plugins ask first

This page explains why Zenmark asks before running a plugin that comes with a
folder.

## The idea

Built-in plugins are part of zenmd and always run. A folder's own plugins
(`.zen/plugins/`) can run programs and scripts as you, with your files and
network, so zenmd asks before the first run, naming what will run. The answer
is kept for that exact set of files: if any file changes, the plugin stops
and asks again.

## Why it is this way

- **Opening a folder must be safe.** You might open a folder someone sent you. Reading it should never run their code without asking.
- **A sandbox wasn't the answer.** Plugins are useful because they can run real tools. Asking once, for an exact version, keeps that and keeps you in charge.
- **Pages can't approve anything.** Only zenmd's own dialog can allow a plugin; a script on a page can't.

> **Missing:** the plugin guide for authors (writing a plugin, its manifest,
> its backend). It lives in the repository's docs/wiki/PLUGINS.md and isn't
> published as user docs yet.
