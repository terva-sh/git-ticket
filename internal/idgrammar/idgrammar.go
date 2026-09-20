// Package idgrammar is the grammar of a ticket ID and nothing else: a series, a
// hyphen, and a 26-character Crockford base32 ULID, per plan 5.6.
//
// It is a leaf so that two packages can agree on it without one importing the
// other. ticket owns the ID and everything around it, and layout checks that a
// frame member is well-formed; since ticket reads a layout back under check,
// layout cannot reach into ticket for the rule, so both read it from here.
package idgrammar

import "strings"

// SeriesMinLen and SeriesMaxLen are the bounds of plan 5.6, restated by
// ticket.SeriesMinLen and ticket.SeriesMaxLen, which are the published ones.
const (
	SeriesMinLen = 2
	SeriesMaxLen = 8
)

// ULIDLen is the length of the Crockford base32 ULID that follows the series.
const ULIDLen = 26

// Crockford is Crockford base32: the digits and the uppercase letters, less I,
// L, O, and U, which are the ones a person misreads.
const Crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// ValidSeries reports whether s is a well-formed series prefix: two to eight
// characters, an uppercase letter first, uppercase letters and digits after.
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

// Split separates an ID into its series and its ULID at the first hyphen. An
// ID with no hyphen has no series.
func Split(id string) (series, ulid string) {
	if i := strings.IndexByte(id, '-'); i >= 0 {
		return id[:i], id[i+1:]
	}
	return "", id
}

// ValidULID reports whether s is 26 characters of Crockford base32.
func ValidULID(s string) bool {
	if len(s) != ULIDLen {
		return false
	}
	for _, r := range s {
		if !strings.ContainsRune(Crockford, r) {
			return false
		}
	}
	return true
}

// Valid reports whether s is a well-formed ticket ID. It is grammar alone:
// whether the series is one a store declares is a different question.
func Valid(s string) bool {
	series, ulid := Split(s)
	return ValidSeries(series) && ValidULID(ulid)
}
