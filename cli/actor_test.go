package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setActors rewrites config.yml with the given roster and declared default.
// A store expresses both only through that file, so a test that wants a
// particular resolution branch has to write one.
func setActors(t *testing.T, dir string, actors []string, declared string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("schema: 1\n")
	if len(actors) == 0 {
		b.WriteString("actors: []\n")
	} else {
		b.WriteString("actors:\n")
		for _, a := range actors {
			b.WriteString("  - id: " + a + "\n    name: \"\"\n")
		}
	}
	b.WriteString("labels: []\nmilestones: []\n")
	b.WriteString("defaults:\n  type: task\n  priority: normal\n")
	if declared == "" {
		b.WriteString("  actor: null\n")
	} else {
		b.WriteString("  actor: " + declared + "\n")
	}
	b.WriteString("  claim_expiry: null\nlock:\n  timeout: 10s\n")

	path := filepath.Join(dir, ".tickets", "config.yml")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write config.yml: %v", err)
	}
}

// createNoActor files a ticket without naming an actor, which is the call an
// agent makes when it has not read the instructions.
func createNoActor(t *testing.T, dir string, extra ...string) result {
	t.Helper()
	args := append([]string{"create", "--title", "Filed by somebody", "--description", "d"}, extra...)
	return runCLI(t, dir, nil, args...)
}

// TestActorResolutionWarnsOnlyWhenNobodyChose covers all four branches of plan
// 4.1 in one table, because the branches are only meaningful against each
// other: the warning earns its place by staying quiet in three of them.
func TestActorResolutionWarnsOnlyWhenNobodyChose(t *testing.T) {
	for _, c := range []struct {
		name     string
		actors   []string
		declared string
		flag     []string
		wantWarn bool
		wantOK   bool
	}{
		{
			name:   "incidental, so it warns",
			actors: []string{"human:sothr"},
			// Nothing chose this actor. It is merely first.
			wantWarn: true, wantOK: true,
		},
		{
			name:   "an explicit --actor is silent",
			actors: []string{"human:sothr"},
			flag:   []string{"--actor", "agent:terva/mieli"},
			// The caller said who it was, so there is nothing to report.
			wantWarn: false, wantOK: true,
		},
		{
			name:     "a declared default is silent",
			actors:   []string{"human:sothr"},
			declared: "human:sothr",
			// This is the single-writer store opting out, which is the whole
			// reason defaults.actor exists.
			wantWarn: false, wantOK: true,
		},
		{
			name:     "a declared default outside the roster still counts",
			actors:   []string{"human:sothr"},
			declared: "agent:terva/nightly",
			// Declaring is the deliberate act. Membership of the roster is not,
			// exactly as --actor accepts an ID the roster does not list.
			wantWarn: false, wantOK: true,
		},
		{
			name:   "nothing at all is refused rather than warned",
			actors: nil,
			// A warning would be the wrong shape here: there is no actor to
			// record, so the write cannot happen at all.
			wantWarn: false, wantOK: false,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := newStore(t)
			setActors(t, dir, c.actors, c.declared)

			got := createNoActor(t, dir, c.flag...)

			if c.wantOK && got.code != exitOK {
				t.Fatalf("create exited %d: %s%s", got.code, got.stdout, got.stderr)
			}
			if !c.wantOK {
				if got.code == exitOK {
					t.Fatalf("create succeeded with no actor anywhere: %s", got.stdout)
				}
				if !strings.Contains(got.stderr, "invalid_field") {
					t.Errorf("stderr does not carry invalid_field:\n%s", got.stderr)
				}
			}

			warned := strings.Contains(got.stderr, "no --actor given")
			if warned != c.wantWarn {
				t.Errorf("warned = %v, want %v\nstderr: %s", warned, c.wantWarn, got.stderr)
			}
		})
	}
}

// TestDefaultActorWarningNamesTheActorAndTheFix is what makes the warning worth
// printing. Saying that something happened without saying to whom, or what to
// do about it, spends the reader's attention and returns nothing.
func TestDefaultActorWarningNamesTheActorAndTheFix(t *testing.T) {
	dir := newStore(t)
	setActors(t, dir, []string{"human:sothr", "agent:terva/other"}, "")

	got := createNoActor(t, dir)
	if got.code != exitOK {
		t.Fatalf("create exited %d: %s", got.code, got.stderr)
	}
	for _, want := range []string{"human:sothr", "--actor", "defaults.actor"} {
		if !strings.Contains(got.stderr, want) {
			t.Errorf("the warning does not carry %q:\n%s", want, got.stderr)
		}
	}
	// The first entry and not any other, since that is the one the store uses.
	if strings.Contains(got.stderr, "agent:terva/other") {
		t.Errorf("the warning names an actor the write will not use:\n%s", got.stderr)
	}
}

// TestDefaultActorWarningReachesEveryWrite holds the warning to the chokepoint
// rather than to one command. create and a mutation take different paths into
// the store, and a warning wired to only one of them is the one a caller
// trusted it for.
func TestDefaultActorWarningReachesEveryWrite(t *testing.T) {
	dir := newStore(t)
	setActors(t, dir, []string{"human:sothr"}, "")
	id := ticketID(t, decode(t, runCLI(t, dir, nil, "--json", "create",
		"--title", "Something to mutate", "--actor", "agent:terva/mieli").stdout))

	for _, args := range [][]string{
		{"note", id, "a note with no actor"},
		{"update", id, "--priority", "high"},
		{"status", id, "ready"},
	} {
		got := runCLI(t, dir, nil, args...)
		if got.code != exitOK {
			t.Fatalf("%v exited %d: %s", args, got.code, got.stderr)
		}
		if !strings.Contains(got.stderr, "no --actor given") {
			t.Errorf("%v wrote without warning:\n%s", args, got.stderr)
		}
	}
}

// TestDefaultActorWarningStaysOffStdout is the constraint that makes this safe
// to emit at all. A consumer parsing stdout must not have to strip prose from
// it, and this warning fires on the most ordinary call there is.
func TestDefaultActorWarningStaysOffStdout(t *testing.T) {
	dir := newStore(t)
	setActors(t, dir, []string{"human:sothr"}, "")

	got := createNoActor(t, dir, "--json")
	if got.code != exitOK {
		t.Fatalf("create exited %d: %s", got.code, got.stderr)
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(got.stdout), &envelope); err != nil {
		t.Fatalf("the warning got into stdout: %v\n%s", err, got.stdout)
	}
	if envelope["kind"] != "mutation-result" {
		t.Errorf("kind = %v, want mutation-result", envelope["kind"])
	}
	if !strings.Contains(got.stderr, "no --actor given") {
		t.Errorf("the warning did not reach stderr:\n%s", got.stderr)
	}
}

// TestDeclaredDefaultActorIsWhoTheWriteRecords checks the branch does what it
// claims rather than merely staying quiet. Silencing a warning by recording the
// wrong actor would be worse than the warning.
func TestDeclaredDefaultActorIsWhoTheWriteRecords(t *testing.T) {
	dir := newStore(t)
	setActors(t, dir, []string{"human:sothr"}, "agent:terva/nightly")

	got := createNoActor(t, dir, "--json")
	if got.code != exitOK {
		t.Fatalf("create exited %d: %s", got.code, got.stderr)
	}
	id := ticketID(t, decode(t, got.stdout))

	shown := decode(t, runCLI(t, dir, nil, "--json", "show", id).stdout)
	tk, _ := shown["ticket"].(map[string]any)
	created, _ := tk["createdBy"].(map[string]any)
	if created["id"] != "agent:terva/nightly" {
		t.Errorf("createdBy = %v, want the declared default", created)
	}
}

// TestConfigPublishesTheDeclaredDefaultActor keeps `config` honest about the
// field, since that command exists so nobody has to read config.yml.
func TestConfigPublishesTheDeclaredDefaultActor(t *testing.T) {
	dir := newStore(t)
	setActors(t, dir, []string{"human:sothr"}, "agent:terva/nightly")

	envelope := decode(t, runCLI(t, dir, nil, "--json", "config").stdout)
	defaults, _ := envelope["defaults"].(map[string]any)
	if defaults["actor"] != "agent:terva/nightly" {
		t.Errorf("defaults.actor = %v, want the declared default", defaults["actor"])
	}

	// Undeclared reads as null rather than as the actor the store would fall
	// back to, because those are different facts.
	setActors(t, dir, []string{"human:sothr"}, "")
	envelope = decode(t, runCLI(t, dir, nil, "--json", "config").stdout)
	defaults, _ = envelope["defaults"].(map[string]any)
	if defaults["actor"] != nil {
		t.Errorf("defaults.actor = %v, want null when nothing is declared", defaults["actor"])
	}
}
