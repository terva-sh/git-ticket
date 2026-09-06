package ticket

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// sampleULID is a well-formed 26-character Crockford body, lifted from the
// store corpus so the tests here and the fixtures agree on what one looks like.
const sampleULID = "01K400HRM0V8QDX5N2WTFB3CJ7"

// TestValidSeriesGrammar is [A-Z][A-Z0-9]{1,7}, per plan 5.6.
func TestValidSeriesGrammar(t *testing.T) {
	for _, s := range []string{"TKT", "IDEA", "K8S", "V2", "S3", "AB", "ABCDEFGH"} {
		if !ValidSeries(s) {
			t.Errorf("ValidSeries(%q) = false, want true", s)
		}
	}
	for _, c := range []struct{ in, why string }{
		{"", "empty"},
		{"A", "one character carries no meaning"},
		{"ABCDEFGHI", "nine exceeds the ceiling"},
		{"2FA", "a leading digit reads as a ULID fragment"},
		{"tkt", "lowercase"},
		{"TK-T", "the separator cannot be inside a series"},
		{"TK_T", "underscore"},
		{"TK T", "space"},
		{"TKÄ", "not ASCII"},
	} {
		if ValidSeries(c.in) {
			t.Errorf("ValidSeries(%q) = true, want false: %s", c.in, c.why)
		}
	}
}

// TestSplitIDSplitsAtTheFirstHyphen covers the bare-fragment case, which 5.6
// resolves across every series and which therefore has to come back with an
// empty series rather than a guess.
func TestSplitIDSplitsAtTheFirstHyphen(t *testing.T) {
	for _, c := range []struct{ in, series, ulid string }{
		{"TKT-" + sampleULID, "TKT", sampleULID},
		{"IDEA-" + sampleULID, "IDEA", sampleULID},
		{sampleULID, "", sampleULID},
		{"01K4", "", "01K4"},
		{"TKT-01K4", "TKT", "01K4"},
		{"", "", ""},
		// The first hyphen, not the last: a malformed reference fails as a bad
		// series rather than quietly swallowing part of one.
		{"A-B-C", "A", "B-C"},
		{"-" + sampleULID, "", sampleULID},
	} {
		series, ulid := SplitID(c.in)
		if series != c.series || ulid != c.ulid {
			t.Errorf("SplitID(%q) = (%q, %q), want (%q, %q)", c.in, series, ulid, c.series, c.ulid)
		}
	}
}

// TestValidIDIsGrammarAlone is the split 5.6 rests on. ValidID cannot read a
// config, so a well-formed ID in a series nobody declared is valid here and
// unknown_series one layer up. Conflating them would make the narrower
// condition unreportable.
func TestValidIDIsGrammarAlone(t *testing.T) {
	if !ValidID("WHATEVER-" + sampleULID) {
		t.Error("an undeclared but well-formed series is a valid ID")
	}
	for _, bad := range []string{
		sampleULID,              // no series at all
		"TKT-" + sampleULID[1:], // 25 characters of body
		"TKT-" + sampleULID + "X",
		"2FA-" + sampleULID,
		"TKT-" + strings.Repeat("I", 26), // I is not Crockford
		"",
	} {
		if ValidID(bad) {
			t.Errorf("ValidID(%q) = true, want false", bad)
		}
	}
}

// TestNewIDMintsInTheGivenSeries covers the default, an explicit series, and
// the refusal. NewID reads no config, so the refusal here is the grammar one.
func TestNewIDMintsInTheGivenSeries(t *testing.T) {
	entropy := func() *bytes.Reader { return bytes.NewReader(make([]byte, 10)) }

	id, err := NewID("IDEA", referenceInstant, entropy())
	if err != nil {
		t.Fatal(err)
	}
	if series, ulid := SplitID(id); series != "IDEA" || len(ulid) != ulidLen {
		t.Fatalf("NewID(IDEA) = %q, want an IDEA series and a %d-character body", id, ulidLen)
	}
	if !ValidID(id) {
		t.Errorf("ValidID(%q) = false", id)
	}

	defaulted, err := NewID("", referenceInstant, entropy())
	if err != nil {
		t.Fatal(err)
	}
	if series, _ := SplitID(defaulted); series != DefaultSeries {
		t.Errorf("an empty series minted %q, want the %s default", defaulted, DefaultSeries)
	}

	_, err = NewID("2FA", referenceInstant, entropy())
	var e *Error
	if !asTicketError(err, &e) || e.Code != CodeInvalidField {
		t.Fatalf("NewID with a malformed series: err = %v, want %s", err, CodeInvalidField)
	}
}

// TestFilenameFallbackRecognizesAnySeries is the deviation from 5.6's first
// wording, recorded in the plan beside it. Checking the store's declaration
// here would make an unparseable file in an undeclared series yield no ID,
// which is the store failing to see a file that is sitting in it.
func TestFilenameFallbackRecognizesAnySeries(t *testing.T) {
	for _, c := range []struct {
		base string
		want string
	}{
		{"TKT-" + sampleULID, "TKT-" + sampleULID},
		{"IDEA-" + sampleULID, "IDEA-" + sampleULID},
		{"NEVERDECLARED-" + sampleULID, ""}, // 13 characters is past the ceiling
		{"notes", ""},
		{"TKT-nope", ""},
	} {
		f := file{Path: "/store/tickets/" + c.base + ".md"}
		if got := f.id(); got != c.want {
			t.Errorf("id() for %s.md = %q, want %q", c.base, got, c.want)
		}
	}
}

// TestCreateRefusesAnUndeclaredSeries is the enforcement half of 5.6, and the
// one place this vocabulary gates a write rather than warning after it. A
// mistyped series is inside an immutable ID, so the only repair is remove and a
// second create.
func TestCreateRefusesAnUndeclaredSeries(t *testing.T) {
	s := newTestStore(t)
	_, err := s.Create(context.Background(), CreateOptions{
		Title:  "Filed into a series nobody declared",
		Series: "IDEA",
		Actor:  Actor{ID: "human:sothr"},
	})
	var e *Error
	if !asTicketError(err, &e) || e.Code != CodeUnknownSeries {
		t.Fatalf("err = %v, want %s", err, CodeUnknownSeries)
	}
	// The message names what the store does declare, so the repair is visible
	// without a second command.
	if !strings.Contains(e.Message, DefaultSeries) {
		t.Errorf("message %q does not name the declared series", e.Message)
	}
}

// TestCreateFilesIntoADeclaredSeries is the same path once the store has said
// yes, and it checks the ID rather than the absence of an error.
func TestCreateFilesIntoADeclaredSeries(t *testing.T) {
	s := newTestStore(t)
	s.config.Series = []string{"TKT", "IDEA"}

	res, err := s.Create(context.Background(), CreateOptions{
		Title:  "An idea, filed as one",
		Series: "IDEA",
		Actor:  Actor{ID: "human:sothr"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if series, _ := SplitID(res.Ticket.ID); series != "IDEA" {
		t.Fatalf("created %s, want an IDEA series", res.Ticket.ID)
	}

	// Naming no series still lands in the default, in a store that declares
	// more than one. Adopting a second series must not move the first.
	defaulted, err := s.Create(context.Background(), CreateOptions{
		Title: "An ordinary ticket in the same store",
		Actor: Actor{ID: "human:sothr"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if series, _ := SplitID(defaulted.Ticket.ID); series != DefaultSeries {
		t.Fatalf("created %s, want the %s default", defaulted.Ticket.ID, DefaultSeries)
	}
}

// seriesIDs is a small store's worth of IDs across two series. tktA and ideaA
// differ only in their last character, which is the case that separates
// abbreviating within a series from abbreviating across the store.
var (
	tktA      = "TKT-01K400HRM0V8QDX5N2WTFB3CJ7"
	tktB      = "TKT-01K400HRN4Y2ZBP7S6DGQ8MXV3"
	ideaA     = "IDEA-01K400HRM0V8QDX5N2WTFB3CJ8"
	seriesIDs = []string{tktA, tktB, ideaA}
)

// TestResolveRefTreatsThePrefixAsIdentity is the decision 5.6 turns on. The
// alternative was a prefix NormalizeRef strips, which would make IDEA-x and
// TKT-x one reference and a wrong prefix a happy answer.
func TestResolveRefTreatsThePrefixAsIdentity(t *testing.T) {
	// tktA and ideaA share their first 25 ULID characters, so a prefixed
	// fragment this long is unambiguous only because the series decides it.
	got, err := ResolveRef("IDEA-01K400HRM", seriesIDs)
	if err != nil || got != ideaA {
		t.Fatalf("IDEA-01K400HRM resolved to (%q, %v), want %s", got, err, ideaA)
	}
	got, err = ResolveRef("TKT-01K400HRM", seriesIDs)
	if err != nil || got != tktA {
		t.Fatalf("TKT-01K400HRM resolved to (%q, %v), want %s", got, err, tktA)
	}

	// The whole point: a prefixed reference matches its own series or nothing.
	// Stripping the prefix would answer tktA here, with exit 0 and nothing in
	// the output saying it guessed.
	_, err = ResolveRef("IDEA-01K400HRN4", seriesIDs)
	var e *Error
	if !asTicketError(err, &e) || e.Code != CodeTicketNotFound {
		t.Fatalf("a fragment of a TKT ULID under IDEA: err = %v, want %s", err, CodeTicketNotFound)
	}
}

// TestResolveRefMatchesABareFragmentAcrossSeries is the compatibility half.
// ULIDs cannot collide, so the commands a person already types keep working in
// a store that adopts a series.
func TestResolveRefMatchesABareFragmentAcrossSeries(t *testing.T) {
	got, err := ResolveRef("01K400HRN4", seriesIDs)
	if err != nil || got != tktB {
		t.Fatalf("a bare fragment resolved to (%q, %v), want %s", got, err, tktB)
	}

	// A whole bare ULID is only the longest such fragment, and resolves the
	// same way. It is not an exact-ID match, because it carries no series.
	if _, body := SplitID(ideaA); true {
		got, err = ResolveRef(body, seriesIDs)
		if err != nil || got != ideaA {
			t.Fatalf("a bare whole ULID resolved to (%q, %v), want %s", got, err, ideaA)
		}
	}

	// A bare fragment two tickets share is still ambiguous_id, and still lists
	// its candidates, even when the two are in different series.
	_, err = ResolveRef("01K400HRM", seriesIDs)
	var e *Error
	if !asTicketError(err, &e) || e.Code != CodeAmbiguousID {
		t.Fatalf("err = %v, want %s", err, CodeAmbiguousID)
	}
	if c := e.Details["candidates"]; !strings.Contains(c, tktA) || !strings.Contains(c, ideaA) {
		t.Errorf("candidates = %q, want both series named", c)
	}
}

// TestResolveRefAppliesTheFloorToTheULIDHalf is 5.6: a series is typed whole or
// not at all, so ID-01M1 is not a shortening of IDEA-, it is the series ID.
func TestResolveRefAppliesTheFloorToTheULIDHalf(t *testing.T) {
	// Four ULID characters is the floor, and the series does not count toward
	// it. Without the split this would pass on length alone.
	if _, err := ResolveRef("IDEA-01K4", seriesIDs); err != nil {
		t.Errorf("IDEA-01K4 is four ULID characters and should resolve: %v", err)
	}
	for _, short := range []string{"IDEA-01K", "TKT-0", "01K", "IDEA-"} {
		_, err := ResolveRef(short, seriesIDs)
		var e *Error
		if !asTicketError(err, &e) || e.Code != CodeTicketNotFound {
			t.Errorf("ResolveRef(%q): err = %v, want %s", short, err, CodeTicketNotFound)
		}
	}
}

// TestResolveRefIgnoresCaseOnBothHalves, per 5.5 and 5.6.
func TestResolveRefIgnoresCaseOnBothHalves(t *testing.T) {
	for _, ref := range []string{"idea-01k400hrm", "IDEA-01k400HRM", "Idea-01K400hrm", " idea-01K400HRM "} {
		got, err := ResolveRef(ref, seriesIDs)
		if err != nil || got != ideaA {
			t.Errorf("ResolveRef(%q) = (%q, %v), want %s", ref, got, err, ideaA)
		}
	}
}

// TestStoreResolveRefRefusesAnUndeclaredSeries is the one resolution rule that
// needs a store. ticket_not_found would send the reader to look for a ticket;
// unknown_series sends them to the config, which is where the repair is.
func TestStoreResolveRefRefusesAnUndeclaredSeries(t *testing.T) {
	s := newTestStore(t)

	_, err := s.resolveRef("IDEA-01K400HRM", seriesIDs)
	var e *Error
	if !asTicketError(err, &e) || e.Code != CodeUnknownSeries {
		t.Fatalf("err = %v, want %s", err, CodeUnknownSeries)
	}

	// Declared, and the same reference resolves. The refusal is about the
	// config and not about which tickets happen to exist.
	s.config.Series = []string{"TKT", "IDEA"}
	if got, err := s.resolveRef("IDEA-01K400HRM", seriesIDs); err != nil || got != ideaA {
		t.Fatalf("after declaring IDEA: (%q, %v), want %s", got, err, ideaA)
	}

	// A bare fragment names no series, so it is never unknown_series.
	s.config.Series = nil
	if got, err := s.resolveRef("01K400HRN4", seriesIDs); err != nil || got != tktB {
		t.Fatalf("a bare fragment: (%q, %v), want %s", got, err, tktB)
	}
}

// TestShortestUniqueAbbreviatesWithinASeries is 5.6's abbreviation rule, and
// the contrast with the store-wide form is the whole argument for it. tktA and
// ideaA share 25 ULID characters, so abbreviating them against each other costs
// both of them their whole ULID for a collision resolution never sees.
func TestShortestUniqueAbbreviatesWithinASeries(t *testing.T) {
	within := ShortestUnique(seriesIDs)
	// tktA competes with tktB alone. They share eight ULID characters, so nine
	// separate them.
	if got := within[tktA]; got != "TKT-01K400HRM" {
		t.Errorf("within-series abbreviation of tktA = %q, want TKT-01K400HRM", got)
	}
	// ideaA competes with nothing, so it gets the eight-character floor even
	// though a TKT ticket shares 25 of its characters.
	if got := within[ideaA]; got != "IDEA-01K400HR" {
		t.Errorf("within-series abbreviation of ideaA = %q, want IDEA-01K400HR", got)
	}

	// Across the store, tktA and ideaA compete, and they differ only in the
	// last character. Both pay their whole ULID for a collision that within a
	// series does not exist, which is the cost 5.6 declines by default.
	across := ShortestUniqueAcrossSeries(seriesIDs)
	if across[tktA] != tktA || across[ideaA] != ideaA {
		t.Errorf("across-series abbreviations = %q and %q, want the full IDs",
			across[tktA], across[ideaA])
	}
	// tktB collides with neither past the timestamp, so the store-wide form
	// leaves it where the within-series form did. It lengthens what collides,
	// not everything.
	if across[tktB] != within[tktB] {
		t.Errorf("across-series abbreviation of tktB = %q, want %q as within",
			across[tktB], within[tktB])
	}
}

// TestAbbreviationsResolveBack is the property that matters more than any exact
// width: what a listing prints, a person pastes straight back into a command.
// Abbreviating and resolving are inverses, which is why they live in one file.
func TestAbbreviationsResolveBack(t *testing.T) {
	for _, mode := range []struct {
		name string
		fn   func([]string) map[string]string
	}{
		{"within a series", ShortestUnique},
		{"across the store", ShortestUniqueAcrossSeries},
	} {
		t.Run(mode.name, func(t *testing.T) {
			for id, short := range mode.fn(seriesIDs) {
				got, err := ResolveRef(short, seriesIDs)
				if err != nil {
					t.Errorf("%s printed as %s, which does not resolve: %v", id, short, err)
					continue
				}
				if got != id {
					t.Errorf("%s printed as %s, which resolves to %s", id, short, got)
				}
			}
		})
	}

	// The store-wide mode promises more than resolving: its ULID half resolves
	// on its own, which is what somebody pasting into a document outside this
	// store wants.
	for id, short := range ShortestUniqueAcrossSeries(seriesIDs) {
		_, body := SplitID(short)
		got, err := ResolveRef(body, seriesIDs)
		if err != nil || got != id {
			t.Errorf("the ULID half of %s is %s, which resolves to (%q, %v)", id, body, got, err)
		}
	}
}

// TestEffectiveSeriesReadsAnEmptyListAsTKT is the inversion at the heart of
// 5.6. Every other allowlist in config.yml reads an empty list as "no opinion,
// permit everything". This one reads it as exactly [TKT], which is what makes
// every store written before series existed already correct.
func TestEffectiveSeriesReadsAnEmptyListAsTKT(t *testing.T) {
	for _, c := range []struct {
		name string
		cfg  Config
		want []string
	}{
		{"nil", Config{}, []string{"TKT"}},
		{"empty", Config{Series: []string{}}, []string{"TKT"}},
		{"declared", Config{Series: []string{"TKT", "IDEA"}}, []string{"TKT", "IDEA"}},
		// A store may drop TKT once nothing uses it. Nothing re-adds it, or
		// the removal it just performed would not stick.
		{"without TKT", Config{Series: []string{"IDEA"}}, []string{"IDEA"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := c.cfg.EffectiveSeries()
			if strings.Join(got, ",") != strings.Join(c.want, ",") {
				t.Fatalf("EffectiveSeries() = %v, want %v", got, c.want)
			}
			for _, s := range c.want {
				if !c.cfg.KnownSeries(s) {
					t.Errorf("KnownSeries(%q) = false, want true", s)
				}
			}
			if c.cfg.KnownSeries("NOPE") {
				t.Error(`KnownSeries("NOPE") = true, want false`)
			}
		})
	}
}

// TestEffectiveSeriesDoesNotAliasTheConfig guards the default case, where
// returning the caller's own slice would let a caller appending to the result
// edit the store's configuration in place.
func TestEffectiveSeriesDoesNotAliasTheConfig(t *testing.T) {
	cfg := Config{Series: []string{"TKT"}}
	got := cfg.EffectiveSeries()
	got[0] = "MANGLED"
	if cfg.Series[0] != "TKT" {
		t.Fatalf("Series[0] = %q after editing the result, want TKT", cfg.Series[0])
	}
}

// TestRenderConfigLeavesASchema1StoreAlone is the compatibility half of 5.6.
// A schema-2 binary rewriting a schema-1 store's config.yml, which `update` and
// every other write does not do but `check --fix` can, must not add a key the
// store's declared level does not have.
//
// The value is still rendered when the store declared one, because config.yml
// has no unknown-field preservation. Declining there would delete it.
func TestRenderConfigLeavesASchema1StoreAlone(t *testing.T) {
	bare := string(RenderConfig(Config{Schema: 1}))
	if strings.Contains(bare, "series:") {
		t.Errorf("schema 1 with no series rendered the key:\n%s", bare)
	}

	declared := string(RenderConfig(Config{Schema: 1, Series: []string{"IDEA"}}))
	if !strings.Contains(declared, "series:") {
		t.Errorf("schema 1 with a declared series dropped it:\n%s", declared)
	}

	at2 := string(RenderConfig(Config{Schema: 2}))
	if !strings.Contains(at2, "series: []") {
		t.Errorf("schema 2 did not render an empty series list:\n%s", at2)
	}
}

// TestConfigRoundTripsSeries holds the pair to each other, the way
// TestConfigRoundTrip does for the rest of the file.
func TestConfigRoundTripsSeries(t *testing.T) {
	// From DefaultConfig rather than a bare literal: ParseConfig fills a zero
	// lock timeout with DefaultLockTimeout, so a hand-built Config fails this
	// round trip on a field that has nothing to do with series.
	base := DefaultConfig()
	base.Schema = 2
	base.Series = []string{"TKT", "IDEA"}
	src := RenderConfig(base)
	cfg, err := ParseConfig(src)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(cfg.Series, ","); got != "TKT,IDEA" {
		t.Fatalf("Series = %q, want TKT,IDEA", got)
	}
	if again := string(RenderConfig(cfg)); again != string(src) {
		t.Errorf("render(parse(render(c))) differs:\n%s", diffLines(string(src), again))
	}
}
