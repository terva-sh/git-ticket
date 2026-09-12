package ticket

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// lifecycleKeys are the frontmatter keys a document may carry and that 4.3
// does not honour. They are listed in the order 5.1 renders them, so a warning
// reads in the same order as the file the reader is looking at.
var lifecycleKeys = []string{"id", "status", "status_reason", "created_at", "updated_at", "created_by", "updated_by", "claim", "archive"}

// Document is a ticket somebody wrote out in full, read from a path per plan
// 4.3. It is the same shape as a template of 4.2 and differs in two ways: it
// carries the title, and it reports the lifecycle keys it is refusing to honour
// instead of dropping them in silence.
type Document struct {
	// Seed is everything a template also gives, per 4.2. Create applies it
	// with the same precedence, so an explicit flag still wins.
	Seed Template
	// Title comes from the document's frontmatter. It is the one field a
	// document gives and a form cannot, which is what makes --title optional
	// with --file and required everywhere else.
	Title string
	// Ignored names the lifecycle keys the document carried, in file order.
	// None of them is honoured. The caller reports them, because the library
	// decides what travels and the wording belongs to whoever is talking to a
	// person.
	Ignored []string
}

// documentFrontmatter is the frontmatter subset a document may seed: the
// template subset of 4.2, plus the title of 4.3. Unknown keys unmarshal
// nowhere and are thereby ignored, which is 4.2's leniency and the reason a
// document can be made by copying a real ticket.
type documentFrontmatter struct {
	Title     string   `yaml:"title"`
	Type      string   `yaml:"type"`
	Priority  string   `yaml:"priority"`
	Labels    []string `yaml:"labels"`
	Assignees []string `yaml:"assignees"`
	Milestone *string  `yaml:"milestone"`
}

// ReadDocument reads an authored ticket from a path, per plan 4.3.
//
// A path that names no readable file refuses, as an unknown template does and
// for the same reason: a ticket that silently lacks the definition of done its
// author expected is worse than a stopped command.
//
// It reads a file and no store, so it is a package function rather than a
// method. A template resolves inside the store and a document does not.
func ReadDocument(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &Error{
				Code:    CodeInvalidField,
				Message: fmt.Sprintf("no file %q to read a ticket from", path),
				Field:   "file",
				Err:     err,
			}
		}
		return nil, &Error{
			Code:    CodeInvalidField,
			Message: fmt.Sprintf("cannot read %q: %v", path, err),
			Field:   "file",
			Err:     err,
		}
	}

	front, bodyText, err := splitFrontmatter(string(data))
	if err != nil {
		return nil, &Error{
			Code:    CodeInvalidField,
			Message: fmt.Sprintf("%s: %v", path, err),
			Field:   "file",
		}
	}

	var fm documentFrontmatter
	if front != "" {
		if err := yaml.Unmarshal([]byte(front), &fm); err != nil {
			return nil, &Error{
				Code:    CodeInvalidField,
				Message: fmt.Sprintf("%s frontmatter: %s", path, yamlMessage(err)),
				Field:   "file",
			}
		}
	}

	body := parseBody(bodyText)
	doc := &Document{
		Title: strings.TrimSpace(fm.Title),
		Seed: Template{
			Type:               fm.Type,
			Priority:           fm.Priority,
			Labels:             fm.Labels,
			Assignees:          fm.Assignees,
			Milestone:          fm.Milestone,
			Description:        strings.TrimSpace(body.Description),
			AcceptanceCriteria: strings.TrimSpace(body.AcceptanceCriteria),
			DefinitionOfDone:   strings.TrimSpace(body.DefinitionOfDone),
			ImplementationPlan: strings.TrimSpace(body.ImplementationPlan),
		},
	}
	if doc.Seed.Milestone != nil && *doc.Seed.Milestone == "" {
		doc.Seed.Milestone = nil
	}
	doc.Ignored = lifecycleKeysIn(front)
	return doc, nil
}

// lifecycleKeysIn names the lifecycle keys the frontmatter states, in the order
// of lifecycleKeys.
//
// A key that is present but null states nothing, so it is not reported. A
// ticket copied out of the store carries `status_reason: null` and `claim:
// null` on the ordinary path, per 5.3, and warning about those would train a
// reader to ignore the warning that matters. What is reported is a key whose
// value somebody wrote.
func lifecycleKeysIn(front string) []string {
	if strings.TrimSpace(front) == "" {
		return nil
	}
	var raw map[string]yaml.Node
	if err := yaml.Unmarshal([]byte(front), &raw); err != nil {
		// The seeding unmarshal above already refused a frontmatter that does
		// not parse, so reaching here means the document parsed and this pass
		// disagreed. Report nothing rather than guess.
		return nil
	}
	var found []string
	for _, key := range lifecycleKeys {
		node, ok := raw[key]
		if !ok {
			continue
		}
		if node.Tag == "!!null" || strings.TrimSpace(node.Value) == "" && node.Kind == yaml.ScalarNode {
			continue
		}
		found = append(found, key)
	}
	return found
}
