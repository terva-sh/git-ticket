package ticket

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
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

// TestAFileRoundTripsWhateverItsLastByte holds the two halves of the marker to
// each other, and it needs both rows to say anything.
//
// AddedFileHunk writes git's "\ No newline at end of file" marker and hashes
// the bytes as they are, so ParseAddedFiles has to honour it. A parser that
// appends a newline to every line it reads rebuilds a file one byte longer
// than the one that went out, and the blob check then fires on an artifact
// nobody touched and tells the receiver it was altered or truncated. That was
// TKT-01M29F9KSAEKDG85KD73X54S04.
//
// The trailing-newline row is the control. A parser that trimmed the last byte
// unconditionally would pass the first row and break every real export, since
// Render ends a ticket file with a newline and so every file the store writes
// is the second row.
func TestAFileRoundTripsWhateverItsLastByte(t *testing.T) {
	for _, tc := range []struct {
		name       string
		data       string
		wantMarker bool
	}{
		{"no trailing newline", "one\ntwo", true},
		{"trailing newline", "one\ntwo\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hunk, _ := AddedFileHunk("a.md", []byte(tc.data))
			if got := strings.Contains(hunk, "\\ No newline at end of file"); got != tc.wantMarker {
				t.Fatalf("hunk carries the marker = %v, want %v; this test is no longer about what it says", got, tc.wantMarker)
			}

			files, err := ParseAddedFiles(hunk)
			if err != nil {
				t.Fatalf("ParseAddedFiles: %v", err)
			}
			if len(files) != 1 {
				t.Fatalf("got %d files, want 1", len(files))
			}
			// The blob name is a hash of the exact bytes, so the parse above
			// already caught any difference. Compare the bytes anyway: a
			// failure that names the length says what went wrong, and two
			// twelve-character hashes do not.
			if files[0].Body != tc.data {
				t.Errorf("body = %q (%d bytes), want %q (%d bytes)",
					files[0].Body, len(files[0].Body), tc.data, len(tc.data))
			}
		})
	}
}

// TestAHunkWithNoTrailingNewlineAppliesWithRealGit is the claim that decided the
// repair, put to the only authority that can settle it.
//
// Honouring the marker was chosen over refusing the input partly because it
// keeps the artifact byte-compatible with git, and plan 12.8 promises a
// receiver can land an export with `git am` alone. Our own parser agreeing with
// our own writer cannot show that: both halves could share a misreading of the
// format and round-trip perfectly. Real git applying the hunk and producing the
// exact bytes is what makes the promise true.
func TestAHunkWithNoTrailingNewlineAppliesWithRealGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	for _, tc := range []struct {
		name string
		data string
	}{
		{"no trailing newline", "one\ntwo"},
		{"trailing newline", "one\ntwo\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			git := func(args ...string) {
				t.Helper()
				cmd := exec.Command("git", args...)
				cmd.Dir = repo
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
				}
			}
			git("init", "-q", "-b", "main")

			hunk, _ := AddedFileHunk("a.md", []byte(tc.data))
			// The patch lives outside the worktree, so applying it cannot be
			// confused by the file it is about to create.
			patch := filepath.Join(t.TempDir(), "one.patch")
			if err := os.WriteFile(patch, []byte(hunk), 0o644); err != nil {
				t.Fatal(err)
			}
			git("apply", patch)

			got, err := os.ReadFile(filepath.Join(repo, "a.md"))
			if err != nil {
				t.Fatalf("git apply wrote no file: %v", err)
			}
			if string(got) != tc.data {
				t.Errorf("git apply produced %q (%d bytes), want %q (%d bytes)",
					got, len(got), tc.data, len(tc.data))
			}
		})
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
