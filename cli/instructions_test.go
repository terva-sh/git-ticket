package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstructionsNeedsNoStore is why the command exists: a person runs it to
// find out how to use the tool, which is before they have a store to use it on.
func TestInstructionsNeedsNoStore(t *testing.T) {
	got := runCLI(t, t.TempDir(), nil, "instructions")
	if got.code != exitOK {
		t.Fatalf("instructions outside a store: %s", got.stderr)
	}
	if !strings.HasPrefix(got.stdout, "## Tickets\n") {
		t.Errorf("the block does not start with its heading: %.40q", got.stdout)
	}
	// It is pasted into a Markdown file, so it ends with exactly one newline
	// and does not run into whatever follows it.
	if !strings.HasSuffix(got.stdout, "\n") || strings.HasSuffix(got.stdout, "\n\n") {
		t.Error("the block should end with exactly one newline")
	}
}

// codeSpans returns the text of every backtick-delimited span in s. The block
// puts each invocation in one, so this is the set of things it tells a reader
// to type, separated from the prose around them.
func codeSpans(s string) []string {
	var out []string
	for {
		i := strings.Index(s, "`")
		if i < 0 {
			return out
		}
		s = s[i+1:]
		j := strings.Index(s, "`")
		if j < 0 {
			return out
		}
		out = append(out, s[:j])
		s = s[j+1:]
	}
}

// TestInstructionsNameRealCommands is the check worth having on prose that
// tells an agent what to type. Every command the block names has to exist, and
// every flag it passes has to be one that command accepts.
//
// The flag half is not hypothetical. The first draft of this block said
// `git ticket files ID --add PATH`, inventing a mutation for what is a query
// from a path back to the tickets that reference it. A version of this test
// that checked only the command name passed, because `files` is a real command.
func TestInstructionsNameRealCommands(t *testing.T) {
	// help is dispatched in Run before the command table, so it is a real
	// invocation that commands() does not list.
	known := map[string]bool{"help": true}
	for _, c := range commands() {
		known[c.name] = true
	}

	checked := 0
	for _, span := range codeSpans(instructionsText) {
		fields := strings.Fields(span)
		if len(fields) < 3 || fields[0] != "git" || fields[1] != "ticket" {
			continue
		}
		name := fields[2]
		if !known[name] {
			t.Errorf("the block says to run `%s`, and %q is not a command", span, name)
			continue
		}
		checked++

		for _, tok := range fields[3:] {
			if !strings.HasPrefix(tok, "--") || tok == "--" {
				continue
			}
			// Running the command with the flag and nothing else is enough.
			// It fails for other reasons, such as a missing value or no store,
			// and those are not what this asserts. Only the flag package's own
			// complaint about an unknown flag is a failure here.
			got := runCLI(t, t.TempDir(), nil, name, tok)
			if strings.Contains(got.stderr, "not defined") {
				t.Errorf("the block says to run `%s`, but %s does not take %s",
					span, name, tok)
			}
		}
	}
	if checked == 0 {
		t.Fatal("the block names no commands, so this test is checking nothing")
	}
}

// TestInstructionsWorkflowRuns drives the workflow the block describes against
// a real store, in the order the block puts it. Each step names the span it
// comes from, so a step that runs is also a step the block still names, and in
// a position after the one before it.
//
// TestInstructionsNameRealCommands checks that every command and flag exists,
// which is not enough. The block used to say "claim it, then status
// in-progress", and a ticket create just wrote is in draft, which cannot be
// claimed. Every command in that sentence was real and only the order was
// wrong, so the block told an agent to run a sequence that fails on its first
// step. This runs the sequence.
func TestInstructionsWorkflowRuns(t *testing.T) {
	dir := newGitStore(t)
	const actor = "agent:test/session"
	id := ticketID(t, createTicket(t, dir))

	// Why the block names the ready step at all. If this ever succeeds, the
	// lifecycle changed and the block's first paragraph is stale prose.
	if got := runCLI(t, dir, nil, "claim", id, "--actor", actor); got.code == exitOK {
		t.Fatal("a draft can be claimed now, so the block's ready step is stale")
	}

	// Setup rather than a step: `ac ID --check N` has nothing to tick without a
	// criterion, and the block puts `ac --add` in a different section.
	if got := runCLI(t, dir, nil, "ac", id, "--add", "the key rotates",
		"--actor", actor); got.code != exitOK {
		t.Fatalf("ac --add: %s", got.stderr)
	}

	steps := []struct {
		span string   // as the block writes it
		args []string // as this test runs it
	}{
		{"git ticket status ID ready", []string{"status", id, "ready"}},
		{"git ticket claim ID", []string{"claim", id}},
		{"git ticket status ID in-progress", []string{"status", id, "in-progress"}},
		{"git ticket plan ID", []string{"plan", id, "1. Reproduce with a stepped clock\n2. Widen the window"}},
		{"git ticket note ID", []string{"note", id, "the skew is 40s, not the 5s we assumed"}},
		{"git ticket ac ID --check N", []string{"ac", id, "--check", "1"}},
		{"git ticket summary ID", []string{"summary", id, "widened the window"}},
		{"git ticket status ID done", []string{"status", id, "done"}},
		{"git ticket release ID", []string{"release", id}},
	}

	prev := 0
	for _, s := range steps {
		at := strings.Index(instructionsText, s.span)
		if at < 0 {
			t.Fatalf("the block no longer names `%s`", s.span)
		}
		if at < prev {
			t.Errorf("the block names `%s` before a step it has to follow", s.span)
		}
		prev = at
		if got := runCLI(t, dir, nil, append(s.args, "--actor", actor)...); got.code != exitOK {
			t.Fatalf("`%s`: exit %d\n%s%s", s.span, got.code, got.stdout, got.stderr)
		}
	}

	// A store the documented workflow produced is a store check accepts.
	if got := runCLI(t, dir, nil, "check", "--strict"); got.code != exitOK {
		t.Errorf("check --strict after the documented workflow:\n%s%s", got.stdout, got.stderr)
	}
}

// TestInitWritesInstructionsOnlyWhenAsked covers plan 12.1: init may write the
// block, and never without being asked.
func TestInitWritesInstructionsOnlyWhenAsked(t *testing.T) {
	plain := t.TempDir()
	if got := runCLI(t, plain, nil, "init", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("init: %s", got.stderr)
	}
	if _, err := os.Stat(filepath.Join(plain, instructionsFile)); !os.IsNotExist(err) {
		t.Errorf("init wrote %s without --instructions", instructionsFile)
	}

	asked := t.TempDir()
	got := runCLI(t, asked, nil, "init", "--instructions", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("init --instructions: %s", got.stderr)
	}
	body, err := os.ReadFile(filepath.Join(asked, instructionsFile))
	if err != nil {
		t.Fatalf("reading the written file: %v", err)
	}
	if string(body) != instructionsText {
		t.Error("the written file is not the block instructions prints")
	}
	if !strings.Contains(got.stdout, instructionsFile) {
		t.Errorf("init did not say it wrote the file: %q", got.stdout)
	}
}

// TestInitRefusesToTouchAMaintainedFile is the half of plan 12.1 that matters.
// AGENTS.md is a file the user writes, so init refuses rather than overwriting
// it, and refuses before making the store so there is nothing to clean up.
func TestInitRefusesToTouchAMaintainedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, instructionsFile)
	const mine = "# my own instructions\n"
	if err := os.WriteFile(path, []byte(mine), 0o644); err != nil {
		t.Fatal(err)
	}

	got := runCLI(t, dir, nil, "--json", "init", "--instructions", "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("init --instructions should refuse when the file exists")
	}
	if code := errCode(t, got); code != codeUsage {
		t.Errorf("code = %v, want %s", code, codeUsage)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != mine {
		t.Error("the user's file was modified")
	}
	// Refused before the store was made, so the repository is as it was.
	if _, err := os.Stat(filepath.Join(dir, ".tickets")); !os.IsNotExist(err) {
		t.Error("a refused init left a store behind")
	}
}

func TestInstructionsTakesNoArguments(t *testing.T) {
	got := runCLI(t, t.TempDir(), nil, "--json", "instructions", "extra")
	if got.code != exitError {
		t.Fatal("a stray argument should be refused")
	}
	if code := errCode(t, got); code != codeUsage {
		t.Errorf("code = %v, want %s", code, codeUsage)
	}
}
