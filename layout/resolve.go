package layout

import (
	"slices"
	"sort"
)

// Routing decides where a card nobody has placed by hand belongs. This file is
// the reference implementation of that decision, per plan 12.10: the canvas
// and every `git ticket canvas` command answer from it, so there is one
// reading of a board rather than two that drift.
//
// The contract, in order, and the order is the whole of it:
//
//  1. A card with a saved coordinate is pinned. Explicit beats implicit, so no
//     rule places it.
//  2. Otherwise the first pen in RuleOrder whose rule the ticket satisfies.
//  3. Otherwise the Inbox.
//
// A rule is satisfied when the ticket satisfies every field the pen's Match
// names: it carries all of the labels, and its status, type and parent are
// each among the values that field lists. Nothing here computes a position;
// that is the canvas's, and the one thing this package must never grow is a
// second copy of it.
//
// An Explanation for a pinned card still carries the rules' answer, marked by
// Pinned, because the question asked of a pin is what would happen on
// releasing it, per plan 10.10. That answer is hypothetical: a consumer that
// places cards reads Pinned first and the rules' answer only when it is nil.

// RuleTicket is what routing needs to know about a ticket. Status, Type and
// Parent are the ticket's own values, and an empty one is a ticket that has
// none: it matches a rule that does not test that field and no rule that
// does.
type RuleTicket struct {
	ID     string
	Labels []string
	Status string
	Type   string
	Parent string
}

// Outcome is why one rule did or did not take a ticket.
type Outcome string

const (
	// Winner is the first rule in order that the ticket satisfied.
	Winner Outcome = "winner"
	// NoMatch is a rule the ticket did not satisfy; Candidate.Failed says
	// which of its fields the ticket failed, and MissingLabels which labels
	// it lacked.
	NoMatch Outcome = "no-match"
	// LaterRule is a rule the ticket satisfied after an earlier one already
	// had. It would take the ticket if the rules above it were removed.
	LaterRule Outcome = "later-rule"
)

// Candidate is one rule as it was considered for one ticket, in resolution
// order. Every pen appears once, so a reader can see what each rule wanted
// rather than only which one won.
type Candidate struct {
	Pen   string `json:"pen"`
	Order int    `json:"order"`
	// Match is the pen's whole rule, so a reader comparing it to the ticket
	// needs nothing else from the board.
	Match Match `json:"match"`
	// MissingLabels is the labels of Match.Labels the ticket lacks, in the
	// order the rule wrote them. Labels is the one field worth reporting per
	// value: the other three hold one value on a ticket, and Match already
	// says what each wanted.
	MissingLabels []string `json:"missingLabels"`
	// Failed names the fields the ticket did not satisfy, among labels,
	// status, type and parent, in that order. It is empty on a rule the
	// ticket matched.
	Failed  []string `json:"failed"`
	Outcome Outcome  `json:"outcome"`
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
	// Destination is the winning pen's id, or empty for the Inbox. When Pinned
	// is set it is where the card would go if released, and nothing places by
	// it.
	Destination string      `json:"destination"`
	Candidates  []Candidate `json:"candidates"`
}

// Inbox reports whether the explanation routes to the Inbox.
func (e Explanation) Inbox() bool { return e.Destination == "" }

// Failures reports how one rule met one ticket: the labels the ticket lacks,
// and the names of the fields it failed. Two empty results are a match.
//
// Neither result is ever nil, because both are serialized in the
// canvas-explain envelope, per plan 10.10, and a consumer reading a match
// there is promised an empty array rather than null.
func (m Match) Failures(t RuleTicket) (missingLabels, failed []string) {
	missingLabels, failed = []string{}, []string{}
	have := make(map[string]bool, len(t.Labels))
	for _, l := range t.Labels {
		have[l] = true
	}
	// Labels conjoin: the ticket carries all of them or the field fails, and
	// which ones it lacks is the part worth naming.
	for _, want := range m.Labels {
		if !have[want] {
			missingLabels = append(missingLabels, want)
		}
	}
	if len(missingLabels) > 0 {
		failed = append(failed, "labels")
	}
	// The other three disjoin, because a ticket holds one of each: a rule
	// listing none of them tests nothing and matches every ticket.
	for _, f := range []struct {
		name  string
		has   string
		wants []string
	}{{"status", t.Status, m.Status}, {"type", t.Type, m.Type}, {"parent", t.Parent, m.Parent}} {
		if len(f.wants) > 0 && !slices.Contains(f.wants, f.has) {
			failed = append(failed, f.name)
		}
	}
	return missingLabels, failed
}

// Explain routes one ticket against a board. A pinned card is reported with
// its pin and with the rules' hypothetical answer; see the file comment.
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
		missing, failed := pen.Match.Failures(t)
		c := Candidate{Pen: id, Order: i, Match: pen.Match, MissingLabels: missing, Failed: failed}
		switch {
		case len(c.Failed) > 0:
			c.Outcome = NoMatch
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
