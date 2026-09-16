package ticket

import (
	"fmt"
	"sort"
)

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
	// It is soft and not hard because nothing mechanical knows which of `auth`
	// and `ui` describes a ticket better. What the tool can see is whether this
	// store has already answered that question everywhere else, which is the
	// boundary the levels exist to draw: the store's own practice is a fact
	// about the store, and whether one ticket should follow it is a judgement.
	//
	// So the rule reads the convention rather than asserting one. It takes the
	// dimension of each ticket's leading label, finds the dimension that leads
	// most often, and reports the tickets that disagree with it. The first
	// label is the primary one: it leads every rendering of the list, and
	// git-ticket-canvas states the same convention independently in
	// TKT-01M27EPDKKW6HGKNKS7A98EQER, "the first label is the ticket's primary
	// label".
	//
	// Until v0.19.1 it asked wherever two or more labels existed, on the
	// argument that a choice was available. That is true and it is not useful:
	// a store with a convention has made the same choice the same way every
	// time, so asking per ticket reports the convention working. Measured on
	// the stores that had adopted it, the rule fired on 120 of terva's 121 open
	// tickets and on all 131 of ketju's. A rule that fires on everything cannot
	// guide a change, which is the only thing doctor is for.
	//
	// The same measurement is what the rule now keys on. terva leads with
	// `area/` on 116 of those 120 and ketju with `area:` on 130 of 131, so the
	// four and the one that disagree are exactly the tickets worth a question.
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
// It does not decide whether the rule fires, only what the finding can say. A
// ticket with more labels than this has some out of sight, which is worth
// naming once there is something to name.
const labelsShownDefault = 2

// labelOrderMinimum is how many labels it takes for the order to be a choice at
// all. One label is not an ordering, so a single-label ticket is neither
// evidence of a convention nor capable of breaking one.
//
// It is a parameter, `min`, and this is the default rather than the rule.
const labelOrderMinimum = 2

// labelOrderConfidence is how dominant the leading dimension must be before
// this store is treated as having a convention at all.
//
// A store that has not settled which dimension leads is not making a mistake
// when its tickets disagree with each other, and reporting them would be the
// tool inventing a rule the store never adopted. Below this share the rule says
// nothing.
//
// Four fifths, and the measurement says the exact value hardly matters: the
// stores that have a convention have a very strong one. terva leads with
// `area/` on 97% of its multi-label tickets and ketju with `area:` on 99%, so
// anything from 0.70 to 0.95 gives both the same answer. Only git-ticket's own
// store, at 88% over eight tickets, sits near a boundary, and 0.8 is what lets
// it speak. It is the `confidence` parameter.
const labelOrderConfidence = 0.8

// labelOrderSample is how many multi-label tickets it takes before a leading
// dimension counts as a convention rather than a coincidence.
//
// Five. Three tickets agreeing is not a practice, and a floor of ten would
// silence git-ticket's own store, which has eight. A store below this gets
// nothing from the rule, which is the right answer for one that has not yet
// labelled enough work to have established anything. It is the `sample`
// parameter.
const labelOrderSample = 5

// labelNoDimension is the leading dimension of a label that carries none.
//
// It is a real value rather than a skip, because "this store leads with bare
// labels" is as much a convention as leading with `area/`, and a store that
// holds it consistently should hear about the one ticket that does not. The
// angle brackets cannot collide with a dimension read off a label, since a
// dimension is the text before the first separator and a label containing one
// would have to be written with it.
const labelNoDimension = "<none>"

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
		Summary: "a ticket leads with a label dimension its store rarely leads with",
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

			// Only the tickets where an ordering exists, which are both the
			// evidence for the convention and the only things that can break
			// it. A single-label ticket votes on nothing and is never reported.
			var ordered []*Ticket
			for _, t := range rc.Tickets {
				if len(t.Labels) >= minLabels {
					ordered = append(ordered, t)
				}
			}

			dominant, share, ok := leadingConvention(ordered)
			if !ok {
				return nil
			}
			if len(ordered) < paramInt(rc.Params, "sample", labelOrderSample) {
				// Too little labelled work to have established anything. Saying
				// nothing is the honest answer, not a missed finding.
				return nil
			}
			if share < paramFloat(rc.Params, "confidence", labelOrderConfidence) {
				// The store has not settled which dimension leads, so its
				// tickets disagreeing with each other is not a mistake and
				// reporting them would be this tool inventing a convention.
				return nil
			}

			var out []DoctorFinding
			for _, t := range ordered {
				lead := leadingDimension(t.Labels[0])
				if lead == dominant {
					continue
				}
				// The finding states the store's own practice and asks whether
				// this ticket meant to depart from it. That is the soft half:
				// the share is a fact, and whether this ticket is the exception
				// is a judgement only a person holds.
				msg := fmt.Sprintf("leads with %s where %.0f%% of this store's multi-label tickets lead with %s",
					describeDimension(t.Labels[0], lead), share*100, namedDimension(dominant))
				if len(t.Labels) > visible {
					// Some are out of sight, which is worth naming because it
					// is a consequence of the order rather than a second
					// question.
					msg += fmt.Sprintf(", and a card shows only %d of its %d, hiding %s",
						visible, len(t.Labels), quoteList(t.Labels[visible:]))
				}
				msg += ": did this one mean to differ?"
				out = append(out, DoctorFinding{
					Rule:    RuleLabelOrder,
					Level:   rc.Level,
					Ticket:  t.ID,
					File:    ticketPath(t),
					Message: msg,
					Remedy: fmt.Sprintf(
						"if not, lead with the usual one using `git ticket update %s --remove-label NAME --add-label NAME`",
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

// leadingDimension is the family a label belongs to: the text up to and
// including the first separator, or labelNoDimension when it carries none.
//
// Both separators are recognised because both are in use and neither is
// blessed. terva writes `area/cli` and ketju writes `area:cli`, and a store
// mixing them has a reconciliation problem this rule is not the place to raise;
// TKT-01M2NSGX6B1WJ56CSBMEG6SFVH carries that question. The separator is kept
// in the returned value so a finding can print `area/` and have it read as a
// prefix rather than as a label.
func leadingDimension(label string) string {
	for i, r := range label {
		if r == '/' || r == ':' {
			return label[:i+1]
		}
	}
	return labelNoDimension
}

// leadingConvention is the dimension this store leads with most often, and the
// share of its ordered tickets that do.
//
// ok is false when there is nothing to read a convention from at all. A share
// is returned even when it is weak, because deciding whether it is strong
// enough belongs to the caller, which is where the parameter lives.
func leadingConvention(ordered []*Ticket) (dominant string, share float64, ok bool) {
	if len(ordered) == 0 {
		return "", 0, false
	}
	counts := map[string]int{}
	for _, t := range ordered {
		counts[leadingDimension(t.Labels[0])]++
	}
	best, bestN := "", -1
	for _, d := range sortedDimensions(counts) {
		// Iterated in sorted order so a tie resolves the same way on every run.
		// A tie means there is no convention and the share will fail the
		// confidence test anyway, but a report that reorders itself between
		// runs is not something anyone should have to debug.
		if counts[d] > bestN {
			best, bestN = d, counts[d]
		}
	}
	return best, float64(bestN) / float64(len(ordered)), true
}

// sortedDimensions orders the tallied dimensions, so a tie between two of them
// resolves the same way on every run. It mirrors sortedRuleIDs, which exists
// for the same reason one directory up in the same feature.
func sortedDimensions(m map[string]int) []string {
	dims := make([]string, 0, len(m))
	for d := range m {
		dims = append(dims, d)
	}
	sort.Strings(dims)
	return dims
}

// describeDimension names what a label leads with, for a sentence. A label with
// no dimension is named by itself, since "<none>" tells a reader nothing about
// which label is sitting in front.
func describeDimension(label, dim string) string {
	if dim == labelNoDimension {
		return fmt.Sprintf("%q, which carries no dimension,", label)
	}
	return fmt.Sprintf("%q", dim)
}

// namedDimension is the same for the store's own convention, where there is no
// one label to fall back on.
func namedDimension(dim string) string {
	if dim == labelNoDimension {
		return "no dimension at all"
	}
	return fmt.Sprintf("%q", dim)
}

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

// paramFloat reads a fractional parameter, falling back on the same terms
// paramInt does and for the same reason.
//
// int is accepted alongside float64 because YAML decodes `confidence: 1` as an
// int, and a store writing that meant certainty rather than a type error.
func paramFloat(params map[string]any, key string, fallback float64) float64 {
	v, ok := params[key]
	if !ok {
		return fallback
	}
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case string:
		var parsed float64
		if _, err := fmt.Sscanf(n, "%g", &parsed); err == nil {
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
