package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// checklistOf returns one section as the CLI's own JSON reports it, which is
// the derived checklist view of plan 10.1 with its explicit indexes.
func checklistOf(t *testing.T, dir, id, section string) []any {
	t.Helper()
	lists := showTicket(t, dir, id)["checklists"].(map[string]any)
	items, _ := lists[section].([]any)
	return items
}

func TestUpdateChangesSeveralFieldsAtOnce(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir, "--label", "auth"))

	got := runCLI(t, dir, nil, "update", id,
		"--title", "Rotate the signing key",
		"--priority", "urgent",
		"--milestone", "v1.2",
		"--add-label", "crypto",
		"--assign", "human:sothr",
		"--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("update: %s", got.stderr)
	}

	tk := showTicket(t, dir, id)
	if tk["title"] != "Rotate the signing key" {
		t.Errorf("title = %v", tk["title"])
	}
	if tk["priority"] != "urgent" {
		t.Errorf("priority = %v", tk["priority"])
	}
	if tk["milestone"] != "v1.2" {
		t.Errorf("milestone = %v", tk["milestone"])
	}
	labels := tk["labels"].([]any)
	if len(labels) != 2 || labels[0] != "auth" || labels[1] != "crypto" {
		t.Errorf("labels = %v, want the original plus the added one", labels)
	}
	if assignees := tk["assignees"].([]any); len(assignees) != 1 || assignees[0] != "human:sothr" {
		t.Errorf("assignees = %v", assignees)
	}

	// One write, so one revision. The human line names what moved.
	if !strings.Contains(got.stdout, "title") || !strings.Contains(got.stdout, "labels") {
		t.Errorf("the summary does not say what changed: %q", got.stdout)
	}
}

// TestUpdateTellsEmptyFromAbsent is why runUpdate reads fs.Visit rather than
// checking for a zero value. Clearing a field and leaving it alone are
// different instructions that both arrive as an empty string.
func TestUpdateTellsEmptyFromAbsent(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	runCLI(t, dir, nil, "update", id, "--milestone", "v1.2", "--actor", "human:sothr")
	if m := showTicket(t, dir, id)["milestone"]; m != "v1.2" {
		t.Fatalf("milestone = %v, want v1.2", m)
	}

	// A different flag entirely must not disturb the milestone.
	runCLI(t, dir, nil, "update", id, "--priority", "high", "--actor", "human:sothr")
	if m := showTicket(t, dir, id)["milestone"]; m != "v1.2" {
		t.Errorf("milestone = %v, want it untouched by an unrelated update", m)
	}

	// An explicit empty value clears it.
	if got := runCLI(t, dir, nil, "update", id, "--milestone", "", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("update --milestone \"\": %s", got.stderr)
	}
	if m := showTicket(t, dir, id)["milestone"]; m != nil {
		t.Errorf("milestone = %v, want null after an explicit clear", m)
	}
}

// TestUpdateChangesTypeAndParent covers the two flags plan 15.8 asked about.
// A bug that turns out to be a chore, and a ticket that belongs under a
// different epic, are both correctable without hand-editing the file.
func TestUpdateChangesTypeAndParent(t *testing.T) {
	dir := newStore(t)
	epic := makeTicket(t, dir, "Authentication")
	id := ticketID(t, createTicket(t, dir, "--type", "bug"))

	if got := runCLI(t, dir, nil, "update", id, "--type", "chore", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("update --type: %s", got.stderr)
	}
	if k := showTicket(t, dir, id)["type"]; k != "chore" {
		t.Errorf("type = %v, want chore", k)
	}

	// The parent goes through resolveID like every other ID the CLI takes, so
	// the TKT- part is optional. A short prefix would work too, but it is not
	// worth a test that flakes: two tickets made in the same millisecond share
	// the first ten characters of their ULID and differ only by chance after.
	if got := runCLI(t, dir, nil, "update", id, "--parent", epic[4:], "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("update --parent: %s", got.stderr)
	}
	if p := showTicket(t, dir, id)["parent"]; p != epic {
		t.Errorf("parent = %v, want the canonical %s", p, epic)
	}

	// An empty --parent clears it, the same rule --milestone follows.
	if got := runCLI(t, dir, nil, "update", id, "--parent", "", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("update --parent \"\": %s", got.stderr)
	}
	if p := showTicket(t, dir, id)["parent"]; p != nil {
		t.Errorf("parent = %v, want null after an explicit clear", p)
	}
}

// TestUpdateRefusesABadParent leaves the two refusals to the library, and
// checks the CLI reports the codes rather than swallowing them.
func TestUpdateRefusesABadParent(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	got := runCLI(t, dir, nil, "--json", "update", id, "--parent", id, "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("a ticket should not be able to parent itself")
	}
	if code := errCode(t, got); code != "parent_cycle" {
		t.Errorf("code = %v, want parent_cycle", code)
	}

	got = runCLI(t, dir, nil, "--json", "update", id,
		"--parent", "TKT-01ZZZZZZZZZZZZZZZZZZZZZZZZ", "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("a parent that is not in the store should be refused")
	}
	if code := errCode(t, got); code != "ticket_not_found" {
		t.Errorf("code = %v, want ticket_not_found", code)
	}
}

// TestUpdateNeedsSomethingToChange refuses a write that would say nothing,
// rather than bumping updated_at for no reason.
func TestUpdateNeedsSomethingToChange(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	got := runCLI(t, dir, nil, "--json", "update", id, "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("update with no flags should be refused")
	}
	if code := errCode(t, got); code != codeUsage {
		t.Errorf("code = %v, want %s", code, codeUsage)
	}

	// And it takes exactly one ID.
	if got := runCLI(t, dir, nil, "--json", "update", "--title", "x"); errCode(t, got) != codeUsage {
		t.Error("update with no ID should be a usage error")
	}
}

func TestUpdateValidatesPriorityBeforeTheStore(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	got := runCLI(t, dir, nil, "--json", "update", id, "--priority", "frobnicate", "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("an invalid priority should be refused")
	}
	if code := errCode(t, got); code != codeUsage {
		t.Errorf("code = %v, want %s", code, codeUsage)
	}

	got = runCLI(t, dir, nil, "--json", "update", id, "--type", "frobnicate", "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("an invalid type should be refused")
	}
	if code := errCode(t, got); code != codeUsage {
		t.Errorf("code = %v, want %s", code, codeUsage)
	}
}

// TestUpdateAppliesNothingWhenAnyPartFails is the property Mutations gives:
// the file is written after the last change succeeds, so a caller that asked
// for two things and got an error got neither.
func TestUpdateAppliesNothingWhenAnyPartFails(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))
	before := showTicket(t, dir, id)

	// A title of only spaces is empty once trimmed, which the library
	// refuses. The priority alongside it is perfectly valid.
	got := runCLI(t, dir, nil, "--json", "update", id,
		"--priority", "urgent", "--title", "   ", "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("an empty title should be refused")
	}
	if code := errCode(t, got); code != "invalid_field" {
		t.Errorf("code = %v, want invalid_field", code)
	}

	after := showTicket(t, dir, id)
	if after["priority"] != before["priority"] {
		t.Errorf("priority = %v, want %v: the valid half of a failed update must not land",
			after["priority"], before["priority"])
	}
	if after["title"] != before["title"] {
		t.Errorf("title = %v, want it unchanged", after["title"])
	}
	if after["revision"] != before["revision"] {
		t.Error("a refused update still wrote the file")
	}
}

func TestUpdateLabelsAndAssignees(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir, "--label", "auth", "--assignee", "human:sothr"))

	// Removals run before additions, so a label named in both ends present
	// whichever order the flags were typed.
	runCLI(t, dir, nil, "update", id, "--add-label", "auth", "--remove-label", "auth",
		"--actor", "human:sothr")
	labels := showTicket(t, dir, id)["labels"].([]any)
	if len(labels) != 1 || labels[0] != "auth" {
		t.Errorf("labels = %v, want auth still present", labels)
	}

	// Repeating a flag adds each value, and a label is a set.
	runCLI(t, dir, nil, "update", id, "--add-label", "crypto", "--add-label", "release",
		"--add-label", "crypto", "--actor", "human:sothr")
	labels = showTicket(t, dir, id)["labels"].([]any)
	if len(labels) != 3 {
		t.Errorf("labels = %v, want three distinct labels", labels)
	}

	runCLI(t, dir, nil, "update", id, "--unassign", "human:sothr", "--actor", "human:sothr")
	if a := showTicket(t, dir, id)["assignees"].([]any); len(a) != 0 {
		t.Errorf("assignees = %v, want empty", a)
	}
	// Removing something that is not there is not an error.
	if got := runCLI(t, dir, nil, "update", id, "--remove-label", "nosuch",
		"--actor", "human:sothr"); got.code != exitOK {
		t.Errorf("removing an absent label: %s", got.stderr)
	}
}

func TestACAddsChecksAndUnchecks(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	if items := checklistOf(t, dir, id, "acceptanceCriteria"); len(items) != 0 {
		t.Fatalf("a new ticket has %d criteria, want none", len(items))
	}

	for _, text := range []string{"The verifier accepts either key", "New tokens use the newer key"} {
		if got := runCLI(t, dir, nil, "ac", id, "--add", text, "--actor", "human:sothr"); got.code != exitOK {
			t.Fatalf("ac --add: %s", got.stderr)
		}
	}
	items := checklistOf(t, dir, id, "acceptanceCriteria")
	if len(items) != 2 {
		t.Fatalf("got %d criteria, want 2", len(items))
	}
	// An added item starts unchecked, and the indexes count from one.
	first := items[0].(map[string]any)
	if first["index"].(float64) != 1 || first["checked"] != false {
		t.Errorf("first item = %v, want index 1 unchecked", first)
	}

	if got := runCLI(t, dir, nil, "ac", id, "--check", "2", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("ac --check: %s", got.stderr)
	}
	items = checklistOf(t, dir, id, "acceptanceCriteria")
	if items[1].(map[string]any)["checked"] != true {
		t.Error("--check 2 did not check the second item")
	}
	if items[0].(map[string]any)["checked"] != false {
		t.Error("--check 2 also touched the first item")
	}

	if got := runCLI(t, dir, nil, "ac", id, "--uncheck", "2", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("ac --uncheck: %s", got.stderr)
	}
	if checklistOf(t, dir, id, "acceptanceCriteria")[1].(map[string]any)["checked"] != false {
		t.Error("--uncheck 2 did not uncheck the second item")
	}
}

// TestACIndexCountsCheckboxesNotLines is the case that made the index explicit
// in plan 10.1. A section may hold prose, and the number a person types counts
// checkbox lines only.
func TestACIndexCountsCheckboxesNotLines(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	// Written the way a person writes it, with the list under a sentence.
	path := filepath.Join(dir, ".tickets", "tickets", id+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	section := "\n## Acceptance criteria\n\nThese all have to hold before release:\n\n" +
		"- [ ] The verifier accepts either key\n- [ ] New tokens use the newer key\n"
	if err := os.WriteFile(path, append(data, []byte(section)...), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := runCLI(t, dir, nil, "ac", id, "--check", "2", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("ac --check 2: %s", got.stderr)
	}

	items := checklistOf(t, dir, id, "acceptanceCriteria")
	if len(items) != 2 {
		t.Fatalf("got %d criteria, want 2", len(items))
	}
	if items[1].(map[string]any)["checked"] != true {
		t.Error("item 2 is the second checkbox, not the second line")
	}
	if items[0].(map[string]any)["checked"] != false {
		t.Error("the first item moved")
	}

	// The prose a person wrote is still there.
	body := showTicket(t, dir, id)["body"].(map[string]any)["acceptanceCriteria"].(string)
	if !strings.Contains(body, "These all have to hold before release:") {
		t.Errorf("editing a checkbox dropped the prose:\n%s", body)
	}
}

// TestACWantsAtLeastOneOperation covers the one thing still refused now that
// the operations combine: an invocation that asks for nothing.
func TestACWantsAtLeastOneOperation(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	got := runCLI(t, dir, nil, "--json", "ac", id, "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("ac with no operation should be refused")
	}
	if code := errCode(t, got); code != codeUsage {
		t.Errorf("code = %v, want %s", code, codeUsage)
	}
	// The message names every operation, because a caller who typed none of
	// them needs the list rather than a complaint.
	msg := decode(t, got.stdout)["error"].(map[string]any)["message"].(string)
	for _, flag := range []string{"--add", "--check", "--uncheck", "--remove"} {
		if !strings.Contains(msg, flag) {
			t.Errorf("the message should name %s: %q", flag, msg)
		}
	}
}

// TestChecklistOperationsCombine is the contract this command now offers: every
// index means the item the caller read when they typed it, whatever else is in
// the same invocation.
//
// The ordering is what makes that true. Checks move nothing, removals renumber
// everything below them so they run highest first, and adds append last. Get it
// wrong and `--check 3 --remove 1` ticks the wrong box.
func TestChecklistOperationsCombine(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir,
		"--ac", "one", "--ac", "two", "--ac", "three", "--ac", "four"))

	got := runCLI(t, dir, nil, "ac", id, "--check", "3", "--remove", "1",
		"--remove", "4", "--add", "five", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("combined operations: %s", got.stderr)
	}

	items := checklistOf(t, dir, id, "acceptanceCriteria")
	want := []struct {
		text    string
		checked bool
	}{
		{"two", false},
		{"three", true},
		{"five", false},
	}
	if len(items) != len(want) {
		t.Fatalf("got %d items, want %d: %v", len(items), len(want), items)
	}
	for i, w := range want {
		it := items[i].(map[string]any)
		if it["text"] != w.text || it["checked"] != w.checked {
			t.Errorf("item %d = %v, want %q checked=%v", i+1, it, w.text, w.checked)
		}
	}
}

// TestChecklistOperationsAreOneWrite covers the atomicity runUpdate already
// promises. One bad index in a combined invocation leaves the ticket alone
// rather than applying the operations that came before it.
func TestChecklistOperationsAreOneWrite(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir, "--ac", "one", "--ac", "two"))
	before, _ := showTicket(t, dir, id)["revision"].(string)

	got := runCLI(t, dir, nil, "--json", "ac", id, "--check", "1", "--check", "9",
		"--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("an index past the end should refuse the whole invocation")
	}
	if code := errCode(t, got); code != "invalid_field" {
		t.Errorf("code = %v, want invalid_field", code)
	}

	if after, _ := showTicket(t, dir, id)["revision"].(string); after != before {
		t.Error("a refused invocation still wrote to the ticket")
	}
	if items := checklistOf(t, dir, id, "acceptanceCriteria"); items[0].(map[string]any)["checked"] != false {
		t.Error("the valid half of a refused invocation was applied")
	}
}

// TestChecklistRemoveDeletesAnItem is the operation ac and dod lacked: a
// criterion that turned out to be wrong could be unchecked but never dropped.
func TestChecklistRemoveDeletesAnItem(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir, "--ac", "keep", "--ac", "wrong"))

	if got := runCLI(t, dir, nil, "ac", id, "--remove", "2", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("remove: %s", got.stderr)
	}
	items := checklistOf(t, dir, id, "acceptanceCriteria")
	if len(items) != 1 || items[0].(map[string]any)["text"] != "keep" {
		t.Errorf("items = %v, want keep alone", items)
	}
}

func TestACRejectsAnIndexThatIsNotThere(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))
	runCLI(t, dir, nil, "ac", id, "--add", "The only item", "--actor", "human:sothr")

	got := runCLI(t, dir, nil, "--json", "ac", id, "--check", "9", "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatal("an index past the end should be refused")
	}
	if code := errCode(t, got); code != "invalid_field" {
		t.Errorf("code = %v, want invalid_field", code)
	}
	// The message says how many there are, which is what a caller needs next.
	msg := decode(t, got.stdout)["error"].(map[string]any)["message"].(string)
	if !strings.Contains(msg, "1") {
		t.Errorf("the message should say how many items exist: %q", msg)
	}

	// Counting starts at one, so zero is a mistake and not the first item.
	got = runCLI(t, dir, nil, "--json", "ac", id, "--check", "0", "--actor", "human:sothr")
	if got.code != exitError {
		t.Error("index 0 should be refused")
	}
	if items := checklistOf(t, dir, id, "acceptanceCriteria"); items[0].(map[string]any)["checked"] != false {
		t.Error("a refused index still changed an item")
	}
}

// TestDoDIsTheOtherSection checks the two commands do not reach into each
// other, since they share every line of implementation but the section.
func TestDoDIsTheOtherSection(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	runCLI(t, dir, nil, "ac", id, "--add", "An acceptance criterion", "--actor", "human:sothr")
	runCLI(t, dir, nil, "dod", id, "--add", "A done criterion", "--actor", "human:sothr")

	ac := checklistOf(t, dir, id, "acceptanceCriteria")
	dod := checklistOf(t, dir, id, "definitionOfDone")
	if len(ac) != 1 || ac[0].(map[string]any)["text"] != "An acceptance criterion" {
		t.Errorf("acceptance criteria = %v", ac)
	}
	if len(dod) != 1 || dod[0].(map[string]any)["text"] != "A done criterion" {
		t.Errorf("definition of done = %v", dod)
	}

	// Checking item 1 of one section leaves the other alone.
	runCLI(t, dir, nil, "dod", id, "--check", "1", "--actor", "human:sothr")
	if checklistOf(t, dir, id, "acceptanceCriteria")[0].(map[string]any)["checked"] != false {
		t.Error("dod --check reached into the acceptance criteria")
	}
	if checklistOf(t, dir, id, "definitionOfDone")[0].(map[string]any)["checked"] != true {
		t.Error("dod --check did not check its own section")
	}
}

// TestEditCommandsEmitMutationResults keeps the new commands inside the JSON
// contract rather than inventing output of their own.
func TestEditCommandsEmitMutationResults(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	for _, args := range [][]string{
		{"update", id, "--priority", "high"},
		{"ac", id, "--add", "A criterion"},
		{"dod", id, "--add", "A criterion"},
	} {
		got := runCLI(t, dir, nil, append([]string{"--json"}, append(args, "--actor", "human:sothr")...)...)
		if got.code != exitOK {
			t.Fatalf("%s: %s", args[0], got.stderr)
		}
		envelope := decode(t, got.stdout)
		if envelope["kind"] != "mutation-result" {
			t.Errorf("%s kind = %v, want mutation-result", args[0], envelope["kind"])
		}
		tk, ok := envelope["ticket"].(map[string]any)
		if !ok {
			t.Fatalf("%s: no ticket in the envelope", args[0])
		}
		if tk["id"] != id {
			t.Errorf("%s ticket id = %v, want %s", args[0], tk["id"], id)
		}
		if rev, _ := tk["revision"].(string); rev == "" {
			t.Errorf("%s: no revision in the envelope", args[0])
		}
		if paths, _ := envelope["pathsChanged"].([]any); len(paths) != 1 {
			t.Errorf("%s pathsChanged = %v, want the one file", args[0], envelope["pathsChanged"])
		}
	}
}

// TestEditCommandsHonourIfRevision covers the two new writes for the flag the
// rest of the mutations already honour.
func TestEditCommandsHonourIfRevision(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))
	stale, _ := showTicket(t, dir, id)["revision"].(string)

	// Move the ticket on, so the revision above is out of date.
	runCLI(t, dir, nil, "update", id, "--priority", "high", "--actor", "human:sothr")

	for _, args := range [][]string{
		{"update", id, "--priority", "low"},
		{"ac", id, "--add", "A criterion"},
		{"dod", id, "--add", "A criterion"},
	} {
		got := runCLI(t, dir, nil, append([]string{"--json"},
			append(args, "--if-revision", stale, "--actor", "human:sothr")...)...)
		if got.code != exitError {
			t.Errorf("%s ignored --if-revision", args[0])
			continue
		}
		if code := errCode(t, got); code != "stale_revision" {
			t.Errorf("%s: code = %v, want stale_revision", args[0], code)
		}
	}
}

// TestChecklistHumanOutput shows the list back with the numbers the next
// command will take.
func TestChecklistHumanOutput(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	runCLI(t, dir, nil, "ac", id, "--add", "The verifier accepts either key", "--actor", "human:sothr")
	got := runCLI(t, dir, nil, "ac", id, "--add", "New tokens use the newer key", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("ac --add: %s", got.stderr)
	}
	for _, want := range []string{"1", "2", "[ ]", "The verifier accepts either key"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("the printed checklist is missing %q:\n%s", want, got.stdout)
		}
	}

	got = runCLI(t, dir, nil, "ac", id, "--check", "1", "--actor", "human:sothr")
	if !strings.Contains(got.stdout, "[x]") {
		t.Errorf("a checked item does not show as checked:\n%s", got.stdout)
	}
}
