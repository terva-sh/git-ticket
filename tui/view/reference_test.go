package view

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

func TestReferenceDetailAndPickerOpenSelectedTarget(t *testing.T) {
	tk := detailTicket()
	tk.References = []ticket.Reference{
		{Ref: "pr:team/docs#12"},
		{Ref: "ledger-ticket:TKT-01K3ZYEE00HV9ZDBB8BEASXBBG"},
		{Ref: "origin-ticket:TKT-01M2NZ88"},
	}
	local := filepath.Join(t.TempDir(), "foreign.md")
	var opened string
	a := NewApp(fixed(tk), Actions{
		References: func(string) ([]ReferenceTarget, error) {
			return []ReferenceTarget{
				{Reference: tk.References[0], URL: "https://example.invalid/team/docs/pulls/12"},
				{Reference: tk.References[1], URL: "https://example.invalid/foreign", LocalPath: local},
				{Reference: tk.References[2]},
			}, nil
		},
		OpenTarget: func(target string) error { opened = target; return nil },
	})
	a.list.Reload()
	a.HandleKey(tui.Key{Kind: tui.KeyEnter})
	detail := strings.Join(renderApp(a, 100, 17), "\n")
	if !strings.Contains(detail, "pr:team/docs#12") || !strings.Contains(detail, "https://example.invalid/team/docs/pulls/12") || !strings.Contains(detail, "origin-ticket:TKT-01M2NZ88") {
		t.Fatalf("resolved detail: %s", detail)
	}
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'r'})
	if a.references == nil || !strings.Contains(strings.Join(renderApp(a, 100, 12), "\n"), "reference targets:") {
		t.Fatal("r did not open the reference picker")
	}
	a.HandleKey(tui.Key{Kind: tui.KeyDown})
	a.HandleKey(tui.Key{Kind: tui.KeyEnter})
	if opened != local || a.references != nil {
		t.Fatalf("picker opened %q, want local %q", opened, local)
	}
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'r'})
	a.HandleKey(tui.Key{Kind: tui.KeyDown})
	a.HandleKey(tui.Key{Kind: tui.KeyDown})
	a.HandleKey(tui.Key{Kind: tui.KeyEnter})
	if foot := renderApp(a, 100, 12)[11]; !strings.Contains(foot, "no resolved target") {
		t.Fatalf("opaque reference feedback: %q", foot)
	}
}

func TestStoreReferencesResolvesPRForeignAndLegacy(t *testing.T) {
	root := t.TempDir()
	actor := ticket.Actor{ID: "human:sothr"}
	s, err := ticket.Init(root, ticket.InitOptions{Actor: actor, Now: func() time.Time { return time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC) }})
	if err != nil {
		t.Fatal(err)
	}
	registry := `version: 1
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
	if err := os.WriteFile(filepath.Join(s.Path(), "references.yml"), []byte(registry), 0o644); err != nil {
		t.Fatal(err)
	}
	created, err := s.Create(context.Background(), ticket.CreateOptions{Title: "Reference reader", Actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	const foreignID = "TKT-01K3ZYEE00HV9ZDBB8BEASXBBG"
	for _, ref := range []string{"pr:team/docs#12", "ledger-ticket:" + foreignID, "origin-ticket:TKT-01M2NZ88"} {
		if _, err := s.Apply(context.Background(), created.Ticket.ID, ticket.AddReference{Ref: ref}, ticket.ApplyOptions{Actor: actor}); err != nil {
			t.Fatal(err)
		}
	}
	refs, err := storeReferences(StoreParams{Store: s})(created.Ticket.ID)
	if err != nil || len(refs) != 3 || refs[0].URL != "https://example.invalid/team/docs/pulls/12" || refs[1].URL == "" || refs[2].Openable() != "" {
		t.Fatalf("portable targets: %+v %v", refs, err)
	}
	checkout := t.TempDir()
	file := filepath.Join(checkout, ".tickets", "done", foreignID+".md")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte("foreign"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Path(), "references.local.yml"), []byte("version: 1\nstores:\n  ledger: "+checkout+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	refs, err = storeReferences(StoreParams{Store: s})(created.Ticket.ID)
	if err != nil || refs[1].Openable() != file || refs[1].URL == "" {
		t.Fatalf("local target did not take precedence: %+v %v", refs, err)
	}
}
