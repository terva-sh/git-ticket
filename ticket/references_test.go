package ticket

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const exampleRegistry = `version: 1
stores:
  ledger:
    repository: https://git.local.example/team/ledger.git
    path: .tickets
    browse: https://git.local.example/team/ledger/search?q={id}
namespaces:
  pr:
    kind: url
    identifier: '(?P<owner>[A-Za-z0-9_-]+)/(?P<repo>[A-Za-z0-9_.-]+)#(?P<number>[0-9]+)'
    template: https://git.local.example/{owner}/{repo}/pulls/{number}
  ledger-ticket:
    kind: ticket-store
    identifier: '(?P<id>[A-Z][A-Z0-9-]*-[0-9A-HJKMNP-TV-Z]{26})'
    store: ledger
`

func putRegistry(t *testing.T, s *Store, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(s.Path(), referencesFile), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReferenceRegistryRoundTripAndConfigRewrite(t *testing.T) {
	r, err := ParseReferenceRegistry([]byte(exampleRegistry))
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := RenderReferenceRegistry(*r)
	if err != nil {
		t.Fatal(err)
	}
	again, err := ParseReferenceRegistry(canonical)
	if err != nil {
		t.Fatal(err)
	}
	second, err := RenderReferenceRegistry(*again)
	if err != nil || string(canonical) != string(second) {
		t.Fatalf("registry render drift: %v", err)
	}

	s := newTestStore(t)
	putRegistry(t, s, exampleRegistry)
	if _, err := s.AddSeries(context.Background(), "IDEA"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(s.Path(), referencesFile))
	if err != nil || string(data) != exampleRegistry {
		t.Fatalf("config rewrite changed registry: %v", err)
	}
	ignore, err := os.ReadFile(filepath.Join(s.Path(), ".gitignore"))
	if err != nil || !strings.Contains(string(ignore), "references.local.yml") {
		t.Fatalf("local binding is not ignored: %v", err)
	}
}

func TestInitPreservesAnAdoptedStoresIgnoreRules(t *testing.T) {
	root := t.TempDir()
	storePath := filepath.Join(root, StoreDirName)
	if err := os.MkdirAll(storePath, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(storePath, ".gitignore")
	if err := os.WriteFile(file, []byte("my-local-cache\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(root, InitOptions{Actor: testActor, Now: fixedClock()}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(file)
	if err != nil || string(data) != "my-local-cache\nreferences.local.yml\n" {
		t.Fatalf("init did not preserve existing ignore rules: %q %v", data, err)
	}
}

func TestInitOverridesLaterLocalReferenceIgnoreNegation(t *testing.T) {
	root := t.TempDir()
	storePath := filepath.Join(root, StoreDirName)
	if err := os.MkdirAll(storePath, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(storePath, ".gitignore")
	if err := os.WriteFile(file, []byte("references.local.yml\n!*.yml\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Init(root, InitOptions{Actor: testActor, Now: fixedClock()}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(file)
	if err != nil || string(data) != "references.local.yml\n!*.yml\nreferences.local.yml\n" {
		t.Fatalf("local binding ignore did not override negation: %q %v", data, err)
	}
}

func TestResolveReferenceUsesPortableURLAndOptionalLocalCheckout(t *testing.T) {
	s := newTestStore(t)
	putRegistry(t, s, exampleRegistry)
	pr, err := s.ResolveReference(context.Background(), Reference{Ref: "pr:team/docs#12"})
	if err != nil {
		t.Fatal(err)
	}
	if pr.Ref != "pr:team/docs#12" || pr.URL != "https://git.local.example/team/docs/pulls/12" || pr.LocalPath != "" {
		t.Fatalf("PR resolution: %+v", pr)
	}
	const id = "TKT-01K3ZYEE00HV9ZDBB8BEASXBBG"
	foreign := Reference{Ref: "ledger-ticket:" + id}
	remote, err := s.ResolveReference(context.Background(), foreign)
	if err != nil {
		t.Fatal(err)
	}
	if remote.Ref != foreign.Ref || remote.URL != "https://git.local.example/team/ledger/search?q="+id || remote.LocalPath != "" {
		t.Fatalf("portable target: %+v", remote)
	}
	checkout := t.TempDir()
	file := filepath.Join(checkout, ".tickets", "done", id+".md")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("ticket"), 0o644); err != nil {
		t.Fatal(err)
	}
	local := LocalReferenceBindings{Version: 1, Stores: map[string]string{"ledger": checkout}}
	data, err := RenderLocalReferenceBindings(local)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseLocalReferenceBindings(data); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Path(), localReferencesFile), data, 0o644); err != nil {
		t.Fatal(err)
	}
	resolved, err := s.ResolveReference(context.Background(), foreign)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.LocalPath != file || resolved.URL != remote.URL {
		t.Fatalf("local resolution: %+v", resolved)
	}
	own := filepath.Join(checkout, "docs", "decision.md")
	if err := os.MkdirAll(filepath.Dir(own), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(own, []byte("decision"), 0o644); err != nil {
		t.Fatal(err)
	}
	withRoot, err := OpenWith(s.Path(), OpenOptions{Root: checkout})
	if err != nil {
		t.Fatal(err)
	}
	relative := "docs/decision.md"
	preferred, err := withRoot.ResolveReference(context.Background(), Reference{Ref: foreign.Ref, Path: &relative})
	if err != nil || preferred.LocalPath != own {
		t.Fatalf("repository-relative path must take precedence: %+v %v", preferred, err)
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	missing, err := s.ResolveReference(context.Background(), foreign)
	if err != nil || missing.LocalPath != "" || missing.URL != remote.URL {
		t.Fatalf("missing checkout must fall back to URL: %+v %v", missing, err)
	}
	if err := os.WriteFile(filepath.Join(s.Path(), localReferencesFile), []byte("version: 9\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	check, err := s.Check(context.Background())
	if err != nil || !check.OK() {
		t.Fatalf("check must ignore machine-local syntax: %+v %v", check, err)
	}
	if _, err := s.ResolveReference(context.Background(), foreign); err == nil {
		t.Fatal("resolver accepted malformed local bindings")
	}
}

func TestURLCaptureIsEncodedAsAPathSegment(t *testing.T) {
	s := newTestStore(t)
	putRegistry(t, s, "version: 1\nstores: {}\nnamespaces:\n  doc:\n    kind: url\n    identifier: '(?P<slug>.+)'\n    template: https://example.invalid/doc/{slug}\n")
	got, err := s.ResolveReference(context.Background(), Reference{Ref: "doc:one/two#three"})
	if err != nil || got.URL != "https://example.invalid/doc/one%2Ftwo%23three" {
		t.Fatalf("path capture escaped a segment: %+v %v", got, err)
	}
	if _, err := s.ResolveReference(context.Background(), Reference{Ref: "doc:.."}); err == nil {
		t.Fatal("a captured dot segment would change the URL path")
	}
}

func TestUndeclaredLegacyReferencesStayOpaque(t *testing.T) {
	s := newTestStore(t)
	putRegistry(t, s, exampleRegistry)
	ticket := mustCreate(t, s, "Legacy references stay valid")
	for _, ref := range []string{"ticket:report", "origin-ticket:TKT-01M2NZ88"} {
		got, err := s.ResolveReference(context.Background(), Reference{Ref: ref})
		if err != nil || got != nil {
			t.Fatalf("legacy %q became invalid or resolved: %+v %v", ref, got, err)
		}
		mustApply(t, s, ticket.ID, AddReference{Ref: ref})
	}
	report, err := s.Check(context.Background())
	if err != nil || !report.OK() {
		t.Fatalf("undeclared legacy references must pass check: %+v %v", report, err)
	}
	if got, err := s.ResolveReference(context.Background(), Reference{Ref: "ledger-ticket:bad"}); got != nil || err == nil {
		t.Fatalf("declared bad identifier: %+v %v", got, err)
	} else {
		var coded *Error
		if !errors.As(err, &coded) || coded.Code != CodeReferenceIdentifierInvalid {
			t.Fatalf("wrong code: %v", err)
		}
	}
}

func TestRegistryRejectsUnsafeTemplatesAndTraversal(t *testing.T) {
	cases := []string{
		strings.Replace(exampleRegistry, "version: 1", "version: 2", 1),
		strings.Replace(exampleRegistry, "{owner}/{repo}", "{owner}.example/{repo}", 1),
		strings.Replace(exampleRegistry, "path: .tickets", "path: ../elsewhere", 1),
		strings.Replace(exampleRegistry, "search?q={id}", "search?q={id}#fragment", 1),
		strings.Replace(exampleRegistry, "https://git.local.example/{owner}", "https://name@git.local.example/{owner}", 1),
		strings.Replace(exampleRegistry, "repository: https://git.local.example/", "repository: https://name@git.local.example/", 1),
		strings.Replace(exampleRegistry, "identifier: '(?P<id>", "identifier: 'prefix(?P<id>", 1),
		strings.Replace(exampleRegistry, "{owner}/{repo}", "%7Bowner%7D/{repo}", 1),
		strings.Replace(exampleRegistry, "search?q={id}", "search?q=%7Bid%7D", 1),
	}
	for i, body := range cases {
		if _, err := ParseReferenceRegistry([]byte(body)); err == nil {
			t.Errorf("case %d accepted unsafe registry", i)
		}
	}
}

func TestCheckRejectsTrackedRegistrySymlinks(t *testing.T) {
	for _, target := range []string{"outside.yml", "missing.yml"} {
		t.Run(target, func(t *testing.T) {
			s := newTestStore(t)
			if target == "outside.yml" {
				if err := os.WriteFile(filepath.Join(filepath.Dir(s.Path()), target), []byte(exampleRegistry), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.Symlink(filepath.Join("..", target), filepath.Join(s.Path(), referencesFile)); err != nil {
				t.Fatal(err)
			}
			report, err := s.Check(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, finding := range report.Errors {
				found = found || finding.Code == CodeReferenceRegistryInvalid
			}
			if !found {
				t.Fatalf("symlink was not reported: %+v", report)
			}
		})
	}
}
