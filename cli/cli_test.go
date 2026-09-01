package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// referenceInstant is the fixed clock the tests write with, the same one the
// fixture corpus is judged against.
var referenceInstant = time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

type result struct {
	stdout string
	stderr string
	code   int
}

// runCLI invokes the CLI the way the binary does, with the process state
// supplied rather than taken from the environment.
func runCLI(t *testing.T, dir string, env map[string]string, args ...string) result {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := Run(args, Env{
		Dir:    dir,
		Getenv: func(k string) string { return env[k] },
		Stdout: &stdout,
		Stderr: &stderr,
		Now:    func() time.Time { return referenceInstant },
	})
	return result{stdout: stdout.String(), stderr: stderr.String(), code: code}
}

// newStore makes a directory with an initialized store in it.
func newStore(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	got := runCLI(t, dir, nil, "init", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("init failed: %s%s", got.stdout, got.stderr)
	}
	return dir
}

// decode reads one JSON envelope, failing the test if it is not valid JSON.
func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, s)
	}
	return out
}

func createTicket(t *testing.T, dir string, extra ...string) map[string]any {
	t.Helper()
	args := append([]string{"--json", "create", "--title", "Rotate the signing key",
		"--actor", "human:sothr"}, extra...)
	got := runCLI(t, dir, nil, args...)
	if got.code != exitOK {
		t.Fatalf("create failed: %s%s", got.stdout, got.stderr)
	}
	return decode(t, got.stdout)
}

func ticketID(t *testing.T, envelope map[string]any) string {
	t.Helper()
	tk, ok := envelope["ticket"].(map[string]any)
	if !ok {
		t.Fatalf("envelope carries no ticket: %v", envelope)
	}
	id, _ := tk["id"].(string)
	if id == "" {
		t.Fatalf("envelope carries no ticket id: %v", envelope)
	}
	return id
}

// TestJSONKindPerOperation is the Phase 2 exit criterion: the JSON contract has
// a test per kind. check-report arrives with the check command.
func TestJSONKindPerOperation(t *testing.T) {
	dir := t.TempDir()

	init := runCLI(t, dir, nil, "--json", "init", "--actor", "human:sothr")
	if init.code != exitOK {
		t.Fatalf("init: %s", init.stderr)
	}
	if kind := decode(t, init.stdout)["kind"]; kind != "mutation-result" {
		t.Errorf("init kind = %v, want mutation-result", kind)
	}
	// init changes the store rather than a ticket, so there is no ticket to
	// name.
	if got := decode(t, init.stdout)["ticket"]; got != nil {
		t.Errorf("init ticket = %v, want null", got)
	}

	created := createTicket(t, dir)
	if created["kind"] != "mutation-result" {
		t.Errorf("create kind = %v, want mutation-result", created["kind"])
	}
	id := ticketID(t, created)

	show := runCLI(t, dir, nil, "--json", "show", id)
	if show.code != exitOK {
		t.Fatalf("show: %s", show.stderr)
	}
	if kind := decode(t, show.stdout)["kind"]; kind != "ticket" {
		t.Errorf("show kind = %v, want ticket", kind)
	}

	list := runCLI(t, dir, nil, "--json", "list")
	if list.code != exitOK {
		t.Fatalf("list: %s", list.stderr)
	}
	if kind := decode(t, list.stdout)["kind"]; kind != "ticket-list" {
		t.Errorf("list kind = %v, want ticket-list", kind)
	}

	bad := runCLI(t, dir, nil, "--json", "frobnicate")
	if bad.code == exitOK {
		t.Error("an unknown command should not succeed")
	}
	envelope := decode(t, bad.stdout)
	if envelope["kind"] != "error" {
		t.Errorf("error kind = %v, want error", envelope["kind"])
	}
	body, ok := envelope["error"].(map[string]any)
	if !ok {
		t.Fatalf("error envelope has no error body: %v", envelope)
	}
	for _, key := range []string{"code", "message", "details"} {
		if _, present := body[key]; !present {
			t.Errorf("the error body omits %q; absent values are null, never missing", key)
		}
	}
	if body["code"] != codeUsage {
		t.Errorf("code = %v, want %s", body["code"], codeUsage)
	}
}

// TestErrorGoesToStderrAndExitsOne pins plan 10.2 and the stdout rule of
// section 10: the message is on stderr in both modes, the envelope is on stdout
// in JSON mode only, and stdout stays empty in human mode.
func TestErrorGoesToStderrAndExitsOne(t *testing.T) {
	dir := t.TempDir() // no store here

	human := runCLI(t, dir, nil, "list")
	if human.code != exitError {
		t.Errorf("exit = %d, want %d", human.code, exitError)
	}
	if human.stdout != "" {
		t.Errorf("human mode wrote to stdout on failure: %q", human.stdout)
	}
	if !strings.Contains(human.stderr, "git-ticket:") {
		t.Errorf("stderr = %q, want the message", human.stderr)
	}

	asJSON := runCLI(t, dir, nil, "--json", "list")
	if asJSON.code != exitError {
		t.Errorf("exit = %d, want %d", asJSON.code, exitError)
	}
	if asJSON.stderr == "" {
		t.Error("JSON mode should still put the message on stderr")
	}
	body := decode(t, asJSON.stdout)["error"].(map[string]any)
	if body["code"] != "store_not_found" {
		t.Errorf("code = %v, want store_not_found", body["code"])
	}
}

// TestGlobalFlagsBeforeOrAfterTheCommand is plan 12.1: both orders are what a
// person types. The standard library's parse stops at the first non-flag word,
// so `show ID --json` needs the permuting parse to see --json at all.
func TestGlobalFlagsBeforeOrAfterTheCommand(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	before := runCLI(t, dir, nil, "--json", "show", id)
	after := runCLI(t, dir, nil, "show", id, "--json")
	if before.code != exitOK || after.code != exitOK {
		t.Fatalf("before: %s\nafter: %s", before.stderr, after.stderr)
	}
	if decode(t, after.stdout)["kind"] != "ticket" {
		t.Errorf("`show ID --json` did not emit the envelope:\n%s", after.stdout)
	}
	if before.stdout != after.stdout {
		t.Errorf("flag order changed the output:\n%s\n%s", before.stdout, after.stdout)
	}
}

// TestStorePrecedence is plan 12.1: the flag beats the environment, the
// environment beats discovery, and a path that names no store is an error
// rather than a reason to go looking elsewhere.
func TestStorePrecedence(t *testing.T) {
	fromFlag := newStore(t)
	fromEnv := newStore(t)
	elsewhere := t.TempDir()

	// The flag wins over the environment.
	got := runCLI(t, elsewhere, map[string]string{"GIT_TICKET_STORE": filepath.Join(fromEnv, ".tickets")},
		"--json", "--store", filepath.Join(fromFlag, ".tickets"),
		"create", "--title", "By the flag", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create: %s", got.stderr)
	}
	if n := countTickets(t, fromFlag); n != 1 {
		t.Errorf("the flagged store holds %d tickets, want 1", n)
	}
	if n := countTickets(t, fromEnv); n != 0 {
		t.Errorf("the environment store holds %d tickets, want 0", n)
	}

	// The environment wins over discovery, which would find nothing here.
	got = runCLI(t, elsewhere, map[string]string{"GIT_TICKET_STORE": filepath.Join(fromEnv, ".tickets")},
		"--json", "create", "--title", "By the environment", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create: %s", got.stderr)
	}
	if n := countTickets(t, fromEnv); n != 1 {
		t.Errorf("the environment store holds %d tickets, want 1", n)
	}

	// A named store that does not exist is store_not_found, and never a
	// silent fall back to discovery.
	got = runCLI(t, fromFlag, nil, "--json", "--store", filepath.Join(elsewhere, "nothing"), "list")
	if got.code != exitError {
		t.Fatal("an explicit --store with no store should fail")
	}
	if code := decode(t, got.stdout)["error"].(map[string]any)["code"]; code != "store_not_found" {
		t.Errorf("code = %v, want store_not_found", code)
	}
}

func countTickets(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, ".tickets", "tickets"))
	if err != nil {
		t.Fatal(err)
	}
	return len(entries)
}

// TestAbsentCollectionsAreEmptyArrays is the rule of section 10 that a consumer
// never has to tell missing from empty.
func TestAbsentCollectionsAreEmptyArrays(t *testing.T) {
	dir := newStore(t)

	empty := runCLI(t, dir, nil, "--json", "list")
	tickets, ok := decode(t, empty.stdout)["tickets"].([]any)
	if !ok || tickets == nil {
		t.Fatalf("an empty list must still carry an array: %s", empty.stdout)
	}
	if len(tickets) != 0 {
		t.Errorf("a fresh store listed %d tickets", len(tickets))
	}

	id := ticketID(t, createTicket(t, dir))
	show := runCLI(t, dir, nil, "--json", "show", id)
	tk := decode(t, show.stdout)["ticket"].(map[string]any)

	for _, key := range []string{"labels", "assignees", "dependencies", "references"} {
		v, present := tk[key]
		if !present {
			t.Errorf("%q is missing; an absent collection is [] and never omitted", key)
			continue
		}
		if _, isArray := v.([]any); !isArray {
			t.Errorf("%q = %v, want an array", key, v)
		}
	}
	for _, key := range []string{"statusReason", "milestone", "parent", "claim", "archive"} {
		v, present := tk[key]
		if !present {
			t.Errorf("%q is missing; an absent scalar is null and never omitted", key)
		}
		if v != nil {
			t.Errorf("%q = %v, want null on a fresh ticket", key, v)
		}
	}
	for _, key := range []string{"extensions", "unknown"} {
		if _, isObject := tk[key].(map[string]any); !isObject {
			t.Errorf("%q = %v, want an object", key, tk[key])
		}
	}
}

// TestBodyAndChecklistsAgree pins plan 10.1: body carries the section as
// written and checklists is the derived view, with the index that ac and dod
// take rather than an array position.
func TestBodyAndChecklistsAgree(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	// The ac command arrives later in Phase 2, so the section is written the
	// way a person would write it by hand: prose above the list, which is the
	// case that makes an array position the wrong index.
	path := filepath.Join(dir, ".tickets", "tickets", id+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	section := "\n## Acceptance criteria\n\nThe rollout has to hold these:\n\n- [x] The verifier accepts either key\n- [ ] New tokens use the newer key\n"
	if err := os.WriteFile(path, append(data, []byte(section)...), 0o644); err != nil {
		t.Fatal(err)
	}

	show := runCLI(t, dir, nil, "--json", "show", id)
	if show.code != exitOK {
		t.Fatalf("show: %s", show.stderr)
	}
	tk := decode(t, show.stdout)["ticket"].(map[string]any)

	body := tk["body"].(map[string]any)["acceptanceCriteria"].(string)
	if !strings.Contains(body, "The rollout has to hold these:") {
		t.Errorf("body dropped the prose a person wrote:\n%s", body)
	}

	items := tk["checklists"].(map[string]any)["acceptanceCriteria"].([]any)
	if len(items) != 2 {
		t.Fatalf("got %d checklist items, want 2: %v", len(items), items)
	}
	first := items[0].(map[string]any)
	if first["index"].(float64) != 1 || first["checked"] != true {
		t.Errorf("first item = %v, want index 1 and checked", first)
	}
	second := items[1].(map[string]any)
	if second["index"].(float64) != 2 || second["checked"] != false {
		t.Errorf("second item = %v, want index 2 and unchecked", second)
	}
	if second["text"] != "New tokens use the newer key" {
		t.Errorf("text = %v", second["text"])
	}
}

// TestListFilters checks the flags reach the library, including that repeating
// one makes its values alternatives.
func TestListFilters(t *testing.T) {
	dir := newStore(t)
	runCLI(t, dir, nil, "create", "--title", "A bug", "--type", "bug", "--priority", "high",
		"--label", "auth", "--actor", "human:sothr")
	runCLI(t, dir, nil, "create", "--title", "A chore", "--type", "chore", "--priority", "low",
		"--actor", "human:sothr")

	count := func(args ...string) int {
		t.Helper()
		got := runCLI(t, dir, nil, append([]string{"--json", "list"}, args...)...)
		if got.code != exitOK {
			t.Fatalf("list: %s", got.stderr)
		}
		return len(decode(t, got.stdout)["tickets"].([]any))
	}

	if n := count(); n != 2 {
		t.Errorf("unfiltered list = %d, want 2", n)
	}
	if n := count("--type", "bug"); n != 1 {
		t.Errorf("type bug = %d, want 1", n)
	}
	if n := count("--type", "bug", "--type", "chore"); n != 2 {
		t.Errorf("repeating a filter makes its values alternatives, got %d, want 2", n)
	}
	if n := count("--label", "auth"); n != 1 {
		t.Errorf("label auth = %d, want 1", n)
	}
	if n := count("--type", "bug", "--priority", "low"); n != 0 {
		t.Errorf("filters across fields all have to hold, got %d, want 0", n)
	}

	// A status outside the set is caught before the store is touched.
	bad := runCLI(t, dir, nil, "list", "--status", "frobnicate")
	if bad.code != exitError {
		t.Error("an invalid status should fail")
	}
}

// TestHumanOutput checks the shape a person sees, which no JSON test covers.
func TestHumanOutput(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir, "--priority", "urgent"))

	list := runCLI(t, dir, nil, "list")
	if list.code != exitOK {
		t.Fatalf("list: %s", list.stderr)
	}
	// A listing abbreviates the ID. Take what it actually printed rather than
	// recomputing it, because the property under test is that the text on the
	// screen resolves.
	abbrev := strings.Fields(list.stdout)[0]
	if !strings.HasPrefix(id, abbrev) {
		t.Errorf("listing shows %q, which is not a prefix of %s", abbrev, id)
	}
	if !strings.Contains(list.stdout, "Rotate the signing key") {
		t.Errorf("listing does not show the title:\n%s", list.stdout)
	}
	resolved := runCLI(t, dir, nil, "show", abbrev)
	if resolved.code != exitOK {
		t.Errorf("the abbreviation a listing prints does not resolve: %s", resolved.stderr)
	}

	empty := runCLI(t, dir, nil, "list", "--status", "done")
	if !strings.Contains(empty.stdout, "No tickets match.") {
		t.Errorf("an empty listing says nothing to a person: %q", empty.stdout)
	}
}

// TestPathsAreRepositoryRelative is plan section 10. It also pins the symlink
// case: a macOS temporary directory is reached through /var, which is a link to
// /private/var, so git reports a root in a different name space than the store.
func TestPathsAreRepositoryRelative(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, so there is no repository root to resolve against")
	}
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if got := runCLI(t, dir, nil, "init", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("init: %s", got.stderr)
	}

	created := createTicket(t, dir)
	paths, ok := created["pathsChanged"].([]any)
	if !ok || len(paths) != 1 {
		t.Fatalf("pathsChanged = %v", created["pathsChanged"])
	}
	got := paths[0].(string)
	if filepath.IsAbs(got) {
		t.Errorf("path = %q, want it relative to the repository root", got)
	}
	if !strings.HasPrefix(got, ".tickets/tickets/") {
		t.Errorf("path = %q, want it under .tickets/tickets/", got)
	}
}

// corpusDir is the fixture corpus, shared with the ticket package. One level
// up, because this package sits at the repository root rather than under
// internal/.
const corpusDir = "../testdata"

type sidecarFinding struct {
	Code   string  `json:"code"`
	File   string  `json:"file"`
	Ticket *string `json:"ticket"`
	Field  *string `json:"field"`
}

type expectation struct {
	Errors   []sidecarFinding `json:"errors"`
	Warnings []sidecarFinding `json:"warnings"`
}

// TestCheckReportKind is the last of the five kinds in plan section 10.
func TestCheckReportKind(t *testing.T) {
	dir := newStore(t)
	createTicket(t, dir)

	got := runCLI(t, dir, nil, "--json", "check")
	if got.code != exitOK {
		t.Fatalf("a store this command just wrote should pass: %s%s", got.stdout, got.stderr)
	}
	envelope := decode(t, got.stdout)
	if envelope["kind"] != "check-report" {
		t.Errorf("kind = %v, want check-report", envelope["kind"])
	}
	if envelope["ok"] != true {
		t.Errorf("ok = %v, want true", envelope["ok"])
	}
	for _, key := range []string{"errors", "warnings"} {
		v, present := envelope[key]
		if !present {
			t.Errorf("%q is missing; an absent collection is [] and never omitted", key)
			continue
		}
		if arr, isArray := v.([]any); !isArray || len(arr) != 0 {
			t.Errorf("%q = %v, want an empty array", key, v)
		}
	}
}

// TestCheckAgreesWithTheCorpus runs the CLI over every fixture store and holds
// it to the sidecar a person reviewed. It also pins the invariant of plan 10.3
// that ok is true exactly when the command exited zero.
func TestCheckAgreesWithTheCorpus(t *testing.T) {
	cases, err := os.ReadDir(filepath.Join(corpusDir, "stores"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		if !c.IsDir() {
			continue
		}
		t.Run(c.Name(), func(t *testing.T) {
			caseDir := filepath.Join(corpusDir, "stores", c.Name())
			var exp expectation
			raw, err := os.ReadFile(filepath.Join(caseDir, "expected.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(raw, &exp); err != nil {
				t.Fatal(err)
			}

			got := runCLI(t, caseDir, nil, "--json", "--store", filepath.Join(caseDir, "store"), "check")
			envelope := decode(t, got.stdout)
			if envelope["kind"] != "check-report" {
				t.Fatalf("kind = %v, want check-report", envelope["kind"])
			}

			// ok mirrors the exit status, whatever the findings are.
			if ok := envelope["ok"] == true; ok != (got.code == exitOK) {
				t.Errorf("ok = %v but exit = %d; plan 10.3 makes them the same fact", envelope["ok"], got.code)
			}
			// A store with errors fails. Warnings alone do not, without --strict.
			if want := exitOK; len(exp.Errors) == 0 && got.code != want {
				t.Errorf("exit = %d, want %d for a store with no errors", got.code, want)
			}
			if len(exp.Errors) > 0 && got.code != exitError {
				t.Errorf("exit = %d, want %d for a store with errors", got.code, exitError)
			}

			// reference_path_unresolved is the one check that depends on where
			// the store sits, per plan 11. The library test injects the case
			// directory as the repository root so the fixture's own
			// docs/present.txt resolves. The CLI takes the root from git and
			// finds this repository instead, where neither path exists, so it
			// reports both references rather than one. Comparing findings here
			// would assert the checkout layout, not the CLI.
			if c.Name() == "reference-unresolved" {
				return
			}
			compareToSidecar(t, "errors", exp.Errors, envelope["errors"].([]any))
			compareToSidecar(t, "warnings", exp.Warnings, envelope["warnings"].([]any))
		})
	}
}

// repoRoot is where this package's tests run from, back up to the repository
// root. A path in the envelope is relative to that, so joining the two has to
// name a real file.
const repoRoot = ".."

// compareToSidecar holds the emitted findings to the reviewed ones. The sidecar
// records a store-relative file and the CLI emits a repository-relative one.
func compareToSidecar(t *testing.T, label string, want []sidecarFinding, got []any) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s: got %d findings, want %d\n%v", label, len(got), len(want), got)
		return
	}
	for i, w := range want {
		g := got[i].(map[string]any)
		if g["code"] != w.Code {
			t.Errorf("%s[%d] code = %v, want %v", label, i, g["code"], w.Code)
		}
		file, _ := g["file"].(string)
		if !strings.HasSuffix(file, w.File) {
			t.Errorf("%s[%d] file = %q, want it to end with %q", label, i, file, w.File)
		}
		if filepath.IsAbs(file) {
			t.Errorf("%s[%d] file = %q, want it relative to the repository root", label, i, file)
		}
		// The path has to name the file from the repository root, which is
		// what makes it useful to a person or an editor. A suffix match alone
		// would still pass if the CLI forgot to convert the store-relative
		// path the library reports.
		if _, err := os.Stat(filepath.Join(repoRoot, file)); err != nil {
			t.Errorf("%s[%d] file = %q, which does not resolve from the repository root: %v",
				label, i, file, err)
		}
		if !sameOptional(g["ticket"], w.Ticket) {
			t.Errorf("%s[%d] ticket = %v, want %v", label, i, g["ticket"], derefOr(w.Ticket))
		}
		if !sameOptional(g["field"], w.Field) {
			t.Errorf("%s[%d] field = %v, want %v", label, i, g["field"], derefOr(w.Field))
		}
	}
}

func sameOptional(got any, want *string) bool {
	if want == nil {
		return got == nil
	}
	return got == *want
}

func derefOr(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}

// TestCheckStrictChangesTheVerdictNotTheArrays is the decision recorded in plan
// 10.3: --strict is a policy on top of the report, so a promoted warning stays
// a warning and only ok and the exit status move.
func TestCheckStrictChangesTheVerdictNotTheArrays(t *testing.T) {
	// label-unknown carries one warning and no errors, which is the case the
	// flag exists for.
	store := filepath.Join(corpusDir, "stores", "label-unknown", "store")

	lenient := runCLI(t, ".", nil, "--json", "--store", store, "check")
	if lenient.code != exitOK {
		t.Errorf("exit = %d, want %d: a warning alone does not fail", lenient.code, exitOK)
	}
	lenientEnvelope := decode(t, lenient.stdout)
	if lenientEnvelope["ok"] != true {
		t.Errorf("ok = %v, want true", lenientEnvelope["ok"])
	}

	strict := runCLI(t, ".", nil, "--json", "--store", store, "check", "--strict")
	if strict.code != exitError {
		t.Errorf("exit = %d, want %d under --strict", strict.code, exitError)
	}
	strictEnvelope := decode(t, strict.stdout)
	if strictEnvelope["ok"] != false {
		t.Errorf("ok = %v, want false under --strict", strictEnvelope["ok"])
	}

	// The finding did not move, and errors is still empty. ok false with an
	// empty errors array is exactly the strict run of a store with warnings.
	if errs := strictEnvelope["errors"].([]any); len(errs) != 0 {
		t.Errorf("--strict moved a finding into errors: %v", errs)
	}
	if warns := strictEnvelope["warnings"].([]any); len(warns) != 1 {
		t.Errorf("warnings = %v, want the one finding to stay put", warns)
	}
	if !equalJSON(lenientEnvelope["warnings"], strictEnvelope["warnings"]) {
		t.Error("--strict changed the warnings array; it should change the verdict only")
	}
}

func equalJSON(a, b any) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

// TestCheckThatCannotRunIsAnError separates the two failures that both exit
// one: a store that fails the check, and a check that could not happen.
func TestCheckThatCannotRunIsAnError(t *testing.T) {
	dir := t.TempDir() // no store here

	got := runCLI(t, dir, nil, "--json", "check")
	if got.code != exitError {
		t.Fatalf("exit = %d, want %d", got.code, exitError)
	}
	envelope := decode(t, got.stdout)
	if envelope["kind"] != "error" {
		t.Errorf("kind = %v, want error: the check could not run at all", envelope["kind"])
	}
	if code := envelope["error"].(map[string]any)["code"]; code != "store_not_found" {
		t.Errorf("code = %v, want store_not_found", code)
	}
}

func TestCheckHumanOutput(t *testing.T) {
	clean := newStore(t)
	createTicket(t, clean)
	got := runCLI(t, clean, nil, "check")
	if got.code != exitOK {
		t.Fatalf("check: %s", got.stderr)
	}
	if !strings.Contains(got.stdout, "No problems found.") {
		t.Errorf("a clean store says nothing to a person: %q", got.stdout)
	}

	// A store with a finding prints it, and the findings go to stdout because
	// the command succeeded at checking. Only the verdict is no.
	broken := filepath.Join(corpusDir, "stores", "parent-missing", "store")
	got = runCLI(t, ".", nil, "--store", broken, "check")
	if got.code != exitError {
		t.Errorf("exit = %d, want %d", got.code, exitError)
	}
	for _, want := range []string{"error", "parent_missing", "1 error"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("the report does not mention %q:\n%s", want, got.stdout)
		}
	}

	// Under --strict a person is told why a store of warnings failed, since
	// the report itself shows no errors.
	warned := filepath.Join(corpusDir, "stores", "label-unknown", "store")
	got = runCLI(t, ".", nil, "--store", warned, "check", "--strict")
	if !strings.Contains(got.stdout, "--strict") {
		t.Errorf("a strict failure does not say why:\n%s", got.stdout)
	}
}

func TestCheckTakesNoArguments(t *testing.T) {
	dir := newStore(t)
	got := runCLI(t, dir, nil, "--json", "check", "TKT-01K3ZZ2JH000GHB4EE6SNRE6MD")
	if got.code != exitError {
		t.Fatalf("exit = %d, want %d", got.code, exitError)
	}
	if code := decode(t, got.stdout)["error"].(map[string]any)["code"]; code != codeUsage {
		t.Errorf("code = %v, want %s", code, codeUsage)
	}
}

// TestHelpAndNoArguments covers the two invocations that are not a command.
func TestHelpAndNoArguments(t *testing.T) {
	dir := t.TempDir()

	help := runCLI(t, dir, nil, "help")
	if help.code != exitOK {
		t.Errorf("help exited %d", help.code)
	}
	for _, want := range []string{"init", "create", "show", "list", "check", "--json"} {
		if !strings.Contains(help.stdout, want) {
			t.Errorf("help does not mention %q", want)
		}
	}

	bare := runCLI(t, dir, nil)
	if bare.code != exitError {
		t.Errorf("a bare invocation exited %d, want %d", bare.code, exitError)
	}
	if bare.stdout != "" {
		t.Errorf("usage belongs on stderr when nothing was asked for: %q", bare.stdout)
	}
}
