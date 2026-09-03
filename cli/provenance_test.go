package cli

import "testing"

// The provenance pair is settled in plan 5.3 and recorded in 12.4: `updated_at`
// and `updated_by` stay, and they are one fact rather than two fields. Nothing
// in this package read either one when that was decided, and no test named
// them, so removing both from the envelope would have passed the whole suite in
// silence. These tests are the guard that decision needs.
//
// The keys are spelled as literal strings on purpose. A test written in terms of
// the Go field names moves with a rename and cannot catch one, and these four
// are a published surface under 12.4.
func TestTicketEnvelopePublishesProvenance(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))
	tk := showTicket(t, dir, id)

	for _, key := range []string{"createdAt", "updatedAt", "createdBy", "updatedBy"} {
		v, ok := tk[key]
		if !ok {
			t.Errorf("the ticket envelope has no %q, which 12.4 covers", key)
			continue
		}
		if v == nil {
			t.Errorf("%s is null on a freshly created ticket", key)
		}
	}

	for _, key := range []string{"createdBy", "updatedBy"} {
		actor, ok := tk[key].(map[string]any)
		if !ok {
			t.Fatalf("%s = %v, want an object with id and name", key, tk[key])
		}
		if actor["id"] != "human:sothr" {
			t.Errorf("%s.id = %v, want human:sothr", key, actor["id"])
		}
	}
}

// TestProvenancePairMovesTogether is the half that matters for 7.5. The driver
// resolves a conflict by taking the later `updated_at` and the actor belonging
// to it, so the two must always describe the same edit. It also pins the
// distinction from the created pair, which a mutation must not touch.
//
// The actors differ rather than the timestamps, because the test clock is the
// fixed reference instant and both stamps would otherwise be equal.
func TestProvenancePairMovesTogether(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))

	if got := runCLI(t, dir, nil, "update", id,
		"--priority", "urgent", "--actor", "agent:terva/mieli"); got.code != exitOK {
		t.Fatalf("update: %s", got.stderr)
	}

	tk := showTicket(t, dir, id)
	updatedBy := tk["updatedBy"].(map[string]any)
	createdBy := tk["createdBy"].(map[string]any)

	if updatedBy["id"] != "agent:terva/mieli" {
		t.Errorf("updatedBy.id = %v, want the actor that made the last change",
			updatedBy["id"])
	}
	// An actor is an agent session, which is the fact Git cannot record: it
	// carries the identity of whoever commits, and nothing here commits.
	if createdBy["id"] != "human:sothr" {
		t.Errorf("createdBy.id = %v, want the original author untouched by a mutation",
			createdBy["id"])
	}
}
