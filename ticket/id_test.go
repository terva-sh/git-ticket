package ticket

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestNewIDShape(t *testing.T) {
	id, err := NewID(referenceInstant, bytes.NewReader(bytes.Repeat([]byte{0xAB}, 10)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(id, IDPrefix) {
		t.Fatalf("id %q does not start with %s", id, IDPrefix)
	}
	body := id[len(IDPrefix):]
	if len(body) != ulidLen {
		t.Errorf("ULID length = %d, want %d", len(body), ulidLen)
	}
	for _, r := range body {
		if !strings.ContainsRune(crockford, r) {
			t.Errorf("character %q is not Crockford base32", r)
		}
	}
	if !ValidID(id) {
		t.Errorf("ValidID(%q) = false", id)
	}
}

// TestNewIDSortsByTime is the property that makes a directory listing
// chronological, which is half the reason for choosing ULIDs.
func TestNewIDSortsByTime(t *testing.T) {
	zeros := func() *bytes.Reader { return bytes.NewReader(make([]byte, 10)) }
	early, err := NewID(referenceInstant, zeros())
	if err != nil {
		t.Fatal(err)
	}
	late, err := NewID(referenceInstant.Add(time.Second), zeros())
	if err != nil {
		t.Fatal(err)
	}
	if !(early < late) {
		t.Errorf("%s should sort before %s", early, late)
	}
}

func TestEncodeULIDBoundaries(t *testing.T) {
	var zero [16]byte
	if got := encodeULID(zero); got != strings.Repeat("0", ulidLen) {
		t.Errorf("all-zero ULID = %q", got)
	}
	var ones [16]byte
	for i := range ones {
		ones[i] = 0xFF
	}
	// 128 bits of ones in a 130-bit space: the first character carries the two
	// pad bits and so tops out at 7, not Z.
	got := encodeULID(ones)
	if got[0] != '7' {
		t.Errorf("first character = %q, want 7", got[0])
	}
	if rest := got[1:]; rest != strings.Repeat("Z", ulidLen-1) {
		t.Errorf("all-ones ULID tail = %q", rest)
	}
}

func TestResolveRef(t *testing.T) {
	ids := []string{
		"TKT-01K3ZZFCP0VJKES58GZG0QDHG0",
		"TKT-01K3ZZH790E1HXA78V5PGPBVQ0",
		"TKT-01K3ZZK1W0XWVS6RYX1JVVWSK3",
	}
	cases := []struct {
		name string
		ref  string
		want string
		code string
	}{
		{"full id", "TKT-01K3ZZH790E1HXA78V5PGPBVQ0", ids[1], ""},
		{"full ulid without the prefix", "01K3ZZH790E1HXA78V5PGPBVQ0", ids[1], ""},
		{"lowercase", "tkt-01k3zzh790e1hxa78v5pgpbvq0", ids[1], ""},
		{"unique prefix", "01K3ZZH7", ids[1], ""},
		{"lowercase prefix", "01k3zzk1", ids[2], ""},
		{"ambiguous prefix", "01K3", "", CodeAmbiguousID},
		{"prefix under four characters", "01K", "", CodeTicketNotFound},
		{"no match", "ZZZZ", "", CodeTicketNotFound},
		{"full id not in the store", "TKT-01K3ZZZZZZZZZZZZZZZZZZZZZZ", "", CodeTicketNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolveRef(c.ref, ids)
			if c.code != "" {
				if CodeOf(err) != c.code {
					t.Fatalf("err = %v, want code %s", err, c.code)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("resolved to %s, want %s", got, c.want)
			}
		})
	}
}

// TestAmbiguousRefListsCandidates is what makes the error actionable: the user
// has to be told which tickets their prefix hit.
func TestAmbiguousRefListsCandidates(t *testing.T) {
	ids := []string{"TKT-01K3ZZFCP0VJKES58GZG0QDHG0", "TKT-01K3ZZH790E1HXA78V5PGPBVQ0"}
	_, err := ResolveRef("01K3", ids)
	var e *Error
	if !asTicketError(err, &e) || e.Code != CodeAmbiguousID {
		t.Fatalf("err = %v, want %s", err, CodeAmbiguousID)
	}
	for _, id := range ids {
		if !strings.Contains(e.Details["candidates"], id) {
			t.Errorf("candidates %q does not name %s", e.Details["candidates"], id)
		}
	}
}
