package ticket

import (
	"crypto/rand"
	"fmt"
	"io"
	"strings"
	"time"
)

// IDPrefix is the fixed part of every ticket ID.
const IDPrefix = "TKT-"

// ulidLen is the length of the Crockford base32 ULID that follows the prefix.
const ulidLen = 26

// minPrefixLen is how much of a ULID a caller must type for a prefix to be
// considered, per plan 5.5. Four characters is enough that a typo does not
// resolve to a real ticket by accident.
const minPrefixLen = 4

// crockford is Crockford base32: the digits and the uppercase letters, less I,
// L, O, and U, which are the ones a person misreads.
const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NewID returns a ticket ID for the given instant: TKT- followed by a
// 26-character ULID. ULIDs need no central counter, so two disconnected agents
// cannot collide, and they sort by creation time.
func NewID(at time.Time, entropy io.Reader) (string, error) {
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
	return IDPrefix + encodeULID(raw), nil
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

// ValidID reports whether s is a well-formed ticket ID.
func ValidID(s string) bool {
	if !strings.HasPrefix(s, IDPrefix) {
		return false
	}
	return validULID(s[len(IDPrefix):])
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

// NormalizeRef puts a user-typed reference into the form IDs are stored in: the
// TKT- prefix removed and the rest uppercased. Matching is case-insensitive
// because a person types an ID in whatever case their terminal gave them.
//
// Crockford's letter substitutions are deliberately not applied. Reading I as 1
// would let two different typos resolve to the same ticket, and git does not do
// it for object hashes either.
func NormalizeRef(ref string) string {
	s := strings.TrimSpace(ref)
	if len(s) >= len(IDPrefix) && strings.EqualFold(s[:len(IDPrefix)], IDPrefix) {
		s = s[len(IDPrefix):]
	}
	return strings.ToUpper(s)
}

// ResolveRef matches a reference against a set of known IDs, per plan 5.5. It
// accepts a full ID or a unique prefix of at least four characters, and returns
// ambiguous_id listing the candidates when more than one ticket matches.
func ResolveRef(ref string, ids []string) (string, error) {
	norm := NormalizeRef(ref)
	if norm == "" {
		return "", codedError(CodeTicketNotFound, "no ticket reference given")
	}
	if validULID(norm) {
		full := IDPrefix + norm
		for _, id := range ids {
			if id == full {
				return id, nil
			}
		}
		return "", &Error{
			Code:    CodeTicketNotFound,
			Message: fmt.Sprintf("no ticket %s in this store", full),
			Ticket:  full,
		}
	}
	if len(norm) < minPrefixLen {
		return "", codedError(CodeTicketNotFound,
			"%q is shorter than the %d characters a prefix needs", ref, minPrefixLen)
	}

	var matches []string
	for _, id := range ids {
		if strings.HasPrefix(NormalizeRef(id), norm) {
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
