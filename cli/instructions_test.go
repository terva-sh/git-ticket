package cli

import (
	"os"
	"os/exec"
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
	// The markers are part of what it prints, so a block a person pastes by
	// hand can be refreshed later by --write. That is the whole point of them.
	if !strings.HasPrefix(got.stdout, instructionsBegin+"\n\n## Tickets\n") {
		t.Errorf("the block does not open with the marker and its heading: %.60q", got.stdout)
	}
	if !strings.HasSuffix(got.stdout, instructionsEnd+"\n") {
		t.Errorf("the block does not close with its marker: %.60q", got.stdout[max(0, len(got.stdout)-60):])
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

// TestInitKeepsWhatIsAlreadyInAMaintainedFile covers the case init used to
// handle badly. AGENTS.md is a file the user writes, and refusing meant a
// project that already had one could never get the block from init at all. It
// appends instead, and every byte the user wrote is still there.
func TestInitKeepsWhatIsAlreadyInAMaintainedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, instructionsFile)
	const mine = "# my own instructions\n\nRun the tests before you push.\n"
	if err := os.WriteFile(path, []byte(mine), 0o644); err != nil {
		t.Fatal(err)
	}

	got := runCLI(t, dir, nil, "init", "--instructions", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("init --instructions: %s", got.stderr)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(body), mine) {
		t.Errorf("the user's prose did not survive:\n%s", body)
	}
	if !strings.Contains(string(body), instructionsBegin) {
		t.Error("the block was not appended")
	}
}

// TestInitRefusesAFileItCannotReadHonestly is the half of plan 12.1 that still
// matters. One marker without its partner leaves no honest reading of where the
// block ends, so init refuses rather than guessing, and refuses before making
// the store so there is nothing to clean up.
func TestInitRefusesAFileItCannotReadHonestly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, instructionsFile)
	mine := "# my own instructions\n\n" + instructionsBegin + "\n\nsomething\n"
	if err := os.WriteFile(path, []byte(mine), 0o644); err != nil {
		t.Fatal(err)
	}

	got := runCLI(t, dir, nil, "--json", "init", "--instructions", "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("init --instructions should refuse a file with one marker")
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

// TestWriteRefreshesTheBlockInPlace is the point of the markers. A store set up
// months ago is stuck with whatever the block said then, and re-pasting it by
// hand is what nobody does. The refresh replaces what sits between the markers
// and every other byte in the file is the byte that was there before.
func TestWriteRefreshesTheBlockInPlace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, instructionsFile)

	const prefix = "# my own instructions\n\nRun the tests before you push.\n\n"
	stale := instructionsBegin + "\n\n## Tickets\n\nWhatever it said last year.\n\n" + instructionsEnd
	const suffix = "\n\n## House rules\n\nKeep this too.\n"
	if err := os.WriteFile(path, []byte(prefix+stale+suffix), 0o644); err != nil {
		t.Fatal(err)
	}

	got := runCLI(t, dir, nil, "instructions", "--write")
	if got.code != exitOK {
		t.Fatalf("instructions --write: %s", got.stderr)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := prefix + strings.TrimRight(instructionsText, "\n") + suffix
	if string(body) != want {
		t.Errorf("the refresh did not land exactly:\n--- got ---\n%s\n--- want ---\n%s", body, want)
	}
	if strings.Contains(string(body), "Whatever it said last year") {
		t.Error("the stale block survived")
	}
}

// TestWriteAppendsWhenThereAreNoMarkers is the case init used to refuse. A
// project with an AGENTS.md written before this tool existed has no markers, so
// the block goes on the end and becomes refreshable from then on.
func TestWriteAppendsWhenThereAreNoMarkers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, instructionsFile)
	const mine = "# my own instructions\n\nRun the tests before you push.\n"
	if err := os.WriteFile(path, []byte(mine), 0o644); err != nil {
		t.Fatal(err)
	}

	got := runCLI(t, dir, nil, "instructions", "--write")
	if got.code != exitOK {
		t.Fatalf("instructions --write: %s", got.stderr)
	}

	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := mine + "\n" + instructionsText; string(body) != want {
		t.Errorf("append did not land exactly:\n--- got ---\n%s", body)
	}

	// Having appended once, the second run has markers to find and replaces
	// rather than appending a second copy.
	if got := runCLI(t, dir, nil, "instructions", "--write"); got.code != exitOK {
		t.Fatalf("second write: %s", got.stderr)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(after), instructionsBegin); n != 1 {
		t.Errorf("the file carries %d copies of the block, want 1", n)
	}
}

// TestWriteChangesNothingWhenItIsCurrent keeps a no-op out of a diff. Rewriting
// identical bytes would touch the file and show up as a change with nothing in
// it, which is how a refresh command becomes one nobody runs.
func TestWriteChangesNothingWhenItIsCurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, instructionsFile)

	if got := runCLI(t, dir, nil, "instructions", "--write"); got.code != exitOK {
		t.Fatalf("first write: %s", got.stderr)
	}
	first, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != instructionsText {
		t.Error("a created file is not the block instructions prints")
	}

	got := runCLI(t, dir, nil, "instructions", "--write")
	if got.code != exitOK {
		t.Fatalf("second write: %s", got.stderr)
	}
	if !strings.Contains(got.stdout, "already current") {
		t.Errorf("the second run should say it did nothing: %q", got.stdout)
	}

	second, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(second) != string(first) {
		t.Error("the second run rewrote the file")
	}
}

// TestWriteRefusesWhatItCannotReadHonestly covers every shape that leaves no
// answer to where the block ends. Guessing would delete prose somebody wrote,
// and a refusal costs them one edit, so this refuses and says what to fix.
func TestWriteRefusesWhatItCannotReadHonestly(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"a begin with no end", "# mine\n\n" + instructionsBegin + "\n\nsomething\n"},
		{"an end with no begin", "# mine\n\nsomething\n\n" + instructionsEnd + "\n"},
		{"the pair reversed", "# mine\n\n" + instructionsEnd + "\n\nx\n\n" + instructionsBegin + "\n"},
		{"two blocks", "# mine\n\n" + instructionsBegin + "\na\n" + instructionsEnd + "\n\n" +
			instructionsBegin + "\nb\n" + instructionsEnd + "\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, instructionsFile)
			if err := os.WriteFile(path, []byte(c.body), 0o644); err != nil {
				t.Fatal(err)
			}

			got := runCLI(t, dir, nil, "--json", "instructions", "--write")
			if got.code != exitError {
				t.Fatal("a file this shape should be refused")
			}
			if code := errCode(t, got); code != codeUsage {
				t.Errorf("code = %v, want %s", code, codeUsage)
			}

			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != c.body {
				t.Error("a refusal changed the file anyway")
			}
		})
	}
}

// TestWriteFindsTheRepositoryRoot pins where the file lands. The block
// describes the repository, so it belongs at the top of it rather than in
// whatever directory the agent happened to be standing in.
func TestWriteFindsTheRepositoryRoot(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, so there is no repository root to find")
	}
	root := t.TempDir()
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	sub := filepath.Join(root, "cmd", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	if got := runCLI(t, sub, nil, "instructions", "--write"); got.code != exitOK {
		t.Fatalf("instructions --write: %s", got.stderr)
	}

	if _, err := os.Stat(filepath.Join(root, instructionsFile)); err != nil {
		t.Errorf("the block did not land at the repository root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sub, instructionsFile)); !os.IsNotExist(err) {
		t.Error("the block landed in the working directory instead")
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
