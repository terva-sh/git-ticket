package ticket

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The framework ships no rules of its own, so every test here registers its
// own through the same Rule shape a shipped rule uses. That is the point: the
// machinery is what this ticket builds, and it would otherwise go out with
// nothing having ever run a rule through it.
func countingRule(id string, level Level, fire bool) Rule {
	return Rule{
		ID:      id,
		Level:   level,
		Summary: "a rule that exists to be run",
		Check: func(rc RuleContext) []DoctorFinding {
			if !fire {
				return nil
			}
			return []DoctorFinding{{
				Rule:    id,
				Level:   rc.Level,
				Message: "fired",
				Remedy:  "do the thing",
			}}
		},
	}
}

// writeDoctorConfig puts a doctor block in the store's config.yml and reopens,
// which is the path a real store takes: a person edits the file, and the next
// command reads it.
func writeDoctorConfig(t *testing.T, s *Store, body string) *Store {
	t.Helper()
	path := filepath.Join(s.Path(), "config.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if err := os.WriteFile(path, append(data, []byte(body)...), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	reopened, err := Open(s.Path())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	return reopened
}

func doctorOn(t *testing.T, s *Store, rules ...Rule) *DoctorReport {
	t.Helper()
	r, err := s.Doctor(context.Background(), rules, time.Now().UTC())
	if err != nil {
		t.Fatalf("doctor: %v", err)
	}
	return r
}

// TestRulesShipOnByDefault is the criterion that makes this a tool with an
// opinion rather than a linter everybody writes themselves.
func TestRulesShipOnByDefault(t *testing.T) {
	s := newTestStore(t)
	mustCreate(t, s, "A ticket")

	r := doctorOn(t, s, countingRule("test_fires", LevelHard, true))
	if len(r.Findings) != 1 {
		t.Fatalf("a rule the store said nothing about did not run: %+v", r.Findings)
	}
	if r.Findings[0].Rule != "test_fires" {
		t.Errorf("finding came from %q", r.Findings[0].Rule)
	}
}

func TestAStoreTurnsARuleOff(t *testing.T) {
	s := newTestStore(t)
	mustCreate(t, s, "A ticket")
	s = writeDoctorConfig(t, s, "\ndoctor:\n  rules:\n    test_fires:\n      enabled: false\n")

	if r := doctorOn(t, s, countingRule("test_fires", LevelHard, true)); len(r.Findings) != 0 {
		t.Errorf("a disabled rule still ran: %+v", r.Findings)
	}
}

// TestAStoreChangesARulesLevel matters more than it looks. The level a rule
// runs at is not only how its finding sorts: a rule reads it to decide whether
// to phrase itself as a question, so a rule moved to soft that kept asserting
// would be making a claim the store said it could not settle.
func TestAStoreChangesARulesLevel(t *testing.T) {
	s := newTestStore(t)
	mustCreate(t, s, "A ticket")
	s = writeDoctorConfig(t, s, "\ndoctor:\n  rules:\n    test_fires:\n      level: soft\n")

	r := doctorOn(t, s, countingRule("test_fires", LevelHard, true))
	if len(r.Findings) != 1 {
		t.Fatalf("want one finding, got %+v", r.Findings)
	}
	if r.Findings[0].Level != LevelSoft {
		t.Errorf("level = %q, want soft: the rule did not see the override", r.Findings[0].Level)
	}
	if r.Grade() != GradeSoft {
		t.Errorf("grade = %v, want soft: a rule moved to soft must stop failing the run", r.Grade())
	}
}

// TestAnUnreadableLevelKeepsTheShippedOne refuses to guess. A level the binary
// does not know is a configuration that says nothing, and defaulting to the
// zero value would silently make every such rule soft.
func TestAnUnreadableLevelKeepsTheShippedOne(t *testing.T) {
	s := newTestStore(t)
	mustCreate(t, s, "A ticket")
	s = writeDoctorConfig(t, s, "\ndoctor:\n  rules:\n    test_fires:\n      level: urgent\n")

	r := doctorOn(t, s, countingRule("test_fires", LevelHard, true))
	if len(r.Findings) != 1 || r.Findings[0].Level != LevelHard {
		t.Errorf("an unreadable level did not fall back to the shipped one: %+v", r.Findings)
	}
}

// TestAStoreSetsARulesParams proves the third thing a store may do. No shipped
// rule reads a parameter yet, so the plumbing would otherwise be untested until
// the first one that does.
func TestAStoreSetsARulesParams(t *testing.T) {
	s := newTestStore(t)
	mustCreate(t, s, "A ticket")
	s = writeDoctorConfig(t, s,
		"\ndoctor:\n  rules:\n    test_params:\n      params:\n        after: 90d\n        count: 3\n")

	var got map[string]any
	rule := Rule{ID: "test_params", Level: LevelHard, Check: func(rc RuleContext) []DoctorFinding {
		got = rc.Params
		return nil
	}}
	doctorOn(t, s, rule)

	if got["after"] != "90d" {
		t.Errorf("after = %v, want 90d", got["after"])
	}
	if got["count"] != 3 {
		t.Errorf("count = %v (%T), want 3", got["count"], got["count"])
	}
}

// TestHardFindingsSortBeforeSoft is the reporting order, which is the whole
// reason the report is ordered at all: the first thing printed should be the
// thing most worth fixing.
func TestHardFindingsSortBeforeSoft(t *testing.T) {
	s := newTestStore(t)
	mustCreate(t, s, "A ticket")

	r := doctorOn(t, s,
		countingRule("a_soft", LevelSoft, true),
		countingRule("z_hard", LevelHard, true),
	)
	if len(r.Findings) != 2 {
		t.Fatalf("want two findings, got %+v", r.Findings)
	}
	// Alphabetically a_soft sorts first, so an order that came out hard-first
	// cannot have come from sorting on the rule ID.
	if r.Findings[0].Level != LevelHard || r.Findings[1].Level != LevelSoft {
		t.Errorf("findings are not hard before soft: %+v", r.Findings)
	}
}

// TestGradeIsTheWorstLevelThatFired is what the exit status encodes, and the
// reason it is a grade rather than a mask: one ordered category.
func TestGradeIsTheWorstLevelThatFired(t *testing.T) {
	s := newTestStore(t)
	mustCreate(t, s, "A ticket")

	for _, c := range []struct {
		what  string
		rules []Rule
		want  Grade
	}{
		{"nothing fired", []Rule{countingRule("quiet", LevelHard, false)}, GradeClean},
		{"soft only", []Rule{countingRule("s", LevelSoft, true)}, GradeSoft},
		{"hard only", []Rule{countingRule("h", LevelHard, true)}, GradeHard},
		{"both", []Rule{countingRule("s", LevelSoft, true), countingRule("h", LevelHard, true)}, GradeHard},
	} {
		if got := doctorOn(t, s, c.rules...).Grade(); got != c.want {
			t.Errorf("%s: grade = %v, want %v", c.what, got, c.want)
		}
	}
}

// TestAnUnknownRuleIDIsReported is what stops a misspelled rule name from being
// a configuration that silently does nothing. ParseConfig does not set
// KnownFields and the rules map accepts any key, so without this the store
// would be configuring a rule that never existed and nothing would say so.
func TestAnUnknownRuleIDIsReported(t *testing.T) {
	s := newTestStore(t)
	mustCreate(t, s, "A ticket")
	s = writeDoctorConfig(t, s, "\ndoctor:\n  rules:\n    labl_presnt:\n      enabled: false\n")

	r := doctorOn(t, s, countingRule("test_fires", LevelHard, true))

	var found *DoctorFinding
	for i := range r.Findings {
		if r.Findings[i].Rule == RuleUnknown {
			found = &r.Findings[i]
		}
	}
	if found == nil {
		t.Fatalf("a misspelled rule ID was not reported: %+v", r.Findings)
	}
	if !strings.Contains(found.Message, "labl_presnt") {
		t.Errorf("the finding does not name the ID that was wrong: %q", found.Message)
	}
	if found.Remedy == "" {
		t.Error("the finding says nothing about what would resolve it")
	}
	// The known rule still ran. A bad entry is not a reason to stop reporting.
	if len(r.Findings) != 2 {
		t.Errorf("an unknown ID stopped the rest of the run: %+v", r.Findings)
	}
}

// TestEveryFindingSaysWhatWouldResolveIt holds the criterion across every
// finding the framework itself can produce, rather than one of them.
func TestEveryFindingSaysWhatWouldResolveIt(t *testing.T) {
	s := newTestStore(t)
	mustCreate(t, s, "A ticket")
	s = writeDoctorConfig(t, s, "\ndoctor:\n  rules:\n    nosuchrule:\n      enabled: true\n")

	r := doctorOn(t, s, countingRule("test_fires", LevelHard, true))
	if len(r.Findings) == 0 {
		t.Fatal("nothing to assert on")
	}
	for _, f := range r.Findings {
		if f.Remedy == "" {
			t.Errorf("%s says nothing about what would resolve it", f.Rule)
		}
		if f.Message == "" {
			t.Errorf("%s has no message", f.Rule)
		}
	}
}

// TestRenderConfigKeepsTheDoctorBlock is a data-loss guard, not a formatting
// one. migrate and `series add` both rewrite config.yml through RenderConfig,
// so a block the emitter does not know how to write back is a block those
// commands delete: a store that turned a rule off would find it on again after
// an unrelated command, with nothing said.
func TestRenderConfigKeepsTheDoctorBlock(t *testing.T) {
	s := newTestStore(t)
	s = writeDoctorConfig(t, s,
		"\ndoctor:\n  rules:\n    a_rule:\n      enabled: false\n      level: soft\n      params:\n        after: 90d\n")

	round, err := ParseConfig(RenderConfig(s.Config()))
	if err != nil {
		t.Fatalf("reparse: %v", err)
	}
	rc, ok := round.Doctor.Rules["a_rule"]
	if !ok {
		t.Fatalf("the doctor block did not survive a render: %s", RenderConfig(s.Config()))
	}
	if rc.Enabled == nil || *rc.Enabled {
		t.Errorf("enabled did not survive: %+v", rc.Enabled)
	}
	if rc.Level == nil || *rc.Level != LevelSoft {
		t.Errorf("level did not survive: %+v", rc.Level)
	}
	if rc.Params["after"] != "90d" {
		t.Errorf("params did not survive: %+v", rc.Params)
	}
}

// TestAStoreWithNoDoctorBlockGrowsNone keeps a fresh config.yml from sprouting
// a block nobody asked for, the same rule the series list follows.
func TestAStoreWithNoDoctorBlockGrowsNone(t *testing.T) {
	s := newTestStore(t)
	if got := string(RenderConfig(s.Config())); strings.Contains(got, "doctor:") {
		t.Errorf("a store that configured nothing grew a doctor block:\n%s", got)
	}
}
