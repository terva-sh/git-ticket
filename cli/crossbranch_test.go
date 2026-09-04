package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// crossBranchCommitDate is when every commit in these tests was made. It has to
// be pinned rather than left to the clock: the ref scan skips a ref whose last
// commit is older than 30 days, and it measures that against the store's
// injected now, which is referenceInstant. A commit dated by the machine would
// put the window's edge wherever the suite happened to run, so this sits five
// days before referenceInstant and the scan reads every ref in these tests.
const crossBranchCommitDate = "2026-09-25T00:00:00Z"

// gitAt runs git with a fixed identity and a fixed commit date.
func gitAt(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com",
		"GIT_AUTHOR_DATE="+crossBranchCommitDate,
		"GIT_COMMITTER_DATE="+crossBranchCommitDate,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// crossRows decodes the rows of a ticket-list envelope.
func crossRows(t *testing.T, r result) []map[string]any {
	t.Helper()
	if r.code != exitOK {
		t.Fatalf("command failed with %d: %s%s", r.code, r.stdout, r.stderr)
	}
	var env struct {
		Tickets []map[string]any `json:"tickets"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &env); err != nil {
		t.Fatalf("decode: %v\n%s", err, r.stdout)
	}
	return env.Tickets
}

// crossRow finds one row by title.
func crossRow(t *testing.T, rows []map[string]any, title string) map[string]any {
	t.Helper()
	for _, row := range rows {
		if row["title"] == title {
			return row
		}
	}
	t.Fatalf("no row titled %q in %d rows", title, len(rows))
	return nil
}

func crossTitles(rows []map[string]any) []string {
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if s, ok := row["title"].(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// newCrossStore builds a repository whose main branch holds one ready ticket,
// and returns the directory and that ticket's ID.
func newCrossStore(t *testing.T, title string) (string, string) {
	t.Helper()
	dir := newGitStore(t)
	id := crossCreate(t, dir, title, "human:sothr")
	if got := runCLI(t, dir, nil, "status", id, "ready", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("status ready: %s%s", got.stdout, got.stderr)
	}
	gitAt(t, dir, "add", "-A")
	gitAt(t, dir, "commit", "-qm", "on main")
	return dir, id
}

// crossCreate files a ticket and returns its ID.
func crossCreate(t *testing.T, dir, title, actor string) string {
	t.Helper()
	got := runCLI(t, dir, nil, "--json", "create", "--title", title, "--actor", actor)
	if got.code != exitOK {
		t.Fatalf("create %q: %s%s", title, got.stdout, got.stderr)
	}
	var env struct {
		Ticket struct {
			ID string `json:"id"`
		} `json:"ticket"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &env); err != nil {
		t.Fatalf("decode create: %v\n%s", err, got.stdout)
	}
	if env.Ticket.ID == "" {
		t.Fatalf("create returned no id: %s", got.stdout)
	}
	return env.Ticket.ID
}

// TestCrossBranchListFindsATicketOnAnotherRef is the first acceptance criterion
// of TKT-01M1Q54B, and the third alongside it: a ticket that exists only on
// another ref appears with --cross-branch and names the ref it came from, and
// every row of an ordinary query carries a null branch.
func TestCrossBranchListFindsATicketOnAnotherRef(t *testing.T) {
	dir, _ := newCrossStore(t, "Lives on main")

	gitAt(t, dir, "switch", "-qc", "feat/other")
	crossCreate(t, dir, "Only on the branch", "agent:terva/other")
	gitAt(t, dir, "add", "-A")
	gitAt(t, dir, "commit", "-qm", "on the branch")
	gitAt(t, dir, "switch", "-q", "main")

	// Without the flag the branch ticket is invisible and nothing claims a ref.
	plain := crossRows(t, runCLI(t, dir, nil, "--json", "list"))
	for _, row := range plain {
		if row["branch"] != nil {
			t.Errorf("an ordinary listing carried branch %v, want null", row["branch"])
		}
	}
	if titles := crossTitles(plain); len(titles) != 1 || titles[0] != "Lives on main" {
		t.Errorf("ordinary listing = %v, want only the main ticket", titles)
	}

	// With it the branch ticket arrives, naming the ref.
	wide := crossRows(t, runCLI(t, dir, nil, "--json", "list", "--cross-branch"))
	row := crossRow(t, wide, "Only on the branch")
	if row["branch"] != "feat/other" {
		t.Errorf("branch = %v, want feat/other", row["branch"])
	}
	if crossRow(t, wide, "Lives on main")["branch"] != nil {
		t.Error("the working-tree row carried a branch, want null")
	}
}

// TestCrossBranchReadyHonoursAClaimOnAnotherRef is the second acceptance
// criterion. The failure this whole feature exists to prevent is two agents
// claiming one ticket because neither could see the other, so a live claim on
// any scanned ref has to take the ticket out of ready.
func TestCrossBranchReadyHonoursAClaimOnAnotherRef(t *testing.T) {
	dir, id := newCrossStore(t, "Contested")

	// Another agent claims it on their own branch and commits.
	gitAt(t, dir, "switch", "-qc", "feat/claimer")
	if got := runCLI(t, dir, nil, "claim", id, "--actor", "agent:terva/other"); got.code != exitOK {
		t.Fatalf("claim: %s%s", got.stdout, got.stderr)
	}
	gitAt(t, dir, "add", "-A")
	gitAt(t, dir, "commit", "-qm", "claimed elsewhere")
	gitAt(t, dir, "switch", "-q", "main")

	// The working tree still says it is free, which is the bug.
	if titles := crossTitles(crossRows(t, runCLI(t, dir, nil, "--json", "ready"))); len(titles) != 1 {
		t.Fatalf("ready without the flag = %v, want the ticket to still look free", titles)
	}

	// The wider read sees the claim.
	if titles := crossTitles(crossRows(t, runCLI(t, dir, nil, "--json", "ready", "--cross-branch"))); len(titles) != 0 {
		t.Errorf("ready --cross-branch = %v, want nothing while another ref holds a claim", titles)
	}

	// And the listing explains itself rather than only omitting the row. The
	// claim is honoured whichever copy won the display, because a claim is
	// never adjudicated: it is reported, not resolved.
	row := crossRow(t, crossRows(t, runCLI(t, dir, nil, "--json", "list", "--cross-branch")), "Contested")
	readiness, ok := row["readiness"].(map[string]any)
	if !ok {
		t.Fatalf("no readiness on the row: %v", row)
	}
	if readiness["isReady"] != false {
		t.Errorf("isReady = %v, want false", readiness["isReady"])
	}
	if readiness["reason"] != "claimed" {
		t.Errorf("reason = %v, want claimed", readiness["reason"])
	}
}

// TestCrossBranchReadsRemoteTrackingRefs is the fourth acceptance criterion,
// and it is the case the feature was actually filed for. Agents here push to a
// remote, so in every other worktree their work is a remote-tracking ref and
// never a local head. The local branch is deleted so the only copy left is
// under refs/remotes/.
func TestCrossBranchReadsRemoteTrackingRefs(t *testing.T) {
	dir, _ := newCrossStore(t, "Lives on main")

	gitAt(t, dir, "switch", "-qc", "feat/pushed")
	crossCreate(t, dir, "Only on the remote", "agent:terva/other")
	gitAt(t, dir, "add", "-A")
	gitAt(t, dir, "commit", "-qm", "on the branch")
	gitAt(t, dir, "switch", "-q", "main")

	bare := t.TempDir()
	gitAt(t, bare, "init", "-q", "--bare", ".")
	gitAt(t, dir, "remote", "add", "origin", bare)
	gitAt(t, dir, "push", "-q", "origin", "feat/pushed")
	gitAt(t, dir, "branch", "-qD", "feat/pushed")

	rows := crossRows(t, runCLI(t, dir, nil, "--json", "list", "--cross-branch"))
	row := crossRow(t, rows, "Only on the remote")
	if row["branch"] != "origin/feat/pushed" {
		t.Errorf("branch = %v, want origin/feat/pushed", row["branch"])
	}
}

// TestCrossBranchRowsAbbreviateUnambiguously pins a bug the first build had. The
// abbreviations were computed from the working tree while the listing also
// printed cross-branch rows, so a working-tree ID could shorten to a prefix that
// matched the row printed beneath it, and a branch-only row printed its whole ID
// because nothing shortened it. A printed prefix a reader can see two matches
// for is worse than a long ID.
func TestCrossBranchRowsAbbreviateUnambiguously(t *testing.T) {
	dir, _ := newCrossStore(t, "Lives on main")

	gitAt(t, dir, "switch", "-qc", "feat/other")
	crossCreate(t, dir, "Only on the branch", "agent:terva/other")
	gitAt(t, dir, "add", "-A")
	gitAt(t, dir, "commit", "-qm", "on the branch")
	gitAt(t, dir, "switch", "-q", "main")

	got := runCLI(t, dir, nil, "list", "--cross-branch")
	if got.code != exitOK {
		t.Fatalf("list: %s%s", got.stdout, got.stderr)
	}
	var printed []string
	for _, line := range strings.Split(strings.TrimSpace(got.stdout), "\n") {
		if fields := strings.Fields(line); len(fields) > 0 {
			printed = append(printed, fields[0])
		}
	}
	if len(printed) < 2 {
		t.Fatalf("expected at least two rows, got %q", got.stdout)
	}
	// Every printed prefix has to identify one row and not another. Comparing
	// each against every other catches the ambiguity whichever row carried it.
	for i, a := range printed {
		for j, b := range printed {
			if i == j {
				continue
			}
			if strings.HasPrefix(b, a) {
				t.Errorf("printed ID %q is a prefix of %q, so it names two rows", a, b)
			}
		}
	}
}

// TestCrossBranchRowsCarryAReadinessReason pins the other bug the first build
// had. A row living only on another ref is absent from a working-tree readiness
// map, so it came back as the zero value: not ready, not blocked, and no reason
// at all. Section 15 records that exact combination as the complaint readiness
// gained a reason to answer, and reaching it again from a wider listing would be
// the same defect with a new cause.
func TestCrossBranchRowsCarryAReadinessReason(t *testing.T) {
	dir, _ := newCrossStore(t, "Lives on main")

	gitAt(t, dir, "switch", "-qc", "feat/other")
	crossCreate(t, dir, "Only on the branch", "agent:terva/other")
	gitAt(t, dir, "add", "-A")
	gitAt(t, dir, "commit", "-qm", "on the branch")
	gitAt(t, dir, "switch", "-q", "main")

	rows := crossRows(t, runCLI(t, dir, nil, "--json", "list", "--cross-branch"))
	for _, row := range rows {
		readiness, ok := row["readiness"].(map[string]any)
		if !ok {
			t.Fatalf("row %v carries no readiness", row["title"])
		}
		if readiness["isReady"] == false && readiness["reason"] == "" {
			t.Errorf("row %v is not ready and says why with an empty string", row["title"])
		}
	}
}
