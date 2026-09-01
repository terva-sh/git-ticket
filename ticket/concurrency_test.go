package ticket

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
)

// The Phase 1 exit criterion says two concurrent processes, not two goroutines.
// Goroutines exercise the same kernel path, since flock belongs to the open
// file description and each writer opens its own, but only separate processes
// prove it. These tests re-execute the test binary to get them.

const (
	envWriterStore    = "GIT_TICKET_TEST_STORE"
	envWriterRef      = "GIT_TICKET_TEST_REF"
	envWriterRevision = "GIT_TICKET_TEST_REVISION"
	envWriterTitle    = "GIT_TICKET_TEST_TITLE"
)

// TestWriterHelper is not a test. It is the child process of
// TestTwoProcessesOneWins, and it skips unless that test started it.
func TestWriterHelper(t *testing.T) {
	store := os.Getenv(envWriterStore)
	if store == "" {
		t.Skip("the child half of TestTwoProcessesOneWins")
	}
	s, err := Open(store)
	if err != nil {
		t.Fatalf("RESULT:open-failed %v", err)
	}
	_, err = s.Apply(context.Background(), os.Getenv(envWriterRef),
		SetTitle{Title: os.Getenv(envWriterTitle)},
		ApplyOptions{Actor: testActor, IfRevision: os.Getenv(envWriterRevision)})

	// The parent reads this line. An exit code would be shorter but the test
	// harness owns the exit code.
	switch {
	case err == nil:
		t.Log("RESULT:ok")
	case CodeOf(err) != "":
		t.Logf("RESULT:%s", CodeOf(err))
	default:
		t.Logf("RESULT:unexpected %v", err)
	}
}

// TestTwoProcessesOneWins is the Phase 1 exit criterion in full: two processes
// writing the same ticket with the same precondition produce one success and
// one stale_revision.
func TestTwoProcessesOneWins(t *testing.T) {
	if os.Getenv(envWriterStore) != "" {
		t.Skip("this is the child process")
	}
	s := newTestStore(t)
	tk := mustCreate(t, s, "Contended by two processes")

	const writers = 2
	var wg sync.WaitGroup
	outputs := make([]string, writers)
	for i := 0; i < writers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cmd := exec.Command(os.Args[0], "-test.run=TestWriterHelper", "-test.v")
			cmd.Env = append(os.Environ(),
				envWriterStore+"="+s.Path(),
				envWriterRef+"="+tk.ID,
				envWriterRevision+"="+tk.Revision,
				envWriterTitle+"=written by process "+string(rune('A'+i)),
			)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("writer %d: %v\n%s", i, err, out)
			}
			outputs[i] = string(out)
		}(i)
	}
	wg.Wait()

	var won, stale int
	for i, out := range outputs {
		switch {
		case strings.Contains(out, "RESULT:ok"):
			won++
		case strings.Contains(out, "RESULT:"+CodeStaleRevision):
			stale++
		default:
			t.Errorf("writer %d reported neither success nor a stale revision:\n%s", i, out)
		}
	}
	if won != 1 || stale != 1 {
		t.Errorf("%d won and %d were stale, want exactly one of each", won, stale)
	}

	// Whichever won, the store is left holding one coherent ticket.
	got, err := s.Get(context.Background(), tk.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.Title, "written by process ") {
		t.Errorf("title = %q, want one of the two writers' titles", got.Title)
	}
	report, err := s.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !report.OK() {
		t.Errorf("the store did not survive the race: %+v", report.Errors)
	}
}
