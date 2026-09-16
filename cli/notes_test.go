package cli

import (
	"strings"
	"testing"
)

// noteTicket files a ticket carrying n notes, numbered in their text so an
// assertion can name the one it means.
func noteTicket(t *testing.T, dir string, n int) string {
	t.Helper()
	got := runCLI(t, dir, nil, "--json", "create", "--title", "A worked ticket", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create: %s%s", got.stdout, got.stderr)
	}
	id, _ := decode(t, got.stdout)["ticket"].(map[string]any)["id"].(string)
	for i := 1; i <= n; i++ {
		text := "note number " + string(rune('0'+i)) + " body"
		if r := runCLI(t, dir, nil, "note", id, text, "--actor", "human:sothr"); r.code != exitOK {
			t.Fatalf("note %d: %s%s", i, r.stdout, r.stderr)
		}
	}
	return id
}

// TestShowCompactsTheNoteHistory is the reason this exists. Reading a ticket
// should cost its current state, not everything ever written on it.
func TestShowCompactsTheNoteHistory(t *testing.T) {
	dir := newStore(t)
	id := noteTicket(t, dir, 4)

	got := runCLI(t, dir, nil, "show", id)
	if got.code != exitOK {
		t.Fatalf("show: %s%s", got.stdout, got.stderr)
	}
	// The newest note is printed whole, because it is nearly always the live one.
	if !strings.Contains(got.stdout, "note number 4 body") {
		t.Errorf("the most recent note was not printed in full:\n%s", got.stdout)
	}
	// The older ones are not.
	for _, gone := range []string{"note number 1 body", "note number 2 body", "note number 3 body"} {
		if strings.Contains(got.stdout, gone) {
			t.Errorf("%q was printed; only the newest note should be:\n%s", gone, got.stdout)
		}
	}
	// A summary a reader cannot act on is worse than the volume it replaced, so
	// the line has to carry the count, the numbers, and the way back.
	for _, want := range []string{"3 earlier notes", "1-3", "note " + id + " --show 1-3", "--list"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("the elision line is missing %q:\n%s", want, got.stdout)
		}
	}
}

// TestShowLeavesAShortHistoryAlone is the control, and it protects the common
// case. Most tickets carry one note, and making those cost a second command
// would trade a real gain on the worst tickets for a tax on the ordinary one.
func TestShowLeavesAShortHistoryAlone(t *testing.T) {
	dir := newStore(t)
	for _, n := range []int{0, 1} {
		id := noteTicket(t, dir, n)
		got := runCLI(t, dir, nil, "show", id)
		if got.code != exitOK {
			t.Fatalf("show with %d notes: %s%s", n, got.stdout, got.stderr)
		}
		if strings.Contains(got.stdout, "hidden") {
			t.Errorf("a ticket with %d notes was compacted:\n%s", n, got.stdout)
		}
		if n == 1 && !strings.Contains(got.stdout, "note number 1 body") {
			t.Errorf("the only note was not printed:\n%s", got.stdout)
		}
	}
}

// TestNoteListIndexesEveryNote covers the surface that makes a range aimable.
// Without it a reader is told to ask for note 3 with no way to learn which is 3.
func TestNoteListIndexesEveryNote(t *testing.T) {
	dir := newStore(t)
	id := noteTicket(t, dir, 3)

	got := runCLI(t, dir, nil, "note", id, "--list")
	if got.code != exitOK {
		t.Fatalf("note --list: %s%s", got.stdout, got.stderr)
	}
	for _, want := range []string{"3 notes", "human:sothr", "note number 1 body", "note number 3 body"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("the index is missing %q:\n%s", want, got.stdout)
		}
	}
	// Numbered from one, the way a person counts.
	if !strings.Contains(got.stdout, "  1  ") || !strings.Contains(got.stdout, "  3  ") {
		t.Errorf("the index is not numbered from one:\n%s", got.stdout)
	}
}

// TestNoteShowPrintsARange is the retrieval half the elision line promises.
func TestNoteShowPrintsARange(t *testing.T) {
	dir := newStore(t)
	id := noteTicket(t, dir, 4)

	one := runCLI(t, dir, nil, "note", id, "--show", "2")
	if one.code != exitOK {
		t.Fatalf("note --show 2: %s%s", one.stdout, one.stderr)
	}
	if !strings.Contains(one.stdout, "note number 2 body") {
		t.Errorf("--show 2 did not print note 2:\n%s", one.stdout)
	}
	if strings.Contains(one.stdout, "note number 3 body") {
		t.Errorf("--show 2 printed a note it was not asked for:\n%s", one.stdout)
	}

	span := runCLI(t, dir, nil, "note", id, "--show", "1-3")
	if span.code != exitOK {
		t.Fatalf("note --show 1-3: %s%s", span.stdout, span.stderr)
	}
	for _, want := range []string{"note number 1 body", "note number 2 body", "note number 3 body"} {
		if !strings.Contains(span.stdout, want) {
			t.Errorf("--show 1-3 is missing %q:\n%s", want, span.stdout)
		}
	}
	if strings.Contains(span.stdout, "note number 4 body") {
		t.Errorf("--show 1-3 printed note 4:\n%s", span.stdout)
	}

	all := runCLI(t, dir, nil, "note", id, "--show", "all")
	if all.code != exitOK {
		t.Fatalf("note --show all: %s%s", all.stdout, all.stderr)
	}
	for i, want := range []string{"note number 1 body", "note number 4 body"} {
		if !strings.Contains(all.stdout, want) {
			t.Errorf("--show all is missing %q (case %d):\n%s", want, i, all.stdout)
		}
	}
}

// TestNoteShowRefusesARangeThatIsNotThere names the bounds in the refusal,
// because "out of range" without saying the range is a second question.
func TestNoteShowRefusesARangeThatIsNotThere(t *testing.T) {
	dir := newStore(t)
	id := noteTicket(t, dir, 2)

	got := runCLI(t, dir, nil, "note", id, "--show", "9")
	if got.code == exitOK {
		t.Fatalf("--show 9 should fail on a ticket with 2 notes:\n%s", got.stdout)
	}
	if !strings.Contains(got.stderr, "1-2") {
		t.Errorf("the refusal does not name the range that exists:\n%s", got.stderr)
	}

	if bad := runCLI(t, dir, nil, "note", id, "--show", "two"); bad.code == exitOK {
		t.Errorf("--show two should fail:\n%s", bad.stdout)
	}
}

// TestNoteStillAppends is the regression guard. The reading flags are picked out
// before the writing path sees them, and the failure that split would produce is
// a note command that silently stopped writing.
func TestNoteStillAppends(t *testing.T) {
	dir := newStore(t)
	id := noteTicket(t, dir, 1)

	if got := runCLI(t, dir, nil, "note", id, "appended after the split", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("note: %s%s", got.stdout, got.stderr)
	}
	got := runCLI(t, dir, nil, "note", id, "--list")
	if !strings.Contains(got.stdout, "appended after the split") {
		t.Errorf("the appended note is not in the index:\n%s", got.stdout)
	}
	if !strings.Contains(got.stdout, "2 notes") {
		t.Errorf("want 2 notes after appending:\n%s", got.stdout)
	}
}

// TestNoteWritesTextThatLooksLikeAFlag holds the escape hatch. A caller with
// prose beginning "--list" says so with --, exactly as they already do for text
// beginning with a dash.
//
// The flags come before the --, which is not this change's rule: everything
// after -- is positional, so the pre-change binary refuses the other order too.
func TestNoteWritesTextThatLooksLikeAFlag(t *testing.T) {
	dir := newStore(t)
	id := noteTicket(t, dir, 1)

	if got := runCLI(t, dir, nil, "note", id, "--actor", "human:sothr", "--", "--list is what I typed"); got.code != exitOK {
		t.Fatalf("note -- text: %s%s", got.stdout, got.stderr)
	}
	if got := runCLI(t, dir, nil, "note", id, "--list"); !strings.Contains(got.stdout, "--list is what I typed") {
		t.Errorf("the literal text was not appended:\n%s", got.stdout)
	}
}

// TestNoteReadingHasNoJSONForm records the gap rather than inventing an
// envelope. body.notes already carries the text whole, and whether a numbered
// form belongs in the contract is a section 15 question.
func TestNoteReadingHasNoJSONForm(t *testing.T) {
	dir := newStore(t)
	id := noteTicket(t, dir, 2)

	for _, args := range [][]string{
		{"--json", "note", id, "--list"},
		{"--json", "note", id, "--show", "1"},
	} {
		got := runCLI(t, dir, nil, args...)
		if got.code == exitOK {
			t.Errorf("%v should be refused:\n%s", args, got.stdout)
		}
		if !strings.Contains(got.stderr, "body.notes") {
			t.Errorf("%v: the refusal does not say where the text already is:\n%s", args, got.stderr)
		}
	}
}

// TestNoteListAndShowAreDifferentQuestions refuses the pair rather than picking
// one, because either choice would be a guess at what the caller meant.
func TestNoteListAndShowAreDifferentQuestions(t *testing.T) {
	dir := newStore(t)
	id := noteTicket(t, dir, 2)

	if got := runCLI(t, dir, nil, "note", id, "--list", "--show", "1"); got.code == exitOK {
		t.Errorf("--list with --show should be refused:\n%s", got.stdout)
	}
}

// TestNoteHelpDocumentsTheReadingForms is the whole point of
// TKT-01M2HVRH9X5S7EMRNK5Z7FWMCP. The only place --list and --show were
// written down was the stand-in line `show` prints when a ticket has more than
// one note, so a reader who met `note --help` first was told that note appends
// and nothing else. They cannot come from the FlagSet walk, because
// noteReadArgs takes them out before any FlagSet exists, so this holds the
// hand-written forms to naming them.
func TestNoteHelpDocumentsTheReadingForms(t *testing.T) {
	// A bare directory, because asking what a command takes must not need a
	// store, the same rule TestSubcommandHelp holds every other command to.
	dir := t.TempDir()

	got := runCLI(t, dir, nil, "note", "--help")
	if got.code != exitOK {
		t.Fatalf("note --help exited %d: %s", got.code, got.stderr)
	}
	for _, want := range []string{
		"usage: git ticket note ID TEXT",
		"git ticket note ID --list",
		"git ticket note ID --show N | N-M | all",
	} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("note --help omits %q:\n%s", want, got.stdout)
		}
	}
	// The epilogue says why the two are absent from the flag list, so a reader
	// who notices is given the reason rather than a stale-looking page.
	if !strings.Contains(got.stdout, "before the flags above are parsed") {
		t.Errorf("note --help does not say why the reading flags are not listed:\n%s", got.stdout)
	}
}

// TestNoteReadingTakesGlobalsAfterTheCommand holds the reading forms to the
// promise the top-level usage makes for every command: a global may come before
// or after the command name.
//
// They did not. noteReadArgs hands everything it did not claim to runNoteRead,
// which counted the words and refused anything but one, so `note ID --list
// --store PATH` failed as though two tickets had been named. Parsing what was
// left behind fixed it without touching the split that keeps a stray --list out
// of a note's text, which TestNoteWritesTextThatLooksLikeAFlag still proves.
func TestNoteReadingTakesGlobalsAfterTheCommand(t *testing.T) {
	dir := newStore(t)
	id := noteTicket(t, dir, 2)

	for _, args := range [][]string{
		{"note", id, "--list", "--actor", "human:sothr"},
		{"note", id, "--show", "1", "--actor", "human:sothr"},
		{"note", id, "--list", "--lock-timeout", "5s"},
	} {
		got := runCLI(t, dir, nil, args...)
		if got.code != exitOK {
			t.Errorf("%v exited %d, want 0: %s", args, got.code, got.stderr)
		}
	}

	// Two tickets are still two tickets. The parse must not have turned the
	// count check into something that accepts anything.
	if got := runCLI(t, dir, nil, "note", id, id, "--list"); got.code == exitOK {
		t.Errorf("two IDs should still be refused:\n%s", got.stdout)
	}
}
