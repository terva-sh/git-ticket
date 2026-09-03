package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAnExistingDirectoryIsNotAStore is the CLI half of plan 4.
//
// TestStorePrecedence already covers a --store path that does not exist. This
// is the case that used to succeed: the path resolves, the directory is there,
// and it is not a store. Both ways of naming one have to refuse it, because a
// typo reaches the tool through either.
func TestAnExistingDirectoryIsNotAStore(t *testing.T) {
	notAStore := t.TempDir()
	if err := os.WriteFile(filepath.Join(notAStore, "notes.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	elsewhere := t.TempDir()

	for _, c := range []struct {
		name string
		env  map[string]string
		args []string
	}{
		{"--store", nil, []string{"--json", "--store", notAStore, "list"}},
		{"GIT_TICKET_STORE", map[string]string{"GIT_TICKET_STORE": notAStore}, []string{"--json", "list"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := runCLI(t, elsewhere, c.env, c.args...)
			if got.code != exitError {
				t.Fatalf("exit = %d, want %d. A directory that is not a store answered:\n%s",
					got.code, exitError, got.stdout)
			}
			errObj, ok := decode(t, got.stdout)["error"].(map[string]any)
			if !ok {
				t.Fatalf("no error object in:\n%s", got.stdout)
			}
			if errObj["code"] != "store_not_found" {
				t.Errorf("code = %v, want store_not_found", errObj["code"])
			}
			// The message names config.yml, so a reader learns what would make
			// the directory a store rather than only that it is not one.
			if msg, _ := errObj["message"].(string); !strings.Contains(msg, "config.yml") {
				t.Errorf("message does not name config.yml: %q", msg)
			}
		})
	}
}

// TestListAndReadyRefuseANonStore holds the behaviour the rule exists for.
//
// The old answers were the danger. `list` emitted ticket-list with an empty
// array and `ready` printed that nothing was ready, both at exit 0, so an agent
// handed the wrong path concluded there was no work. Asserting the exit code
// alone would not catch a regression that kept exit 0 and changed the wording.
func TestListAndReadyRefuseANonStore(t *testing.T) {
	notAStore := t.TempDir()
	elsewhere := t.TempDir()

	for _, command := range []string{"list", "ready"} {
		t.Run(command, func(t *testing.T) {
			got := runCLI(t, elsewhere, nil, "--store", notAStore, command)
			if got.code != exitError {
				t.Fatalf("%s exit = %d, want %d", command, got.code, exitError)
			}
			if strings.Contains(got.stdout, "No tickets") || strings.Contains(got.stdout, "Nothing is ready") {
				t.Errorf("%s answered about work rather than about the store:\n%s", command, got.stdout)
			}
			if !strings.Contains(got.stderr, "store_not_found") {
				t.Errorf("%s stderr does not report store_not_found:\n%s", command, got.stderr)
			}
		})
	}
}

// TestCheckRefusesANonStoreBeforeReportingOnIt covers the finding that used to
// answer this question by accident.
//
// `check --strict` already failed on an empty directory, but through
// epics_index_stale, whose message says epics.md "does not match the epics in
// this store" while failing to be a store. Plain `check` exited 0 on the same
// directory, because that finding is a warning. Both now stop at the store.
func TestCheckRefusesANonStoreBeforeReportingOnIt(t *testing.T) {
	notAStore := t.TempDir()
	elsewhere := t.TempDir()

	for _, args := range [][]string{
		{"--store", notAStore, "check"},
		{"--store", notAStore, "check", "--strict"},
	} {
		t.Run(strings.Join(args[2:], " "), func(t *testing.T) {
			got := runCLI(t, elsewhere, nil, args...)
			if got.code != exitError {
				t.Fatalf("exit = %d, want %d:\n%s", got.code, exitError, got.stdout)
			}
			if strings.Contains(got.stdout, "epics_index_stale") || strings.Contains(got.stderr, "epics_index_stale") {
				t.Errorf("check blamed the epics index for a directory that is not a store:\n%s%s",
					got.stdout, got.stderr)
			}
			if !strings.Contains(got.stderr, "store_not_found") {
				t.Errorf("stderr does not report store_not_found:\n%s", got.stderr)
			}
		})
	}
}
