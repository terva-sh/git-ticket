package ticket

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMovedDonePrerequisiteGatesUntilManualResolution(t *testing.T) {
	s := newTestStore(t)
	source := mustCreate(t, s, "Work moving to another store")
	dependent := mustCreate(t, s, "Local work waiting on the move")
	replacement := mustCreate(t, s, "Replacement local prerequisite")
	mustApply(t, s, source.ID, SetStatus{Status: StatusDone, Reason: "local work ended"})
	mustApply(t, s, dependent.ID, AddDependency{ID: source.ID})
	mustApply(t, s, dependent.ID, SetStatus{Status: StatusReady})
	before, err := s.Readiness(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !before[dependent.ID].Ready {
		t.Fatal("done source should satisfy the dependency before it moves")
	}

	const dest = "ledger-ticket:TKT-01K4ADX6V0YWD5KSGS7500YDAP"
	moved := mustApply(t, s, source.ID, MoveTo{Ref: dest, Reason: "ownership transferred"})
	after, err := s.Readiness(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if after[dependent.ID].Ready || len(after[dependent.ID].Blocking) != 1 || after[dependent.ID].Blocking[0] != source.ID {
		t.Fatalf("moved done source must block dependent: %+v", after[dependent.ID])
	}
	report, err := s.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range report.Errors {
		if f.Code == CodeDependencyMoved && f.Ticket == dependent.ID && strings.Contains(f.Message, dest) {
			found = true
		}
	}
	if !found {
		t.Fatalf("check omitted the moved dependency: %+v", report.Errors)
	}

	_, err = s.ResolveMove(context.Background(), dependent.ID, ResolveMoveOptions{
		From: source.ID, IfSourceRevision: source.Revision, Reason: "foreign ticket inspected", Actor: testActor,
	})
	var stale *Error
	if !errors.As(err, &stale) || stale.Code != CodeStaleRevision || stale.Ticket != source.ID {
		t.Fatalf("stale source revision: %v", err)
	}

	resolved, err := s.ResolveMove(context.Background(), dependent.ID, ResolveMoveOptions{
		From: source.ID, IfSourceRevision: moved.Ticket.Revision, WaitOn: replacement.ID,
		Reason: "wait for local integration", Actor: testActor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved.Ticket.Dependencies) != 1 || resolved.Ticket.Dependencies[0] != replacement.ID ||
		len(resolved.Ticket.References) != 1 || resolved.Ticket.References[0].Ref != dest ||
		!strings.Contains(resolved.Ticket.Body.Notes, "wait for local integration") {
		t.Fatalf("resolution lost its decision: %+v", resolved.Ticket)
	}
	report, err = s.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range report.Errors {
		if f.Code == CodeDependencyMoved {
			t.Fatalf("resolved edge still reported: %+v", f)
		}
	}
	after, err = s.Readiness(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if after[dependent.ID].Ready || len(after[dependent.ID].Blocking) != 1 || after[dependent.ID].Blocking[0] != replacement.ID {
		t.Fatalf("replacement must continue gating: %+v", after[dependent.ID])
	}
}

func TestMoveRequiresExplicitSchemaFourMigration(t *testing.T) {
	s := newTestStore(t)
	s.config.Schema = 3
	if err := os.WriteFile(filepath.Join(s.path, configFile), RenderConfig(s.config), 0o644); err != nil {
		t.Fatal(err)
	}
	old := mustCreate(t, s, "Old schema source")
	_, err := s.Apply(context.Background(), old.ID, MoveTo{Ref: "ledger-ticket:destination", Reason: "transfer"}, ApplyOptions{Actor: testActor})
	var coded *Error
	if !errors.As(err, &coded) || coded.Code != CodeSchemaUnsupported {
		t.Fatalf("move below schema 4: %v", err)
	}
	if _, err := s.Migrate(context.Background(), MigrateOptions{}); err != nil {
		t.Fatal(err)
	}
	upgraded, err := s.Get(context.Background(), old.ID)
	if err != nil {
		t.Fatal(err)
	}
	if upgraded.Schema != 4 || upgraded.MovedTo != nil || !strings.Contains(string(Render(upgraded)), "moved_to: null") {
		t.Fatalf("migration did not add the schema-4 field: %+v", upgraded)
	}
	if _, err := s.Apply(context.Background(), old.ID, MoveTo{Ref: "ledger-ticket:destination", Reason: "transfer"}, ApplyOptions{Actor: testActor}); err != nil {
		t.Fatal(err)
	}
}

func TestMovedReadyOriginalIsNeverReady(t *testing.T) {
	ref := "ledger-ticket:destination"
	original := &Ticket{ID: "TKT-01K4AEAAA000000000000001", Status: StatusReady, MovedTo: &ref}
	r := readinessOf([]*Ticket{original}, referenceInstant, nil)[original.ID]
	if r.Ready || r.Reason != ReasonMoved {
		t.Fatalf("moved original: %+v", r)
	}
}
