package ticket

import "fmt"

// The rules that ship with the binary. They are the tool's opinion about what a
// well-kept store looks like, which is the only reason to have a hygiene
// command rather than a linter everybody writes themselves. Defaults that try
// to offend nobody say nothing, so these are meant to be worth overriding.

const (
	// RuleLabelMissing is the first hard rule: a ticket with no label at all.
	//
	// Presence only. A label outside the store's allowlist is already
	// label_unknown in check, and a rule that repeated it would be teaching a
	// reader that two commands disagree about whose job that is. A ticket with
	// one unknown label is labelled, and this rule says nothing about it.
	RuleLabelMissing = "label_missing"

	// RuleLabelOrder is the first soft rule: whether the label a reader sees
	// first is the one that describes the ticket best.
	//
	// It is the soft rule and not a hard one because nothing mechanical knows
	// which of `auth` and `ui` describes a ticket better. The tool can see that
	// a choice was available and cannot see whether it was made well, which is
	// exactly the boundary the levels exist to draw.
	RuleLabelOrder = "label_order"
)

// labelsShownDefault is how many labels a card shows before the rest collapse
// behind a +N disclosure.
//
// Two, because that is what git-ticket-canvas renders: CardView shows
// labels.slice(0, 2), and three when the cards are compact. A store whose board
// is compact, or which reads its tickets somewhere else entirely, sets `visible`
// in params rather than living with this number.
const labelsShownDefault = 2

// DefaultRules is the set that ships with the binary, on by default. A store
// that configures nothing gets exactly this.
func DefaultRules() []Rule {
	return []Rule{labelMissingRule(), labelOrderRule()}
}

func labelMissingRule() Rule {
	return Rule{
		ID:      RuleLabelMissing,
		Level:   LevelHard,
		Summary: "a ticket carries no label",
		Check: func(rc RuleContext) []DoctorFinding {
			var out []DoctorFinding
			for _, t := range rc.Tickets {
				if len(t.Labels) > 0 {
					continue
				}
				out = append(out, DoctorFinding{
					Rule:    RuleLabelMissing,
					Level:   rc.Level,
					Ticket:  t.ID,
					File:    ticketPath(t),
					Message: "carries no label, so nothing but its title says what it is about",
					// The allowlist is pointed at rather than listed. A store may
					// allow more labels than fit on a line, and `config` is the
					// command whose job is answering what this store allows.
					Remedy: fmt.Sprintf(
						"add one with `git ticket update %s --add-label NAME`; `git ticket config` lists what this store allows",
						t.ID),
				})
			}
			return out
		},
	}
}

func labelOrderRule() Rule {
	return Rule{
		ID:      RuleLabelOrder,
		Level:   LevelSoft,
		Summary: "a ticket has more labels than a card shows, so the order decides which are seen",
		Check: func(rc RuleContext) []DoctorFinding {
			visible := paramInt(rc.Params, "visible", labelsShownDefault)
			if visible < 1 {
				// A store that set this to zero or less is asking for a finding
				// on every labelled ticket, which is not a thing this rule can
				// say anything useful about. Treated as the default rather than
				// obeyed, because obeying it produces noise and no information.
				visible = labelsShownDefault
			}

			var out []DoctorFinding
			for _, t := range rc.Tickets {
				// Only when a label is actually hidden. With two labels and a
				// card that shows two, order changes nothing a reader sees, and
				// a rule that fired there would be asking about a choice that
				// does not exist.
				if len(t.Labels) <= visible {
					continue
				}
				hidden := t.Labels[visible:]
				out = append(out, DoctorFinding{
					Rule:   RuleLabelOrder,
					Level:  rc.Level,
					Ticket: t.ID,
					File:   ticketPath(t),
					// A question, not a verdict. The tool can say which labels
					// are out of sight and cannot say whether that is wrong, so
					// it reports the first and asks about the second.
					Message: fmt.Sprintf(
						"a card shows the first %d of its %d labels, hiding %s: is %q the one that describes it best?",
						visible, len(t.Labels), quoteList(hidden), t.Labels[0]),
					Remedy: fmt.Sprintf(
						"if not, move a label to the end with `git ticket update %s --remove-label NAME --add-label NAME`",
						t.ID),
				})
			}
			return out
		},
	}
}

// ticketPath is where a ticket's file sits, store-relative, the way a check
// finding names one.
func ticketPath(t *Ticket) string { return statusDir(t.Status) + "/" + t.ID + ".md" }

// paramInt reads an integer parameter, falling back when the store set nothing
// or set something this rule cannot read.
//
// A value of the wrong shape falls back rather than failing the run. A rule is
// advice, and refusing to give any because one parameter was misspelled would
// spend the whole report on a typo. YAML decodes a plain integer as int, and a
// quoted one as a string, so both are accepted: a person writing `visible: "3"`
// meant three.
func paramInt(params map[string]any, key string, fallback int) int {
	v, ok := params[key]
	if !ok {
		return fallback
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		var parsed int
		if _, err := fmt.Sscanf(n, "%d", &parsed); err == nil {
			return parsed
		}
	}
	return fallback
}

// quoteList renders labels for a sentence, quoted so a label containing a space
// reads as one thing.
func quoteList(labels []string) string {
	switch len(labels) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf("%q", labels[0])
	}
	out := ""
	for i, l := range labels {
		switch {
		case i == 0:
			out = fmt.Sprintf("%q", l)
		case i == len(labels)-1:
			out += fmt.Sprintf(" and %q", l)
		default:
			out += fmt.Sprintf(", %q", l)
		}
	}
	return out
}
