package layout

import "sort"

// Routing decides where a card nobody has placed by hand belongs. This file is
// the reference implementation of that decision, per plan 12.10: the canvas
// and every `git ticket canvas` command answer from it, so there is one
// reading of a board rather than two that drift.
//
// The contract, in order, and the order is the whole of it:
//
//  1. A card with a saved coordinate is pinned. Explicit beats implicit, so no
//     rule is consulted.
//  2. Otherwise the first pen in RuleOrder whose rule the ticket satisfies.
//  3. Otherwise the Inbox.
//
// A rule is satisfied when the ticket carries every one of the pen's
// RequiredLabels. Nothing here computes a position; that is the canvas's, and
// the one thing this package must never grow is a second copy of it.

// RuleTicket is what routing needs to know about a ticket.
type RuleTicket struct {
	ID     string
	Labels []string
}

// Outcome is why one rule did or did not take a ticket.
type Outcome string

const (
	// Winner is the first rule in order that the ticket satisfied.
	Winner Outcome = "winner"
	// MissingLabels is a rule the ticket did not satisfy; Candidate.Missing
	// says which labels it lacked.
	MissingLabels Outcome = "missing-labels"
	// LaterRule is a rule the ticket satisfied after an earlier one already
	// had. It would take the ticket if the rules above it were removed.
	LaterRule Outcome = "later-rule"
)

// Candidate is one rule as it was considered for one ticket, in resolution
// order. Every pen appears once, so a reader can see what each rule was
// missing rather than only which one won.
type Candidate struct {
	Pen      string   `json:"pen"`
	Order    int      `json:"order"`
	Required []string `json:"requiredLabels"`
	Missing  []string `json:"missingLabels"`
	Outcome  Outcome  `json:"outcome"`
}

// Explanation is where one ticket's card belongs and why.
type Explanation struct {
	ID string `json:"id"`
	// Pinned is the saved coordinate, and nil for an automatic card. When it
	// is set, Destination and Candidates describe where the card would go if
	// somebody handed it back to automatic placement; they are reported
	// rather than hidden because that is the question a person asks before
	// releasing a pin.
	Pinned *Card `json:"pinned"`
	// Destination is the winning pen's id, or empty for the Inbox.
	Destination string      `json:"destination"`
	Candidates  []Candidate `json:"candidates"`
}

// Inbox reports whether the explanation routes to the Inbox.
func (e Explanation) Inbox() bool { return e.Destination == "" }

// Match reports which of a pen's required labels a ticket lacks. An empty
// result is a match. The order follows the pen's rule, so the answer reads the
// way the rule was written.
func Match(pen Pen, labels []string) []string {
	have := make(map[string]bool, len(labels))
	for _, l := range labels {
		have[l] = true
	}
	var missing []string
	for _, want := range pen.RequiredLabels {
		if !have[want] {
			missing = append(missing, want)
		}
	}
	return missing
}

// Explain routes one ticket against a board.
func Explain(b *Board, t RuleTicket) Explanation {
	e := Explanation{ID: t.ID, Candidates: make([]Candidate, 0, len(b.RuleOrder))}
	if c, ok := b.Cards[t.ID]; ok {
		pinned := c
		e.Pinned = &pinned
	}
	won := false
	for i, id := range b.RuleOrder {
		pen, ok := b.Pens[id]
		if !ok {
			// A validated board cannot reach here; an unvalidated one is
			// reported rather than crashed on, and the rule is skipped.
			continue
		}
		c := Candidate{Pen: id, Order: i, Required: append([]string(nil), pen.RequiredLabels...), Missing: Match(pen, t.Labels)}
		switch {
		case len(c.Missing) > 0:
			c.Outcome = MissingLabels
		case won:
			c.Outcome = LaterRule
		default:
			c.Outcome = Winner
			e.Destination = id
			won = true
		}
		e.Candidates = append(e.Candidates, c)
	}
	return e
}

// Route explains every ticket, sorted by ID so that output is stable and a
// diff between two runs is a diff between two boards.
func Route(b *Board, tickets []RuleTicket) []Explanation {
	out := make([]Explanation, 0, len(tickets))
	for _, t := range tickets {
		out = append(out, Explain(b, t))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
