# Your first steps

In this tutorial you will turn a folder of three markdown files into a small
site, read it in Zenmark and export it as a website. It takes about ten
minutes and needs only a text editor.

## Before you start

- Zenmark and `zenmd`, from [Installation](installation.md).
- An empty folder, such as `~/Docs/hello`.

## Step 1: Write a home page

Make `~/Docs/hello/index.md`:

```markdown
# Hello

My first zenmd site.

- [Install](install.md)
- [Use](use.md)
```

`index.md` is the folder's home. Its H1 is the page's title.

## Step 2: Add the two pages

Make `install.md` with `# Install` and a line of text, and `use.md` with
`# Use`. Nothing else is needed: no config file, no list of pages.

## Step 3: Open the folder

```sh
zenmd ~/Docs/hello
```

Zenmark opens the folder. The sidebar shows Hello with Install and Use under
it, in the order the home page links them. Change a page in any editor and
save: the page updates on screen.

## Step 4: Export it

```sh
zenmd export site ~/Docs/hello
```

The website is in `~/Docs/hello/exports/html`. Open its `index.html` in a
browser: it works from disk, with no server.

## What you built

A folder that is a site: three pages, a sidebar made from your links, and a
website you can put anywhere. Next, start a real project from a blueprint:
[Start from a blueprint](../how-tos/start-from-a-blueprint.md).
