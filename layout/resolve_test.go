package layout

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

const epicID = "TKT-01K3ZZ2JH000GHB4EE6SNRE6MD"

func routingBoard() *Board {
	b := Empty("default")
	b.Pens = map[string]Pen{
		"frontend": {Title: "Frontend", Match: Match{Labels: []string{"frontend"}}},
		"fe-bugs":  {Title: "Frontend bugs", Match: Match{Labels: []string{"frontend", "bug"}}},
		"backend":  {Title: "Backend", Match: Match{Labels: []string{"backend"}}},
	}
	b.RuleOrder = []string{"frontend", "fe-bugs", "backend"}
	b.Inbox = &Point{X: 10, Y: 20}
	return b
}

func outcomes(e Explanation) []Outcome {
	out := make([]Outcome, 0, len(e.Candidates))
	for _, c := range e.Candidates {
		out = append(out, c.Outcome)
	}
	return out
}

// TestRouteIsFirstMatchInRuleOrder holds the contract of plan 12.10: the first
// pen in RuleOrder that the ticket satisfies wins, and a more specific rule
// later in the order does not outrank it. That is the decision recorded on the
// canvas repository's TKT-01M2ND1RJ and it is the opposite of what an earlier
// TypeScript evaluator did, which is why the test says it in its name.
func TestRouteIsFirstMatchInRuleOrder(t *testing.T) {
	b := routingBoard()
	got := Explain(b, RuleTicket{ID: "TKT-A", Labels: []string{"bug", "frontend"}})
	if got.Destination != "frontend" {
		t.Fatalf("destination = %q, want frontend, the earlier rule", got.Destination)
	}
	if want := []Outcome{Winner, LaterRule, NoMatch}; !reflect.DeepEqual(outcomes(got), want) {
		t.Fatalf("outcomes = %v, want %v", outcomes(got), want)
	}
	if got.Candidates[2].MissingLabels[0] != "backend" {
		t.Fatalf("backend rule missingLabels = %v, want [backend]", got.Candidates[2].MissingLabels)
	}
	if !reflect.DeepEqual(got.Candidates[2].Failed, []string{"labels"}) {
		t.Fatalf("backend rule failed = %v, want [labels]", got.Candidates[2].Failed)
	}
}

func TestRouteNoMatchGoesToInbox(t *testing.T) {
	got := Explain(routingBoard(), RuleTicket{ID: "TKT-B", Labels: []string{"docs"}})
	if !got.Inbox() || got.Destination != "" {
		t.Fatalf("destination = %q, want the inbox", got.Destination)
	}
	for _, c := range got.Candidates {
		if c.Outcome != NoMatch {
			t.Fatalf("%s outcome = %s, want no-match", c.Pen, c.Outcome)
		}
	}
}

// A pinned card keeps its coordinate and routing is still reported, because
// where the card would go is the question somebody asks before releasing it.
func TestRoutePinnedCardKeepsItsCoordinate(t *testing.T) {
	b := routingBoard()
	b.Cards["TKT-C"] = Card{X: 100, Y: -40}
	got := Explain(b, RuleTicket{ID: "TKT-C", Labels: []string{"backend"}})
	if got.Pinned == nil || got.Pinned.X != 100 || got.Pinned.Y != -40 {
		t.Fatalf("pinned = %+v, want (100, -40)", got.Pinned)
	}
	if got.Destination != "backend" {
		t.Fatalf("destination = %q, want backend reported alongside the pin", got.Destination)
	}
}

func TestRouteLegacyBoardWithoutRoutingSendsEverythingToInbox(t *testing.T) {
	b, err := Parse("default", []byte("schema: 1\nboard: default\ncards:\n  TKT-D: {x: 1, y: 2}\n"))
	if err != nil {
		t.Fatal(err)
	}
	got := Route(b, []RuleTicket{{ID: "TKT-E"}, {ID: "TKT-D"}})
	if len(got) != 2 || got[0].ID != "TKT-D" || got[1].ID != "TKT-E" {
		t.Fatalf("Route returned %+v, want two explanations sorted by ID", got)
	}
	if got[0].Pinned == nil || got[1].Pinned != nil {
		t.Fatalf("pins = %v/%v, want TKT-D pinned and TKT-E automatic", got[0].Pinned, got[1].Pinned)
	}
	for _, e := range got {
		if !e.Inbox() || len(e.Candidates) != 0 {
			t.Fatalf("%s: destination %q with %d candidates, want the inbox and none", e.ID, e.Destination, len(e.Candidates))
		}
	}
}

// Every field of the rule, the two ways a field can be read, and the two ways
// a rule can be wider or narrower than one field. Labels conjoin because a
// ticket carries many at once; status, type and parent disjoin because a
// ticket holds one of each, per plan 12.10.
func TestMatchChecksEveryFieldOfTheRule(t *testing.T) {
	full := RuleTicket{ID: "T-1", Labels: []string{"frontend", "bug"}, Status: "ready", Type: "task", Parent: epicID}
	cases := []struct {
		name    string
		rule    Match
		ticket  RuleTicket
		failed  []string
		missing []string
	}{
		{"labels carried", Match{Labels: []string{"frontend", "bug"}}, full, nil, nil},
		{"one label short", Match{Labels: []string{"frontend", "docs"}}, full, []string{"labels"}, []string{"docs"}},
		{"status hit", Match{Status: []string{"ready"}}, full, nil, nil},
		{"status missed", Match{Status: []string{"draft"}}, full, []string{"status"}, nil},
		{"status is a disjunction", Match{Status: []string{"blocked", "ready"}}, full, nil, nil},
		{"type hit", Match{Type: []string{"task"}}, full, nil, nil},
		{"type missed", Match{Type: []string{"bug"}}, full, []string{"type"}, nil},
		{"type is a disjunction", Match{Type: []string{"bug", "task"}}, full, nil, nil},
		{"parent hit", Match{Parent: []string{epicID}}, full, nil, nil},
		{"parent missed", Match{Parent: []string{"TKT-01K3ZZ2JH000GHB4EE6SNRE6ME"}}, full, []string{"parent"}, nil},
		{"a ticket with no parent fails a parent rule", Match{Parent: []string{epicID}},
			RuleTicket{ID: "T-2", Status: "ready"}, []string{"parent"}, nil},
		{"fields conjoin and both hold", Match{Labels: []string{"bug"}, Status: []string{"ready"}, Type: []string{"task"}}, full, nil, nil},
		{"fields conjoin and one fails", Match{Labels: []string{"bug"}, Status: []string{"draft"}}, full, []string{"status"}, nil},
		{"every field fails, in one order", Match{Labels: []string{"docs"}, Status: []string{"draft"}, Type: []string{"epic"}, Parent: []string{epicID}},
			RuleTicket{ID: "T-3"}, []string{"labels", "status", "type", "parent"}, []string{"docs"}},
		{"a field the rule leaves out matches everything", Match{Labels: []string{"bug"}},
			RuleTicket{ID: "T-4", Labels: []string{"bug"}, Status: "whatever", Type: "whatever"}, nil, nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			missing, failed := c.rule.Failures(c.ticket)
			if !reflect.DeepEqual(failed, orEmpty(c.failed)) {
				t.Errorf("failed = %v, want %v", failed, orEmpty(c.failed))
			}
			if !reflect.DeepEqual(missing, orEmpty(c.missing)) {
				t.Errorf("missingLabels = %v, want %v", missing, orEmpty(c.missing))
			}
		})
	}
}

func TestMatchReportsMissingLabelsInRuleOrder(t *testing.T) {
	rule := Match{Labels: []string{"a", "b", "c"}}
	if got, _ := rule.Failures(RuleTicket{Labels: []string{"b"}}); !reflect.DeepEqual(got, []string{"a", "c"}) {
		t.Fatalf("missing = %v, want [a c]", got)
	}
	if got, failed := rule.Failures(RuleTicket{Labels: []string{"c", "a", "b", "extra"}}); len(got) != 0 || len(failed) != 0 {
		t.Fatalf("missing = %v, failed = %v; extra labels are allowed", got, failed)
	}
}

// A board written before schema 4 routes exactly as it did: requiredLabels is
// read as match.labels and nothing else about the rule changes.
func TestRouteReadsASchema3BoardByItsLabels(t *testing.T) {
	b, err := Parse("default", []byte(`schema: 3
board: "default"
cards: {}
frames: {}
pens:
  "fe": {title: "Frontend", x: 0, y: 0, w: 10, h: 10, color: "#759bcc", pin: {x: 0, y: 0}, requiredLabels: ["frontend", "bug"]}
ruleOrder: ["fe"]
inbox: {x: 0, y: 0}
`))
	if err != nil {
		t.Fatal(err)
	}
	if got := b.Pens["fe"].Match; !reflect.DeepEqual(got, Match{Labels: []string{"frontend", "bug"}}) {
		t.Fatalf("requiredLabels read as %+v, want match.labels", got)
	}
	if got := Explain(b, RuleTicket{ID: "T-1", Labels: []string{"bug", "frontend"}}); got.Destination != "fe" {
		t.Fatalf("destination = %q, want fe: a legacy rule still catches what it caught", got.Destination)
	}
	// Labels conjoin, at schema 3 as at schema 4.
	got := Explain(b, RuleTicket{ID: "T-2", Labels: []string{"frontend"}})
	if !got.Inbox() || !reflect.DeepEqual(got.Candidates[0].MissingLabels, []string{"bug"}) {
		t.Fatalf("a ticket carrying one of two labels routed to %q missing %v", got.Destination, got.Candidates[0].MissingLabels)
	}
}

// The envelope contract in plan 10.10 promises arrays, so a candidate carries
// an empty missingLabels and failed, and a match carries all four of its
// fields, never null. A test on the Go value would pass with nil.
func TestCandidatesSerializeArraysNotNull(t *testing.T) {
	b := routingBoard()
	b.Pens["ready"] = Pen{Title: "Ready", Match: Match{Status: []string{"ready"}}}
	b.RuleOrder = append(b.RuleOrder, "ready")
	e := Explain(b, RuleTicket{ID: "T-1", Labels: []string{"frontend", "bug"}, Status: "ready"})
	data, err := json.Marshal(e.Candidates)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "null") {
		t.Fatalf("candidates carry null:\n%s", data)
	}
	var got []map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	for _, c := range got {
		match, ok := c["match"].(map[string]any)
		if !ok {
			t.Fatalf("candidate %v carries no match record", c)
		}
		for _, field := range []string{"labels", "status", "type", "parent"} {
			if _, ok := match[field].([]any); !ok {
				t.Errorf("match.%s = %v, want an array on every pen", field, match[field])
			}
		}
		for _, field := range []string{"missingLabels", "failed"} {
			if _, ok := c[field].([]any); !ok {
				t.Errorf("%s = %v, want an array", field, c[field])
			}
		}
	}
}
