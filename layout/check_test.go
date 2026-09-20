package layout

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeBoardFile(t *testing.T, store, name, content string) {
	t.Helper()
	dir := filepath.Join(store, DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const canonicalBoard = `# git-ticket canvas layout. Positions and frames; tickets live in their own files.
# One line per card, sorted by ticket ID, so a drag is a one-line diff.
schema: 3
board: "default"
cards:
  "TKT-01K3ZZ2JH000GHB4EE6SNRE6MD": {x: 120, y: -40}
frames:
  "f1": {title: "Auth", x: 0, y: 0, w: 400, h: 300, color: "#759bcc", members: [TKT-01K3ZZ2JH000GHB4EE6SNRE6MD]}
pens:
  "fe": {title: "Frontend", x: 0, y: 0, w: 600, h: 400, color: "#759bcc", pin: {x: 0, y: 0}, requiredLabels: ["frontend"]}
ruleOrder: ["fe"]
inbox: {x: -300, y: 0}
`

// Modify is the CLI's one way to write, and its promise is that a refused
// write leaves the file as it was: validation runs before the rename, not
// after a partial write.
func TestModifyRefusesBeforeWriting(t *testing.T) {
	store := t.TempDir()
	writeBoardFile(t, store, "default.yml", canonicalBoard)
	s := New(store)
	_, err := s.Modify("default", func(b *Board) error {
		b.RuleOrder = []string{"fe", "fe"}
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "ruleOrder") {
		t.Fatalf("Modify accepted a board validation refuses: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(store, DirName, "default.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != canonicalBoard {
		t.Fatalf("a refused Modify changed the file:\n%s", got)
	}

	b, err := s.Modify("default", func(b *Board) error {
		b.Inbox = &Point{X: 1.234, Y: 5}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if b.Inbox.X != 1.23 {
		t.Fatalf("Modify did not normalise: inbox x = %v", b.Inbox.X)
	}
	got, _ = os.ReadFile(filepath.Join(store, DirName, "default.yml"))
	if !strings.Contains(string(got), "inbox: {x: 1.23, y: 5}") {
		t.Fatalf("Modify did not write:\n%s", got)
	}
}

func kinds(ps []Problem) []ProblemKind {
	out := make([]ProblemKind, 0, len(ps))
	for _, p := range ps {
		out = append(out, p.Kind)
	}
	return out
}

func TestCheckIsSilentOnACanonicalBoardAndAMissingDirectory(t *testing.T) {
	store := t.TempDir()
	yes := func(string) bool { return true }
	got, err := Check(store, yes, yes)
	if err != nil || len(got) != 0 {
		t.Fatalf("no canvas directory: %v, %v", got, err)
	}
	writeBoardFile(t, store, "default.yml", canonicalBoard)
	writeBoardFile(t, store, ".default.123.tmp", "half a save")
	writeBoardFile(t, store, "notes.txt", "not a board")
	got, err = Check(store, yes, yes)
	if err != nil || len(got) != 0 {
		t.Fatalf("canonical board with stray files: %v, %v", got, err)
	}
}

func TestCheckReportsEachProblemAgainstItsRecord(t *testing.T) {
	store := t.TempDir()
	writeBoardFile(t, store, "default.yml", canonicalBoard)
	writeBoardFile(t, store, "bad name.yml", canonicalBoard)
	writeBoardFile(t, store, "broken.yml", "schema: 3\nboard: \"broken\"\ncards: []\n")
	no := func(string) bool { return false }
	got, err := Check(store, no, no)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		kind  ProblemKind
		file  string
		field string
	}{
		{Invalid, "canvas/bad name.yml", ""},
		{Invalid, "canvas/broken.yml", ""},
		{TicketMissing, "canvas/default.yml", "cards.TKT-01K3ZZ2JH000GHB4EE6SNRE6MD"},
		{TicketMissing, "canvas/default.yml", "frames.f1.members"},
		{LabelUnknown, "canvas/default.yml", "pens.fe.requiredLabels"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d problems %v, want %d", len(got), got, len(want))
	}
	for i, w := range want {
		if got[i].Kind != w.kind || got[i].File != w.file || got[i].Field != w.field {
			t.Errorf("problem %d = %+v, want %+v", i, got[i], w)
		}
	}
}

// A board that is valid but not in the form a save writes is reported with the
// bytes a save would write, which is the whole of the repair. Ordering, an
// old schema, a comment, and float noise are each that condition.
func TestCheckReportsANonCanonicalBoardWithItsCanonicalBytes(t *testing.T) {
	store := t.TempDir()
	writeBoardFile(t, store, "default.yml", `schema: 2
board: default
# somebody's comment
cards:
  TKT-01K3ZZ2JH000GHB4EE6SNRE6MD: {x: 120.004, y: -40}
frames: {}
`)
	yes := func(string) bool { return true }
	got, err := Check(store, yes, yes)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Kind != NotCanonical {
		t.Fatalf("got %v, want one NotCanonical", kinds(got))
	}
	canonical := string(got[0].Canonical)
	for _, line := range []string{"schema: 3\n", `  "TKT-01K3ZZ2JH000GHB4EE6SNRE6MD": {x: 120, y: -40}` + "\n", "pens: {}\n", "inbox: {x: 0, y: 0}\n"} {
		if !strings.Contains(canonical, line) {
			t.Errorf("canonical bytes lack %q:\n%s", line, canonical)
		}
	}
	if strings.Contains(canonical, "comment") {
		t.Errorf("canonical bytes kept the comment:\n%s", canonical)
	}
	// Writing the canonical bytes back is a fixed point.
	writeBoardFile(t, store, "default.yml", canonical)
	got, err = Check(store, yes, yes)
	if err != nil || len(got) != 0 {
		t.Fatalf("the canonical bytes are not canonical: %v, %v", got, err)
	}
}

// Two Stores over one directory stand in for two processes: a canvas and a
// CLI, or two CLIs. The file lock is per open file, so the second Store's
// write waits on the first's and times out rather than renaming over it, and
// goes through once the first is done.
func TestWritersInSeparateStoresWaitOnTheFileLock(t *testing.T) {
	store := t.TempDir()
	writeBoardFile(t, store, "default.yml", canonicalBoard)
	a, b := New(store), New(store)
	b.SetLockTimeout(150 * time.Millisecond)

	hold := make(chan struct{})
	entered := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := a.Modify("default", func(bd *Board) error {
			close(entered)
			<-hold
			bd.Inbox = &Point{X: 1, Y: 1}
			return nil
		})
		done <- err
	}()
	<-entered
	_, err := b.Modify("default", func(bd *Board) error { bd.Inbox = &Point{X: 2, Y: 2}; return nil })
	if !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("second writer got %v, want a lock timeout while the first holds the file", err)
	}
	close(hold)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	got, err := b.Modify("default", func(bd *Board) error { bd.Inbox = &Point{X: 2, Y: 2}; return nil })
	if err != nil || got.Inbox.X != 2 {
		t.Fatalf("after release: %v, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(store, DirName, ".lock")); err != nil {
		t.Fatalf("outside a repository the lock is a dot-file in the canvas directory: %v", err)
	}
	if problems, _ := Check(store, func(string) bool { return true }, func(string) bool { return true }); len(problems) != 0 {
		t.Fatalf("the lock file is reported by Check: %v", problems)
	}
}
