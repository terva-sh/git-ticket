package view

import (
	"strings"

	"github.com/terva-sh/git-ticket/ticket"
)

// filter is the parsed form of the filter line. The semantics mirror
// ticket.Filter, because a person who learned `git ticket list` should
// not have to learn a second rule here: within one field the values
// are alternatives, and across fields they all have to hold. Bare
// words are the one addition, a case-insensitive title match, because
// the first thing anybody types into a filter is a word they remember
// from a title.
type filter struct {
	status   []string
	label    []string
	assignee []string
	words    []string
}

// parseFilter reads the filter line. Tokens are separated by spaces;
// `status:S`, `label:L`, and `assignee:A` go to their fields, and
// anything else is a title word. A token with an empty value, like a
// bare `status:`, is dropped rather than matching nothing.
func parseFilter(line string) filter {
	var f filter
	for _, tok := range strings.Fields(line) {
		key, val, found := strings.Cut(tok, ":")
		if !found {
			f.words = append(f.words, strings.ToLower(tok))
			continue
		}
		switch key {
		case "status":
			if val != "" {
				f.status = append(f.status, val)
			}
		case "label":
			if val != "" {
				f.label = append(f.label, val)
			}
		case "assignee":
			// An assignee ID carries its own colon, human:sothr, so
			// the value is everything after the first cut put back
			// together. strings.Cut split at the first colon only,
			// and val already holds the rest intact.
			if val != "" {
				f.assignee = append(f.assignee, val)
			}
		default:
			// Not a field this filter knows: the whole token is a
			// title word, colon included, so searching for "fix:" or
			// an ID fragment still works.
			f.words = append(f.words, strings.ToLower(tok))
		}
	}
	return f
}

func (f filter) empty() bool {
	return len(f.status) == 0 && len(f.label) == 0 && len(f.assignee) == 0 && len(f.words) == 0
}

// match reports whether t passes every clause.
func (f filter) match(t *ticket.Ticket) bool {
	if len(f.status) > 0 && !oneOf(t.Status, f.status) {
		return false
	}
	if len(f.label) > 0 && !anyIn(t.Labels, f.label) {
		return false
	}
	if len(f.assignee) > 0 && !anyIn(t.Assignees, f.assignee) {
		return false
	}
	title := strings.ToLower(t.Title)
	for _, w := range f.words {
		if !strings.Contains(title, w) {
			return false
		}
	}
	return true
}

// oneOf reports whether have is any of the wanted values.
func oneOf(have string, want []string) bool {
	for _, w := range want {
		if have == w {
			return true
		}
	}
	return false
}

// anyIn reports whether the two sets intersect: the ticket carries at
// least one of the wanted values.
func anyIn(have, want []string) bool {
	for _, h := range have {
		if oneOf(h, want) {
			return true
		}
	}
	return false
}
