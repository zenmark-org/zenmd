// Package pagetemplates holds zen's built-in page templates (overview,
// tutorial, how-to, explanation, reference, troubleshooting, term and release-notes): one YAML
// file each, with the layout a new page gets, its frontmatter fields and the
// skeleton it starts with. A blueprint copies the ones it names into the new
// folder's .zen/templates.
package pagetemplates

import "embed"

// FS has one <id>.yml per page template.
//
//go:embed *.yml
var FS embed.FS

// IDs lists the built-in page templates.
var IDs = []string{"overview", "tutorial", "how-to", "explanation", "reference", "troubleshooting", "term", "release-notes"}
