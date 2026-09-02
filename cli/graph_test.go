package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/ticket"
)

// makeTicket creates a ticket with the given title and returns its full ID.
func makeTicket(t *testing.T, dir, title string) string {
	t.Helper()
	got := runCLI(t, dir, nil, "--json", "create", "--title", title, "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create %q: %s", title, got.stderr)
	}
	return ticketID(t, decode(t, got.stdout))
}

// dependenciesOf reads the dependency list back through the JSON contract.
func dependenciesOf(t *testing.T, dir, id string) []string {
	t.Helper()
	var out []string
	for _, d := range showTicket(t, dir, id)["dependencies"].([]any) {
		out = append(out, d.(string))
	}
	return out
}

// wantIDs checks the answer holds exactly these tickets and is sorted.
//
// It compares as a set, because the order is by ID and IDs created in the same
// millisecond share their timestamp characters and differ only in the random
// suffix. Asserting creation order would pass or fail on the entropy.
func wantIDs(t *testing.T, got []string, want ...string) {
	t.Helper()
	if !slices.IsSorted(got) {
		t.Errorf("result is not sorted by ID: %v", got)
	}
	gotSorted := slices.Clone(got)
	wantSorted := slices.Clone(want)
	slices.Sort(gotSorted)
	slices.Sort(wantSorted)
	if !slices.Equal(gotSorted, wantSorted) {
		t.Errorf("got %v, want exactly %v", got, want)
	}
}

// idsOf pulls the IDs out of a ticket-list envelope.
func idsOf(t *testing.T, r result) []string {
	t.Helper()
	var out []string
	for _, item := range decode(t, r.stdout)["tickets"].([]any) {
		out = append(out, item.(map[string]any)["id"].(string))
	}
	return out
}

func TestLinkAddsADependency(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A, the top")
	b := makeTicket(t, dir, "B, the middle")

	if got := runCLI(t, dir, nil, "link", a, "--depends-on", b, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("link: %s", got.stderr)
	}
	deps := dependenciesOf(t, dir, a)
	if len(deps) != 1 || deps[0] != b {
		t.Errorf("dependencies = %v, want [%s]", deps, b)
	}

	// Linking the same pair twice leaves one edge: a dependency is a set.
	runCLI(t, dir, nil, "link", a, "--depends-on", b, "--actor", "human:sothr")
	if deps := dependenciesOf(t, dir, a); len(deps) != 1 {
		t.Errorf("dependencies = %v, want the edge recorded once", deps)
	}
}

// TestLinkAcceptsAnIDPrefix is plan 5.5: any command taking an ID takes a
// unique prefix. The library's mutation wants a canonical ID, so the CLI is
// what turns one into the other.
func TestLinkAcceptsAnIDPrefix(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A, the top")
	b := makeTicket(t, dir, "B, the middle")

	// Long enough to be unique, since these two were made in the same
	// millisecond and share their timestamp characters.
	prefix := b[:len(b)-8]
	if got := runCLI(t, dir, nil, "link", a, "--depends-on", prefix, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("link with a prefix: %s", got.stderr)
	}
	// What lands in the file is the full ID, never the prefix that was typed.
	if deps := dependenciesOf(t, dir, a); len(deps) != 1 || deps[0] != b {
		t.Errorf("dependencies = %v, want the resolved [%s]", deps, b)
	}
}

func TestLinkAddsAReference(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A")

	if got := runCLI(t, dir, nil, "link", a, "--ref", "proposal:git-ticket",
		"--path", "docs/plan.md", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("link --ref: %s", got.stderr)
	}
	refs := showTicket(t, dir, a)["references"].([]any)
	if len(refs) != 1 {
		t.Fatalf("references = %v, want one", refs)
	}
	first := refs[0].(map[string]any)
	if first["ref"] != "proposal:git-ticket" || first["path"] != "docs/plan.md" {
		t.Errorf("reference = %v", first)
	}

	// A reference with no path is legal, and the path is null rather than "".
	if got := runCLI(t, dir, nil, "link", a, "--ref", "spec:rfc9999",
		"--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("link --ref with no path: %s", got.stderr)
	}
	refs = showTicket(t, dir, a)["references"].([]any)
	if len(refs) != 2 {
		t.Fatalf("references = %v, want two", refs)
	}
	if p := refs[1].(map[string]any)["path"]; p != nil {
		t.Errorf("path = %v, want null", p)
	}

	// Re-linking a ref that is already there replaces its path, so link is
	// idempotent rather than accumulating duplicates.
	runCLI(t, dir, nil, "link", a, "--ref", "proposal:git-ticket",
		"--path", "docs/other.md", "--actor", "human:sothr")
	refs = showTicket(t, dir, a)["references"].([]any)
	if len(refs) != 2 {
		t.Errorf("references = %v, want the ref replaced rather than repeated", refs)
	}
	if p := refs[0].(map[string]any)["path"]; p != "docs/other.md" {
		t.Errorf("path = %v, want the new one", p)
	}
}

func TestLinkRefusesWhatCannotBeLinked(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A")

	got := runCLI(t, dir, nil, "--json", "link", a, "--depends-on", a, "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("a ticket depending on itself should be refused")
	}
	if code := errCode(t, got); code != "dependency_cycle" {
		t.Errorf("code = %v, want dependency_cycle", code)
	}

	// A dependency on a ticket that does not exist fails before any write,
	// because the CLI resolves the ID first.
	got = runCLI(t, dir, nil, "--json", "link", a,
		"--depends-on", "TKT-01JZZZZZZZZZZZZZZZZZZZZZZZ", "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("a dependency on a missing ticket should be refused")
	}
	if code := errCode(t, got); code != "ticket_not_found" {
		t.Errorf("code = %v, want ticket_not_found", code)
	}
}

func TestLinkFlagCombinations(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A")
	b := makeTicket(t, dir, "B")

	// Neither alternative.
	if got := runCLI(t, dir, nil, "--json", "link", a, "--actor", "human:sothr"); errCode(t, got) != codeUsage {
		t.Error("link with no operation should be a usage error")
	}
	// Both alternatives, which names both rather than picking one.
	got := runCLI(t, dir, nil, "--json", "link", a, "--depends-on", b, "--ref", "x",
		"--actor", "human:sothr")
	if errCode(t, got) != codeUsage {
		t.Error("link with both operations should be a usage error")
	}
	msg := decode(t, got.stdout)["error"].(map[string]any)["message"].(string)
	if !strings.Contains(msg, "--depends-on") || !strings.Contains(msg, "--ref") {
		t.Errorf("the message should name both flags: %q", msg)
	}
	// A path with no ref to attach it to.
	got = runCLI(t, dir, nil, "--json", "link", a, "--depends-on", b,
		"--path", "docs/plan.md", "--actor", "human:sothr")
	if errCode(t, got) != codeUsage {
		t.Error("--path without --ref should be a usage error")
	}
}

func TestUnlinkRemovesDependenciesAndReferences(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A")
	b := makeTicket(t, dir, "B")
	runCLI(t, dir, nil, "link", a, "--depends-on", b, "--actor", "human:sothr")
	runCLI(t, dir, nil, "link", a, "--ref", "proposal:x", "--actor", "human:sothr")

	if got := runCLI(t, dir, nil, "unlink", a, "--depends-on", b, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("unlink --depends-on: %s", got.stderr)
	}
	if deps := dependenciesOf(t, dir, a); len(deps) != 0 {
		t.Errorf("dependencies = %v, want empty", deps)
	}

	if got := runCLI(t, dir, nil, "unlink", a, "--ref", "proposal:x", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("unlink --ref: %s", got.stderr)
	}
	if refs := showTicket(t, dir, a)["references"].([]any); len(refs) != 0 {
		t.Errorf("references = %v, want empty", refs)
	}

	// Removing what is not there succeeds: the caller's intent is already true.
	if got := runCLI(t, dir, nil, "unlink", a, "--ref", "proposal:x", "--actor", "human:sothr"); got.code != exitOK {
		t.Errorf("unlinking an absent reference: %s", got.stderr)
	}
}

// TestUnlinkRepairsADanglingDependency is why unlink resolves a prefix against
// the ticket's own dependency list rather than against the store.
//
// A dependency naming a ticket that does not exist is the dependency_missing
// that check reports, and unlink is the repair. Resolving through the store
// would fail to find it and leave the store permanently broken.
func TestUnlinkRepairsADanglingDependency(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A")

	const ghost = "TKT-01JZZZZZZZZZZZZZZZZZZZZZZZ"
	path := filepath.Join(dir, ".tickets", "tickets", a+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	edited := strings.Replace(string(data), "dependencies: []",
		"dependencies:\n  - "+ghost, 1)
	if edited == string(data) {
		t.Fatal("the fixture did not have the empty dependency list this test edits")
	}
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	// check sees it, which is how a person finds out.
	if got := runCLI(t, dir, nil, "--json", "check"); got.code != exitError {
		t.Fatal("check should report a dependency on a missing ticket")
	}

	if got := runCLI(t, dir, nil, "unlink", a, "--depends-on", ghost, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("unlink a dangling dependency: %s", got.stderr)
	}
	if deps := dependenciesOf(t, dir, a); len(deps) != 0 {
		t.Errorf("dependencies = %v, want the dangling one gone", deps)
	}
	if got := runCLI(t, dir, nil, "--json", "check"); got.code != exitOK {
		t.Errorf("check is still failing after the repair:\n%s", got.stdout)
	}
}

func TestUnlinkFlagCombinations(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A")

	if got := runCLI(t, dir, nil, "--json", "unlink", a, "--actor", "human:sothr"); errCode(t, got) != codeUsage {
		t.Error("unlink with no operation should be a usage error")
	}
	got := runCLI(t, dir, nil, "--json", "unlink", a, "--depends-on", "x", "--ref", "y",
		"--actor", "human:sothr")
	if errCode(t, got) != codeUsage {
		t.Error("unlink with both operations should be a usage error")
	}
}

// chain builds A depends on B depends on C and returns the three IDs.
func chain(t *testing.T, dir string) (a, b, c string) {
	t.Helper()
	a = makeTicket(t, dir, "A, the top")
	b = makeTicket(t, dir, "B, the middle")
	c = makeTicket(t, dir, "C, the bottom")
	runCLI(t, dir, nil, "link", a, "--depends-on", b, "--actor", "human:sothr")
	runCLI(t, dir, nil, "link", b, "--depends-on", c, "--actor", "human:sothr")
	return a, b, c
}

func TestDepsDirectAndTransitive(t *testing.T) {
	dir := newStore(t)
	a, b, c := chain(t, dir)

	direct := runCLI(t, dir, nil, "--json", "deps", a)
	if direct.code != exitOK {
		t.Fatalf("deps: %s", direct.stderr)
	}
	if ids := idsOf(t, direct); len(ids) != 1 || ids[0] != b {
		t.Errorf("deps = %v, want the direct edge [%s]", ids, b)
	}

	all := runCLI(t, dir, nil, "--json", "deps", a, "--transitive")
	if all.code != exitOK {
		t.Fatalf("deps --transitive: %s", all.stderr)
	}
	ids := idsOf(t, all)
	wantIDs(t, ids, b, c)

	// The starting ticket is never in its own answer.
	for _, id := range ids {
		if id == a {
			t.Error("deps included the ticket it was asked about")
		}
	}
}

func TestDepsDependents(t *testing.T) {
	dir := newStore(t)
	a, b, c := chain(t, dir)

	direct := runCLI(t, dir, nil, "--json", "deps", c, "--dependents")
	if ids := idsOf(t, direct); len(ids) != 1 || ids[0] != b {
		t.Errorf("dependents = %v, want [%s]", ids, b)
	}

	all := runCLI(t, dir, nil, "--json", "deps", c, "--dependents", "--transitive")
	wantIDs(t, idsOf(t, all), a, b)
}

// TestDepsTerminatesOnACycle is the property the library's visited set gives.
// A cycle is a real state a store can be in, reported by check, and a walk that
// hangs on one would make deps unusable exactly when it is needed.
func TestDepsTerminatesOnACycle(t *testing.T) {
	dir := newStore(t)
	a := makeTicket(t, dir, "A")
	b := makeTicket(t, dir, "B")
	runCLI(t, dir, nil, "link", a, "--depends-on", b, "--actor", "human:sothr")
	runCLI(t, dir, nil, "link", b, "--depends-on", a, "--actor", "human:sothr")

	// If this loops the test binary hangs and the panic on timeout says where.
	for _, args := range [][]string{
		{"deps", a, "--transitive"},
		{"deps", a, "--dependents", "--transitive"},
	} {
		got := runCLI(t, dir, nil, append([]string{"--json"}, args...)...)
		if got.code != exitOK {
			t.Fatalf("%v: %s", args, got.stderr)
		}
		if ids := idsOf(t, got); len(ids) != 1 || ids[0] != b {
			t.Errorf("%v = %v, want just [%s]", args, ids, b)
		}
	}
}

func TestDepsOutput(t *testing.T) {
	dir := newStore(t)
	a, b, _ := chain(t, dir)

	// A read, so it is a ticket-list and not a mutation.
	got := runCLI(t, dir, nil, "--json", "deps", a)
	if kind := decode(t, got.stdout)["kind"]; kind != "ticket-list" {
		t.Errorf("kind = %v, want ticket-list", kind)
	}

	// An empty answer still carries an array.
	got = runCLI(t, dir, nil, "--json", "deps", a, "--dependents")
	tickets, ok := decode(t, got.stdout)["tickets"].([]any)
	if !ok || len(tickets) != 0 {
		t.Errorf("tickets = %v, want an empty array", decode(t, got.stdout)["tickets"])
	}

	// Human mode says which direction found nothing, since an empty list on
	// its own does not say whether it looked up or down.
	human := runCLI(t, dir, nil, "deps", a, "--dependents")
	if !strings.Contains(human.stdout, "Nothing depends on it.") {
		t.Errorf("dependents with no answer: %q", human.stdout)
	}
	human = runCLI(t, dir, nil, "deps", b, "--dependents")
	if !strings.Contains(human.stdout, "A, the top") {
		t.Errorf("dependents should list the ticket above:\n%s", human.stdout)
	}

	if got := runCLI(t, dir, nil, "--json", "deps"); errCode(t, got) != codeUsage {
		t.Error("deps with no ID should be a usage error")
	}
}

// TestDepsPointsAtChildren covers the section 8 rule that an empty deps answer
// names the ticket's children, because "It depends on nothing." is true and
// useless on an epic. deps still walks dependencies alone: this is a pointer at
// list --parent, not a second edge kind mixed into the result.
func TestDepsPointsAtChildren(t *testing.T) {
	dir := newStore(t)
	withParent := func(title, parent string) string {
		t.Helper()
		got := runCLI(t, dir, nil, "--json", "create", "--title", title,
			"--parent", parent, "--actor", "human:sothr")
		if got.code != exitOK {
			t.Fatalf("create %q: %s", title, got.stderr)
		}
		return ticketID(t, decode(t, got.stdout))
	}

	epic := makeTicket(t, dir, "An epic with two slices")
	withParent("First slice", epic)
	withParent("Second slice", epic)
	solo := makeTicket(t, dir, "An epic with one slice")
	withParent("The only slice", solo)
	leaf := makeTicket(t, dir, "No children at all")

	// The case the plan complains about: an epic answering "nothing".
	got := runCLI(t, dir, nil, "deps", epic)
	if !strings.Contains(got.stdout, "It depends on nothing.") {
		t.Errorf("the direction message went missing:\n%s", got.stdout)
	}
	if !strings.Contains(got.stdout, "2 children") ||
		!strings.Contains(got.stdout, "list --parent "+epic) {
		t.Errorf("deps on an epic should name the children and the command:\n%s", got.stdout)
	}

	// One child reads as English rather than as "1 children".
	if got := runCLI(t, dir, nil, "deps", solo); !strings.Contains(got.stdout, "1 child;") {
		t.Errorf("a single child should be singular:\n%s", got.stdout)
	}

	// A childless ticket keeps the bare message it always had.
	if got := runCLI(t, dir, nil, "deps", leaf); strings.Contains(got.stdout, "child") {
		t.Errorf("a childless ticket should get no hint:\n%s", got.stdout)
	}

	// The reverse direction keeps its own wording and still points.
	got = runCLI(t, dir, nil, "deps", epic, "--dependents")
	if !strings.Contains(got.stdout, "Nothing depends on it.") ||
		!strings.Contains(got.stdout, "2 children") {
		t.Errorf("dependents on an epic:\n%s", got.stdout)
	}

	// The hint is for a person. The JSON contract does not carry it, so the
	// envelope keeps exactly the four keys 10.1 gives a ticket-list.
	envelope := decode(t, runCLI(t, dir, nil, "--json", "deps", epic).stdout)
	if len(envelope) != 4 || envelope["kind"] != "ticket-list" {
		t.Errorf("the JSON envelope gained something: %v", envelope)
	}
	// Every file in this store parses, so the channel is present and empty
	// rather than absent, per 10.1.
	if broken, ok := envelope["unreadable"].([]any); !ok || len(broken) != 0 {
		t.Errorf("unreadable = %v, want an empty array", envelope["unreadable"])
	}
	if tickets, ok := envelope["tickets"].([]any); !ok || len(tickets) != 0 {
		t.Errorf("tickets = %v, want an empty array", envelope["tickets"])
	}
}

// TestCreateAcceptsAnIDPrefix covers the same 5.5 rule on create, which passed
// its --depends-on and --parent through to the library unresolved.
func TestCreateAcceptsAnIDPrefix(t *testing.T) {
	dir := newStore(t)
	parent := makeTicket(t, dir, "The epic")
	dep := makeTicket(t, dir, "The dependency")
	prefix := func(id string) string { return id[:len(id)-8] }

	got := runCLI(t, dir, nil, "--json", "create", "--title", "The child",
		"--parent", prefix(parent), "--depends-on", prefix(dep), "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create with prefixes: %s", got.stderr)
	}
	child := ticketID(t, decode(t, got.stdout))

	tk := showTicket(t, dir, child)
	if tk["parent"] != parent {
		t.Errorf("parent = %v, want the resolved %s", tk["parent"], parent)
	}
	deps := dependenciesOf(t, dir, child)
	if len(deps) != 1 || deps[0] != dep {
		t.Errorf("dependencies = %v, want the resolved [%s]", deps, dep)
	}
}

// TestShortestUniqueAbbreviation is a unit test of the rule, and then the
// property that matters: what a listing prints has to resolve.
func TestShortestUniqueAbbreviation(t *testing.T) {
	// Two IDs that agree for the first twelve characters, which is what
	// tickets created in the same millisecond look like.
	ids := []string{
		"TKT-01M1F5JY1MCPGAQPMCWK23HASQ",
		"TKT-01M1F5JY345C084Q3KP4RRY4EJ",
		"TKT-01ZZZZZZZZZZZZZZZZZZZZZZZZ",
	}
	short := shortestUnique(ids)
	seen := map[string]bool{}
	for _, id := range ids {
		s := short[id]
		if !strings.HasPrefix(id, s) {
			t.Errorf("%s abbreviated to %q, which is not a prefix of it", id, s)
		}
		if seen[s] {
			t.Errorf("%q is the abbreviation of more than one ID", s)
		}
		seen[s] = true
	}
	// The far-apart one stays at the floor; the two close ones have to grow.
	floor := len(ticket.IDPrefix) + abbrevLen
	if got := short[ids[2]]; len(got) != floor {
		t.Errorf("a distinct ID abbreviated to %q, want the floor length", got)
	}
	if len(short[ids[0]]) <= floor {
		t.Errorf("two IDs sharing a long prefix must grow past the floor, got %q", short[ids[0]])
	}
}

// TestListingAbbreviationsResolve is the regression test. Tickets created in
// one run share their ULID timestamp characters, so a fixed-width abbreviation
// printed the same prefix on more than one row.
func TestListingAbbreviationsResolve(t *testing.T) {
	dir := newStore(t)
	for i := range 5 {
		makeTicket(t, dir, string(rune('A'+i))+", a ticket")
	}

	list := runCLI(t, dir, nil, "list")
	if list.code != exitOK {
		t.Fatalf("list: %s", list.stderr)
	}
	lines := strings.Split(strings.TrimSpace(list.stdout), "\n")
	if len(lines) != 5 {
		t.Fatalf("got %d rows, want 5:\n%s", len(lines), list.stdout)
	}

	seen := map[string]bool{}
	for _, line := range lines {
		abbrev := strings.Fields(line)[0]
		if seen[abbrev] {
			t.Errorf("%q appears on more than one row", abbrev)
		}
		seen[abbrev] = true

		// The whole point: it has to resolve to exactly one ticket.
		if got := runCLI(t, dir, nil, "show", abbrev); got.code != exitOK {
			t.Errorf("the listing printed %q, which does not resolve: %s", abbrev, got.stderr)
		}
	}
}
