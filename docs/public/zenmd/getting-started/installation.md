# Installation

In this tutorial you will install Zenmark and the `zenmd` command, and check
that both work. It takes about ten minutes. You need a Mac with macOS 13 or
later.

## Before you start

- A Mac with macOS 13 (Ventura) or later.
- To build from source instead: Go 1.26 or later, Node 22 or later, and a terminal.

## Step 1: Install the app

Download `Zenmark-<version>.dmg` from the
[releases page](https://github.com/zenmark-org/zenmd/releases), open it and
drag Zenmark to Applications. Releases are signed and notarised, so the app
opens without a warning.

> **Missing:** the public download page on zenmark.org, and whether releases
> stay on GitHub once the repository is private.

## Step 2: Put `zenmd` on your path

The `zenmd` command lives inside the app. Link it into a folder on your path:

```sh
mkdir -p ~/.local/bin
ln -sf /Applications/Zenmark.app/Contents/MacOS/zenmd ~/.local/bin/zenmd
```

> **Missing:** an "Install command line tool" item in the app, so nobody has
> to type this. The plan lists it for 0.1.0-beta.1 (docs/wiki/VERSIONING.md) but it
> isn't built.

## Step 3: Check that it works

```sh
zenmd version
```

You should see `zenmd` and a version number, such as `zenmd 0.5.1`.

## Build from source

If you work on zenmd itself:

```sh
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
go install github.com/go-task/task/v3/cmd/task@latest
git clone https://github.com/zenmark-org/zenmd && cd zenmd
task build        # the frontend, the runtime and bin/zenmd
task install      # links zenmd into ~/.local/bin
task package      # bin/Zenmark.app, for this Mac
```

## What you installed

You have Zenmark in Applications and `zenmd` on your path. Next,
[open your first folder](first-steps.md).
