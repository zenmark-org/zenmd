# sandbox

Runs a block's code in a container, with Apple's `container` tool (not
Docker). The block shows the code, a Run button and a console. On by default,
and it works after a one-time setup (start the service with
`container system start`, pull the image once).

## Names

| Name | Written as |
|---|---|
| `sandbox` | `:::sandbox`, a fence ```` ~~~sandbox {image=python …} ````, or in MyST ```` ```{sandbox} ```` |

## Options of a block

| Option | Default | What it does |
|---|---|---|
| `image` | none, required | A key of the `images` setting. A page cannot name an image itself. |
| `cmd` | none, required | The command to run, such as `python main.py`. |
| `file` | none | The file name the code is written to, a plain name. Its extension also picks the colouring. |
| `lang` | from `file` | The language used to colour the code. |
| `editable` | off | The code is a text box you can change before running. |

A block with no image, or an unknown one, shows what to add to the global
`config.yml` instead of a Run button. An option the plugin does not know is
ignored.

## Options of the plugin

These are read from your global `config.yml` only. A page or a folder's
`.zenmd.yml` cannot set them.

| Option | Default | What it does |
|---|---|---|
| `images` | none | Image keys a block may use, mapped to image references. |
| `timeout` | `30` | Seconds a run may take, 1 to 600. |
| `network` | `false` | Give the container network access. |
| `memory` | `512M` | The memory limit, such as `512M` or `1G`. |
| `cpus` | `1` | CPUs for the container, 1 to 8. |

## Example

````markdown
~~~sandbox {image=python cmd="python main.py" file=main.py editable}
print("hello from a container")
~~~
````
