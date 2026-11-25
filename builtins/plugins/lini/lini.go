// Package lini renders ```lini and :::lini blocks as Lini figures. Go emits an
// escaped placeholder; assets/lini.js compiles it to an SVG in the browser with
// the vendored Lini compiler (see VENDOR.md).
package lini

import (
	"embed"
	"errors"
	"html"
	"io/fs"
	"sort"
	"strings"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

//go:embed assets/lini.js assets/lini.css assets/lini_wasm.js assets/lini_wasm.d.ts assets/lini_wasm_bg.wasm assets/lini_wasm_bg.wasm.d.ts assets/LICENSE-lini-wasm
var assets embed.FS

// The three figures of the help (and of the demo page). They are plain
// strings so that tests can compile them with the vendored compiler.
const (
	flowSnippet = `~~~lini
{ direction: row; gap: 40; }
|box#open| "Open folder"
|box#read| "Read a page"
|box#share| "Share it"
open -> read "pick"
read -> share "export"
~~~`
	mindmapSnippet = `~~~lini
|mindmap| "Zenmark" [
  |topic| "Read"
  |topic| "Write" [
    |topic| "Editor"
    |topic| "Preview"
  ]
  |topic| "Share"
]
~~~`
	sequenceSnippet = `~~~lini
{ layout: sequence; }
|box#you| "You"
|box#editor| "Editor"
you -> editor "type"
editor -> you "page reloads"
~~~`
)

type plugin struct{}

// New returns the lini plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{
		ID:          "lini",
		Version:     "0.1.0",
		APIVersion:  plugins.APIVersion,
		Description: "Lini figures: flows, mindmaps, sequences, charts",
		Default:     false,
		Help: plugins.Help{
			Summary: "Draws a Lini figure from a lini block: a flow, a mindmap, a sequence, a chart, a floor plan or a circuit, written as text. The figure follows the light or dark theme. A wrong figure keeps its source visible, with the compiler's message under it. Turn it on with plugins: { lini: {} }.",
			Snippets: []plugins.Snippet{
				{Title: "A flow", Markdown: plugins.Fences(flowSnippet)},
				{Title: "A mindmap", Markdown: plugins.Fences(mindmapSnippet)},
				{Title: "A sequence", Markdown: plugins.Fences(sequenceSnippet)},
			},
		},
		Blocks: []string{"lini"},
	}
}

// knownAttrs are the block attributes lini takes.
// (arg and title are set by the page, as for every fence).
var knownAttrs = map[string]bool{"file": true, "arg": true, "title": true}

func (plugin) RenderBlock(ctx plugins.Context, b plugins.Block) (string, error) {
	if ctx.Warn != nil {
		names := make([]string, 0, len(b.Attrs))
		for k := range b.Attrs {
			if !knownAttrs[k] {
				names = append(names, k)
			}
		}
		sort.Strings(names)
		for _, k := range names {
			ctx.Warn("unknown option :" + k + ":")
		}
	}
	body := b.Body
	if p, own, ok := plugins.FileRef(b); ok {
		if ctx.ReadFile == nil {
			return "", errors.New("needs an open folder")
		}
		data, err := ctx.ReadFile(p)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(own) != "" && ctx.Warn != nil {
			ctx.Warn("the block's own text is ignored because it names a file")
		}
		body = strings.ReplaceAll(string(data), "\r\n", "\n")
		if strings.TrimSpace(body) == "" {
			if ctx.Warn != nil {
				ctx.Warn(p + " is empty")
			}
			return "", nil
		}
	}
	return `<div class="zen-lini"><pre>` + html.EscapeString(body) + `</pre></div>`, nil
}

func (plugin) Assets() fs.FS {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}
	return sub
}

func (plugin) Entry() plugins.Assets {
	return plugins.Assets{CSS: []string{"lini.css"}, JS: []string{"lini.js"}}
}
