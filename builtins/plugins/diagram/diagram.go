// Package diagram renders ```mermaid and :::diagram blocks as diagrams. Go
// emits an escaped placeholder; assets/diagram.js turns it into an SVG with
// the vendored mermaid bundle (see VENDOR.md).
package diagram

import (
	"embed"
	"errors"
	"html"
	"io/fs"
	"strings"

	"github.com/zenmark-org/zenmd/engine/plugins"
)

//go:embed assets/diagram.js assets/diagram.css assets/mermaid.min.js
var assets embed.FS

var themes = map[string]bool{"neutral": true, "dark": true, "default": true}

type plugin struct{}

// New returns the diagram plugin.
func New() plugins.Plugin { return plugin{} }

func (plugin) Manifest() plugins.Manifest {
	return plugins.Manifest{
		ID:          "diagram",
		Version:     "0.1.0",
		APIVersion:  plugins.APIVersion,
		Description: "Mermaid diagrams",
		Default:     true,
		Help: plugins.Help{
			Summary: "Draws mermaid diagrams from a mermaid (or diagram) block, written in the page or read from a file of the folder (a :file: line, {file=...} or a path after mermaid; / starts at the folder's root). Diagrams follow the light or dark theme. A wrong diagram keeps its source visible, with mermaid's error under it. A drawn diagram can be zoomed and panned where it sits (corner controls, pinch, drag, keys) and opened over the whole window.",
			Snippets: []plugins.Snippet{
				{Title: "A flowchart", Markdown: plugins.Fences("~~~mermaid\ngraph LR\n  A[Open folder] --> B{README.md?}\n  B -- yes --> C[Show README]\n  B -- no --> D[Pick a page]\n~~~")},
				{Title: "A sequence diagram", Markdown: plugins.Fences("~~~mermaid\nsequenceDiagram\n  You->>Editor: type\n  You->>Editor: save\n  Editor-->>You: page reloads\n~~~")},
				{Title: "A diagram from a file", Markdown: plugins.Fences("~~~mermaid\n:file: /diagrams/flow.mmd\n~~~\n\nA path that starts with / is from the folder's root; any other is from the page's folder.")},
				{Title: "Another theme for one page", Markdown: plugins.Fences("---\nplugins:\n  diagram: { theme: dark }\n---\n# A page\n\n~~~mermaid\ngraph TD\n  A --> B\n~~~")},
			},
		},
		Blocks: []string{"mermaid", "diagram"},
		Allow:  []plugins.AllowRule{{Element: "div", Attrs: []string{"data-theme"}}},
		Config: map[string]plugins.Option{
			"theme": {Type: "string", Default: "neutral", Description: "mermaid theme: neutral, dark or default"},
		},
	}
}

func (plugin) RenderBlock(ctx plugins.Context, b plugins.Block) (string, error) {
	theme, _ := ctx.Config["theme"].(string)
	if !themes[theme] {
		theme = "neutral"
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
	return `<div class="zen-diagram" data-theme="` + theme + `"><pre>` +
		html.EscapeString(body) + `</pre></div>`, nil
}

func (plugin) Assets() fs.FS {
	sub, err := fs.Sub(assets, "assets")
	if err != nil {
		panic(err)
	}
	return sub
}

func (plugin) Entry() plugins.Assets {
	return plugins.Assets{CSS: []string{"diagram.css"}, JS: []string{"diagram.js"}}
}
