package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const portablePRRegistry = `version: 1
stores: {}
namespaces:
  pr:
    kind: url
    identifier: '(?P<owner>[A-Za-z0-9_-]+)/(?P<repo>[A-Za-z0-9_-]+)#(?P<number>[0-9]+)'
    template: https://example.invalid/{owner}/{repo}/pulls/{number}
`

func mappedExport(t *testing.T) (string, string) {
	t.Helper()
	src, id := newExportSource(t, "Ticket with a PR reference")
	if err := os.WriteFile(filepath.Join(src, ".tickets", "references.yml"), []byte(portablePRRegistry), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := runCLI(t, src, nil, "link", id, "--ref", "pr:team/docs#12", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("add reference: %s%s", got.stdout, got.stderr)
	}
	out := filepath.Join(t.TempDir(), "export")
	if got := runCLI(t, src, nil, "export", id, "--out", out); got.code != exitOK {
		t.Fatalf("export: %s%s", got.stdout, got.stderr)
	}
	return out, id
}

func TestImportCanAliasPortableReferenceMapping(t *testing.T) {
	dir, _ := mappedExport(t)
	dest := newGitStore(t)
	preview := runCLI(t, dest, nil, "import", dir)
	if preview.code != exitOK || !strings.Contains(preview.stdout, "pr:team/docs#12 -> https://example.invalid/team/docs/pulls/12") || !strings.Contains(preview.stdout, "receiver: absent; choice: decline") {
		t.Fatalf("mapping preview: %s%s", preview.stdout, preview.stderr)
	}
	if _, err := os.Stat(filepath.Join(dest, ".tickets", "references.yml")); !os.IsNotExist(err) {
		t.Fatalf("preview wrote registry: %v", err)
	}
	adopt := runCLI(t, dest, nil, "import", dir, "--adopt-mappings", "--adopt", "--map", "pr=alias:foreign-pr", "--actor", "human:sothr")
	if adopt.code != exitOK || !strings.Contains(adopt.stderr, "adopted reference mappings") {
		t.Fatalf("mapping adoption: %s%s", adopt.stdout, adopt.stderr)
	}
	data, err := os.ReadFile(filepath.Join(dest, ".tickets", "references.yml"))
	if err != nil || !strings.Contains(string(data), "foreign-pr:") {
		t.Fatalf("adopted registry: %s %v", data, err)
	}
	rows := crossRows(t, runCLI(t, dest, nil, "--json", "list", "--all"))
	if len(rows) != 1 {
		t.Fatalf("adopted rows: %v", rows)
	}
	id, _ := rows[0]["id"].(string)
	shown := runCLI(t, dest, nil, "--json", "show", id)
	if shown.code != exitOK || !strings.Contains(shown.stdout, "foreign-pr:team/docs#12") {
		t.Fatalf("aliased reference: %s%s", shown.stdout, shown.stderr)
	}
	if checked := runCLI(t, dest, nil, "check", "--strict"); checked.code != exitOK {
		t.Fatalf("adopted store check: %s%s", checked.stdout, checked.stderr)
	}
}

func TestConflictingMappingPreviewAndOpaqueDecline(t *testing.T) {
	dir, _ := mappedExport(t)
	dest := newGitStore(t)
	other := strings.Replace(portablePRRegistry, "https://example.invalid/", "https://other.invalid/", 1)
	if err := os.WriteFile(filepath.Join(dest, ".tickets", "references.yml"), []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}
	preview := runCLI(t, dest, nil, "import", dir)
	if preview.code != exitOK || !strings.Contains(preview.stdout, "receiver: conflicting") || !strings.Contains(preview.stdout, "decline:LOCAL") {
		t.Fatalf("conflict preview: %s%s", preview.stdout, preview.stderr)
	}
	if got := runCLI(t, dest, nil, "import", dir, "--adopt", "--actor", "human:sothr"); got.code == exitOK {
		t.Fatalf("conflicting reference adopted without an opaque target: %s", got.stdout)
	}
	adopt := runCLI(t, dest, nil, "import", dir, "--adopt", "--map", "pr=decline:unmapped-pr", "--actor", "human:sothr")
	if adopt.code != exitOK || !strings.Contains(adopt.stdout, "Ticket with a PR reference") {
		t.Fatalf("opaque decline: %s%s", adopt.stdout, adopt.stderr)
	}
	rows := crossRows(t, runCLI(t, dest, nil, "--json", "list", "--all"))
	id, _ := rows[0]["id"].(string)
	shown := runCLI(t, dest, nil, "--json", "show", id)
	if !strings.Contains(shown.stdout, "unmapped-pr:team/docs#12") || strings.Contains(shown.stdout, "https://other.invalid/team/docs/pulls/12") {
		t.Fatalf("declined reference resolved to unrelated local meaning: %s", shown.stdout)
	}
}

func TestLegacyExportWithoutLookupStillImports(t *testing.T) {
	dir, _ := mappedExport(t)
	if err := os.Remove(filepath.Join(dir, exportReferenceLookup)); err != nil {
		t.Fatal(err)
	}
	dest := newGitStore(t)
	if got := runCLI(t, dest, nil, "import", dir); got.code != exitOK || strings.Contains(got.stdout, "Reference lookup offered") {
		t.Fatalf("old export preview: %s%s", got.stdout, got.stderr)
	}
	if got := runCLI(t, dest, nil, "import", dir, "--adopt", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("old export adoption: %s%s", got.stdout, got.stderr)
	}
}

func TestMappingsCanBeAcceptedBeforeGitAm(t *testing.T) {
	dir, _ := mappedExport(t)
	dest := newGitStore(t)
	accepted := runCLI(t, dest, nil, "import", dir, "--adopt-mappings", "--map", "pr=adopt")
	if accepted.code != exitOK || !strings.Contains(accepted.stdout, "No tickets written") ||
		!strings.Contains(accepted.stdout, "receiver before mapping write: absent") ||
		!strings.Contains(accepted.stdout, "receiver now: mapping installed as pr") {
		t.Fatalf("mapping-only adoption: %s%s", accepted.stdout, accepted.stderr)
	}
	if rows := crossRows(t, runCLI(t, dest, nil, "--json", "list", "--all")); len(rows) != 0 {
		t.Fatalf("mapping-only adoption filed tickets: %v", rows)
	}
	exportGit(t, dest, "add", ".tickets/references.yml")
	exportGit(t, dest, "commit", "-qm", "accept reference mapping")
	exportGit(t, dest, "am", filepath.Join(dir, exportTicketPatch))
	if checked := runCLI(t, dest, nil, "check", "--strict"); checked.code != exitOK {
		t.Fatalf("git am route with accepted mapping: %s%s", checked.stdout, checked.stderr)
	}
}

func TestOldExportCannotSilentlyUseReceivingNamespace(t *testing.T) {
	dir, _ := mappedExport(t)
	if err := os.Remove(filepath.Join(dir, exportReferenceLookup)); err != nil {
		t.Fatal(err)
	}
	dest := newGitStore(t)
	other := strings.Replace(portablePRRegistry, "https://example.invalid/", "https://other.invalid/", 1)
	if err := os.WriteFile(filepath.Join(dest, ".tickets", "references.yml"), []byte(other), 0o644); err != nil {
		t.Fatal(err)
	}
	preview := runCLI(t, dest, nil, "import", dir)
	if preview.code != exitOK || !strings.Contains(preview.stdout, "destinations are unknown") {
		t.Fatalf("old export conflict preview: %s%s", preview.stdout, preview.stderr)
	}
	if got := runCLI(t, dest, nil, "import", dir, "--adopt", "--actor", "human:sothr"); got.code == exitOK {
		t.Fatal("old export inherited a local namespace without a receiver choice")
	}
	if got := runCLI(t, dest, nil, "import", dir, "--adopt", "--map", "pr=decline:old-pr", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("old export opaque import: %s%s", got.stdout, got.stderr)
	}
}
