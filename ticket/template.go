package ticket

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// templatesDir is where a store keeps its templates, per plan 4.2:
// ticket-shaped Markdown files a person reviewed, versioned and diffed
// like everything else in the store.
const templatesDir = "templates"

// Template is the seedable subset of a ticket file, per plan 4.2. It
// deliberately carries nothing lifecycle-shaped: no status and no
// created instant, because a template that files things as done is a
// foot-gun, and 6.2.1 is where a backport says those things
// explicitly. The loader ignores every other field in the file, which
// is what lets a template be made by copying a real ticket.
type Template struct {
	Type               string
	Priority           string
	Labels             []string
	Assignees          []string
	Milestone          *string
	Description        string
	AcceptanceCriteria string
	DefinitionOfDone   string
	ImplementationPlan string
}

// Templates lists the names in the store's templates directory,
// sorted. A store with no directory has no templates, which is not an
// error: most stores never define one.
func (s *Store) Templates() ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(s.path, templatesDir))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, &Error{Code: CodeValidationFailed, Message: err.Error(), Err: err}
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".md"))
	}
	sort.Strings(names)
	return names, nil
}

// templateFrontmatter is the frontmatter subset a template may seed.
// Unknown keys unmarshal nowhere and are thereby ignored, per 4.2.
type templateFrontmatter struct {
	Type      string   `yaml:"type"`
	Priority  string   `yaml:"priority"`
	Labels    []string `yaml:"labels"`
	Assignees []string `yaml:"assignees"`
	Milestone *string  `yaml:"milestone"`
}

// Template reads one template by name, for a consumer building a
// create form: the TUI prefills its description editor from this, so
// the person edits the skeleton instead of typing over an invisible
// one. Create resolves its own template internally through the same
// loader.
func (s *Store) Template(name string) (*Template, error) {
	return s.loadTemplate(name)
}

// loadTemplate reads one template by name. A name that resolves to no
// file refuses with invalid_field naming the directory, per 4.2: a
// ticket that silently lacks the definition of done its author
// expected is worse than a stopped command.
func (s *Store) loadTemplate(name string) (*Template, error) {
	path := filepath.Join(s.path, templatesDir, name+".md")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, &Error{
			Code:    CodeInvalidField,
			Message: fmt.Sprintf("no template %q in %s", name, filepath.Join(s.path, templatesDir)),
			Field:   "template",
		}
	}
	if err != nil {
		return nil, &Error{Code: CodeValidationFailed, Message: err.Error(), Err: err}
	}

	front, bodyText, err := splitFrontmatter(string(data))
	if err != nil {
		return nil, &Error{
			Code:    CodeInvalidField,
			Message: fmt.Sprintf("template %q: %v", name, err),
			Field:   "template",
		}
	}
	var fm templateFrontmatter
	if front != "" {
		if err := yaml.Unmarshal([]byte(front), &fm); err != nil {
			return nil, &Error{
				Code:    CodeInvalidField,
				Message: fmt.Sprintf("template %q frontmatter: %s", name, yamlMessage(err)),
				Field:   "template",
			}
		}
	}
	body := parseBody(bodyText)

	tpl := &Template{
		Type:               fm.Type,
		Priority:           fm.Priority,
		Labels:             fm.Labels,
		Assignees:          fm.Assignees,
		Milestone:          fm.Milestone,
		Description:        strings.TrimSpace(body.Description),
		AcceptanceCriteria: strings.TrimSpace(body.AcceptanceCriteria),
		DefinitionOfDone:   strings.TrimSpace(body.DefinitionOfDone),
		ImplementationPlan: strings.TrimSpace(body.ImplementationPlan),
	}
	if tpl.Milestone != nil && *tpl.Milestone == "" {
		tpl.Milestone = nil
	}
	return tpl, nil
}
