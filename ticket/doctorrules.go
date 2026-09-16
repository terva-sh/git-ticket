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
	//
	// It asks wherever a choice exists, which is any ticket carrying two or
	// more labels, rather than only where a card hides one. The first label is
	// the primary one: it leads every rendering of the list, and
	// git-ticket-canvas states the same convention independently in
	// TKT-01M27EPDKKW6HGKNKS7A98EQER, "the first label is the ticket's primary
	// label". A ticket with two labels has a real choice about which leads even
	// though both are on screen, so a threshold tied to hiding was asking about
	// visibility when the question is about primacy.
	RuleLabelOrder = "label_order"
)

// labelsShownDefault is how many labels a card shows before the rest collapse
// behind a +N disclosure.
//
// Two, because that is what git-ticket-canvas renders: CardView shows
// labels.slice(0, 2), and three when the cards are compact. A store whose board
// is compact, or which reads its tickets somewhere else entirely, sets `visible`
// in params rather than living with this number.
//
// It no longer decides whether the rule fires, only what the finding can say.
// A ticket with more labels than this has some out of sight, which is worth
// naming; one with fewer still has a first label, which is the part the rule is
// actually about.
const labelsShownDefault = 2

// labelOrderMinimum is how many labels it takes for the order to be a choice at
// all. One label is not an ordering.
//
// It is a parameter, `min`, and this is the default rather than the rule. A
// store whose convention already settles which dimension leads has answered this
// question once for every ticket, and asking it again per ticket is the noise
// the framework's configurability exists to let a store switch off. Measured
// when this default moved from 3 to 2: terva, which documents `area/` before
// `scope/` and carries two labels on nearly every ticket, went from 0 soft
// findings to 48. Raising `min` is the narrow answer there; disabling the rule
// is the blunt one, and a store should not have to reach for the blunt one.
const labelOrderMinimum = 2

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
		Summary: "a ticket carries more than one label, so which one leads is a choice",
		Check: func(rc RuleContext) []DoctorFinding {
			minLabels := paramInt(rc.Params, "min", labelOrderMinimum)
			if minLabels < labelOrderMinimum {
				// Below two there is no ordering to ask about, so a store
				// setting that is asking for a finding on every labelled
				// ticket and would get no information from any of them.
				minLabels = labelOrderMinimum
			}
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
				// Wherever the order is a choice. One label is not an ordering,
				// and nothing below asks about a ticket that has no decision to
				// make.
				if len(t.Labels) < minLabels {
					continue
				}
				// A question, not a verdict. The tool can see that a choice was
				// available and cannot see whether it was made well, so it names
				// the label that leads and asks about that one thing.
				msg := fmt.Sprintf("%q leads its %d labels: is that the one that describes it best?",
					t.Labels[0], len(t.Labels))
				if len(t.Labels) > visible {
					// Some are out of sight, which is worth naming because it is
					// a consequence of the order rather than a second question.
					msg = fmt.Sprintf("%q leads its %d labels and a card shows only %d, hiding %s: is that the one that describes it best?",
						t.Labels[0], len(t.Labels), visible, quoteList(t.Labels[visible:]))
				}
				out = append(out, DoctorFinding{
					Rule:    RuleLabelOrder,
					Level:   rc.Level,
					Ticket:  t.ID,
					File:    ticketPath(t),
					Message: msg,
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
