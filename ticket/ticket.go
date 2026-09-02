// Package ticket implements the git-ticket format and store: Markdown tickets
// with YAML frontmatter under .tickets/, parsed, validated, and rendered
// deterministically.
//
// The format is specified in docs/plan.md sections 4 through 11. Where this
// code and that document disagree, the document is right and this code is a
// bug.
package ticket

import (
	"time"

	"gopkg.in/yaml.v3"
)

// SchemaVersion is the ticket schema this package reads and writes. A file
// declaring a higher version is refused with SchemaUnsupported rather than
// parsed, because a major bump may remove a field or change its meaning.
const SchemaVersion = 1

// Status values, per plan 6.1.
const (
	StatusDraft      = "draft"
	StatusReady      = "ready"
	StatusInProgress = "in-progress"
	StatusBlocked    = "blocked"
	StatusReview     = "review"
	StatusDone       = "done"
	StatusArchived   = "archived"
)

// Statuses lists every valid status in lifecycle order.
var Statuses = []string{
	StatusDraft, StatusReady, StatusInProgress,
	StatusBlocked, StatusReview, StatusDone, StatusArchived,
}

// Types lists every valid ticket type, per plan 5.1.
var Types = []string{"task", "bug", "chore", "spike", "epic"}

// Priorities lists every valid priority, per plan 5.1.
var Priorities = []string{"low", "normal", "high", "urgent"}

// BlocksOn values, per plan 5.1. The field says which edges gate a ticket in
// addition to its dependencies, which always gate, per 6.3.
//
// It is additive rather than selective on purpose. A value that switched
// dependency gating off would let one field quietly undo the rule 6.3 defines,
// and since the default renders on every ticket in the store that would be a
// store-wide change spelled as a default. BlocksOnChildren therefore adds the
// direct children to the gating set and takes nothing away.
const (
	BlocksOnNone     = "none"
	BlocksOnChildren = "children"
)

// BlocksOnValues lists every valid blocks_on value, per plan 5.1.
var BlocksOnValues = []string{BlocksOnNone, BlocksOnChildren}

func validValue(set []string, v string) bool {
	for _, s := range set {
		if s == v {
			return true
		}
	}
	return false
}

// ValidStatus reports whether s is in the set of plan 6.1.
func ValidStatus(s string) bool { return validValue(Statuses, s) }

// ValidType reports whether t is in the set of plan 5.1.
func ValidType(t string) bool { return validValue(Types, t) }

// ValidPriority reports whether p is in the set of plan 5.1.
func ValidPriority(p string) bool { return validValue(Priorities, p) }

// DateLayout is the shape of a due_on value, per plan 5.1: a calendar date
// rather than an instant, meaning the end of that day in UTC.
const DateLayout = "2006-01-02"

// ValidDueOn reports whether s is a due_on this format accepts.
//
// It demands the exact YYYY-MM-DD shape rather than whatever time.Parse
// tolerates, so 2026-1-4 is refused along with an RFC3339 instant. Truncating
// an instant to its date would throw away a distinction the author can be seen
// making, per 5.1, so nothing here truncates.
func ValidDueOn(s string) bool {
	d, err := time.Parse(DateLayout, s)
	return err == nil && d.Format(DateLayout) == s
}

// ValidBlocksOn reports whether b is in the set of plan 5.1.
func ValidBlocksOn(b string) bool { return validValue(BlocksOnValues, b) }

// Timestamp is an instant that remembers how it was written. Rendering emits
// Raw when it is set, so a file that spells an instant differently than this
// package would still round-trips byte for byte. A timestamp this package
// creates has an empty Raw and renders as RFC 3339 in UTC.
type Timestamp struct {
	Time time.Time
	Raw  string
}

// Now returns a Timestamp for t with no recorded spelling, so it renders
// canonically.
func Now(t time.Time) Timestamp { return Timestamp{Time: t.UTC()} }

// String returns the rendered form: the original spelling when there is one,
// and RFC 3339 in UTC otherwise.
func (t Timestamp) String() string {
	if t.Raw != "" {
		return t.Raw
	}
	return t.Time.UTC().Format(time.RFC3339)
}

// Actor identifies who did something: a person or an agent session.
type Actor struct {
	ID   string
	Name string
}

// Reference is a typed stable identifier with an optional repository-relative
// path. The core preserves the namespace without interpreting it.
type Reference struct {
	Ref  string
	Path *string
}

// Claim records that an actor is working a ticket. It is metadata and not a
// status, per plan 6.4.
type Claim struct {
	Actor     string
	Branch    *string
	Worktree  *string
	Commit    *string
	ClaimedAt *Timestamp
	ExpiresAt *Timestamp
}

// Expired reports whether the claim is past its expiry at the given instant. A
// claim with no expiry never expires.
func (c *Claim) Expired(at time.Time) bool {
	if c == nil || c.ExpiresAt == nil {
		return false
	}
	return at.After(c.ExpiresAt.Time)
}

// Archive records that a ticket was archived and what status it held first.
// FromStatus is what keeps archiving from silently blocking dependents, per
// plan 6.3.
type Archive struct {
	ArchivedAt *Timestamp
	FromStatus *string
	Reason     *string
}

// UnknownField is a top-level frontmatter key this version does not define. It
// is preserved through a write so that a newer reader sharing the repository
// does not lose data, per plan 5.4.
type UnknownField struct {
	Key   string
	Value *yaml.Node
}

// Section is a body section this version does not define. Unknown sections
// survive a round trip and render after the known ones in their original
// relative order.
type Section struct {
	Heading string
	Text    string
}

// Body holds the Markdown below the frontmatter. Each known section is stored
// as its raw text so that a round trip cannot lose formatting the parser did
// not anticipate. Checklist sections are read through Items and written
// through the checklist mutations.
type Body struct {
	// Preamble is text before the first section heading. The renderer emits
	// none, but a hand-edited file may carry some and must not lose it.
	Preamble           string
	Description        string
	AcceptanceCriteria string
	DefinitionOfDone   string
	ImplementationPlan string
	Notes              string
	Comments           string
	Summary            string
	Extra              []Section
}

// Ticket is one parsed ticket file.
type Ticket struct {
	Schema int
	ID     string
	Title  string
	Type   string
	Status string
	// StatusReason is the reason 6.2 requires for entering blocked and for
	// reopening from done. It holds the current reason only: a transition that
	// requires a reason overwrites it and any other transition clears it. The
	// history lives in Notes.
	StatusReason *string
	Priority     string
	// DueOn is a deadline as a YYYY-MM-DD date meaning the end of that day in
	// UTC, per plan 5.1. Nil is no deadline. It is the one time value in this
	// format that is not an instant, which is why the key ends _on rather than
	// _at, and it is stored exactly as written: a deadline is a claim about a
	// calendar day, and expanding it would have to pick a zone at write time.
	DueOn        *string
	Labels       []string
	Assignees    []string
	Milestone    *string
	Parent       *string
	Dependencies []string
	// BlocksOn names the edges that gate this ticket beyond its dependencies,
	// per plan 5.1. It is an enum and always carries a value, like Type and
	// Priority and unlike Milestone, so an absent key parses as BlocksOnNone
	// rather than as null. None means dependencies alone, which is what every
	// ticket did before this field existed.
	BlocksOn   string
	References []Reference
	Claim      *Claim
	Archive    *Archive
	CreatedAt  Timestamp
	UpdatedAt  Timestamp
	CreatedBy  *Actor
	UpdatedBy  *Actor
	Extensions *yaml.Node

	// Unknown holds top-level frontmatter keys this version does not define,
	// in the order they appeared.
	Unknown []UnknownField

	Body Body

	// Path is where the ticket was read from, and Revision is the SHA-256 of
	// the bytes on disk at that moment. Both are empty for a ticket that was
	// built rather than read. Revision is computed and never stored in the
	// file, per plan 7.1.
	Path     string
	Revision string
}

// Archived reports whether the status is archived. The status is
// authoritative when it disagrees with the directory, per plan 6.3.
func (t *Ticket) Archived() bool { return t.Status == StatusArchived }

// ExtensionsMap decodes the extensions mapping into plain Go values, for a
// consumer that speaks JSON rather than YAML nodes. It returns an empty map
// when the ticket carries no extensions.
func (t *Ticket) ExtensionsMap() map[string]any { return decodeNodeMap(t.Extensions) }

// UnknownMap decodes the top-level fields this version does not define, keyed
// by their field names. They are preserved on write either way, per plan 5.4;
// this is only how a consumer reads them.
func (t *Ticket) UnknownMap() map[string]any {
	out := map[string]any{}
	for _, u := range t.Unknown {
		var v any
		if u.Value != nil {
			_ = u.Value.Decode(&v)
		}
		out[u.Key] = v
	}
	return out
}

func decodeNodeMap(n *yaml.Node) map[string]any {
	out := map[string]any{}
	if n == nil || n.Kind != yaml.MappingNode {
		return out
	}
	_ = n.Decode(&out)
	return out
}

// SatisfiesDependency reports whether a ticket depending on t may proceed: t is
// done, or archived from done, per plan 6.3.
func (t *Ticket) SatisfiesDependency() bool {
	if t.Status == StatusDone {
		return true
	}
	if t.Status == StatusArchived && t.Archive != nil && t.Archive.FromStatus != nil {
		return *t.Archive.FromStatus == StatusDone
	}
	return false
}
