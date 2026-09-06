package ticket

import (
	"crypto/rand"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

// DefaultSeries is the series every store has, per plan 5.6. A store that
// declares nothing has this one, so every store written before series existed
// is already correct.
const DefaultSeries = "TKT"

// IDPrefix is the default series with its separator. It is what NewID mints
// when no series is named, and it is no longer the fixed part of every ID:
// since 5.6 a store may declare others.
const IDPrefix = DefaultSeries + "-"

// ulidLen is the length of the Crockford base32 ULID that follows the prefix.
const ulidLen = 26

// minPrefixLen is how much of a ULID a caller must type for a prefix to be
// considered, per plan 5.5. Four characters is enough that a typo does not
// resolve to a real ticket by accident.
const minPrefixLen = 4

// abbrevLen is the fewest characters of a ULID a listing ever shows. Four is
// the minimum a prefix may be, per 5.5, and eight still looks like an ID.
const abbrevLen = 8

// crockford is Crockford base32: the digits and the uppercase letters, less I,
// L, O, and U, which are the ones a person misreads.
const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// SeriesMinLen and SeriesMaxLen are the bounds of plan 5.6. Two is the floor
// because one letter carries no meaning and spends a whole namespace on a
// character nobody can guess the expansion of. Eight is the ceiling because the
// prefix is quoted beside 26 characters of ULID everywhere it appears.
//
// They are exported because `schema` publishes them, per 10.4, so a consumer
// learns what is legal before it opens a store.
const (
	SeriesMinLen = 2
	SeriesMaxLen = 8
)

// SeriesPattern is the same grammar as a regular expression, for a consumer
// that would otherwise reimplement ValidSeries and get an edge wrong.
//
// ValidSeries does not use it. A hand-written check is cheaper on a path every
// create takes, and compiling a package-level regexp to validate three
// characters buys nothing. That makes this a second statement of one rule, so
// TestSeriesPatternAgreesWithValidSeries holds the two together and fails if
// either moves without the other.
const SeriesPattern = "^[A-Z][A-Z0-9]{1,7}$"

// ValidSeries reports whether s is a well-formed series prefix, per plan 5.6:
// [A-Z][A-Z0-9]{1,7}.
//
// This is grammar alone and says nothing about whether a store declares the
// series. The two are different failures with different repairs, so they have
// different codes: a malformed prefix is invalid_field and a well-formed one
// the store has not declared is unknown_series.
//
// A leading digit is refused because a ULID's body opens with ten characters of
// timestamp that are mostly digits, so 2FA-01M1 and a ULID fragment that lost
// its prefix look alike at the speed anybody reads an ID.
func ValidSeries(s string) bool {
	if len(s) < SeriesMinLen || len(s) > SeriesMaxLen {
		return false
	}
	if s[0] < 'A' || s[0] > 'Z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		if (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// SplitID divides an ID at its separator, per plan 5.6. A reference carrying no
// separator is all ULID and returns an empty series, which is the bare fragment
// 5.6 resolves across every series.
//
// It splits at the first hyphen rather than the last. Crockford base32 has no
// hyphen in it, so a well-formed ID has exactly one, and splitting at the first
// makes a malformed reference fail as a bad series rather than silently
// swallowing part of one.
func SplitID(id string) (series, ulid string) {
	if i := strings.IndexByte(id, '-'); i >= 0 {
		return id[:i], id[i+1:]
	}
	return "", id
}

// NewID returns a ticket ID for the given instant: the series, a hyphen, and a
// 26-character ULID. ULIDs need no central counter, so two disconnected agents
// cannot collide, and they sort by creation time.
//
// An empty series means DefaultSeries, so a caller that has no opinion gets the
// series every store has. Whether the store declares the series is the caller's
// question, because this function reads no config.
func NewID(series string, at time.Time, entropy io.Reader) (string, error) {
	if series == "" {
		series = DefaultSeries
	}
	if !ValidSeries(series) {
		return "", codedError(CodeInvalidField,
			"%q is not a series: two to eight characters, uppercase letters and digits, letter first", series)
	}
	if entropy == nil {
		entropy = rand.Reader
	}
	var raw [16]byte
	ms := uint64(at.UTC().UnixMilli())
	raw[0] = byte(ms >> 40)
	raw[1] = byte(ms >> 32)
	raw[2] = byte(ms >> 24)
	raw[3] = byte(ms >> 16)
	raw[4] = byte(ms >> 8)
	raw[5] = byte(ms)
	if _, err := io.ReadFull(entropy, raw[6:]); err != nil {
		return "", fmt.Errorf("ticket: reading entropy for an ID: %w", err)
	}
	return series + "-" + encodeULID(raw), nil
}

// encodeULID writes 128 bits as 26 base32 characters. The 26 characters hold
// 130 bits, so the value is padded with two leading zero bits and the first
// character is never above 7.
func encodeULID(raw [16]byte) string {
	out := make([]byte, ulidLen)
	for i := range out {
		var v byte
		for k := 0; k < 5; k++ {
			pos := i*5 + k - 2 // bit index into the 128-bit value
			var bit byte
			if pos >= 0 {
				bit = (raw[pos/8] >> (7 - uint(pos%8))) & 1
			}
			v = v<<1 | bit
		}
		out[i] = crockford[v]
	}
	return string(out)
}

// ValidID reports whether s is a well-formed ticket ID: a legal series, a
// hyphen, and a 26-character ULID.
//
// It is grammar alone, per 5.6. An ID in a series this store does not declare
// is well-formed and is unknown_series, which is the narrower condition and a
// different repair.
func ValidID(s string) bool {
	series, ulid := SplitID(s)
	return ValidSeries(series) && validULID(ulid)
}

func validULID(s string) bool {
	if len(s) != ulidLen {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune(crockford, r) {
			return false
		}
	}
	return true
}

// NormalizeRef puts a user-typed reference into the form IDs are stored in:
// trimmed and uppercased. Matching is case-insensitive on both halves, per 5.5
// and 5.6, because a person types an ID in whatever case their terminal gave
// them.
//
// It no longer strips a prefix. Under 5.6 the series is part of the ID rather
// than decoration around it, so stripping would make IDEA-x and TKT-x one
// reference and a wrong prefix a happy answer. SplitID is what separates the
// halves, and ResolveRef compares them one at a time.
//
// Crockford's letter substitutions are deliberately not applied. Reading I as 1
// would let two different typos resolve to the same ticket, and git does not do
// it for object hashes either.
func NormalizeRef(ref string) string {
	return strings.ToUpper(strings.TrimSpace(ref))
}

// ResolveRef matches a reference against a set of known IDs, per plan 5.5 and
// 5.6. It accepts a full ID or a unique prefix of at least four characters, and
// returns ambiguous_id listing the candidates when more than one ticket
// matches.
//
// Series changes three of 5.5's rules. A bare ULID fragment resolves across
// every series, because ULIDs cannot collide and the commands a person already
// types have to keep working in a store that adopts one. A prefixed fragment
// matches on both halves, the series exactly and the ULID as a prefix, because
// under 5.6 the prefix is identity: IDEA-01M1SH never resolves to a TKT ticket.
// And the four-character floor applies to the ULID half alone, since a series
// is typed whole or not at all.
//
// It does not check whether the store declares the series, because it reads no
// config. That is unknown_series and it belongs to Store.resolveRef.
func ResolveRef(ref string, ids []string) (string, error) {
	norm := NormalizeRef(ref)
	if norm == "" {
		return "", codedError(CodeTicketNotFound, "no ticket reference given")
	}
	series, body := SplitID(norm)

	// A whole ID, prefix and all. A bare 26-character ULID is not this case: it
	// carries no series, so it falls through to the scan below and matches
	// across every one of them.
	if series != "" && validULID(body) {
		for _, id := range ids {
			if id == norm {
				return id, nil
			}
		}
		return "", &Error{
			Code: CodeTicketNotFound,
			// The ID is not repeated here. Error() prefixes it from Ticket, and
			// naming it in both places printed it twice in one sentence.
			Message: "no such ticket in this store",
			Ticket:  norm,
		}
	}
	if len(body) < minPrefixLen {
		return "", codedError(CodeTicketNotFound,
			"%q is shorter than the %d characters a prefix needs", ref, minPrefixLen)
	}

	var matches []string
	for _, id := range ids {
		idSeries, idBody := SplitID(id)
		// An empty series in the reference means the caller named none, which
		// matches every series. A named one matches only itself.
		if series != "" && idSeries != series {
			continue
		}
		if strings.HasPrefix(idBody, body) {
			matches = append(matches, id)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return "", codedError(CodeTicketNotFound, "no ticket matches %q", ref)
	default:
		return "", &Error{
			Code:    CodeAmbiguousID,
			Message: fmt.Sprintf("%q matches %d tickets", ref, len(matches)),
			Details: map[string]string{"candidates": strings.Join(matches, " ")},
		}
	}
}

// resolveRef is ResolveRef plus the one rule that needs a store: a reference
// naming a series this store does not declare returns unknown_series and not
// ticket_not_found, per plan 5.6. The two send a reader to different places,
// one to look for a ticket and the other to look at the config.
//
// It is checked before the scan rather than after it, so an undeclared series
// answers the same way whether or not some ticket happens to carry it.
//
// Not every resolution goes through here. `unlink --depends-on` resolves
// against the ticket's own dependency list, and a dependency naming an
// undeclared series is exactly the dangling edge unlink exists to repair, so
// refusing it there would make it unrepairable.
func (s *Store) resolveRef(ref string, ids []string) (string, error) {
	if series, _ := SplitID(NormalizeRef(ref)); series != "" {
		if cfg := s.Config(); !cfg.KnownSeries(series) {
			return "", &Error{
				Code: CodeUnknownSeries,
				Message: fmt.Sprintf("this store does not declare the series %q; it declares %s",
					series, strings.Join(cfg.EffectiveSeries(), ", ")),
				Field: "series",
			}
		}
	}
	return ResolveRef(ref, ids)
}

// ShortestUnique maps each ID to the fewest characters that still resolve to
// it, never fewer than abbrevLen. It is the inverse of ResolveRef and lives
// beside it because the two have to agree: what a listing prints, a person
// pastes straight back into a command.
//
// A fixed width cannot do this. A ULID opens with ten characters of timestamp,
// so two tickets created in the same millisecond are identical that far in, and
// a listing printing eight shows one abbreviation on two rows. That tells the
// reader to type something that comes back ambiguous_id. Shortening to what is
// actually unique is git's rule for object hashes, which 5.5 already invokes
// for prefixes.
//
// It takes IDs rather than tickets because it needs nothing else from one.
//
// Under 5.6 it shortens within a series rather than across the store, because
// the prefix already carries the rest of the disambiguation. Two tickets minted
// in the same millisecond in different series then print short abbreviations
// that both resolve, where a store-wide computation would lengthen both for a
// collision resolution never sees. Every ID carries its own series, so the
// grouping comes out of the argument and the signature does not change.
func ShortestUnique(ids []string) map[string]string {
	groups := make(map[string][]string)
	for _, id := range ids {
		series, _ := SplitID(id)
		groups[series] = append(groups[series], id)
	}
	out := make(map[string]string, len(ids))
	for _, group := range groups {
		for id, short := range shortestWithin(group) {
			out[id] = short
		}
	}
	return out
}

// ShortestUniqueAcrossSeries is the same computation over the whole store at
// once, which is what `--ids store` prints, per 5.6. The ULID half of every
// result resolves on its own, which is what somebody pasting an ID into a
// document read outside this store wants.
//
// It is a second function rather than a flag on the first because the two
// answer different questions and both are wanted at once: a listing chooses,
// and neither choice is a mode the other should have to carry.
func ShortestUniqueAcrossSeries(ids []string) map[string]string {
	return shortestWithin(ids)
}

// shortestWithin abbreviates one set of IDs against each other, ignoring the
// series. Both exported forms are this function over a different grouping.
func shortestWithin(ids []string) map[string]string {
	bodies := make([]string, 0, len(ids))
	for _, id := range ids {
		_, body := SplitID(NormalizeRef(id))
		bodies = append(bodies, body)
	}
	sorted := append([]string{}, bodies...)
	sort.Strings(sorted)

	// In sorted order the longest prefix an ID shares with any other is shared
	// with one of its two neighbours, so the pairs are enough.
	need := make(map[string]int, len(sorted))
	for i, b := range sorted {
		n := abbrevLen
		if i > 0 {
			n = max(n, commonPrefixLen(b, sorted[i-1])+1)
		}
		if i+1 < len(sorted) {
			n = max(n, commonPrefixLen(b, sorted[i+1])+1)
		}
		need[b] = n
	}

	out := make(map[string]string, len(ids))
	for i, id := range ids {
		b := bodies[i]
		n := min(need[b], len(b))
		series, _ := SplitID(id)
		if series == "" {
			// Nothing in a store produces this, since every ID carries a
			// series. A caller passing bare ULIDs gets bare ULIDs back rather
			// than a leading hyphen.
			out[id] = b[:n]
			continue
		}
		out[id] = series + "-" + b[:n]
	}
	return out
}

func commonPrefixLen(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}
