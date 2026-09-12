package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeDoc puts an authored ticket outside the store. A document is a file
// somebody wrote, per plan 4.3, and it deliberately does not live in
// .tickets/templates/.
func writeDoc(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "authored.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const cliWholeTicket = `---
title: Add token refresh handling
type: bug
priority: high
labels:
  - regression
---

## Description

The daemon drops the refresh token on a 401 and never retries.

## Acceptance criteria

- [ ] A 401 refreshes once and retries the original request
- [ ] A second 401 gives up

## Definition of done

- [ ] The retry is covered by a test that fails without it
`

// TestCreateFileFilesAWholeTicket is the first criterion from the outside: one
// command, no --title, and every section arrives.
func TestCreateFileFilesAWholeTicket(t *testing.T) {
	dir := newGitStore(t)
	path := writeDoc(t, cliWholeTicket)

	got := runCLI(t, dir, nil, "--json", "--actor", "human:sothr", "create", "--file", path)
	if got.code != exitOK {
		t.Fatalf("create --file: exit %d\nstderr: %s", got.code, got.stderr)
	}
	id := decode(t, got.stdout)["ticket"].(map[string]any)["id"].(string)

	show := runCLI(t, dir, nil, "show", id, "--body")
	for _, want := range []string{
		"drops the refresh token",
		"A 401 refreshes once and retries",
		"A second 401 gives up",
		"covered by a test that fails without it",
	} {
		if !strings.Contains(show.stdout, want) {
			t.Fatalf("the document did not survive, missing %q:\n%s", want, show.stdout)
		}
	}

	head := runCLI(t, dir, nil, "show", id)
	for _, want := range []string{"Add token refresh handling", "bug", "high", "regression"} {
		if !strings.Contains(head.stdout, want) {
			t.Fatalf("show lacks %q:\n%s", want, head.stdout)
		}
	}
}

// TestCreateFileWarnsAboutLifecycleKeys is the reporting half of 4.3. The keys
// are dropped, the create still succeeds, and the warning says so rather than
// leaving the author to notice.
func TestCreateFileWarnsAboutLifecycleKeys(t *testing.T) {
	dir := newGitStore(t)
	path := writeDoc(t, `---
id: TKT-01K3ZZ67Q0PT427VFD1F4WFWSH
title: Copied out of a real store
status: done
created_at: 2024-01-01T00:00:00Z
---

## Description

Copied, which is the ordinary way to write one of these.
`)

	got := runCLI(t, dir, nil, "--json", "--actor", "human:sothr", "create", "--file", path)
	if got.code != exitOK {
		t.Fatalf("a document with lifecycle keys refused: exit %d\nstderr: %s", got.code, got.stderr)
	}
	for _, want := range []string{"id", "status", "created_at", "--status", "--created"} {
		if !strings.Contains(got.stderr, want) {
			t.Fatalf("the warning does not name %q:\n%s", want, got.stderr)
		}
	}

	// The gate of 6.2.1 holds: the document asked for done and got draft.
	id := decode(t, got.stdout)["ticket"].(map[string]any)["id"].(string)
	show := runCLI(t, dir, nil, "--json", "show", id)
	if status := decode(t, show.stdout)["ticket"].(map[string]any)["status"].(string); status != "draft" {
		t.Fatalf("status = %q, want draft: a document must not promote itself", status)
	}
	if !strings.Contains(got.stderr, "draft") {
		t.Fatalf("the warning does not say where it landed:\n%s", got.stderr)
	}
}

// TestCreateFileKeepsTheLeniency is the second criterion. A key the loader does
// not know is ignored rather than refused, which is what lets a document be
// written by copying a real ticket or by a model that invented a field.
func TestCreateFileKeepsTheLeniency(t *testing.T) {
	dir := newGitStore(t)
	path := writeDoc(t, `---
title: Still files
severity: catastrophic
sprint: 14
---

## Description

Filed anyway.
`)

	got := runCLI(t, dir, nil, "--json", "--actor", "human:sothr", "create", "--file", path)
	if got.code != exitOK {
		t.Fatalf("an unknown key refused the create: exit %d\nstderr: %s", got.code, got.stderr)
	}
	if strings.Contains(got.stderr, "severity") || strings.Contains(got.stderr, "sprint") {
		t.Fatalf("an unknown key was reported as a lifecycle key:\n%s", got.stderr)
	}
}

// TestCreateFileLetsAnExplicitFlagWin is the third criterion, and the rule a
// template already follows: a document is a starting point, not an argument.
func TestCreateFileLetsAnExplicitFlagWin(t *testing.T) {
	dir := newGitStore(t)
	path := writeDoc(t, cliWholeTicket)

	got := runCLI(t, dir, nil, "--json", "--actor", "human:sothr",
		"create", "--file", path, "--title", "What the caller typed", "--priority", "urgent")
	if got.code != exitOK {
		t.Fatalf("create: exit %d\nstderr: %s", got.code, got.stderr)
	}
	// The mutation envelope carries the ID and the revision only, so the fields
	// are read back from the ticket itself.
	id := decode(t, got.stdout)["ticket"].(map[string]any)["id"].(string)
	tk := decode(t, runCLI(t, dir, nil, "--json", "show", id).stdout)["ticket"].(map[string]any)
	if tk["title"].(string) != "What the caller typed" {
		t.Errorf("title = %q, want the flag to win", tk["title"])
	}
	if tk["priority"].(string) != "urgent" {
		t.Errorf("priority = %q, want the flag to win", tk["priority"])
	}
	// The document's own type still applies, since the caller named none.
	if tk["type"].(string) != "bug" {
		t.Errorf("type = %q, want the document's", tk["type"])
	}

	// What the caller did not name still comes from the document.
	show := runCLI(t, dir, nil, "show", id, "--body")
	if !strings.Contains(show.stdout, "drops the refresh token") {
		t.Fatalf("the document's description was lost:\n%s", show.stdout)
	}
}

// TestCreateFileRefusesASecondSeedSource is the rule --from and --template
// already follow, extended to the third source by 4.3.
func TestCreateFileRefusesASecondSeedSource(t *testing.T) {
	dir := newGitStore(t)
	path := writeDoc(t, cliWholeTicket)
	writeStoreTemplate(t, dir, "bug", cliBugTemplate)

	seed := runCLI(t, dir, nil, "--json", "--actor", "human:sothr", "create", "--title", "A source")
	if seed.code != exitOK {
		t.Fatalf("seed create: exit %d\nstderr: %s", seed.code, seed.stderr)
	}
	sourceID := decode(t, seed.stdout)["ticket"].(map[string]any)["id"].(string)

	for _, tc := range []struct {
		name string
		args []string
	}{
		{"with --template", []string{"create", "--file", path, "--template", "bug"}},
		{"with --from", []string{"create", "--file", path, "--from", sourceID}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := runCLI(t, dir, nil, append([]string{"--actor", "human:sothr"}, tc.args...)...)
			if got.code == exitOK {
				t.Fatalf("two seed sources were accepted:\n%s", got.stdout)
			}
			if !strings.Contains(got.stderr, "seed sources") {
				t.Fatalf("the refusal does not say why: %s", got.stderr)
			}
		})
	}
}

// TestCreateFileMissingPathRefuses matches the template refusal of 4.2: a
// ticket that silently lacks what its author wrote is worse than a stopped
// command.
func TestCreateFileMissingPathRefuses(t *testing.T) {
	dir := newGitStore(t)
	missing := filepath.Join(t.TempDir(), "nope.md")

	got := runCLI(t, dir, nil, "--actor", "human:sothr", "create", "--file", missing)
	if got.code == exitOK {
		t.Fatal("create --file with a missing path succeeded")
	}
	if !strings.Contains(got.stderr, "nope.md") {
		t.Fatalf("the refusal does not name the path: %s", got.stderr)
	}
}

// TestCreateStillNeedsATitleWithoutASource keeps the relaxation narrow. --title
// is optional with --file because the document supplies it, and nowhere else.
func TestCreateStillNeedsATitleWithoutASource(t *testing.T) {
	dir := newGitStore(t)
	got := runCLI(t, dir, nil, "--actor", "human:sothr", "create")
	if got.code == exitOK {
		t.Fatal("create with no title and no source succeeded")
	}
	if !strings.Contains(got.stderr, "--title") {
		t.Fatalf("the refusal does not name --title: %s", got.stderr)
	}
}
