package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const readerRegistry = `version: 1
stores:
  ledger:
    repository: https://example.invalid/team/ledger.git
    path: .tickets
    browse: https://example.invalid/team/ledger/search?q={id}
namespaces:
  pr:
    kind: url
    identifier: '(?P<owner>[a-z]+)/(?P<repo>[a-z]+)#(?P<number>[0-9]+)'
    template: https://example.invalid/{owner}/{repo}/pulls/{number}
  ledger-ticket:
    kind: ticket-store
    identifier: '(?P<id>[A-Z][A-Z0-9-]*-[0-9A-HJKMNP-TV-Z]{26})'
    store: ledger
`

func referenceReaderStore(t *testing.T) (string, string) {
	t.Helper()
	dir := newGitStore(t)
	if err := os.WriteFile(filepath.Join(dir, ".tickets", "references.yml"), []byte(readerRegistry), 0o644); err != nil {
		t.Fatal(err)
	}
	id := ticketID(t, createTicket(t, dir))
	for _, ref := range []string{
		"pr:team/docs#12",
		"ledger-ticket:TKT-01K3ZYEE00HV9ZDBB8BEASXBBG",
		"origin-ticket:TKT-01M2NZ88",
	} {
		got := runCLI(t, dir, nil, "link", id, "--ref", ref, "--actor", "human:sothr")
		if got.code != exitOK {
			t.Fatalf("add %s: %s%s", ref, got.stdout, got.stderr)
		}
	}
	return dir, id
}

func TestShowAndRefsExposeResolvedTargetsWithoutChangingStoredRefs(t *testing.T) {
	dir, id := referenceReaderStore(t)
	const prURL = "https://example.invalid/team/docs/pulls/12"
	const foreignURL = "https://example.invalid/team/ledger/search?q=TKT-01K3ZYEE00HV9ZDBB8BEASXBBG"
	shown := runCLI(t, dir, nil, "show", id)
	if shown.code != exitOK || !strings.Contains(shown.stdout, "pr:team/docs#12") || !strings.Contains(shown.stdout, prURL) || !strings.Contains(shown.stdout, foreignURL) {
		t.Fatalf("human show: %s%s", shown.stdout, shown.stderr)
	}
	jsonShow := runCLI(t, dir, nil, "--json", "show", id)
	if jsonShow.code != exitOK {
		t.Fatalf("JSON show: %s%s", jsonShow.stdout, jsonShow.stderr)
	}
	env := decode(t, jsonShow.stdout)
	tk := env["ticket"].(map[string]any)
	refs := tk["references"].([]any)
	got := map[string]map[string]any{}
	for _, item := range refs {
		r := item.(map[string]any)
		got[r["ref"].(string)] = r
	}
	if got["pr:team/docs#12"]["resolvedUrl"] != prURL || got["ledger-ticket:TKT-01K3ZYEE00HV9ZDBB8BEASXBBG"]["resolvedUrl"] != foreignURL {
		t.Fatalf("JSON targets: %+v", got)
	}
	if _, resolved := got["origin-ticket:TKT-01M2NZ88"]["resolvedUrl"]; resolved {
		t.Fatalf("undeclared legacy ref became resolved: %+v", got)
	}
	if got["pr:team/docs#12"]["path"] != nil {
		t.Fatalf("stored path changed: %+v", got["pr:team/docs#12"])
	}
	plain := runCLI(t, dir, nil, "refs", "pr:")
	resolved := runCLI(t, dir, nil, "refs", "pr:", "--resolve")
	if plain.code != exitOK || resolved.code != exitOK || strings.Contains(plain.stdout, prURL) || !strings.Contains(resolved.stdout, prURL) || strings.Contains(resolved.stdout, foreignURL) {
		t.Fatalf("refs lookup changed: plain=%s resolved=%s", plain.stdout, resolved.stdout)
	}
	jsonRefs := runCLI(t, dir, nil, "--json", "refs", "pr:", "--resolve")
	if jsonRefs.code != exitOK {
		t.Fatalf("JSON refs resolution: %s%s", jsonRefs.stdout, jsonRefs.stderr)
	}
	listed := decode(t, jsonRefs.stdout)
	if listed["kind"] != "ticket-list" || len(listed["tickets"].([]any)) != 1 {
		t.Fatalf("JSON refs changed its ticket-list contract: %+v", listed)
	}
	legacy := runCLI(t, dir, nil, "refs", "origin-ticket:", "--resolve")
	if legacy.code != exitOK || !strings.Contains(legacy.stdout, "origin-ticket:TKT-01M2NZ88") || strings.Contains(legacy.stdout, "->") {
		t.Fatalf("legacy query: %s%s", legacy.stdout, legacy.stderr)
	}
}

func TestForeignTicketResolutionUsesLocalCheckoutWhenAvailable(t *testing.T) {
	dir, id := referenceReaderStore(t)
	checkout := t.TempDir()
	const foreignID = "TKT-01K3ZYEE00HV9ZDBB8BEASXBBG"
	file := filepath.Join(checkout, ".tickets", "done", foreignID+".md")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("foreign ticket"), 0o644); err != nil {
		t.Fatal(err)
	}
	local := "version: 1\nstores:\n  ledger: " + checkout + "\n"
	if err := os.WriteFile(filepath.Join(dir, ".tickets", "references.local.yml"), []byte(local), 0o644); err != nil {
		t.Fatal(err)
	}
	shown := runCLI(t, dir, nil, "--json", "show", id)
	if shown.code != exitOK {
		t.Fatalf("show: %s%s", shown.stdout, shown.stderr)
	}
	refs := decode(t, shown.stdout)["ticket"].(map[string]any)["references"].([]any)
	found := false
	for _, item := range refs {
		ref := item.(map[string]any)
		if ref["ref"] != "ledger-ticket:"+foreignID {
			continue
		}
		found = true
		if ref["resolvedLocalPath"] != file || !strings.Contains(ref["resolvedUrl"].(string), foreignID) {
			t.Fatalf("local and portable targets: %+v", ref)
		}
	}
	if !found {
		t.Fatal("foreign reference missing from JSON")
	}
	resolved := runCLI(t, dir, nil, "refs", "ledger-ticket:", "--resolve")
	if resolved.code != exitOK || !strings.Contains(resolved.stdout, file) {
		t.Fatalf("local refs resolution: %s%s", resolved.stdout, resolved.stderr)
	}
}
