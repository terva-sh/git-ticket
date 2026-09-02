package cli

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newGitStore makes a real repository with a store and one commit in it. A
// claim records the branch and the commit it was based on, so those tests need
// a repository rather than a bare directory.
func newGitStore(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		// A test must not depend on the developer's git identity, and commit
		// refuses without one.
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test", "GIT_COMMITTER_EMAIL=test@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	run("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "-A")
	run("commit", "-qm", "first")

	if got := runCLI(t, dir, nil, "init", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("init: %s", got.stderr)
	}
	return dir
}

// readyTicket creates a ticket and moves it to ready, which is the first status
// a claim is permitted from.
func readyTicket(t *testing.T, dir string) string {
	t.Helper()
	id := ticketID(t, createTicket(t, dir))
	if got := runCLI(t, dir, nil, "status", id, "ready", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("status ready: %s", got.stderr)
	}
	return id
}

func showTicket(t *testing.T, dir, id string) map[string]any {
	t.Helper()
	got := runCLI(t, dir, nil, "--json", "show", id)
	if got.code != exitOK {
		t.Fatalf("show: %s", got.stderr)
	}
	return decode(t, got.stdout)["ticket"].(map[string]any)
}

func errCode(t *testing.T, r result) string {
	t.Helper()
	body, ok := decode(t, r.stdout)["error"].(map[string]any)
	if !ok {
		t.Fatalf("no error envelope in:\n%s", r.stdout)
	}
	code, _ := body["code"].(string)
	return code
}

// TestStatusFollowsTheTransitionTable pins plan 6.2 at the CLI boundary. The
// table itself is the library's, so this checks that a refusal arrives with the
// code and the detail a caller needs to act on.
func TestStatusFollowsTheTransitionTable(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	// A new ticket is a draft, and draft goes to ready.
	if got := runCLI(t, dir, nil, "status", id, "ready", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("draft to ready: %s", got.stderr)
	}
	if s := showTicket(t, dir, id)["status"]; s != "ready" {
		t.Errorf("status = %v, want ready", s)
	}

	// Ready does not go straight to done.
	got := runCLI(t, dir, nil, "--json", "status", id, "done", "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("ready to done should be refused")
	}
	if code := errCode(t, got); code != "invalid_transition" {
		t.Errorf("code = %v, want invalid_transition", code)
	}
	// The refusal names where the ticket may go instead, so a caller does not
	// have to hold the table itself.
	details := decode(t, got.stdout)["error"].(map[string]any)["details"].(map[string]any)
	if permitted, _ := details["permitted"].(string); !strings.Contains(permitted, "in-progress") {
		t.Errorf("permitted = %q, want it to mention in-progress", permitted)
	}

	// An unknown status is caught before the store is touched.
	if got := runCLI(t, dir, nil, "--json", "status", id, "frobnicate"); errCode(t, got) != codeUsage {
		t.Error("an unknown status should be a usage error")
	}
	// status takes two arguments, not one.
	if got := runCLI(t, dir, nil, "--json", "status", id); errCode(t, got) != codeUsage {
		t.Error("status with no target should be a usage error")
	}
}

// TestStatusReasonRules covers the two transitions plan 6.2 will not let
// through without a reason, and where the reason lands.
func TestStatusReasonRules(t *testing.T) {
	dir := newStore(t)
	id := readyTicket(t, dir)

	// Blocked needs a reason.
	got := runCLI(t, dir, nil, "--json", "status", id, "blocked", "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("blocked with no reason should be refused")
	}
	if code := errCode(t, got); code != "invalid_field" {
		t.Errorf("code = %v, want invalid_field", code)
	}

	if got := runCLI(t, dir, nil, "status", id, "blocked",
		"--reason", "waiting on the vendor", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("blocked with a reason: %s", got.stderr)
	}
	tk := showTicket(t, dir, id)
	if tk["statusReason"] != "waiting on the vendor" {
		t.Errorf("statusReason = %v", tk["statusReason"])
	}
	// The reason lands in Notes as well, which is what survives the next
	// transition clearing the field.
	notes := tk["body"].(map[string]any)["notes"].(string)
	if !strings.Contains(notes, "waiting on the vendor") {
		t.Errorf("the reason did not reach Notes:\n%s", notes)
	}

	// Moving on without a reason clears the field but keeps the history.
	if got := runCLI(t, dir, nil, "status", id, "ready", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("blocked to ready: %s", got.stderr)
	}
	tk = showTicket(t, dir, id)
	if tk["statusReason"] != nil {
		t.Errorf("statusReason = %v, want null once the ticket moved on", tk["statusReason"])
	}
	if notes := tk["body"].(map[string]any)["notes"].(string); !strings.Contains(notes, "waiting on the vendor") {
		t.Error("Notes lost the history the status_reason field no longer carries")
	}
}

// TestStatusCannotArchive is the one status the command refuses, because
// archiving also moves the file.
func TestStatusCannotArchive(t *testing.T) {
	dir := newStore(t)
	id := readyTicket(t, dir)

	got := runCLI(t, dir, nil, "--json", "status", id, "archived", "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("status archived should be refused")
	}
	if code := errCode(t, got); code != "invalid_transition" {
		t.Errorf("code = %v, want invalid_transition", code)
	}
	if msg := decode(t, got.stdout)["error"].(map[string]any)["message"].(string); !strings.Contains(msg, "archive") {
		t.Errorf("the message should point at the archive command: %q", msg)
	}
}

// TestClaimRecordsWhereTheWorkIs is plan 6.4: the branch and worktree when they
// can be determined, and the commit the claim was based on.
func TestClaimRecordsWhereTheWorkIs(t *testing.T) {
	dir := newGitStore(t)
	id := readyTicket(t, dir)

	if got := runCLI(t, dir, nil, "claim", id, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("claim: %s", got.stderr)
	}
	claim, ok := showTicket(t, dir, id)["claim"].(map[string]any)
	if !ok {
		t.Fatal("the ticket carries no claim")
	}
	if claim["actor"] != "human:sothr" {
		t.Errorf("actor = %v", claim["actor"])
	}
	if claim["branch"] != "main" {
		t.Errorf("branch = %v, want main", claim["branch"])
	}
	if commit, _ := claim["commit"].(string); len(commit) != 40 {
		t.Errorf("commit = %v, want the full HEAD hash", claim["commit"])
	}
	if claim["worktree"] == nil {
		t.Error("worktree is null inside a repository")
	}
	// There is no default expiry, per 6.4.
	if claim["expiresAt"] != nil {
		t.Errorf("expiresAt = %v, want null with no --expires-in", claim["expiresAt"])
	}

	if got := runCLI(t, dir, nil, "claim", id, "--expires-in", "2h",
		"--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("claim --expires-in: %s", got.stderr)
	}
	if expires := showTicket(t, dir, id)["claim"].(map[string]any)["expiresAt"]; expires == nil {
		t.Error("expiresAt is null after --expires-in")
	}
}

// TestReclaimKeepsAnExpiryNothingReplaced covers plan 6.4 at the surface a
// person types. Re-running claim to stay alive on a long task must not turn a
// bounded claim into an unbounded one.
func TestReclaimKeepsAnExpiryNothingReplaced(t *testing.T) {
	dir := newGitStore(t)
	id := readyTicket(t, dir)

	if got := runCLI(t, dir, nil, "claim", id, "--expires-in", "2h",
		"--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("claim --expires-in: %s", got.stderr)
	}
	// Only the expiry is asserted here. runCLI pins Now to one instant, so a
	// preserved claimed_at and a freshly computed one are identical bytes and
	// the check could never fail. TestReclaimRenewsRatherThanReplaces in the
	// ticket package moves the clock and covers claimed_at properly.
	first := showTicket(t, dir, id)["claim"].(map[string]any)
	expires := first["expiresAt"]
	if expires == nil {
		t.Fatal("the first claim recorded no expiry")
	}

	// The same actor claims again and names no duration.
	if got := runCLI(t, dir, nil, "claim", id, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("re-claim: %s", got.stderr)
	}
	again := showTicket(t, dir, id)["claim"].(map[string]any)
	if again["expiresAt"] != expires {
		t.Errorf("expiresAt = %v after a re-claim, want it kept at %v", again["expiresAt"], expires)
	}
}

// TestShowSaysWhyATicketCannotBeRead covers plan section 8 at the surface a
// person types. A hand-edited ticket with a YAML typo must not report as a
// ticket that was never filed.
func TestShowSaysWhyATicketCannotBeRead(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))
	path := filepath.Join(dir, ".tickets", "tickets", id+".md")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(data), "title:", "title: [unclosed", 1)
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	got := runCLI(t, dir, nil, "show", id)
	if got.code == exitOK {
		t.Fatalf("show on a broken ticket succeeded:\n%s", got.stdout)
	}
	if !strings.Contains(got.stderr, "parse_error") {
		t.Errorf("stderr = %q, want it to name parse_error", got.stderr)
	}
	if strings.Contains(got.stderr, "ticket_not_found") {
		t.Errorf("show still calls a present ticket absent: %q", got.stderr)
	}

	// The JSON envelope carries the same code, since a host reads that one.
	got = runCLI(t, dir, nil, "show", id, "--json")
	var env struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &env); err != nil {
		t.Fatalf("the --json envelope did not parse: %v\n%s", err, got.stdout)
	}
	if env.Error.Code != "parse_error" {
		t.Errorf("envelope code = %q, want parse_error", env.Error.Code)
	}
}

// TestListReportsWhatItHadToLeaveOut covers the unreadable channel in 10.1. A
// query drops a file it cannot parse, so without this a host building a board
// cannot tell a short listing from a complete one.
func TestListReportsWhatItHadToLeaveOut(t *testing.T) {
	dir := newStore(t)
	keep := ticketID(t, createTicket(t, dir))
	breakID := ticketID(t, createTicket(t, dir))
	path := filepath.Join(dir, ".tickets", "tickets", breakID+".md")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	broken := strings.Replace(string(data), "title:", "title: [unclosed", 1)
	if err := os.WriteFile(path, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	got := runCLI(t, dir, nil, "--json", "list")
	if got.code != exitOK {
		t.Fatalf("list: %s", got.stderr)
	}
	var env struct {
		Tickets []struct {
			ID string `json:"id"`
		} `json:"tickets"`
		Unreadable []struct {
			Code  string  `json:"code"`
			File  string  `json:"file"`
			Field *string `json:"field"`
		} `json:"unreadable"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &env); err != nil {
		t.Fatalf("envelope did not parse: %v\n%s", err, got.stdout)
	}

	// The readable ticket is still listed, and the broken one still is not.
	if len(env.Tickets) != 1 || env.Tickets[0].ID != keep {
		t.Errorf("tickets = %+v, want only %s", env.Tickets, keep)
	}
	// But the envelope now says why the listing is short.
	if len(env.Unreadable) != 1 {
		t.Fatalf("unreadable = %+v, want one entry", env.Unreadable)
	}
	if env.Unreadable[0].Code != "parse_error" {
		t.Errorf("code = %q, want parse_error", env.Unreadable[0].Code)
	}
	if !strings.HasSuffix(env.Unreadable[0].File, breakID+".md") {
		t.Errorf("file = %q, want the broken ticket's path", env.Unreadable[0].File)
	}
}

// TestClaimOutsideARepositoryStillWorks holds the best-effort promise: a claim
// records what it can rather than failing where git has nothing to say.
func TestClaimOutsideARepositoryStillWorks(t *testing.T) {
	dir := newStore(t) // a store, but no repository around it
	id := readyTicket(t, dir)

	if got := runCLI(t, dir, nil, "claim", id, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("claim outside a repository: %s", got.stderr)
	}
	claim := showTicket(t, dir, id)["claim"].(map[string]any)
	if claim["actor"] != "human:sothr" {
		t.Errorf("actor = %v", claim["actor"])
	}
	for _, key := range []string{"branch", "worktree", "commit"} {
		if claim[key] != nil {
			t.Errorf("%s = %v, want null with no repository to read", key, claim[key])
		}
	}
}

// TestClaimConflictAndForce is the rest of 6.4: a live claim by somebody else
// is refused, and taking it leaves a trace.
func TestClaimConflictAndForce(t *testing.T) {
	dir := newStore(t)
	id := readyTicket(t, dir)

	if got := runCLI(t, dir, nil, "claim", id, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("first claim: %s", got.stderr)
	}

	got := runCLI(t, dir, nil, "--json", "claim", id, "--actor", "agent:terva/s1")
	if got.code != exitError {
		t.Fatal("claiming another actor's live claim should be refused")
	}
	if code := errCode(t, got); code != "claim_conflict" {
		t.Errorf("code = %v, want claim_conflict", code)
	}
	details := decode(t, got.stdout)["error"].(map[string]any)["details"].(map[string]any)
	if details["actor"] != "human:sothr" {
		t.Errorf("details.actor = %v, want the holder", details["actor"])
	}

	// Re-claiming your own ticket is not a conflict.
	if got := runCLI(t, dir, nil, "claim", id, "--actor", "human:sothr"); got.code != exitOK {
		t.Errorf("re-claiming your own ticket: %s", got.stderr)
	}

	if got := runCLI(t, dir, nil, "claim", id, "--actor", "agent:terva/s1", "--force"); got.code != exitOK {
		t.Fatalf("--force: %s", got.stderr)
	}
	tk := showTicket(t, dir, id)
	if actor := tk["claim"].(map[string]any)["actor"]; actor != "agent:terva/s1" {
		t.Errorf("actor = %v, want the forcing actor", actor)
	}
	notes := tk["body"].(map[string]any)["notes"].(string)
	if !strings.Contains(notes, "human:sothr") || !strings.Contains(notes, "agent:terva/s1") {
		t.Errorf("taking a claim should name both actors in Notes:\n%s", notes)
	}
}

// TestClaimRefusedOnDraftAndDone covers the statuses 6.4 will not claim from.
func TestClaimRefusedOnDraftAndDone(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir)) // still a draft

	if got := runCLI(t, dir, nil, "--json", "claim", id, "--actor", "human:sothr"); got.code != exitError {
		t.Error("a draft cannot be claimed")
	}

	for _, s := range []string{"ready", "in-progress", "done"} {
		if got := runCLI(t, dir, nil, "status", id, s, "--actor", "human:sothr"); got.code != exitOK {
			t.Fatalf("status %s: %s", s, got.stderr)
		}
	}
	if got := runCLI(t, dir, nil, "--json", "claim", id, "--actor", "human:sothr"); got.code != exitError {
		t.Error("a done ticket cannot be claimed")
	}
}

// TestRelease drops a claim, and says nothing when there was none.
func TestRelease(t *testing.T) {
	dir := newStore(t)
	id := readyTicket(t, dir)
	runCLI(t, dir, nil, "claim", id, "--actor", "human:sothr")

	if got := runCLI(t, dir, nil, "release", id, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("release: %s", got.stderr)
	}
	if claim := showTicket(t, dir, id)["claim"]; claim != nil {
		t.Errorf("claim = %v, want null after release", claim)
	}
	// Releasing an unclaimed ticket succeeds: what the caller asked for is
	// already true.
	if got := runCLI(t, dir, nil, "release", id, "--actor", "human:sothr"); got.code != exitOK {
		t.Errorf("releasing an unclaimed ticket: %s", got.stderr)
	}
}

// TestArchiveMovesTheFile is why archive is its own command rather than a
// status: the status alone would leave the file in tickets/.
func TestArchiveMovesTheFile(t *testing.T) {
	dir := newStore(t)
	id := readyTicket(t, dir)

	live := filepath.Join(dir, ".tickets", "tickets", id+".md")
	archived := filepath.Join(dir, ".tickets", "archive", id+".md")
	if _, err := os.Stat(live); err != nil {
		t.Fatalf("the ticket should start in tickets/: %v", err)
	}

	got := runCLI(t, dir, nil, "--json", "archive", id, "--reason", "superseded", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("archive: %s", got.stderr)
	}
	if _, err := os.Stat(archived); err != nil {
		t.Errorf("the file did not move to archive/: %v", err)
	}
	if _, err := os.Stat(live); !os.IsNotExist(err) {
		t.Error("the old file is still in tickets/")
	}

	// Both ends of the move are reported.
	if paths := decode(t, got.stdout)["pathsChanged"].([]any); len(paths) != 2 {
		t.Fatalf("pathsChanged = %v, want both ends of the move", paths)
	}

	tk := showTicket(t, dir, id)
	if tk["status"] != "archived" {
		t.Errorf("status = %v, want archived", tk["status"])
	}
	archive, ok := tk["archive"].(map[string]any)
	if !ok {
		t.Fatal("no archive block")
	}
	// from_status is what keeps an archived ticket from silently blocking its
	// dependents.
	if archive["fromStatus"] != "ready" {
		t.Errorf("fromStatus = %v, want ready", archive["fromStatus"])
	}
	if archive["reason"] != "superseded" {
		t.Errorf("reason = %v", archive["reason"])
	}

	// Archiving twice is refused rather than silently repeated.
	if got := runCLI(t, dir, nil, "--json", "archive", id, "--actor", "human:sothr"); got.code != exitError {
		t.Error("archiving an archived ticket should be refused")
	}
}

// TestArchivePathsStayRepositoryRelative is a regression test.
//
// displayPath resolves symlinks on both sides before making a path relative,
// because a macOS temporary directory is reached through /var while git answers
// with /private/var. EvalSymlinks fails on a path that is not there, and
// archiving reports the old location after deleting it, so the new path came
// back relative and the old one absolute in the same array.
//
// This needs a real repository: with no root at all, section 10 says an
// absolute path is correct.
func TestArchivePathsStayRepositoryRelative(t *testing.T) {
	dir := newGitStore(t)
	id := readyTicket(t, dir)

	got := runCLI(t, dir, nil, "--json", "archive", id, "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("archive: %s", got.stderr)
	}
	paths := decode(t, got.stdout)["pathsChanged"].([]any)
	if len(paths) != 2 {
		t.Fatalf("pathsChanged = %v, want both ends of the move", paths)
	}
	for _, p := range paths {
		path := p.(string)
		if filepath.IsAbs(path) {
			t.Errorf("pathsChanged holds an absolute path: %q", path)
			continue
		}
		// The surviving end has to resolve from the root. The removed end
		// cannot be stat'd, which is the whole difficulty.
		if !strings.HasPrefix(path, ".tickets/") {
			t.Errorf("path = %q, want it under .tickets/", path)
		}
	}
}

// TestUnarchiveRestoresToReady moves the file back and drops the archive block.
func TestUnarchiveRestoresToReady(t *testing.T) {
	dir := newStore(t)
	id := readyTicket(t, dir)
	runCLI(t, dir, nil, "archive", id, "--actor", "human:sothr")

	if got := runCLI(t, dir, nil, "unarchive", id, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("unarchive: %s", got.stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, ".tickets", "tickets", id+".md")); err != nil {
		t.Errorf("the file did not move back: %v", err)
	}
	tk := showTicket(t, dir, id)
	if tk["status"] != "ready" {
		t.Errorf("status = %v, want ready", tk["status"])
	}
	if tk["archive"] != nil {
		t.Errorf("archive = %v, want null once the ticket is live again", tk["archive"])
	}
	// The block goes, so Notes is what keeps the history.
	if notes := tk["body"].(map[string]any)["notes"].(string); !strings.Contains(notes, "unarchived") {
		t.Errorf("Notes does not record the unarchive:\n%s", notes)
	}

	if got := runCLI(t, dir, nil, "--json", "unarchive", id, "--actor", "human:sothr"); got.code != exitError {
		t.Error("unarchiving a live ticket should be refused")
	}
}

// TestIfRevisionRefusesAStaleWrite is the flag working at all. It was
// registered from the first commit of the CLI and reached no mutation until
// there were mutations to reach.
func TestIfRevisionRefusesAStaleWrite(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))
	stale, _ := showTicket(t, dir, id)["revision"].(string)
	if stale == "" {
		t.Fatal("no revision to work from")
	}

	// The precondition holds, so the write goes through.
	if got := runCLI(t, dir, nil, "status", id, "ready",
		"--if-revision", stale, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("a current revision should be accepted: %s", got.stderr)
	}

	// That write moved the ticket on, so the same revision is now stale.
	got := runCLI(t, dir, nil, "--json", "status", id, "in-progress",
		"--if-revision", stale, "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("a stale revision should be refused")
	}
	if code := errCode(t, got); code != "stale_revision" {
		t.Errorf("code = %v, want stale_revision", code)
	}
	// The ticket did not move.
	if s := showTicket(t, dir, id)["status"]; s != "ready" {
		t.Errorf("status = %v, want ready: the refused write should change nothing", s)
	}

	// Every mutation honours it, not just status.
	for _, args := range [][]string{
		{"claim", id},
		{"archive", id},
		{"release", id},
	} {
		got := runCLI(t, dir, nil, append([]string{"--json"}, append(args,
			"--if-revision", stale, "--actor", "human:sothr")...)...)
		if got.code != exitError {
			t.Errorf("%s ignored --if-revision", args[0])
			continue
		}
		if code := errCode(t, got); code != "stale_revision" {
			t.Errorf("%s: code = %v, want stale_revision", args[0], code)
		}
	}
}
