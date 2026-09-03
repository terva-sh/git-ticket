package ticket

import (
	"fmt"
	"slices"
	"testing"
	"time"
)

// liveClaim is a claim with no expiry, which never expires and therefore always
// holds the ticket.
func liveClaim() *Claim { return &Claim{Actor: "agent:other/session"} }

// TestReadinessReasonNamesEveryStatus is the gap this field was filed for. Ten
// of the eleven open tickets in this repository's own store reported not ready,
// not blocked, and three empty slices, which is correct and says nothing.
//
// Building the tickets directly rather than walking transitions keeps the table
// exhaustive: every status in 6.1 appears, including the ones a lifecycle walk
// would make tedious to reach.
func TestReadinessReasonNamesEveryStatus(t *testing.T) {
	for _, status := range Statuses {
		t.Run(status, func(t *testing.T) {
			tk := &Ticket{ID: "TKT-01K3ZZAAA000000000000001", Status: status}
			r := readinessOf([]*Ticket{tk}, referenceInstant)[tk.ID]

			if status == StatusReady {
				if !r.Ready || r.Reason != "" {
					t.Errorf("ready = %+v, want ready with no reason", r)
				}
				return
			}
			if r.Ready {
				t.Errorf("%s should not be ready", status)
			}
			// The status is its own reason, so a status added to 6.1 needs no
			// decision here.
			if r.Reason != status {
				t.Errorf("reason = %q, want %q", r.Reason, status)
			}
		})
	}
}

// TestReadinessReasonPrecedence pins the order the enum exists to settle. More
// than one thing can be true at once, and a set of booleans would leave every
// consumer to decide which wins, which is the same rule answered in several
// places and eventually answered differently.
func TestReadinessReasonPrecedence(t *testing.T) {
	blocker := &Ticket{ID: "TKT-01K3ZZAAA000000000000001", Status: StatusReady}

	cases := []struct {
		name   string
		ticket *Ticket
		want   string
	}{
		{
			// Promotion is the move that comes first. Nobody acts on the
			// dependency of a ticket that is not in the queue yet.
			name: "status beats a dependency",
			ticket: &Ticket{
				ID: "TKT-01K3ZZAAA000000000000002", Status: StatusDraft,
				Dependencies: []string{blocker.ID},
			},
			want: StatusDraft,
		},
		{
			name: "status beats a claim",
			ticket: &Ticket{
				ID: "TKT-01K3ZZAAA000000000000003", Status: StatusDraft,
				Claim: liveClaim(),
			},
			want: StatusDraft,
		},
		{
			// A claim is advisory and reserves nothing, so the dependency is
			// the thing that actually has to change.
			name: "a dependency beats a claim",
			ticket: &Ticket{
				ID: "TKT-01K3ZZAAA000000000000004", Status: StatusReady,
				Dependencies: []string{blocker.ID}, Claim: liveClaim(),
			},
			want: ReasonWaitingOnDependencies,
		},
		{
			name: "a claim alone",
			ticket: &Ticket{
				ID: "TKT-01K3ZZAAA000000000000005", Status: StatusReady,
				Claim: liveClaim(),
			},
			want: ReasonClaimed,
		},
		{
			name:   "nothing in the way",
			ticket: &Ticket{ID: "TKT-01K3ZZAAA000000000000006", Status: StatusReady},
			want:   "",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			all := []*Ticket{blocker, c.ticket}
			r := readinessOf(all, referenceInstant)[c.ticket.ID]
			if r.Reason != c.want {
				t.Errorf("reason = %q, want %q", r.Reason, c.want)
			}
		})
	}
}

// TestReadinessReasonPopulatesTheSlicesToo covers the half of the precedence
// that could quietly lose information. Naming the status as the reason must not
// stop the dependency arrays being filled, or the choice would hide a blocker
// rather than rank it.
func TestReadinessReasonPopulatesTheSlicesToo(t *testing.T) {
	blocker := &Ticket{ID: "TKT-01K3ZZAAA000000000000001", Status: StatusReady}
	draft := &Ticket{
		ID: "TKT-01K3ZZAAA000000000000002", Status: StatusDraft,
		Dependencies: []string{blocker.ID},
	}

	r := readinessOf([]*Ticket{blocker, draft}, referenceInstant)[draft.ID]

	if r.Reason != StatusDraft {
		t.Errorf("reason = %q, want draft", r.Reason)
	}
	if !r.Blocked {
		t.Error("isBlocked should still be true: the dependency is real")
	}
	if !slices.Equal(r.Blocking, []string{blocker.ID}) {
		t.Errorf("blocking = %v, want the dependency named", r.Blocking)
	}
}

// TestReasonIsEmptyExactlyWhenReady is the invariant worth holding, rather than
// the list of values. The reason and the verdict are built from the same three
// inputs, so they cannot come to disagree about a ticket.
func TestReasonIsEmptyExactlyWhenReady(t *testing.T) {
	blocker := &Ticket{ID: "TKT-01K3ZZAAA000000000000001", Status: StatusDraft}

	var all []*Ticket
	all = append(all, blocker)
	id := 2
	for _, status := range Statuses {
		for _, deps := range [][]string{nil, {blocker.ID}} {
			for _, claim := range []*Claim{nil, liveClaim()} {
				all = append(all, &Ticket{
					ID:           fmt.Sprintf("TKT-01K3ZZAAA%017d", id),
					Status:       status,
					Dependencies: deps,
					Claim:        claim,
				})
				id++
			}
		}
	}

	got := readinessOf(all, referenceInstant)
	for _, tk := range all {
		r := got[tk.ID]
		if r.Ready != (r.Reason == "") {
			t.Errorf("%s status=%s: ready=%v but reason=%q",
				tk.ID, tk.Status, r.Ready, r.Reason)
		}
	}
}

// TestUnreadyReasonsPublishesEveryValue is the guard on the published list. A
// value the code can emit and schema does not name would leave a consumer
// switching on the field with no case for it, which is the failure publishing
// the list exists to prevent.
func TestUnreadyReasonsPublishesEveryValue(t *testing.T) {
	seen := map[string]bool{}
	for _, status := range Statuses {
		for _, blocked := range []bool{false, true} {
			for _, held := range []bool{false, true} {
				if r := unreadyReason(status, blocked, held); r != "" {
					seen[r] = true
				}
			}
		}
	}

	for reason := range seen {
		if !slices.Contains(UnreadyReasons, reason) {
			t.Errorf("%q can be emitted and UnreadyReasons does not name it", reason)
		}
	}
	// And the other direction, so the list does not advertise a value nothing
	// produces.
	for _, reason := range UnreadyReasons {
		if !seen[reason] {
			t.Errorf("UnreadyReasons names %q and nothing emits it", reason)
		}
	}
	if !slices.Contains(UnreadyReasons, ReasonWaitingOnDependencies) ||
		!slices.Contains(UnreadyReasons, ReasonClaimed) {
		t.Errorf("the two non-status reasons are missing: %v", UnreadyReasons)
	}
	if slices.Contains(UnreadyReasons, StatusReady) {
		t.Errorf("ready is not a reason to be unready: %v", UnreadyReasons)
	}
}

// TestExpiredClaimIsNotAReason follows the rule 6.4 already sets. An expired
// claim grants no exclusivity, so it cannot be what stands in the way.
func TestExpiredClaimIsNotAReason(t *testing.T) {
	past := Timestamp{Time: referenceInstant.Add(-time.Hour)}
	tk := &Ticket{
		ID: "TKT-01K3ZZAAA000000000000001", Status: StatusReady,
		Claim: &Claim{Actor: "agent:other/session", ExpiresAt: &past},
	}

	r := readinessOf([]*Ticket{tk}, referenceInstant)[tk.ID]

	if !r.Ready || r.Reason != "" {
		t.Errorf("got %+v, want ready with no reason: the claim has expired", r)
	}
}
