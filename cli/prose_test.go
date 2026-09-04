package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// runCLIStdin is runCLI with something on stdin, for the `-` path of plan 12.1.
func runCLIStdin(t *testing.T, dir, stdin string, args ...string) result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, Env{
		Dir:    dir,
		Getenv: func(string) string { return "" },
		Stdout: &stdout,
		Stderr: &stderr,
		Stdin:  strings.NewReader(stdin),
		Now:    func() time.Time { return referenceInstant },
	})
	return result{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

// writeFile drops one prose file into the store directory and returns its path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
	return path
}

// sectionOf reads one section out of a ticket's JSON body as text.
func sectionOf(t *testing.T, dir, id, section string) string {
	t.Helper()
	text, _ := bodyOf(t, dir, id)[section].(string)
	return text
}

// hazard is the text TKT-01M1PCXW was filed about: an apostrophe ends a
// single-quoted shell string and a backtick inside a double-quoted one runs a
// command. Passing it through a file means neither is the caller's problem.
const hazard = "The shell isn't the authoring surface. A backtick `date` and a\n" +
	"double quote \" all survive, because nothing here is parsed by a shell."

// TestCreateReadsProseFromFiles covers --description-file and --plan-file, the
// two sections create takes, per plan 12.1.
func TestCreateReadsProseFromFiles(t *testing.T) {
	dir := newStore(t)
	desc := writeFile(t, dir, "desc.md", hazard+"\n")
	plan := writeFile(t, dir, "plan.md", "Step one.\nStep two.\n")

	got := runCLI(t, dir, nil, "--json", "create", "--title", "From files",
		"--description-file", desc, "--plan-file", plan, "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create failed: %s%s", got.stdout, got.stderr)
	}
	id := decode(t, got.stdout)["ticket"].(map[string]any)["id"].(string)

	if want, got := hazard, sectionOf(t, dir, id, "description"); got != want {
		t.Errorf("description:\n got %q\nwant %q", got, want)
	}
	if want, got := "Step one.\nStep two.", sectionOf(t, dir, id, "implementationPlan"); got != want {
		t.Errorf("implementation plan:\n got %q\nwant %q", got, want)
	}
}

// TestProseFileTrimsTrailingNewline holds the rule in 12.1 that a final newline
// terminates a text file's last line rather than being content. printf and a
// heredoc disagree about whether to emit one, so a section written both ways
// has to land identically.
//
// Two layers do this and the test covers both on purpose. parseBody already
// drops the blank lines at the end of a section, so the newline cases pass with
// readProse trimming nothing; they are here to catch that going away. The
// trailing-space case is the one only readProse handles, and removing the
// TrimRight fails it alone.
func TestProseFileTrimsTrailingNewline(t *testing.T) {
	dir := newStore(t)
	for _, tc := range []struct {
		name    string
		content string
	}{
		{"no newline", "Just the prose."},
		{"one newline", "Just the prose.\n"},
		{"several", "Just the prose.\n\n\n"},
		{"trailing spaces", "Just the prose.  \n\t\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := writeFile(t, dir, "d.md", tc.content)
			got := runCLI(t, dir, nil, "--json", "create", "--title", "T",
				"--description-file", f, "--actor", "human:sothr")
			if got.code != exitOK {
				t.Fatalf("create failed: %s%s", got.stdout, got.stderr)
			}
			id := decode(t, got.stdout)["ticket"].(map[string]any)["id"].(string)
			if want, got := "Just the prose.", sectionOf(t, dir, id, "description"); got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	}
}

// TestProseFileKeepsIndentation is the other half of the trimming rule. Only
// trailing whitespace goes, because indentation inside the prose is content: a
// fenced code block is the case that would break.
func TestProseFileKeepsIndentation(t *testing.T) {
	dir := newStore(t)
	const content = "Here is the call:\n\n```go\nif err != nil {\n\treturn err\n}\n```\n"
	f := writeFile(t, dir, "d.md", content)

	got := runCLI(t, dir, nil, "--json", "create", "--title", "T",
		"--description-file", f, "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create failed: %s%s", got.stdout, got.stderr)
	}
	id := decode(t, got.stdout)["ticket"].(map[string]any)["id"].(string)

	want := strings.TrimRight(content, "\n")
	if got := sectionOf(t, dir, id, "description"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestProseFromStdin covers the `-` path on both flag shapes: the named sibling
// on create and the single --file on a command whose text is a positional.
func TestProseFromStdin(t *testing.T) {
	dir := newStore(t)

	got := runCLIStdin(t, dir, hazard+"\n", "--json", "create", "--title", "T",
		"--description-file", "-", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create failed: %s%s", got.stdout, got.stderr)
	}
	id := decode(t, got.stdout)["ticket"].(map[string]any)["id"].(string)
	if want, got := hazard, sectionOf(t, dir, id, "description"); got != want {
		t.Errorf("description from stdin:\n got %q\nwant %q", got, want)
	}

	got = runCLIStdin(t, dir, "Where it landed.\n", "summary", id, "--file", "-",
		"--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("summary failed: %s%s", got.stdout, got.stderr)
	}
	if want, got := "Where it landed.", sectionOf(t, dir, id, "summary"); got != want {
		t.Errorf("summary from stdin:\n got %q\nwant %q", got, want)
	}
}

// TestTextEntryCommandsTakeAFile covers the four commands of 12.1 that take
// their text as a positional. Each one gets one --file, since there is only one
// section it could fill.
func TestTextEntryCommandsTakeAFile(t *testing.T) {
	dir := newStore(t)
	id := createTicket(t, dir)["ticket"].(map[string]any)["id"].(string)
	f := writeFile(t, dir, "text.md", hazard+"\n")

	for _, tc := range []struct {
		command string
		section string
	}{
		{"plan", "implementationPlan"},
		{"summary", "summary"},
		{"note", "notes"},
		{"comment", "comments"},
	} {
		t.Run(tc.command, func(t *testing.T) {
			got := runCLI(t, dir, nil, tc.command, id, "--file", f, "--actor", "human:sothr")
			if got.code != exitOK {
				t.Fatalf("%s failed: %s%s", tc.command, got.stdout, got.stderr)
			}
			// note and comment append under an attribution line, so the text is
			// contained rather than equal.
			if body := sectionOf(t, dir, id, tc.section); !strings.Contains(body, hazard) {
				t.Errorf("%s did not store the file's text, got %q", tc.command, body)
			}
		})
	}
}

// TestTextEntryFileWorksBeforeTheID guards the parseFlags loop of 12.1, which
// consumes one positional per pass so a flag can sit on either side of the ID.
func TestTextEntryFileWorksBeforeTheID(t *testing.T) {
	dir := newStore(t)
	id := createTicket(t, dir)["ticket"].(map[string]any)["id"].(string)
	f := writeFile(t, dir, "p.md", "Ordered before the ID.\n")

	got := runCLI(t, dir, nil, "plan", "--file", f, id, "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("plan failed: %s%s", got.stdout, got.stderr)
	}
	if want, got := "Ordered before the ID.", sectionOf(t, dir, id, "implementationPlan"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestUpdateReadsDescriptionFromFile covers update's sibling flag, and holds the
// reported field list to the write. The two flags are one field, so a
// --description-file write has to report "description" the way --description
// does.
func TestUpdateReadsDescriptionFromFile(t *testing.T) {
	dir := newStore(t)
	id := createTicket(t, dir)["ticket"].(map[string]any)["id"].(string)
	f := writeFile(t, dir, "d.md", hazard+"\n")

	got := runCLI(t, dir, nil, "update", id, "--description-file", f, "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("update failed: %s%s", got.stdout, got.stderr)
	}
	if !strings.Contains(got.stdout, "description") {
		t.Errorf("update did not report the changed field: %q", got.stdout)
	}
	if want, got := hazard, sectionOf(t, dir, id, "description"); got != want {
		t.Errorf("description:\n got %q\nwant %q", got, want)
	}
}

// TestProseRefusesBothSpellings holds the rule in 12.1 that a section handed
// both an inline value and a file is a usage error naming both. A caller who
// typed two meant one, and resolving by precedence writes something nobody
// asked for.
func TestProseRefusesBothSpellings(t *testing.T) {
	dir := newStore(t)
	id := createTicket(t, dir)["ticket"].(map[string]any)["id"].(string)
	f := writeFile(t, dir, "d.md", "from the file\n")

	for _, tc := range []struct {
		name string
		args []string
		want []string
	}{
		{
			"create description",
			[]string{"create", "--title", "T", "--description", "inline", "--description-file", f},
			[]string{"--description", "--description-file"},
		},
		{
			"create plan",
			[]string{"create", "--title", "T", "--plan", "inline", "--plan-file", f},
			[]string{"--plan", "--plan-file"},
		},
		{
			"update description",
			[]string{"update", id, "--description", "inline", "--description-file", f},
			[]string{"--description", "--description-file"},
		},
		{
			"note text and file",
			[]string{"note", id, "inline", "--file", f},
			[]string{"--file"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := runCLI(t, dir, nil, append(tc.args, "--actor", "human:sothr")...)
			if got.code != exitError {
				t.Fatalf("expected a refusal, got code %d: %s%s", got.code, got.stdout, got.stderr)
			}
			for _, want := range tc.want {
				if !strings.Contains(got.stderr, want) {
					t.Errorf("the refusal does not name %s: %q", want, got.stderr)
				}
			}
			if !strings.Contains(got.stderr, "not both") {
				t.Errorf("the refusal does not say both were given: %q", got.stderr)
			}
		})
	}
}

// TestProseRefusesTwoReadsOfStdin holds the other group rule of 12.1. Stdin is a
// stream and the second read returns nothing, so a command asking for it twice
// cannot be given what the caller meant.
func TestProseRefusesTwoReadsOfStdin(t *testing.T) {
	dir := newStore(t)
	got := runCLIStdin(t, dir, "some prose\n", "create", "--title", "T",
		"--description-file", "-", "--plan-file", "-", "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatalf("expected a refusal, got code %d: %s%s", got.code, got.stdout, got.stderr)
	}
	for _, want := range []string{"--description-file", "--plan-file", "stdin"} {
		if !strings.Contains(got.stderr, want) {
			t.Errorf("the refusal does not name %s: %q", want, got.stderr)
		}
	}
	// Nothing was written, so the store still holds no ticket.
	if list := runCLI(t, dir, nil, "--json", "list", "--all"); strings.Contains(list.stdout, "\"id\"") {
		t.Errorf("the refused create wrote a ticket anyway: %s", list.stdout)
	}
}

// TestProseFileMissingIsAUsageError covers the code choice in 12.1: section 10
// lists no code for a failed read, and a path the caller can correct is what
// `usage` covers. The message names the flag, since the os error already
// carries the path and the reason.
func TestProseFileMissingIsAUsageError(t *testing.T) {
	dir := newStore(t)
	got := runCLI(t, dir, nil, "--json", "create", "--title", "T",
		"--description-file", filepath.Join(dir, "absent.md"), "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatalf("expected a refusal, got code %d: %s%s", got.code, got.stdout, got.stderr)
	}
	env := decode(t, got.stdout)
	errObj, ok := env["error"].(map[string]any)
	if !ok {
		t.Fatalf("no error in the envelope: %s", got.stdout)
	}
	if code, _ := errObj["code"].(string); code != codeUsage {
		t.Errorf("code is %q, want %q", code, codeUsage)
	}
	if msg, _ := errObj["message"].(string); !strings.Contains(msg, "--description-file") {
		t.Errorf("the message does not name the flag: %q", msg)
	}
}

// TestHeadingWarningNamesTheFileFlag holds the warning to the flag that carried
// the text. A file is where somebody writes enough prose to want subheadings,
// so the trap is likelier here, and a warning naming --description would send
// the reader to a flag they did not type.
func TestHeadingWarningNamesTheFileFlag(t *testing.T) {
	dir := newStore(t)
	id := createTicket(t, dir)["ticket"].(map[string]any)["id"].(string)
	f := writeFile(t, dir, "bad.md", "Intro prose.\n\n## Risks\n\nA second section.\n")

	got := runCLI(t, dir, nil, "create", "--title", "T", "--description-file", f,
		"--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create failed: %s%s", got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, "--description-file") {
		t.Errorf("the warning does not name --description-file: %q", got.stderr)
	}

	got = runCLI(t, dir, nil, "note", id, "--file", f, "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("note failed: %s%s", got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, "--file") {
		t.Errorf("the warning does not name --file: %q", got.stderr)
	}

	// A positional has no flag to name, so the command is what the warning
	// says. This is the behaviour that shipped, and it must not move.
	got = runCLI(t, dir, nil, "note", id, "Intro.\n\n## Risks\n\ninline",
		"--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("note failed: %s%s", got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, "the text for note") {
		t.Errorf("the warning does not name the command: %q", got.stderr)
	}
}

// TestTextEntryStillNeedsText holds the refusal for a command given neither a
// positional nor a file, so --file did not turn a required argument into an
// optional one.
func TestTextEntryStillNeedsText(t *testing.T) {
	dir := newStore(t)
	id := createTicket(t, dir)["ticket"].(map[string]any)["id"].(string)

	got := runCLI(t, dir, nil, "note", id, "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatalf("expected a refusal, got code %d: %s%s", got.code, got.stdout, got.stderr)
	}
	if !strings.Contains(got.stderr, "--file") {
		t.Errorf("the refusal does not offer --file: %q", got.stderr)
	}
}
