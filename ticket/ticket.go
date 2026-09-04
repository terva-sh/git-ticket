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
	"unicode/utf8"

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

// TerminalStatuses are the statuses a listing leaves out by default, per plan
// section 8. Done and archived both mean the work is over, and a list of work
// is about work that is still live.
var TerminalStatuses = []string{StatusDone, StatusArchived}

// OpenStatuses is what a listing answers with by default: every status that is
// not terminal.
//
// It is derived rather than written out, because section 8 requires the rule to
// be an exclusion. A status added to Statuses is open unless somebody also
// names it terminal, where a hand-written list of the open five would drop it
// silently. Whether this format grows custom statuses is still open in section
// 15, so that is not hypothetical.
var OpenStatuses = openStatuses()

func openStatuses() []string {
	out := make([]string, 0, len(Statuses))
	for _, s := range Statuses {
		if !TerminalStatus(s) {
			out = append(out, s)
		}
	}
	return out
}

// TerminalStatus reports whether a status means the work is over.
func TerminalStatus(status string) bool {
	for _, s := range TerminalStatuses {
		if s == status {
			return true
		}
	}
	return false
}

// The two unready reasons that are not a status echo, per plan section 8.
//
// ReasonWaitingOnDependencies is spelled out rather than called "blocked"
// because Readiness.Blocked and the isBlocked key already mean "a dependency or
// a blocking child is in the way", while StatusBlocked means a person marked
// the ticket blocked and wrote a reason. Both senses of the word are load
// bearing and neither can be renamed, so the new field takes a third name and
// each word keeps one meaning per field: "blocked" in a reason is always the
// status, isBlocked is always the dependency graph.
//
// ReasonClaimed is "claimed" and not "claimed_by_other", because readiness is
// computed from the store and a clock with no actor anywhere in the call. It
// cannot tell your own claim from somebody else's, so a name asserting the
// difference would be a claim the code cannot check. A caller that wants it
// compares claim.by against itself, which it can do and this cannot.
const (
	ReasonWaitingOnDependencies = "waiting_on_dependencies"
	ReasonClaimed               = "claimed"
)

// UnreadyReasons lists every value Readiness.Reason can carry. The empty string
// a ready ticket carries is not in it, because it is the absence of a reason
// rather than one of them.
//
// The status half is derived rather than written out, for the reason
// OpenStatuses is: a status added to Statuses becomes a reason here with no
// edit, where a hand-written list would silently lack it and a consumer
// switching on the field would fall through. Whether this format grows custom
// statuses is still open in section 15, so that is not hypothetical.
var UnreadyReasons = unreadyReasons()

func unreadyReasons() []string {
	out := make([]string, 0, len(Statuses)+1)
	for _, s := range Statuses {
		// Every status except ready is its own reason. Ready is missing because a
		// ticket that is ready and startable has no reason at all, and one that is
		// ready but held or waiting reports what actually stands in the way.
		if s != StatusReady {
			out = append(out, s)
		}
	}
	return append(out, ReasonWaitingOnDependencies, ReasonClaimed)
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

// Title length thresholds, per plan 5.1. A title is the only part of a ticket
// reference that means anything to a person, since a ULID does not, so it is
// bounded to stay readable beside an ID rather than to save space.
//
// Two thresholds rather than one because a single useful cap would be
// retroactive. TitleWarn names a title that has grown into a sentence and
// TitleMax is where a write is refused. The gap between them is deliberate: a
// store that predates this rule keeps working and surfaces its long titles as
// warnings, while only a title nobody meant to write reaches the refusal.
const (
	TitleWarn = 72
	TitleMax  = 120
)

// TitleLength counts a title the way plan 5.1 measures it, in characters rather
// than bytes. A title written in a language that needs more bytes per character
// is not longer to read, and measuring bytes would cap it shorter for no reason
// the author could see.
func TitleLength(title string) int { return utf8.RuneCountInString(title) }

// TitleTooLong reports whether a title exceeds the length a write may store.
func TitleTooLong(title string) bool { return TitleLength(title) > TitleMax }

// TitleLong reports whether a title is past the point check warns about. It is
// false once TitleTooLong is true, so one title raises one finding rather than
// both: the error already says the title is too long and a warning beside it
// would be the same sentence twice.
func TitleLong(title string) bool {
	n := TitleLength(title)
	return n > TitleWarn && n <= TitleMax
}

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

	// Branch is the ref this copy came from, in the short form for-each-ref
	// prints, and empty when the copy came from the working tree. Only a
	// cross-branch query sets it, per plan section 8, so every row of an
	// ordinary query carries the empty string.
	//
	// When it is set, Path is where the file sits on that ref rather than a
	// path in the caller's working tree, and opening it will fail. That pairing
	// is the point: a row naming a ticket the caller cannot open has to say so.
	Branch string
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
