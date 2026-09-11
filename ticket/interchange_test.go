package ticket

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestBlobSHAMatchesGit checks the index line against git's own hashing. A wrong
// blob name still applies, but it takes away the object git needs for a --3way
// fallback, and the failure would only appear on a patch that conflicts.
//
// It asks git through `hash-object --stdin`, which needs no repository, so the
// check is about the hashing rule and nothing else.
func TestBlobSHAMatchesGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	body := []byte("---\nid: TKT-1\n---\n\n## Description\n\nhi\n")

	cmd := exec.Command("git", "hash-object", "--stdin")
	cmd.Stdin = bytes.NewReader(body)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git hash-object: %v", err)
	}
	want := strings.TrimSpace(string(out))

	if got := BlobSHA(body); got != want {
		t.Errorf("BlobSHA = %s, want %s", got, want)
	}
}

// TestAddedFileHunkRoundTrips is the pair the wire format has to satisfy: what
// one side writes, the other reads back unchanged.
//
// The blob check inside ParseAddedFiles makes this stronger than it looks. A
// hunk whose body drifted from its index line fails there rather than here, so
// this passing means the two halves agree on the bytes and on the name of the
// bytes.
func TestAddedFileHunkRoundTrips(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"trailing newline", "---\nid: TKT-1\n---\n\n## Description\n\nhi\n"},
		{"blank lines inside", "---\nid: TKT-3\n---\n\n## Description\n\none\n\ntwo\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			hunk, lines := AddedFileHunk(".tickets/tickets/TKT-1.md", []byte(c.body))
			if lines == 0 {
				t.Fatal("no lines counted")
			}
			got, err := ParseAddedFiles(hunk)
			if err != nil {
				t.Fatalf("ParseAddedFiles: %v", err)
			}
			if len(got) != 1 {
				t.Fatalf("got %d files, want 1", len(got))
			}
			if got[0].Path != ".tickets/tickets/TKT-1.md" {
				t.Errorf("path = %q", got[0].Path)
			}
			if got[0].Body != c.body {
				t.Errorf("body = %q, want %q", got[0].Body, c.body)
			}
		})
	}
}

// TestAFileWithNoTrailingNewlineDoesNotRoundTrip pins a defect rather than a
// design, and it is pinned because it is worth somebody's attention.
//
// AddedFileHunk writes git's "\ No newline at end of file" marker and hashes
// the bytes as they are. ParseAddedFiles skips that marker and appends a
// newline to every line it reads, so it rebuilds a file one byte longer than
// the one that went out, hashes that, and reports a mismatch. Export writes an
// artifact its own import refuses, and the error tells the receiver the patch
// was altered or truncated when nobody touched it.
//
// It is latent because Render always ends a ticket file with a newline, so no
// file the store writes takes this path. That is why this is a test and not a
// fix in the same commit: the fix belongs with whoever decides whether the
// parser should honour the marker or the writer should refuse such a file, and
// changing the wire format inside a move that must be byte-identical is exactly
// the wrong moment.
func TestAFileWithNoTrailingNewlineDoesNotRoundTrip(t *testing.T) {
	hunk, _ := AddedFileHunk("a.md", []byte("one\ntwo"))
	if !strings.Contains(hunk, "\\ No newline at end of file") {
		t.Fatal("the hunk does not carry git's marker, so this test is no longer about what it says")
	}

	_, err := ParseAddedFiles(hunk)
	if err == nil {
		t.Fatal("the round trip now works; delete this test and tick the defect off")
	}
	if !strings.Contains(err.Error(), "does not match its blob name") {
		t.Errorf("error = %v, want the blob mismatch this defect produces", err)
	}
}

// TestParseAddedFilesRefusesAnAlteredPatch is the reason the index line is
// computed rather than left blank. A patch somebody edited by hand, or one that
// arrived truncated, must not become a ticket that looks fine.
func TestParseAddedFilesRefusesAnAlteredPatch(t *testing.T) {
	hunk, _ := AddedFileHunk("a.md", []byte("one\ntwo\n"))
	altered := strings.Replace(hunk, "+two", "+three", 1)

	_, err := ParseAddedFiles(altered)
	if err == nil {
		t.Fatal("an altered body parsed without complaint")
	}
	if !strings.Contains(err.Error(), "does not match its blob name") {
		t.Errorf("error = %v, want it to name the blob mismatch", err)
	}
}

// TestMboxMessageOpensWhereMailsplitLooks keeps the framing git needs. The From
// line and the --- separator are what turn this from text into something `git
// am` will take.
func TestMboxMessageOpensWhereMailsplitLooks(t *testing.T) {
	when := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	got := MboxMessage("A Sender <a@example.com>", when, "A subject", "A body", "a diff\n")

	if !strings.HasPrefix(got, MboxFromLine+"\n") {
		t.Errorf("message does not open with the From line:\n%s", got)
	}
	for _, want := range []string{
		"From: A Sender <a@example.com>\n",
		"Subject: [PATCH] A subject\n",
		"\n---\na diff\n",
		"-- \ngit-ticket\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("message is missing %q:\n%s", want, got)
		}
	}
}

// TestMboxMessageFlattensASubject keeps a multi-line title out of the headers.
// A newline in a Subject: is a header break, so the rest of the message would
// arrive as body and the patch would not apply.
func TestMboxMessageFlattensASubject(t *testing.T) {
	got := MboxMessage("x <x@example.com>", time.Now(), "first\nsecond", "body", "")
	if !strings.Contains(got, "Subject: [PATCH] first second\n") {
		t.Errorf("subject was not flattened:\n%s", got)
	}
}
