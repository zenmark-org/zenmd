package session

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/zenmark-org/zenmd/engine/config"
	"github.com/zenmark-org/zenmd/engine/layouts"
)

// releaseNotes is the id of the page template whose folder suggests v<next version>.
const releaseNotes = "release-notes"

// ruleFor is the page template id the open folder's path rules give the page rel ("" for none).
func (s *Session) ruleFor(rel string) string {
	_, rules := s.templateContext()
	for _, r := range rules {
		if layouts.Match(r.Pattern, rel) {
			return r.Name
		}
	}
	return ""
}

// nameVersionRE is a page name that is v<semver> (the v is optional).
var nameVersionRE = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)$`)

type semver [3]int

func parseNameVersion(stem string) (semver, bool) {
	m := nameVersionRE.FindStringSubmatch(stem)
	if m == nil {
		return semver{}, false
	}
	var v semver
	for i := range v {
		n, err := strconv.Atoi(m[i+1])
		if err != nil {
			return semver{}, false
		}
		v[i] = n
	}
	return v, true
}

func (a semver) less(b semver) bool {
	for i := range a {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

// SuggestPageName is the name New Page offers in folder ("releases/", "" for the top):
// for a folder whose path rule is release-notes it is v plus the next minor version after the
// newest release there (v0.1.0 when there is none). It is "" for any other folder, which keeps
// the dialog's own suggestion.
func (s *Session) SuggestPageName(folder string) (r0 string) {
	folder = strings.Trim(strings.ReplaceAll(folder, `\`, "/"), "/")
	if strings.HasPrefix(folder, "..") || strings.ContainsRune(folder, 0) {
		return ""
	}
	if s.ruleFor(path.Join(folder, "x.md")) != releaseNotes {
		return ""
	}
	root, _ := s.templateContext()
	if root == "" {
		return ""
	}
	var newest semver
	found := false
	entries, _ := os.ReadDir(filepath.Join(root, filepath.FromSlash(folder)))
	for _, e := range entries {
		name := e.Name()
		if !e.Type().IsRegular() || !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		v, ok := parseNameVersion(strings.TrimSuffix(name, name[len(name)-3:]))
		if !ok { // a page with another name: its version: field counts
			b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(folder), name))
			if err != nil {
				continue
			}
			meta, _, _ := config.SplitFrontmatter(b)
			sv, _ := meta["version"].(string)
			if v, ok = parseNameVersion(strings.TrimSpace(sv)); !ok {
				continue
			}
		}
		if !found || newest.less(v) {
			newest, found = v, true
		}
	}
	next := fmt.Sprintf("v%d.%d.%d", newest[0], newest[1]+1, 0)
	if !found {
		next = "v0.1.0"
	}
	if folder == "" {
		return next
	}
	return folder + "/" + next
}

// pagePlaceholderRE is one {{name}} of a page template skeleton.
var pagePlaceholderRE = regexp.MustCompile(`\{\{([A-Za-z][A-Za-z0-9_-]*)\}\}`)

// ruledPageText is the starting text of the new page rel when a path rule of the open folder
// names a page template: its layout and fields as frontmatter, then its skeleton. The version
// comes from a v<semver> name and the date defaults to today. false when no rule or template applies.
func (s *Session) ruledPageText(rel string) (string, bool) {
	id := s.ruleFor(rel)
	if id == "" {
		return "", false
	}
	var tpl *PageTemplateInfo
	list := s.Templates().Pages().Templates
	for i := range list {
		if list[i].ID == id && list[i].Error == "" {
			tpl = &list[i]
			break
		}
	}
	if tpl == nil {
		return "", false
	}
	title := titleOf(rel)
	today := s.now().Format("2006-01-02")
	vals := map[string]string{"title": title, "today": today, "date": today, "folder": path.Dir(rel)}
	stem := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
	if v, ok := parseNameVersion(stem); ok {
		vals["version"] = fmt.Sprintf("%d.%d.%d", v[0], v[1], v[2])
	}
	fill := func(t string) string {
		return pagePlaceholderRE.ReplaceAllStringFunc(t, func(m string) string {
			if v, ok := vals[m[2:len(m)-2]]; ok {
				return v
			}
			return m
		})
	}
	var fm []string
	if tpl.Layout != "" && tpl.Layout != "article" {
		fm = append(fm, "layout: "+tpl.Layout)
	}
	for _, f := range tpl.Fields {
		v, ok := vals[f.Name]
		if !ok || f.Name != "version" && f.Name != "date" {
			v = fill(f.Default)
		}
		switch {
		case f.Name == "version" && (v != "" || f.Required): // always quoted: 1.10 stays 1.10
			fm = append(fm, "version: "+quoteYAML(v))
		case v != "":
			fm = append(fm, f.Name+": "+yamlString(v))
		}
	}
	text := fill(tpl.Skeleton)
	if len(fm) == 0 {
		return text, true
	}
	return "---\n" + strings.Join(fm, "\n") + "\n---\n" + text, true
}

func quoteYAML(s string) string {
	return "\"" + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + "\""
}
