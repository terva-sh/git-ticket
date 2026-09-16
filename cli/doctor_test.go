package cli

import (
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/ticket"
)

// withRules installs a rule set for one test. DefaultRules is empty until
// TKT-01M2NJDAVHTXKEPJ0ZCAJP3QY0 lands the first two, so without this seam the
// grades and the output below would ship with nothing having exercised them.
func withRules(t *testing.T, rules ...ticket.Rule) {
	t.Helper()
	prev := doctorRules
	doctorRules = func() []ticket.Rule { return rules }
	t.Cleanup(func() { doctorRules = prev })
}

func firing(id string, level ticket.Level) ticket.Rule {
	return ticket.Rule{
		ID:      id,
		Level:   level,
		Summary: "a rule that exists to be run",
		Check: func(rc ticket.RuleContext) []ticket.DoctorFinding {
			out := make([]ticket.DoctorFinding, 0, len(rc.Tickets))
			for _, tk := range rc.Tickets {
				out = append(out, ticket.DoctorFinding{
					Rule:    id,
					Level:   rc.Level,
					Ticket:  tk.ID,
					Message: "this ticket tripped " + id,
					Remedy:  "tidy it",
				})
			}
			return out
		},
	}
}

func TestDoctorSaysNothingWhenThereIsNothing(t *testing.T) {
	dir := newStore(t)
	withRules(t)

	got := runCLI(t, dir, nil, "doctor")
	if got.code != exitOK {
		t.Fatalf("doctor exited %d: %s", got.code, got.stderr)
	}
	if !strings.Contains(got.stdout, "Nothing to tidy.") {
		t.Errorf("a clean store did not say so: %q", got.stdout)
	}
}

// TestDoctorNeverFailsABuildByDefault is the separation from check stated as an
// exit status. Hygiene is advice, and advice that fails a build is a rule.
func TestDoctorNeverFailsABuildByDefault(t *testing.T) {
	dir := newStore(t)
	createTicket(t, dir, "--title", "A ticket")
	withRules(t, firing("test_hard", ticket.LevelHard))

	got := runCLI(t, dir, nil, "doctor")
	if got.code != exitOK {
		t.Errorf("doctor without --strict exited %d, want 0: %s", got.code, got.stdout)
	}
	if !strings.Contains(got.stdout, "test_hard") {
		t.Errorf("the finding was not reported: %q", got.stdout)
	}
}

// TestDoctorStrictExitsByTheGradedBucket is the reserved bucket of plan 10.2.
// A grade and not a mask: one ordered category, readable in a shell comparison.
func TestDoctorStrictExitsByTheGradedBucket(t *testing.T) {
	for _, c := range []struct {
		what  string
		rules []ticket.Rule
		want  int
	}{
		{"clean", nil, exitOK},
		{"soft only", []ticket.Rule{firing("test_soft", ticket.LevelSoft)}, exitDoctorSoft},
		{"hard", []ticket.Rule{firing("test_hard", ticket.LevelHard)}, exitDoctorHard},
		{
			"both, worst wins",
			[]ticket.Rule{firing("test_soft", ticket.LevelSoft), firing("test_hard", ticket.LevelHard)},
			exitDoctorHard,
		},
	} {
		t.Run(c.what, func(t *testing.T) {
			dir := newStore(t)
			createTicket(t, dir, "--title", "A ticket")
			withRules(t, c.rules...)

			got := runCLI(t, dir, nil, "doctor", "--strict")
			if got.code != c.want {
				t.Errorf("exit = %d, want %d: %s", got.code, c.want, got.stdout)
			}
		})
	}
}

// TestDoctorStrictSaysWhichNumberItExitedWith closes the gap a bare status
// leaves: a reader who gets 20 from a shell has nowhere to look it up.
func TestDoctorStrictSaysWhichNumberItExitedWith(t *testing.T) {
	dir := newStore(t)
	createTicket(t, dir, "--title", "A ticket")
	withRules(t, firing("test_soft", ticket.LevelSoft))

	got := runCLI(t, dir, nil, "doctor", "--strict")
	if !strings.Contains(got.stdout, "Exiting 20") {
		t.Errorf("the output does not name the status it exited with: %q", got.stdout)
	}
	// And it says why that is not a failure, since a non-zero status normally is.
	if !strings.Contains(got.stdout, "questions rather than failures") {
		t.Errorf("the output does not say why a soft grade is not a failure: %q", got.stdout)
	}
}

// TestDoctorSoftFindingNeverFailsTheRun is the criterion that survives the
// bucket. A soft finding may put the command in the informational grade. It may
// not make the run fail, and 20 is below the bucket a failure would use.
func TestDoctorSoftFindingNeverFailsTheRun(t *testing.T) {
	dir := newStore(t)
	createTicket(t, dir, "--title", "A ticket")
	withRules(t, firing("test_soft", ticket.LevelSoft))

	if got := runCLI(t, dir, nil, "doctor"); got.code != exitOK {
		t.Errorf("a soft finding failed a default run: %d", got.code)
	}
	if got := runCLI(t, dir, nil, "doctor", "--strict"); got.code == exitError {
		t.Errorf("a soft finding produced the failure status rather than the informational one")
	}
}

func TestDoctorReportsEveryFindingWithARemedy(t *testing.T) {
	dir := newStore(t)
	createTicket(t, dir, "--title", "A ticket")
	withRules(t, firing("test_hard", ticket.LevelHard))

	got := runCLI(t, dir, nil, "doctor")
	if !strings.Contains(got.stdout, "tidy it") {
		t.Errorf("the remedy was not printed: %q", got.stdout)
	}
	if !strings.Contains(got.stdout, "1 hard finding, 0 soft findings") {
		t.Errorf("the tally is wrong or absent: %q", got.stdout)
	}
}

func TestDoctorJSONCarriesTheGradeAndTheFindings(t *testing.T) {
	dir := newStore(t)
	createTicket(t, dir, "--title", "A ticket")
	withRules(t, firing("test_soft", ticket.LevelSoft))

	env := decode(t, runCLI(t, dir, nil, "--json", "doctor").stdout)
	if env["kind"] != "doctor-report" {
		t.Fatalf("kind = %v, want doctor-report", env["kind"])
	}
	// Spelled rather than numbered: a consumer reading JSON has no use for the
	// exit number, and the spelling survives the reserved numbers moving.
	if env["grade"] != "soft" {
		t.Errorf("grade = %v, want soft", env["grade"])
	}
	findings, _ := env["findings"].([]any)
	if len(findings) != 1 {
		t.Fatalf("findings = %v", env["findings"])
	}
	f, _ := findings[0].(map[string]any)
	for _, key := range []string{"rule", "level", "ticket", "message", "remedy"} {
		if f[key] == nil || f[key] == "" {
			t.Errorf("finding is missing %s: %v", key, f)
		}
	}
}

// TestDoctorRuleIDsAndCheckCodesShareOneNamespace is the decision made explicit.
// Sharing is what lets a rule name a check code when it has to say where its
// own boundary is; the cost is that neither side may spend a name the other
// has, and nothing but this test would notice a collision.
func TestDoctorRuleIDsAndCheckCodesShareOneNamespace(t *testing.T) {
	spent := map[string]string{}
	for _, c := range ticket.CheckErrorCodes {
		spent[c] = "a check error code"
	}
	for _, c := range ticket.CheckWarningCodes {
		spent[c] = "a check warning code"
	}

	ids := []string{ticket.RuleUnknown}
	for _, r := range ticket.DefaultRules() {
		ids = append(ids, r.ID)
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if what, ok := spent[id]; ok {
			t.Errorf("doctor rule %q is already %s", id, what)
		}
		if seen[id] {
			t.Errorf("doctor rule %q is declared twice", id)
		}
		seen[id] = true
	}
}

// TestSchemaPublishesTheRuleIdentifiers is what makes an identifier a promise.
// A store writes these into config.yml, so a name that is not published is a
// name nobody could have known was safe to depend on.
func TestSchemaPublishesTheRuleIdentifiers(t *testing.T) {
	dir := t.TempDir()

	env := decode(t, runCLI(t, dir, nil, "--json", "schema").stdout)
	rules, _ := env["doctorRules"].([]any)
	if len(rules) == 0 {
		t.Fatalf("schema publishes no doctor rules: %v", env["doctorRules"])
	}
	found := false
	for _, r := range rules {
		m, _ := r.(map[string]any)
		if m["id"] == ticket.RuleUnknown {
			found = true
			if m["level"] != string(ticket.LevelHard) {
				t.Errorf("%s published at level %v", ticket.RuleUnknown, m["level"])
			}
			if m["summary"] == "" {
				t.Errorf("%s published with no summary", ticket.RuleUnknown)
			}
		}
	}
	if !found {
		t.Errorf("schema does not publish %s: %v", ticket.RuleUnknown, rules)
	}

	// The human form carries them too, since the reader most likely to be
	// choosing a name to put in config.yml is a person.
	if out := runCLI(t, dir, nil, "schema").stdout; !strings.Contains(out, "doctor rules") {
		t.Errorf("the human schema has no doctor rules section: %s", out)
	}
}
