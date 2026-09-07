package ticket

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestClaimRefusalNamesTheTransition covers terva's third handoff, finding 1.
// The refusal already named the status it refused on, and every word of it was
// true. What it did not name was the move that clears it, so an agent that had
// just filed a ticket had to find `status ready` somewhere other than the
// message. Each refused status wants a different answer, which is why this is a
// table and not one appended sentence: a draft is promoted to ready, a done
// ticket reopens to in-progress, and an archived one comes back through
// unarchive. The notWant column is the point of the table. It fails the build if
// somebody later collapses these into a single sentence that names ready to
// every caller.
//
// The remedy column matters more than the strings. Checking only that a message
// names a next step proves the sentence exists, not that following it works.
// The first version of this change told a done ticket to run status in-progress
// and stopped there, and a real binary answered that with a second refusal,
// because reopening from done needs a reason. Every case here now performs what
// its message describes and claims afterwards.
func TestClaimRefusalNamesTheTransition(t *testing.T) {
	cases := []struct {
		status  string
		want    string
		notWant string
		remedy  func(t *testing.T, s *Store, id string)
	}{
		{StatusDraft, "status ID ready", "in-progress", func(t *testing.T, s *Store, id string) {
			mustApply(t, s, id, SetStatus{Status: StatusReady})
		}},
		{StatusDone, "status ID in-progress --reason R", "unarchive", func(t *testing.T, s *Store, id string) {
			mustApply(t, s, id, SetStatus{Status: StatusInProgress, Reason: "reopened in a test"})
		}},
		{StatusArchived, "unarchive ID", "status ID ready", func(t *testing.T, s *Store, id string) {
			mustApply(t, s, id, UnarchiveTicket{})
		}},
	}
	for _, c := range cases {
		t.Run(c.status, func(t *testing.T) {
			s := newTestStore(t)
			tk := mustCreate(t, s, "Rotate the signing key before it expires")
			// Each status is reached the way the store actually reaches it.
			// SetStatus refuses to archive, with a refusal that names its own
			// remedy: "archiving also moves the file; use archive".
			switch c.status {
			case StatusDraft:
				// create leaves it here.
			case StatusArchived:
				mustApply(t, s, tk.ID, ArchiveTicket{Reason: "swept out of the working set in a test"})
			default:
				mustApply(t, s, tk.ID, SetStatus{
					Status: c.status,
					Reason: "closed in a test, which draft to done requires",
				})
			}

			_, err := s.Apply(context.Background(), tk.ID, ClaimTicket{}, ApplyOptions{Actor: testActor})
			if CodeOf(err) != CodeValidationFailed {
				t.Fatalf("claiming a %s ticket = %v, want %s", c.status, err, CodeValidationFailed)
			}
			msg := err.Error()
			if !strings.Contains(msg, c.status) {
				t.Errorf("the message no longer names the status it refused on: %s", msg)
			}
			if !strings.Contains(msg, c.want) {
				t.Errorf("the message does not name the remedy %q: %s", c.want, msg)
			}
			if strings.Contains(msg, c.notWant) {
				t.Errorf("the message offers %q, which does not apply to a %s ticket: %s",
					c.notWant, c.status, msg)
			}

			// And the advice works. A remedy that lands the reader on a second
			// refusal has not saved them the trip it exists to save.
			c.remedy(t, s, tk.ID)
			if _, err := s.Apply(context.Background(), tk.ID, ClaimTicket{}, ApplyOptions{Actor: testActor}); err != nil {
				t.Errorf("following the remedy for a %s ticket still refused the claim: %v", c.status, err)
			}
		})
	}
}

// TestReopeningFromDoneNeedsAReason pins the constraint that makes the done
// remedy read "--reason R" while the draft one does not. Without this, a later
// tidy-up that makes all three remedies look alike would drop the flag and the
// message would send its reader into a second refusal again.
func TestReopeningFromDoneNeedsAReason(t *testing.T) {
	s := newTestStore(t)
	tk := mustCreate(t, s, "Rotate the signing key before it expires")
	mustApply(t, s, tk.ID, SetStatus{Status: StatusDone, Reason: "shipped in a test"})

	_, err := s.Apply(context.Background(), tk.ID,
		SetStatus{Status: StatusInProgress}, ApplyOptions{Actor: testActor})
	if err == nil {
		t.Fatal("reopening from done with no reason was allowed, so the claim remedy should stop naming --reason R")
	}
}

// TestSchemaUnsupportedNamesTheUpgrade covers terva's third handoff, finding 2.
// Two sites raise the code, config.yml and a ticket file, and terva measured
// only the first, so this covers both.
//
// The forbidden words are the substance. Every neighbouring schema refusal says
// to run migrate, because there the store is behind the reader. This one fires
// when the store is ahead, where migrate and check --fix are equally useless,
// and terva reported a reader reaching for check --fix against a sentence that
// was true and named no next step.
func TestSchemaUnsupportedNamesTheUpgrade(t *testing.T) {
	namesTheUpgrade := func(t *testing.T, where string, err error) {
		t.Helper()
		if CodeOf(err) != CodeSchemaUnsupported {
			t.Fatalf("%s = %v, want %s", where, err, CodeSchemaUnsupported)
		}
		msg := err.Error()
		if !strings.Contains(msg, "upgrade git-ticket") {
			t.Errorf("%s names no remedy: %s", where, msg)
		}
		if !strings.Contains(msg, fmt.Sprintf("%d", SchemaVersion)) {
			t.Errorf("%s no longer names the level this reader supports: %s", where, msg)
		}
		for _, wrong := range []string{"migrate", "check --fix"} {
			if strings.Contains(msg, wrong) {
				t.Errorf("%s offers %q, which cannot clear a store that is ahead: %s",
					where, wrong, msg)
			}
		}
	}

	t.Run("config.yml", func(t *testing.T) {
		s := newTestStore(t)
		cfg := s.Config()
		cfg.Schema = SchemaVersion + 1
		if err := os.WriteFile(filepath.Join(s.Path(), "config.yml"), RenderConfig(cfg), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := OpenWith(s.Path(), OpenOptions{Now: fixedClock(), NoRoot: true})
		namesTheUpgrade(t, "opening a store one level ahead", err)
	})

	t.Run("ticket file", func(t *testing.T) {
		s := newTestStore(t)
		tk := mustCreate(t, s, "Rotate the signing key before it expires")

		files, err := filepath.Glob(filepath.Join(s.Path(), "draft", "*.md"))
		if err != nil || len(files) != 1 {
			t.Fatalf("want one draft file, got %v (%v)", files, err)
		}
		data, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatal(err)
		}
		// Raising the level relative to SchemaVersion rather than writing a
		// number, and failing when the replacement matched nothing. A
		// hard-coded pair stops matching the next time the schema moves, and a
		// silent no-op leaves this passing against a file it never raised.
		ahead := strings.Replace(string(data),
			fmt.Sprintf("schema: %d", SchemaVersion),
			fmt.Sprintf("schema: %d", SchemaVersion+1), 1)
		if ahead == string(data) {
			t.Fatalf("no schema %d line to raise, so the file was left readable", SchemaVersion)
		}
		if err := os.WriteFile(files[0], []byte(ahead), 0o644); err != nil {
			t.Fatal(err)
		}

		_, err = s.Get(context.Background(), tk.ID)
		namesTheUpgrade(t, "reading a ticket one level ahead", err)
	})
}
