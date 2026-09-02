package cli

import (
	"slices"
	"testing"
)

// readinessOf pulls the readiness block out of a shown ticket.
func readinessOf(t *testing.T, dir, id string) map[string]any {
	t.Helper()
	r, ok := showTicket(t, dir, id)["readiness"].(map[string]any)
	if !ok {
		t.Fatalf("ticket %s carries no readiness block", id)
	}
	return r
}

// strList reads a JSON array of strings, treating a missing key as a failure
// rather than as an empty list, because plan 10 renders an absent collection as
// [] and never omits it.
func strList(t *testing.T, m map[string]any, key string) []string {
	t.Helper()
	raw, ok := m[key]
	if !ok {
		t.Fatalf("key %q is missing from %v", key, m)
	}
	if raw == nil {
		t.Fatalf("key %q is null, want [] per plan 10", key)
	}
	items, ok := raw.([]any)
	if !ok {
		t.Fatalf("key %q is %T, want an array", key, raw)
	}
	out := make([]string, 0, len(items))
	for _, i := range items {
		out = append(out, i.(string))
	}
	return out
}

// TestBlocksOnThroughTheCLI covers the whole path: the flag on create, the flag
// on update, the frontmatter field, and the two JSON keys.
//
// blockingChildren is a key of its own rather than more entries in
// blockingDependencies. A consumer rendering "waiting on" from that older key
// would otherwise print a child ID labelled as a dependency, with nothing in
// the payload to tell the two apart.
func TestBlocksOnThroughTheCLI(t *testing.T) {
	dir := newStore(t)

	epic := ticketID(t, createTicket(t, dir, "--title", "Ship token refresh", "--type", "epic", "--blocks-on", "children"))
	if got := showTicket(t, dir, epic)["blocksOn"]; got != "children" {
		t.Errorf("blocksOn = %v, want children", got)
	}

	// With no children yet it is not blocked, which is the settled answer:
	// blocking it would name no blocker.
	r := readinessOf(t, dir, epic)
	if r["isBlocked"] != false {
		t.Errorf("isBlocked = %v, want false for an epic with no children", r["isBlocked"])
	}
	if got := strList(t, r, "blockingChildren"); len(got) != 0 {
		t.Errorf("blockingChildren = %v, want empty", got)
	}

	// check says so instead.
	report := decode(t, runCLI(t, dir, nil, "--json", "check").stdout)
	if !hasCode(t, report, "warnings", "blocks_on_no_children") {
		t.Errorf("check should warn blocks_on_no_children: %v", report)
	}

	// A child arrives and now the epic waits on it.
	kid := ticketID(t, createTicket(t, dir, "--title", "Refresh endpoint", "--parent", epic))
	runCLI(t, dir, nil, "status", epic, "ready", "--actor", "human:sothr")

	r = readinessOf(t, dir, epic)
	if r["isBlocked"] != true {
		t.Errorf("isBlocked = %v, want true once a child exists", r["isBlocked"])
	}
	if got := strList(t, r, "blockingChildren"); len(got) != 1 || got[0] != kid {
		t.Errorf("blockingChildren = %v, want [%s]", got, kid)
	}
	// The older key keeps its old meaning.
	if got := strList(t, r, "blockingDependencies"); len(got) != 0 {
		t.Errorf("blockingDependencies = %v, want empty: a child is not a dependency", got)
	}
	if ids := idsOf(t, runCLI(t, dir, nil, "--json", "ready")); slices.Contains(ids, epic) {
		t.Errorf("ready = %v, should not offer a blocked epic", ids)
	}

	// update --blocks-on none puts it back, and the epic becomes startable.
	if got := runCLI(t, dir, nil, "update", epic, "--blocks-on", "none", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("update --blocks-on none: %s", got.stderr)
	}
	if got := showTicket(t, dir, epic)["blocksOn"]; got != "none" {
		t.Errorf("blocksOn = %v, want none", got)
	}
	if r := readinessOf(t, dir, epic); r["isBlocked"] != false {
		t.Errorf("isBlocked = %v, want false once it stops gating on children", r["isBlocked"])
	}

	// A value outside the set is a usage error on both commands, not a write
	// that fails later.
	for _, args := range [][]string{
		{"create", "--title", "x", "--blocks-on", "childrn"},
		{"update", epic, "--blocks-on", "childrn"},
	} {
		if got := runCLI(t, dir, nil, append(args, "--actor", "human:sothr")...); got.code == exitOK {
			t.Errorf("%v should have been rejected", args)
		}
	}
}

// hasCode reports whether a check envelope carries a finding with this code in
// the named collection.
func hasCode(t *testing.T, envelope map[string]any, collection, code string) bool {
	t.Helper()
	items, ok := envelope[collection].([]any)
	if !ok {
		t.Fatalf("envelope carries no %s: %v", collection, envelope)
	}
	for _, i := range items {
		if i.(map[string]any)["code"] == code {
			return true
		}
	}
	return false
}
