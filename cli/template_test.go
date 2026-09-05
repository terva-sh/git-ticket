package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeStoreTemplate puts a template into a CLI test store's
// .tickets/templates/.
func writeStoreTemplate(t *testing.T, dir, name, content string) {
	t.Helper()
	tdir := filepath.Join(dir, ".tickets", "templates")
	if err := os.MkdirAll(tdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tdir, name+".md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

const cliBugTemplate = `---
type: bug
priority: high
---

## Description

Steps to reproduce:

## Definition of done

- [ ] the fix names the cause
`

// TestCreateTemplateFromTheCLI drives plan 4.2 end to end: the seeded
// fields arrive, and the explicit flag wins.
func TestCreateTemplateFromTheCLI(t *testing.T) {
	dir := newGitStore(t)
	writeStoreTemplate(t, dir, "bug", cliBugTemplate)

	got := runCLI(t, dir, nil, "--json", "--actor", "human:sothr",
		"create", "--title", "Clock skew breaks refresh",
		"--template", "bug", "--priority", "urgent")
	if got.code != exitOK {
		t.Fatalf("create --template: exit %d\nstderr: %s", got.code, got.stderr)
	}
	id := decode(t, got.stdout)["ticket"].(map[string]any)["id"].(string)

	show := runCLI(t, dir, nil, "show", id)
	for _, want := range []string{"bug", "urgent", "Steps to reproduce", "names the cause"} {
		if !strings.Contains(show.stdout, want) {
			t.Fatalf("show lacks %q:\n%s", want, show.stdout)
		}
	}
	if strings.Contains(show.stdout, "  high  ") {
		t.Fatalf("the template's priority beat the explicit flag:\n%s", show.stdout)
	}
}

func TestCreateMissingTemplateRefuses(t *testing.T) {
	dir := newGitStore(t)
	got := runCLI(t, dir, nil, "--actor", "human:sothr",
		"create", "--title", "No seed", "--template", "ghost")
	if got.code == exitOK {
		t.Fatalf("create --template ghost succeeded")
	}
	if !strings.Contains(got.stderr, "templates") {
		t.Fatalf("refusal does not name the templates directory: %s", got.stderr)
	}
}

// TestConfigPublishesTemplates, per plan 10.6: a bare sorted list,
// never null, and the human form names the same set.
func TestConfigPublishesTemplates(t *testing.T) {
	dir := newGitStore(t)

	got := runCLI(t, dir, nil, "--json", "config")
	if got.code != exitOK {
		t.Fatalf("config: %s", got.stderr)
	}
	tmpls, ok := decode(t, got.stdout)["templates"].([]any)
	if !ok || len(tmpls) != 0 {
		t.Fatalf("templates with none defined = %v, want an empty list, not null", tmpls)
	}

	writeStoreTemplate(t, dir, "bug", cliBugTemplate)
	writeStoreTemplate(t, dir, "adr", "---\ntype: spike\n---\n\n## Description\n\nContext:\n")

	got = runCLI(t, dir, nil, "--json", "config")
	tmpls = decode(t, got.stdout)["templates"].([]any)
	if len(tmpls) != 2 || tmpls[0] != "adr" || tmpls[1] != "bug" {
		t.Fatalf("templates = %v, want [adr bug]", tmpls)
	}

	human := runCLI(t, dir, nil, "config")
	if !strings.Contains(human.stdout, "templates   adr bug") {
		t.Fatalf("human config lacks the template list:\n%s", human.stdout)
	}
}
