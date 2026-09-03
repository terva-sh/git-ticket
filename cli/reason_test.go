package cli

import (
	"testing"
)

// reasonOf pulls readiness.reason out of a ticket envelope.
func reasonOf(t *testing.T, env map[string]any) (reason string, isReady bool) {
	t.Helper()
	tk, ok := env["ticket"].(map[string]any)
	if !ok {
		t.Fatalf("envelope carries no ticket: %v", env)
	}
	r, ok := tk["readiness"].(map[string]any)
	if !ok {
		t.Fatalf("ticket carries no readiness: %v", tk)
	}
	got, ok := r["reason"].(string)
	if !ok {
		t.Fatalf("reason is %T, want a string: %v", r["reason"], r)
	}
	ready, _ := r["isReady"].(bool)
	return got, ready
}

func showJSON(t *testing.T, dir, id string) map[string]any {
	t.Helper()
	got := runCLI(t, dir, nil, "--json", "show", id)
	if got.code != exitOK {
		t.Fatalf("show: %s%s", got.stdout, got.stderr)
	}
	return decode(t, got.stdout)
}

// TestReadinessReasonIsPublished covers the key end to end. The library
// computing a reason is worth nothing if the envelope does not carry it, and
// this is the surface a consumer drawing a board actually reads.
func TestReadinessReasonIsPublished(t *testing.T) {
	dir := newStore(t)
	const actor = "human:sothr"

	// A draft, which is the case that reported nothing before this field. Ten
	// of the eleven open tickets in this repository's own store were this.
	draft := ticketID(t, createTicket(t, dir))
	if got, ready := reasonOf(t, showJSON(t, dir, draft)); got != "draft" || ready {
		t.Errorf("draft: reason = %q, isReady = %v, want \"draft\" and false", got, ready)
	}

	// Ready and startable, the one case with no reason at all.
	open := ticketID(t, createTicket(t, dir))
	runCLI(t, dir, nil, "status", open, "ready", "--actor", actor)
	if got, ready := reasonOf(t, showJSON(t, dir, open)); got != "" || !ready {
		t.Errorf("open: reason = %q, isReady = %v, want empty and true", got, ready)
	}

	// A live claim on an otherwise startable ticket.
	held := ticketID(t, createTicket(t, dir))
	runCLI(t, dir, nil, "status", held, "ready", "--actor", actor)
	runCLI(t, dir, nil, "claim", held, "--actor", actor)
	if got, ready := reasonOf(t, showJSON(t, dir, held)); got != "claimed" || ready {
		t.Errorf("held: reason = %q, isReady = %v, want \"claimed\" and false", got, ready)
	}

	// A dependency that is not done. Spelled out rather than called "blocked",
	// because isBlocked already means the graph and the blocked status means a
	// person marked it.
	waiter := ticketID(t, createTicket(t, dir))
	runCLI(t, dir, nil, "link", waiter, "--depends-on", draft, "--actor", actor)
	runCLI(t, dir, nil, "status", waiter, "ready", "--actor", actor)
	if got, ready := reasonOf(t, showJSON(t, dir, waiter)); got != "waiting_on_dependencies" || ready {
		t.Errorf("waiter: reason = %q, isReady = %v, want \"waiting_on_dependencies\" and false",
			got, ready)
	}
}

// TestReadinessReasonAppearsInAListing checks the other kind that carries
// readiness. A consumer greying out cards reads a listing, not one ticket at a
// time, which is the call this field exists to save.
func TestReadinessReasonAppearsInAListing(t *testing.T) {
	dir := newStore(t)
	createTicket(t, dir)

	got := runCLI(t, dir, nil, "--json", "list")
	if got.code != exitOK {
		t.Fatalf("list: %s%s", got.stdout, got.stderr)
	}
	env := decode(t, got.stdout)

	tickets, ok := env["tickets"].([]any)
	if !ok || len(tickets) == 0 {
		t.Fatalf("tickets = %v, want at least one", env["tickets"])
	}
	for _, v := range tickets {
		r, ok := v.(map[string]any)["readiness"].(map[string]any)
		if !ok {
			t.Fatalf("a row carries no readiness: %v", v)
		}
		if _, ok := r["reason"].(string); !ok {
			t.Errorf("a listing row carries no reason: %v", r)
		}
	}
}
