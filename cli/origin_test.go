package cli

import (
	"strings"
	"testing"
)

// originOf reads the origin field back through the JSON contract. It returns
// the empty string for null, which is what a ticket that came from nowhere
// carries.
func originOf(t *testing.T, dir, id string) string {
	t.Helper()
	raw, ok := showTicket(t, dir, id)["origin"]
	if !ok {
		t.Fatalf("%s carries no origin key at all", id)
	}
	if raw == nil {
		return ""
	}
	s, _ := raw.(string)
	return s
}

// TestCreateFromSeedsTheStatementAndNotTheRouting is 5.6's list, exactly. Title,
// description, and acceptance criteria are the statement of the work, while
// priority, labels, and milestone are routing that the receiving series decides
// for itself.
func TestCreateFromSeedsTheStatementAndNotTheRouting(t *testing.T) {
	dir := newStore(t)
	source := makeTicket(t, dir, "Work out how ID series should subdivide a store",
		"--description", "Several components share one repository.",
		"--ac", "The prefix travels with the reference",
		"--ac", "A store that declares nothing keeps working",
		"--priority", "high",
		"--label", "format",
		"--milestone", "v0.12.0")

	got := runCLI(t, dir, nil, "--json", "create", "--from", source, "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create --from: %s%s", got.stdout, got.stderr)
	}
	id := ticketID(t, decode(t, got.stdout))

	child := showTicket(t, dir, id)
	if child["title"] != "Work out how ID series should subdivide a store" {
		t.Errorf("title = %v, want the source's", child["title"])
	}
	if got := sectionOf(t, dir, id, "description"); !strings.Contains(got, "Several components") {
		t.Errorf("description = %q, want the source's", got)
	}
	ac := sectionOf(t, dir, id, "acceptanceCriteria")
	for _, want := range []string{"The prefix travels", "declares nothing"} {
		if !strings.Contains(ac, want) {
			t.Errorf("acceptance criteria %q does not carry %q", ac, want)
		}
	}
	// Unticked, whatever the source had done about them. Carrying a tick
	// across would claim evidence for work this ticket has not begun.
	if strings.Contains(ac, "[x]") {
		t.Errorf("a seeded criterion arrived ticked: %q", ac)
	}

	// The routing is the store's defaults and not the source's.
	if child["priority"] == "high" {
		t.Error("priority was seeded from the source, and 5.6 says it should not be")
	}
	if labels, _ := child["labels"].([]any); len(labels) != 0 {
		t.Errorf("labels = %v, want none seeded", labels)
	}
	if child["milestone"] != nil {
		t.Errorf("milestone = %v, want none seeded", child["milestone"])
	}

	if originOf(t, dir, id) != source {
		t.Errorf("origin = %q, want %s", originOf(t, dir, id), source)
	}
}

// TestCreateFromNeverTouchesTheSource. An idea that fans out to three
// implementation tickets must not close on the first, and nothing here closes
// it, because the create that is the last one is not distinguishable at create
// time from the create that is not.
func TestCreateFromNeverTouchesTheSource(t *testing.T) {
	dir := newStore(t)
	source := makeTicket(t, dir, "An idea worth splitting three ways")
	before := showTicket(t, dir, source)

	for _, title := range []string{"First slice", "Second slice", "Third slice"} {
		got := runCLI(t, dir, nil, "--json", "create",
			"--from", source, "--title", title, "--actor", "human:sothr")
		if got.code != exitOK {
			t.Fatalf("create --from %q: %s%s", title, got.stdout, got.stderr)
		}
	}

	after := showTicket(t, dir, source)
	if after["status"] != before["status"] {
		t.Errorf("the source moved to %v from %v", after["status"], before["status"])
	}
	if after["revision"] != before["revision"] {
		t.Errorf("the source was rewritten: revision %v became %v", before["revision"], after["revision"])
	}

	// All three record it, and the source enumerates none of them. The reverse
	// direction is derived, per 5.6, because a source edited by every
	// descendant is the one file every agent conflicts on.
	list := runCLI(t, dir, nil, "--json", "list", "--origin", source)
	if list.code != exitOK {
		t.Fatalf("list --origin: %s", list.stderr)
	}
	tickets, _ := decode(t, list.stdout)["tickets"].([]any)
	if len(tickets) != 3 {
		t.Errorf("list --origin found %d tickets, want 3", len(tickets))
	}
}

// TestCreateFromRefusesATemplateToo. Two seed sources overlapping on the same
// fields need a precedence rule nobody would remember, so 5.6 refuses the pair.
func TestCreateFromRefusesATemplateToo(t *testing.T) {
	dir := newStore(t)
	source := makeTicket(t, dir, "A source to seed from")

	got := runCLI(t, dir, nil, "--json", "create",
		"--from", source, "--template", "adr", "--title", "Both at once", "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatalf("--from with --template should be refused, got exit %d", got.code)
	}
	if code := errCode(t, got); code != codeUsage {
		t.Errorf("code = %v, want %s", code, codeUsage)
	}
}

// TestCreateFromLetsAnExplicitFlagWin, which is the precedence --template
// already uses: the seed fills what the caller did not name.
func TestCreateFromLetsAnExplicitFlagWin(t *testing.T) {
	dir := newStore(t)
	source := makeTicket(t, dir, "The source title",
		"--description", "The source description",
		"--ac", "The source criterion")

	got := runCLI(t, dir, nil, "--json", "create",
		"--from", source,
		"--title", "A title of my own",
		"--description", "A description of my own",
		"--ac", "A criterion of my own",
		"--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create --from: %s%s", got.stdout, got.stderr)
	}
	id := ticketID(t, decode(t, got.stdout))

	if title := showTicket(t, dir, id)["title"]; title != "A title of my own" {
		t.Errorf("title = %v, want the explicit one", title)
	}
	if d := sectionOf(t, dir, id, "description"); !strings.Contains(d, "of my own") {
		t.Errorf("description = %q, want the explicit one", d)
	}
	if ac := sectionOf(t, dir, id, "acceptanceCriteria"); strings.Contains(ac, "The source criterion") {
		t.Errorf("acceptance criteria took the seed over the explicit flag: %q", ac)
	}
	// The link is recorded whichever fields the seed actually supplied.
	if originOf(t, dir, id) != source {
		t.Errorf("origin = %q, want %s", originOf(t, dir, id), source)
	}
}

// TestCreateFromCrossesSeriesAndDoesNotHaveTo. 5.6 does not restrict --from to
// a cross-series create: splitting a ticket within one series is the same
// operation and earns the same record.
func TestCreateFromCrossesSeriesAndDoesNotHaveTo(t *testing.T) {
	dir := newStore(t)
	seriesEnvelopeOf(t, dir, "add", "IDEA")
	idea := makeTicket(t, dir, "An idea in its own series", "--series", "IDEA")

	across := runCLI(t, dir, nil, "--json", "create",
		"--from", idea, "--title", "The implementation of it", "--actor", "human:sothr")
	if across.code != exitOK {
		t.Fatalf("cross-series create --from: %s%s", across.stdout, across.stderr)
	}
	id := ticketID(t, decode(t, across.stdout))
	if !strings.HasPrefix(id, "TKT-") {
		t.Errorf("created %s, want the default series", id)
	}
	if originOf(t, dir, id) != idea {
		t.Errorf("origin = %q, want %s", originOf(t, dir, id), idea)
	}

	within := runCLI(t, dir, nil, "--json", "create",
		"--from", id, "--title", "A slice of the implementation", "--actor", "human:sothr")
	if within.code != exitOK {
		t.Fatalf("same-series create --from: %s%s", within.stdout, within.stderr)
	}
	if got := originOf(t, dir, ticketID(t, decode(t, within.stdout))); got != id {
		t.Errorf("origin = %q, want %s", got, id)
	}
}

// TestUpdateOriginSetsAndClears, with the empty value clearing it the way
// --parent and --milestone do.
func TestUpdateOriginSetsAndClears(t *testing.T) {
	dir := newStore(t)
	source := makeTicket(t, dir, "Where the work came from")
	id := makeTicket(t, dir, "The work itself")

	if got := runCLI(t, dir, nil, "update", id, "--origin", source, "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("update --origin: %s%s", got.stdout, got.stderr)
	}
	if originOf(t, dir, id) != source {
		t.Errorf("origin = %q, want %s", originOf(t, dir, id), source)
	}

	if got := runCLI(t, dir, nil, "update", id, "--origin", "", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("update --origin '': %s%s", got.stdout, got.stderr)
	}
	if got := originOf(t, dir, id); got != "" {
		t.Errorf("origin = %q after clearing, want null", got)
	}
}

// TestUpdateOriginRefusesTheTicketItself is the one case anybody reaches by
// accident. It is invalid_field and not a cycle code, because 5.6 declines an
// origin_cycle outright: nothing walks origin transitively, so a cycle in it
// gates nothing.
func TestUpdateOriginRefusesTheTicketItself(t *testing.T) {
	dir := newStore(t)
	id := makeTicket(t, dir, "A ticket that came from itself")

	got := runCLI(t, dir, nil, "--json", "update", id, "--origin", id, "--actor", "human:sothr")
	if got.code != exitError {
		t.Fatalf("a self-origin should be refused, got exit %d", got.code)
	}
	if code := errCode(t, got); code != "invalid_field" {
		t.Errorf("code = %v, want invalid_field", code)
	}
}

// TestListSeriesFiltersByPrefix. A series is not a field, so this filters on
// part of the ID, and a typo fails as usage rather than matching nothing and
// reading as an empty store.
func TestListSeriesFiltersByPrefix(t *testing.T) {
	dir := newStore(t)
	seriesEnvelopeOf(t, dir, "add", "IDEA")
	makeTicket(t, dir, "An ordinary ticket")
	makeTicket(t, dir, "An idea", "--series", "IDEA")
	makeTicket(t, dir, "Another idea", "--series", "IDEA")

	for _, c := range []struct {
		series string
		want   int
	}{{"IDEA", 2}, {"TKT", 1}} {
		got := runCLI(t, dir, nil, "--json", "list", "--series", c.series)
		if got.code != exitOK {
			t.Fatalf("list --series %s: %s", c.series, got.stderr)
		}
		tickets, _ := decode(t, got.stdout)["tickets"].([]any)
		if len(tickets) != c.want {
			t.Errorf("list --series %s found %d, want %d", c.series, len(tickets), c.want)
		}
	}

	// Lowercase is accepted, like every other place a series is typed.
	lower := runCLI(t, dir, nil, "--json", "list", "--series", "idea")
	if tickets, _ := decode(t, lower.stdout)["tickets"].([]any); len(tickets) != 2 {
		t.Errorf("list --series idea found %d, want 2", len(tickets))
	}

	bad := runCLI(t, dir, nil, "--json", "list", "--series", "2FA")
	if bad.code != exitError {
		t.Fatalf("a malformed series should be refused, got exit %d", bad.code)
	}
	if code := errCode(t, bad); code != codeUsage {
		t.Errorf("code = %v, want %s", code, codeUsage)
	}
}

// TestListOriginRefusesAnUnresolvableID, for the reason --parent does: an
// unresolvable one returning an empty list reads exactly like a ticket that
// nothing came out of.
func TestListOriginRefusesAnUnresolvableID(t *testing.T) {
	dir := newStore(t)
	makeTicket(t, dir, "Something to not match")

	got := runCLI(t, dir, nil, "--json", "list", "--origin", "TKT-01ZZZZZZZZZZZZZZZZZZZZZZZZ")
	if got.code != exitError {
		t.Fatalf("an unresolvable --origin should be refused, got exit %d", got.code)
	}
	if code := errCode(t, got); code != "ticket_not_found" {
		t.Errorf("code = %v, want ticket_not_found", code)
	}
}
