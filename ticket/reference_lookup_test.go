package ticket

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func lookupFixture(t *testing.T) (*Export, string) {
	t.Helper()
	s := newTestStore(t)
	putRegistry(t, s, exampleRegistry)
	tk := mustCreate(t, s, "Portable reference")
	mustApply(t, s, tk.ID, AddReference{Ref: "pr:team/docs#12"})
	mustApply(t, s, tk.ID, AddReference{Ref: "legacy:opaque"})
	art, err := s.Export(context.Background(), ExportOptions{IDs: []string{tk.ID}, From: "Sender <sender@example.invalid>", Now: fixedClock()()})
	if err != nil {
		t.Fatal(err)
	}
	return art, tk.ID
}

func TestReferenceLookupBindsExactPatchAndUse(t *testing.T) {
	art, _ := lookupFixture(t)
	lookup, err := ParseReferenceLookup(art.Patch, art.References)
	if err != nil {
		t.Fatal(err)
	}
	if len(lookup.Namespaces) != 1 || lookup.Namespaces["pr"].Kind != "url" || len(lookup.Stores) != 0 || len(lookup.Undeclared) != 1 || lookup.Undeclared[0] != "legacy" {
		t.Fatalf("offered lookup = %+v", lookup)
	}
	if _, err := ParseReferenceLookup(append([]byte{}, art.Patch...), art.References); err != nil {
		t.Fatalf("exact patch rejected: %v", err)
	}
	if _, err := ParseReferenceLookup(append(append([]byte{}, art.Patch...), '\n'), art.References); err == nil || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("altered patch accepted: %v", err)
	}
	changed := strings.Replace(string(art.References), `"legacy"`, `"other"`, 1)
	if _, err := ParseReferenceLookup(art.Patch, []byte(changed)); err == nil || !strings.Contains(err.Error(), "namespace") {
		t.Fatalf("wrong namespace use accepted: %v", err)
	}
}

func TestMappingChoicesWriteRegistryAndRewriteAdoptedRefs(t *testing.T) {
	art, _ := lookupFixture(t)
	dest := newTestStore(t)
	choices := []MappingSelection{{Namespace: "pr", Action: "alias", Local: "source-pr"}}
	plan, err := dest.PlanReferenceMappings(art.Patch, art.References, choices)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Changed || plan.Rewrites["pr"] != "source-pr" || plan.Offers[0].Destination != "https://git.local.example/team/docs/pulls/12" {
		t.Fatalf("mapping plan = %+v", plan)
	}
	if _, err := dest.ApplyReferenceMappings(context.Background(), plan, "sha256:wrong"); CodeOf(err) != CodeStaleRevision {
		t.Fatalf("stale map revision accepted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest.Path(), referencesFile)); !os.IsNotExist(err) {
		t.Fatalf("stale precondition wrote registry: %v", err)
	}
	result, err := dest.ApplyReferenceMappings(context.Background(), plan, plan.MapRevision)
	if err != nil || !result.Changed {
		t.Fatalf("mapping write: %+v %v", result, err)
	}
	registry, err := dest.ReadReferenceRegistry()
	if err != nil || registry.Namespaces["source-pr"].Kind != "url" {
		t.Fatalf("registry after write: %+v %v", registry, err)
	}
	importPlan, err := dest.PlanImport(context.Background(), ImportOptions{
		Patch: string(art.Patch), ReferenceRewrites: plan.Rewrites, ReferenceRegistry: &plan.Registry, Actor: testActor,
	})
	if err != nil {
		t.Fatal(err)
	}
	refs := importPlan.Tickets[0].Refs
	if refs[0].Ref != "source-pr:team/docs#12" || refs[1].Ref != "legacy:opaque" {
		t.Fatalf("adopted refs = %+v", refs)
	}
	filed, err := dest.ApplyImport(context.Background(), importPlan)
	if err != nil || len(filed.Filed) != 1 {
		t.Fatalf("ticket adoption after mapping write: %+v %v", filed, err)
	}
	landed, err := dest.Get(context.Background(), filed.Filed[0].ID)
	if err != nil || landed.References[0].Ref != "source-pr:team/docs#12" {
		t.Fatalf("adopted ticket lost its mapped reference: %+v %v", landed, err)
	}
}

func TestConflictingMappingNeedsOpaqueTargetForTicketAdoption(t *testing.T) {
	art, _ := lookupFixture(t)
	dest := newTestStore(t)
	putRegistry(t, dest, strings.Replace(exampleRegistry, "https://git.local.example/{owner}/{repo}/pulls/{number}", "https://another.example/{owner}/{repo}/pulls/{number}", 1))
	preview, err := dest.PlanReferenceMappings(art.Patch, art.References, nil)
	if err != nil || len(preview.Unresolved) != 1 || preview.Unresolved[0] != "pr" {
		t.Fatalf("conflict preview: %+v %v", preview, err)
	}
	choices := []MappingSelection{{Namespace: "pr", Action: "decline", Local: "unmapped-pr"}}
	plan, err := dest.PlanReferenceMappings(art.Patch, art.References, choices)
	if err != nil || plan.Changed || plan.Rewrites["pr"] != "unmapped-pr" {
		t.Fatalf("opaque decline: %+v %v", plan, err)
	}
}

func TestTicketStoreAliasSeparatesConflictingStoreKeys(t *testing.T) {
	src := newTestStore(t)
	putRegistry(t, src, exampleRegistry)
	tk := mustCreate(t, src, "Foreign ticket link")
	mustApply(t, src, tk.ID, AddReference{Ref: "ledger-ticket:TKT-01K3ZYEE00HV9ZDBB8BEASXBBG"})
	art, err := src.Export(context.Background(), ExportOptions{IDs: []string{tk.ID}, From: "Sender <sender@example.invalid>", Now: fixedClock()()})
	if err != nil {
		t.Fatal(err)
	}
	dest := newTestStore(t)
	putRegistry(t, dest, "version: 1\nstores:\n  ledger:\n    repository: https://another.invalid/team/ledger.git\n    path: .tickets\n    browse: https://another.invalid/team/ledger/search?q={id}\nnamespaces: {}\n")
	plan, err := dest.PlanReferenceMappings(art.Patch, art.References, []MappingSelection{{Namespace: "ledger-ticket", Action: "alias", Local: "source-ticket"}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Registry.Namespaces["source-ticket"].Store != "source-ticket_ledger" || plan.Registry.Stores["source-ticket_ledger"].Repository != "https://git.local.example/team/ledger.git" {
		t.Fatalf("foreign store key collision was not separated: %+v", plan.Registry)
	}
	if plan.Registry.Stores["ledger"].Repository != "https://another.invalid/team/ledger.git" {
		t.Fatal("local store key was overwritten")
	}
}

func TestFromStoreAddsResolvableReferenceWhenDeclared(t *testing.T) {
	art, incomingID := lookupFixture(t)
	dest := newTestStore(t)
	putRegistry(t, dest, exampleRegistry)
	registry, err := dest.ReadReferenceRegistry()
	if err != nil {
		t.Fatal(err)
	}
	plan, err := dest.PlanImport(context.Background(), ImportOptions{
		Patch: string(art.Patch), FromStore: "ledger", ReferenceRegistry: registry,
	})
	if err != nil {
		t.Fatal(err)
	}
	refs := plan.Tickets[0].Refs
	want := "ledger-ticket:" + incomingID
	found := false
	for _, ref := range refs {
		found = found || ref.Ref == want
	}
	if !found {
		t.Fatalf("declared foreign-ticket provenance %q missing from %+v", want, refs)
	}
}

func TestUndeclaredIncomingNamespaceCannotAcquireLocalMeaning(t *testing.T) {
	art, _ := lookupFixture(t)
	dest := newTestStore(t)
	putRegistry(t, dest, "version: 1\nstores: {}\nnamespaces:\n  legacy:\n    kind: url\n    identifier: '(?P<slug>[a-z]+)'\n    template: https://local.invalid/{slug}\n")
	preview, err := dest.PlanReferenceMappings(art.Patch, art.References, nil)
	if err != nil || len(preview.Unresolved) != 1 || preview.Unresolved[0] != "legacy" {
		t.Fatalf("undeclared collision preview: %+v %v", preview, err)
	}
	plan, err := dest.PlanReferenceMappings(art.Patch, art.References, []MappingSelection{{Namespace: "legacy", Action: "decline", Local: "source-legacy"}})
	if err != nil || plan.Rewrites["legacy"] != "source-legacy" {
		t.Fatalf("opaque rewrite for undeclared ref: %+v %v", plan, err)
	}
}

func TestDecliningIdenticalMappingKeepsExistingResolution(t *testing.T) {
	art, _ := lookupFixture(t)
	dest := newTestStore(t)
	putRegistry(t, dest, exampleRegistry)
	plan, err := dest.PlanReferenceMappings(art.Patch, art.References, []MappingSelection{{Namespace: "pr", Action: "decline"}})
	if err != nil || plan.Changed || len(plan.Rewrites) != 0 || plan.Offers[0].Existing != "identical" {
		t.Fatalf("identical decline: %+v %v", plan, err)
	}
	resolved, err := dest.ResolveReference(context.Background(), Reference{Ref: "pr:team/docs#12"})
	if err != nil || resolved == nil || resolved.URL != "https://git.local.example/team/docs/pulls/12" {
		t.Fatalf("existing local meaning did not remain active: %+v %v", resolved, err)
	}
}

func TestMappingWriteSurvivesPartialTicketAdoption(t *testing.T) {
	src := newTestStore(t)
	putRegistry(t, src, exampleRegistry)
	first := mustCreate(t, src, "First incoming ticket")
	second := mustCreate(t, src, "Second incoming ticket")
	mustApply(t, src, first.ID, AddReference{Ref: "pr:team/docs#12"})
	art, err := src.Export(context.Background(), ExportOptions{IDs: []string{first.ID, second.ID}, From: "Sender <sender@example.invalid>", Now: fixedClock()()})
	if err != nil {
		t.Fatal(err)
	}
	dest := newTestStore(t)
	mappings, err := dest.PlanReferenceMappings(art.Patch, art.References, []MappingSelection{{Namespace: "pr", Action: "adopt"}})
	if err != nil {
		t.Fatal(err)
	}
	mapResult, err := dest.ApplyReferenceMappings(context.Background(), mappings, "")
	if err != nil || !mapResult.Changed {
		t.Fatalf("mapping write: %+v %v", mapResult, err)
	}
	plan, err := dest.PlanImport(context.Background(), ImportOptions{Patch: string(art.Patch), Actor: testActor, ReferenceRegistry: &mappings.Registry})
	if err != nil {
		t.Fatal(err)
	}
	plan.Tickets[1].Create.Title = "" // simulate a later ticket refusing to file
	filed, err := dest.ApplyImport(context.Background(), plan)
	if err == nil || filed == nil || len(filed.Filed) != 1 {
		t.Fatalf("partial adoption result: %+v %v", filed, err)
	}
	if filed.Filed[0].FromID != plan.Tickets[0].Incoming.ID {
		t.Fatalf("partial result lost the source ID: %+v", filed.Filed)
	}
	registry, err := dest.ReadReferenceRegistry()
	if err != nil || registry.Namespaces["pr"].Kind != "url" {
		t.Fatalf("accepted mapping was lost after a ticket failure: %+v %v", registry, err)
	}
}

func TestApplyImportRejectsChangedRegistryBeforeFiling(t *testing.T) {
	art, _ := lookupFixture(t)
	dest := newTestStore(t)
	plan, err := dest.PlanImport(context.Background(), ImportOptions{Patch: string(art.Patch), Actor: testActor})
	if err != nil {
		t.Fatal(err)
	}
	putRegistry(t, dest, exampleRegistry)
	result, err := dest.ApplyImport(context.Background(), plan)
	if CodeOf(err) != CodeStaleRevision || result == nil || len(result.Filed) != 0 {
		t.Fatalf("changed registry filed tickets: %+v %v", result, err)
	}
	all, err := dest.List(context.Background(), Filter{All: true})
	if err != nil || len(all) != 0 {
		t.Fatalf("stale import changed ticket store: %+v %v", all, err)
	}
}

func TestMappingPlanKeepsRegistryAndRevisionFromOneSnapshot(t *testing.T) {
	art, _ := lookupFixture(t)
	dest := newTestStore(t)
	first, err := ParseReferenceRegistry([]byte(exampleRegistry))
	if err != nil {
		t.Fatal(err)
	}
	second, err := ParseReferenceRegistry([]byte(exampleRegistry))
	if err != nil {
		t.Fatal(err)
	}
	pr := second.Namespaces["pr"]
	pr.Template = "https://other.invalid/{owner}/{repo}/pulls/{number}"
	second.Namespaces["pr"] = pr
	a, err := RenderReferenceRegistry(*first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := RenderReferenceRegistry(*second)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dest.Path(), referencesFile)
	if err := os.WriteFile(file, a, 0o644); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		written := 0
		for i := 0; i < 300; i++ {
			err := writeFileAtomic(file, [][]byte{a, b}[i%2])
			// Windows refuses to rename over a file another handle holds open,
			// and the reader below opens this one on every pass. A refused
			// write leaves the old bytes whole, which is a snapshot the reader
			// must still agree with, so it is not this test's failure.
			if runtime.GOOS == "windows" && errors.Is(err, fs.ErrPermission) {
				continue
			}
			if err != nil {
				done <- err
				return
			}
			written++
		}
		// If every write were refused, the reader would only ever see the
		// first revision and the test would prove nothing.
		if written < 2 {
			done <- fmt.Errorf("only %d of 300 registry writes landed", written)
			return
		}
		done <- nil
	}()
	for i := 0; i < 300; i++ {
		plan, err := dest.PlanReferenceMappings(art.Patch, art.References, nil)
		if err != nil {
			t.Fatal(err)
		}
		template := plan.Registry.Namespaces["pr"].Template
		if template == first.Namespaces["pr"].Template && plan.MapRevision == Revision(a) {
			continue
		}
		if template == second.Namespaces["pr"].Template && plan.MapRevision == Revision(b) {
			continue
		}
		t.Fatalf("parsed registry %q with unrelated revision %q", template, plan.MapRevision)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}
