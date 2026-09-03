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
