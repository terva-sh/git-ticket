package cli

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

// copyStore builds a store holding one ticket and returns its directory, the
// ticket's ID, and the body exactly as the file stores it below the
// frontmatter. The expectation is computed by splitting the file by hand
// rather than through ticket.RawBody, because RawBody is half of what these
// tests are checking.
func copyStore(t *testing.T) (dir, id, body string) {
	t.Helper()
	dir = newGitStore(t)
	got := runCLI(t, dir, nil, "--json", "--actor", "human:sothr",
		"create", "--title", "Copy me", "--description", "Prose to paste.\n\nWith a second paragraph.")
	if got.code != exitOK {
		t.Fatalf("create: exit %d\nstderr: %s", got.code, got.stderr)
	}
	env := decode(t, got.stdout)
	tk := env["ticket"].(map[string]any)
	id = tk["id"].(string)
	// The mutation envelope's ticket object carries id and revision only,
	// per plan 10.2; the file lands in pathsChanged.
	path := env["pathsChanged"].([]any)[0].(string)

	data, err := os.ReadFile(dir + "/" + path)
	if err != nil {
		t.Fatalf("reading the ticket file: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("---\n")) {
		t.Fatalf("ticket file does not open with a frontmatter fence")
	}
	idx := bytes.Index(data[4:], []byte("\n---\n"))
	if idx < 0 {
		t.Fatalf("ticket file does not close its frontmatter")
	}
	return dir, id, string(data[4+idx+5:])
}

// runCopyCLI is runCLI with the clipboard seams of Env exposed, the way
// selfupdate_test exposes ReleaseAPI.
func runCopyCLI(t *testing.T, dir string, env Env, args ...string) result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	env.Dir = dir
	env.Getenv = func(string) string { return "" }
	env.Stdout = &stdout
	env.Stderr = &stderr
	env.Now = func() time.Time { return referenceInstant }
	code := Run(args, env)
	return result{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

type nopWriteCloser struct{ *bytes.Buffer }

func (nopWriteCloser) Close() error { return nil }

// TestShowBodyPrintsTheStoredBody holds --body to plan 12.7: the file's own
// bytes below the frontmatter, nothing else, so a pipe carries what a reader
// of the file sees.
func TestShowBodyPrintsTheStoredBody(t *testing.T) {
	dir, id, body := copyStore(t)

	got := runCLI(t, dir, nil, "show", id, "--body")
	if got.code != exitOK {
		t.Fatalf("show --body: exit %d\nstderr: %s", got.code, got.stderr)
	}
	if got.stdout != body {
		t.Fatalf("show --body is not the stored body.\ngot:  %q\nwant: %q", got.stdout, body)
	}
	if strings.Contains(got.stdout, "\x1b") {
		t.Fatalf("show --body carries ANSI: %q", got.stdout)
	}
}

// TestShowBodyRefusesJSON: two spellings for one thing means drift, and
// show --json already carries every section.
func TestShowBodyRefusesJSON(t *testing.T) {
	dir, id, _ := copyStore(t)

	got := runCLI(t, dir, nil, "--json", "show", id, "--body")
	if got.code == exitOK {
		t.Fatalf("show --body --json succeeded; it must refuse")
	}
	if !strings.Contains(got.stderr, "--json") {
		t.Fatalf("refusal does not name --json: %s", got.stderr)
	}
}

// TestCopyProbeOrder holds the fixed order of plan 12.7, in particular that
// the Linux tools beat clip.exe, because WSL2 is Linux with clip.exe on PATH.
func TestCopyProbeOrder(t *testing.T) {
	cases := []struct {
		name      string
		available map[string]bool
		wantTool  string
		wantArgs  []string
	}{
		{"wl-copy beats clip.exe", map[string]bool{"wl-copy": true, "clip.exe": true}, "wl-copy", nil},
		{"pbcopy beats everything", map[string]bool{"pbcopy": true, "wl-copy": true, "xclip": true}, "pbcopy", nil},
		{"xclip names the clipboard selection", map[string]bool{"xclip": true}, "xclip", []string{"-selection", "clipboard"}},
		{"xsel names input and clipboard", map[string]bool{"xsel": true}, "xsel", []string{"--input", "--clipboard"}},
		{"clip.exe when it is all there is", map[string]bool{"clip.exe": true}, "clip.exe", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir, id, body := copyStore(t)
			var ranPath string
			var ranArgs []string
			var ranStdin []byte
			env := Env{
				LookPath: func(name string) (string, error) {
					if tc.available[name] {
						return "/fake/bin/" + name, nil
					}
					return "", errors.New("not found")
				},
				RunTool: func(path string, args []string, stdin []byte) error {
					ranPath, ranArgs, ranStdin = path, args, stdin
					return nil
				},
			}
			got := runCopyCLI(t, dir, env, "copy", id)
			if got.code != exitOK {
				t.Fatalf("copy: exit %d\nstderr: %s", got.code, got.stderr)
			}
			if want := "/fake/bin/" + tc.wantTool; ranPath != want {
				t.Fatalf("ran %q, want %q", ranPath, want)
			}
			if fmt.Sprint(ranArgs) != fmt.Sprint(tc.wantArgs) {
				t.Fatalf("args = %v, want %v", ranArgs, tc.wantArgs)
			}
			if string(ranStdin) != body {
				t.Fatalf("the tool was fed %q, want the stored body %q", ranStdin, body)
			}
			if got.stdout != "" {
				t.Fatalf("stdout must stay silent, got %q", got.stdout)
			}
			want := fmt.Sprintf("copied %s (%d bytes) via %s", id, len(body), tc.wantTool)
			if !strings.Contains(got.stderr, want) {
				t.Fatalf("stderr = %q, want it to contain %q", got.stderr, want)
			}
		})
	}
}

// TestCopyToolFailureIsLoud: a tool that is found but fails ends the command,
// because pretending the clipboard changed is the one wrong answer.
func TestCopyToolFailureIsLoud(t *testing.T) {
	dir, id, _ := copyStore(t)
	env := Env{
		LookPath: func(name string) (string, error) {
			if name == "wl-copy" {
				return "/fake/bin/wl-copy", nil
			}
			return "", errors.New("not found")
		},
		RunTool: func(string, []string, []byte) error {
			return errors.New("wayland compositor said no")
		},
	}
	got := runCopyCLI(t, dir, env, "copy", id)
	if got.code == exitOK {
		t.Fatalf("copy succeeded although the tool failed")
	}
	if !strings.Contains(got.stderr, "wl-copy failed") || !strings.Contains(got.stderr, "compositor said no") {
		t.Fatalf("failure does not name the tool and its words: %s", got.stderr)
	}
}

// TestCopyFallsBackToOSC52: no tool on PATH, so the body goes through the
// controlling terminal, base64 inside the OSC 52 sequence.
func TestCopyFallsBackToOSC52(t *testing.T) {
	dir, id, body := copyStore(t)
	var tty bytes.Buffer
	env := Env{
		LookPath:     func(string) (string, error) { return "", errors.New("not found") },
		ClipboardTTY: func() (io.WriteCloser, error) { return nopWriteCloser{&tty}, nil },
	}
	got := runCopyCLI(t, dir, env, "copy", id)
	if got.code != exitOK {
		t.Fatalf("copy: exit %d\nstderr: %s", got.code, got.stderr)
	}
	want := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(body)) + "\a"
	if tty.String() != want {
		t.Fatalf("the terminal got %q, want the OSC 52 sequence %q", tty.String(), want)
	}
	if !strings.Contains(got.stderr, "via OSC 52") {
		t.Fatalf("stderr does not say which path was taken: %s", got.stderr)
	}
}

// TestCopyNoToolNoTTY: the failure names every tool the probe looked for,
// because installing one of them is the user's repair.
func TestCopyNoToolNoTTY(t *testing.T) {
	dir, id, _ := copyStore(t)
	env := Env{
		LookPath:     func(string) (string, error) { return "", errors.New("not found") },
		ClipboardTTY: func() (io.WriteCloser, error) { return nil, errors.New("no tty") },
	}
	got := runCopyCLI(t, dir, env, "copy", id)
	if got.code == exitOK {
		t.Fatalf("copy succeeded with no clipboard path at all")
	}
	for _, tool := range []string{"pbcopy", "wl-copy", "xclip", "xsel", "clip.exe"} {
		if !strings.Contains(got.stderr, tool) {
			t.Fatalf("failure does not name %s: %s", tool, got.stderr)
		}
	}
}

// TestCopyRefusesNativeWindows, per plan 12.7. show --body still works
// there, so the refusal points at it.
func TestCopyRefusesNativeWindows(t *testing.T) {
	dir, id, _ := copyStore(t)
	old := copyGOOS
	copyGOOS = "windows"
	defer func() { copyGOOS = old }()

	got := runCLI(t, dir, nil, "copy", id)
	if got.code == exitOK {
		t.Fatalf("copy succeeded on native Windows")
	}
	for _, want := range []string{"WSL2", "Linux", "macOS"} {
		if !strings.Contains(got.stderr, want) {
			t.Fatalf("refusal does not name %s: %s", want, got.stderr)
		}
	}
}

// TestRunClipboardToolOutlivesNoForkedChild reproduces the hang the first
// real-world run hit. wl-copy forks a child that serves the selection until
// something replaces it, and the child holds the inherited descriptors open.
// With pipes on stdout and stderr, Wait blocks until that child dies, which
// read as a two-minute timeout. The stand-in tool here does the same fork
// shape: it backgrounds a sleep and exits at once. The assertion is that
// runClipboardTool comes back with the parent rather than the child.
func TestRunClipboardToolOutlivesNoForkedChild(t *testing.T) {
	done := make(chan error, 1)
	go func() {
		done <- runClipboardTool("/bin/sh", []string{"-c", "sleep 30 & exit 0"}, []byte("body"))
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("runClipboardTool: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("runClipboardTool waited for the forked child instead of the tool itself")
	}
}

// TestRunClipboardToolReportsTheToolsWords: "exit status 3" alone tells the
// user nothing, so the tool's own stderr rides in the error.
func TestRunClipboardToolReportsTheToolsWords(t *testing.T) {
	err := runClipboardTool("/bin/sh", []string{"-c", "echo boom >&2; exit 3"}, nil)
	if err == nil {
		t.Fatal("a failing tool did not fail the call")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("the tool's words are missing: %v", err)
	}
}

// TestCopyRefusesJSON: a clipboard write has nothing to say in JSON, and
// show --json is the JSON path to the body.
func TestCopyRefusesJSON(t *testing.T) {
	dir, id, _ := copyStore(t)

	got := runCLI(t, dir, nil, "--json", "copy", id)
	if got.code == exitOK {
		t.Fatalf("copy --json succeeded; it must refuse")
	}
	if !strings.Contains(got.stderr, "show --json") {
		t.Fatalf("refusal does not point at show --json: %s", got.stderr)
	}
}
