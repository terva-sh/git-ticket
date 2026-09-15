package cli

import (
	"strings"
	"testing"
)

// withHeading is text destined for one body section that carries a line the
// parser reads as the start of another one. Everything below "## Risks" leaves
// the section it was written for.
const withHeading = "Prose that was meant to be the whole section.\n\n## Risks\n\nThis lands somewhere else."

// TestHeadingWarningNamesTheFix is the whole point of the warning. Reporting
// that something is wrong without saying what to type instead leaves the reader
// exactly where they started, because the rule is invisible: nothing else
// reports it and the file reads correctly afterwards.
func TestHeadingWarningNamesTheFix(t *testing.T) {
	dir := newStore(t)

	got := runCLI(t, dir, nil, "create", "--title", "Rotate the signing key",
		"--description", withHeading, "--actor", "human:sothr")

	if got.code != exitOK {
		t.Fatalf("the write should still succeed: %s%s", got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, "## Risks") {
		t.Errorf("the warning does not name the heading it found:\n%s", got.stderr)
	}
	if !strings.Contains(got.stderr, "### Risks") {
		t.Errorf("the warning does not name the fix:\n%s", got.stderr)
	}
}

// TestHeadingWarningCoversEveryWritingCommand walks the list in the ticket. A
// warning on create alone would be worth little, because the command that
// filed TKT-01M1HVMQ wrongly is not the only one that takes section text.
func TestHeadingWarningCoversEveryWritingCommand(t *testing.T) {
	cases := []struct {
		name string
		args func(id string) []string
	}{
		{"create --description", func(string) []string {
			return []string{"create", "--title", "T", "--description", withHeading}
		}},
		{"create --plan", func(string) []string {
			return []string{"create", "--title", "T", "--plan", withHeading}
		}},
		{"update --description", func(id string) []string {
			return []string{"update", id, "--description", withHeading}
		}},
		{"plan", func(id string) []string { return []string{"plan", id, withHeading} }},
		{"summary", func(id string) []string { return []string{"summary", id, withHeading} }},
		{"note", func(id string) []string { return []string{"note", id, withHeading} }},
		{"comment", func(id string) []string { return []string{"comment", id, withHeading} }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := newStore(t)
			id := ticketID(t, createTicket(t, dir))

			args := append(c.args(id), "--actor", "human:sothr")
			got := runCLI(t, dir, nil, args...)

			if got.code != exitOK {
				t.Fatalf("%s failed: %s%s", c.name, got.stdout, got.stderr)
			}
			if !strings.Contains(got.stderr, "### Risks") {
				t.Errorf("%s wrote a split section and said nothing:\n%s", c.name, got.stderr)
			}
		})
	}
}

// TestHeadingWarningStaysOffStdout is the constraint that makes this safe to
// add at all. A caller parsing the envelope must not have to care that a
// warning exists, so it goes to stderr in both modes and moves no exit status.
func TestHeadingWarningStaysOffStdout(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	got := runCLI(t, dir, nil, "--json", "summary", id, withHeading,
		"--actor", "human:sothr")

	if got.code != exitOK {
		t.Fatalf("a warning must not change the exit status: %d", got.code)
	}
	if !strings.Contains(got.stderr, "### Risks") {
		t.Fatalf("the warning is missing under --json:\n%s", got.stderr)
	}
	if strings.Contains(got.stdout, "warning") {
		t.Errorf("the warning leaked into the envelope:\n%s", got.stdout)
	}
	// Parses, so the envelope survived having a warning printed alongside it.
	env := decode(t, got.stdout)
	if env["kind"] != "mutation-result" {
		t.Errorf("kind = %v, want mutation-result", env["kind"])
	}
}

// TestHeadingWarningIsSilentWhereTheParserIs covers the way a guard like this
// fails in practice. Warning on text the parser would not split teaches the
// reader to ignore it, and a ticket quoting Markdown is not unusual.
func TestHeadingWarningIsSilentWhereTheParserIs(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{"backtick fence", "Prose.\n\n```\n## NotAHeading\n```\n\nMore."},
		{"tilde fence", "Prose.\n\n~~~\n## NotAHeading\n~~~\n\nMore."},
		{"indented", "Prose.\n\n  ## Indented\n\nMore."},
		{"a real subheading", "Prose.\n\n### Risks\n\nMore."},
		{"no heading at all", "Prose with ## in the middle of a line."},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := newStore(t)
			id := ticketID(t, createTicket(t, dir))

			got := runCLI(t, dir, nil, "summary", id, c.text, "--actor", "human:sothr")

			if got.code != exitOK {
				t.Fatalf("summary failed: %s%s", got.stdout, got.stderr)
			}
			if strings.Contains(got.stderr, "warning") {
				t.Errorf("warned about text the parser does not split:\n%s", got.stderr)
			}
		})
	}
}

// TestChecklistAddDoesNotWarn holds the scope line the ticket drew. An item is
// rendered as "- [ ] TEXT", so the line cannot open with "## " however the text
// starts, and warning here would be noise about an impossible condition.
func TestChecklistAddDoesNotWarn(t *testing.T) {
	for _, cmd := range []string{"ac", "dod"} {
		t.Run(cmd, func(t *testing.T) {
			dir := newStore(t)
			id := ticketID(t, createTicket(t, dir))

			got := runCLI(t, dir, nil, cmd, id, "--add", "## looks like a heading",
				"--actor", "human:sothr")

			if got.code != exitOK {
				t.Fatalf("%s --add failed: %s%s", cmd, got.stdout, got.stderr)
			}
			if strings.Contains(got.stderr, "warning") {
				t.Errorf("%s --add warned, but a checkbox line cannot open a section:\n%s",
					cmd, got.stderr)
			}
		})
	}
}

// demotedCriteria is a description carrying the mistake this whole check exists
// for: a "###" heading named for a section the format owns, with items under it
// that read as criteria and are prose.
const demotedCriteria = "Body.\n\n### Acceptance criteria\n\n- [ ] first\n- [ ] second\n"

// TestCheckReportsADemotedSectionHeading is the reported defect, end to end.
//
// It asserts the exit status as well as the text, because the whole failure was
// that a ticket in this shape passed check --strict in CI for several sessions
// while a person read it as having criteria.
func TestCheckReportsADemotedSectionHeading(t *testing.T) {
	dir := newStore(t)
	runCLI(t, dir, nil, "create", "--title", "Reproduction",
		"--description", demotedCriteria, "--actor", "human:sothr")

	// Plain check passes: the store is valid and nothing dangles. --strict is
	// where a judgement about what the author meant is allowed to bite.
	if got := runCLI(t, dir, nil, "check"); got.code != exitOK {
		t.Errorf("plain check should pass, got %d: %s%s", got.code, got.stdout, got.stderr)
	}
	got := runCLI(t, dir, nil, "check", "--strict")
	if got.code != exitError {
		t.Fatalf("check --strict should fail, got %d: %s%s", got.code, got.stdout, got.stderr)
	}
	out := got.stdout + got.stderr
	for _, want := range []string{
		"section_heading_demoted",
		"### Acceptance criteria",
		"## Acceptance criteria",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the report does not carry %q, so the fix is not obvious from it:\n%s", want, out)
		}
	}
}

// TestCheckLeavesOrdinarySubheadingsAlone is the control, and it is the reason
// the check is keyed on the name rather than the level.
//
// This repository's own store carries 14 "### Trigger", 8 "### Open questions"
// and 7 "### Scope". A rule that flagged the level would fire on every one of
// them, which is a check that arrives with a backlog and teaches its readers to
// ignore it.
func TestCheckLeavesOrdinarySubheadingsAlone(t *testing.T) {
	dir := newStore(t)
	body := "Body.\n\n### Trigger\n\nA real report.\n\n### Scope\n\nWhat this covers.\n\n### Open questions\n\nNone.\n"
	runCLI(t, dir, nil, "create", "--title", "Ordinary subheadings",
		"--description", body, "--actor", "human:sothr")

	got := runCLI(t, dir, nil, "check", "--strict")
	if got.code != exitOK {
		t.Fatalf("check --strict should pass on ordinary subheadings: %s%s", got.stdout, got.stderr)
	}
	if strings.Contains(got.stdout+got.stderr, "section_heading_demoted") {
		t.Errorf("an ordinary subheading was reported:\n%s%s", got.stdout, got.stderr)
	}
}

// TestCheckIgnoresASubheadingInsideAFence holds the fence rule.
//
// Documentation about this very defect is the likeliest place to write
// "### Acceptance criteria" inside a code block, so the check that reports it
// must not fire on the prose explaining it.
func TestCheckIgnoresASubheadingInsideAFence(t *testing.T) {
	dir := newStore(t)
	body := "How the mistake looks:\n\n```markdown\n### Acceptance criteria\n\n- [ ] not a real one\n```\n"
	runCLI(t, dir, nil, "create", "--title", "Documenting the trap",
		"--description", body, "--actor", "human:sothr")

	if got := runCLI(t, dir, nil, "check", "--strict"); got.code != exitOK {
		t.Fatalf("a fenced heading is text, not a heading: %s%s", got.stdout, got.stderr)
	}
}

// TestTheWarningDoesNotAdviseBreakingAWorkingTicket is the half the triage
// added, and it is the defect that produced the report in the first place.
//
// "## Acceptance criteria" opens the real section and its items tick. The
// warning used to answer that by telling the author to write
// "### Acceptance criteria" instead, which is the unreachable form, so following
// the tool's own advice was how a ticket arrived broken.
func TestTheWarningDoesNotAdviseBreakingAWorkingTicket(t *testing.T) {
	dir := newStore(t)
	body := "Body.\n\n## Acceptance criteria\n\n- [ ] first\n"
	got := runCLI(t, dir, nil, "--json", "create", "--title", "Two hashes",
		"--description", body, "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create: %s%s", got.stdout, got.stderr)
	}

	// The advice must not be the thing that breaks it.
	if strings.Contains(got.stderr, `Write "### Acceptance criteria" instead`) {
		t.Errorf("the warning still advises the unreachable form:\n%s", got.stderr)
	}
	// And it should say what actually happened, because the section did move.
	for _, want := range []string{"## Acceptance criteria", "real"} {
		if !strings.Contains(got.stderr, want) {
			t.Errorf("stderr does not say the real section was opened (missing %q):\n%s", want, got.stderr)
		}
	}

	// The claim the message makes has to be true: the items are live.
	id, _ := decode(t, got.stdout)["ticket"].(map[string]any)["id"].(string)
	if r := runCLI(t, dir, nil, "ac", id, "--check", "1", "--actor", "human:sothr"); r.code != exitOK {
		t.Errorf("the message says the items are live, but ticking one failed: %s%s", r.stdout, r.stderr)
	}
	// And the ticket it produced is not the one check complains about.
	if r := runCLI(t, dir, nil, "check", "--strict"); r.code != exitOK {
		t.Errorf("check --strict should pass on the working form: %s%s", r.stdout, r.stderr)
	}
}

// TestTheWarningStillWarnsAboutAnOrdinaryHeading keeps the original behaviour
// for every name the format does not own. The note above is a narrowing, not a
// replacement, and without this the narrowing could have swallowed the warning.
func TestTheWarningStillWarnsAboutAnOrdinaryHeading(t *testing.T) {
	dir := newStore(t)
	got := runCLI(t, dir, nil, "create", "--title", "Rotate the signing key",
		"--description", withHeading, "--actor", "human:sothr")
	if !strings.Contains(got.stderr, "### Risks") {
		t.Errorf("the ordinary warning lost its advice:\n%s", got.stderr)
	}
}
