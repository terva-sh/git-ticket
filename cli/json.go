package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/terva-sh/git-ticket/ticket"
)

// The JSON contract of plan section 10. Absent scalars are null and absent
// collections are [], always present rather than omitted, so a consumer never
// has to tell missing from empty. That rule is why so many fields here are
// pointers and why every slice is built with make rather than left nil.

const schemaVersion = 1

type ticketEnvelope struct {
	SchemaVersion int         `json:"schemaVersion"`
	Kind          string      `json:"kind"`
	Ticket        *ticketJSON `json:"ticket"`
}

type ticketListEnvelope struct {
	SchemaVersion int           `json:"schemaVersion"`
	Kind          string        `json:"kind"`
	Tickets       []*ticketJSON `json:"tickets"`
	// Unreadable names the ticket files a query had to leave out. Without it a
	// short listing is indistinguishable from a complete one, and a host
	// building a board on this envelope would show neither the ticket nor the
	// reason it is missing. It carries the same shape check reports.
	Unreadable []findingJSON `json:"unreadable"`
}

// versionEnvelope answers --version for a host. It describes the binary rather
// than a store, so it carries no ticket and no findings.
type versionEnvelope struct {
	SchemaVersion int    `json:"schemaVersion"`
	Kind          string `json:"kind"`
	Version       string `json:"version"`
	Commit        string `json:"commit"`
	Go            string `json:"go"`
	Modified      bool   `json:"modified"`
}

type mutationEnvelope struct {
	SchemaVersion int             `json:"schemaVersion"`
	Kind          string          `json:"kind"`
	Ticket        *mutationTicket `json:"ticket"`
	PathsChanged  []string        `json:"pathsChanged"`
}

// mutationTicket is the identity of what changed. A caller that wants the whole
// ticket back asks for it with show, which is the operation that returns one.
type mutationTicket struct {
	ID       string `json:"id"`
	Revision string `json:"revision"`
}

type checkEnvelope struct {
	SchemaVersion int           `json:"schemaVersion"`
	Kind          string        `json:"kind"`
	OK            bool          `json:"ok"`
	Errors        []findingJSON `json:"errors"`
	Warnings      []findingJSON `json:"warnings"`
	// PathsChanged names every file --fix moved, both ends of each move, the
	// way a mutation reports one. It is empty without --fix, and empty under
	// --dry-run because nothing was written.
	PathsChanged []string `json:"pathsChanged"`
	// Repairs is what --fix did, or under --dry-run what it would do. It is
	// the only place a dry run says anything, since the findings it would
	// clear are still in errors.
	Repairs []repairJSON `json:"repairs"`
	DryRun  bool         `json:"dryRun"`
}

// repairJSON is one move. codes names the findings it clears, as a list because
// a file in the wrong directory under the wrong name raises both and one move
// settles them together.
type repairJSON struct {
	Codes  []string `json:"codes"`
	Ticket string   `json:"ticket"`
	From   string   `json:"from"`
	To     string   `json:"to"`
}

type findingJSON struct {
	Code   string  `json:"code"`
	File   string  `json:"file"`
	Ticket *string `json:"ticket"`
	Field  *string `json:"field"`
}

// schemaEnvelope is what `git ticket schema` emits: the values a consumer would
// otherwise have to read the plan to learn. Everything in it comes from the
// code that enforces it, so it cannot drift from what the binary does.
type schemaEnvelope struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Kind          string              `json:"kind"`
	TicketSchema  int                 `json:"ticketSchema"`
	Kinds         []string            `json:"kinds"`
	Statuses      []string            `json:"statuses"`
	Types         []string            `json:"types"`
	Priorities    []string            `json:"priorities"`
	Transitions   map[string][]string `json:"transitions"`
	ErrorCodes    []string            `json:"errorCodes"`
	FindingCodes  []findingCodeJSON   `json:"findingCodes"`
}

type findingCodeJSON struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
}

// instructionsEnvelope carries the agent workflow block. The block is prose, so
// the envelope holds it as one string rather than pretending it has structure.
type instructionsEnvelope struct {
	SchemaVersion int    `json:"schemaVersion"`
	Kind          string `json:"kind"`
	Text          string `json:"text"`
}

type errorEnvelope struct {
	SchemaVersion int       `json:"schemaVersion"`
	Kind          string    `json:"kind"`
	Error         errorBody `json:"error"`
}

type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details"`
}

type ticketJSON struct {
	ID           string          `json:"id"`
	Revision     string          `json:"revision"`
	Path         *string         `json:"path"`
	Schema       int             `json:"schema"`
	Title        string          `json:"title"`
	Type         string          `json:"type"`
	Status       string          `json:"status"`
	StatusReason *string         `json:"statusReason"`
	Priority     string          `json:"priority"`
	Labels       []string        `json:"labels"`
	Assignees    []string        `json:"assignees"`
	Milestone    *string         `json:"milestone"`
	Parent       *string         `json:"parent"`
	Dependencies []string        `json:"dependencies"`
	References   []referenceJSON `json:"references"`
	Claim        *claimJSON      `json:"claim"`
	Archive      *archiveJSON    `json:"archive"`
	CreatedAt    *string         `json:"createdAt"`
	UpdatedAt    *string         `json:"updatedAt"`
	CreatedBy    *actorJSON      `json:"createdBy"`
	UpdatedBy    *actorJSON      `json:"updatedBy"`
	Extensions   map[string]any  `json:"extensions"`
	Unknown      map[string]any  `json:"unknown"`
	Body         bodyJSON        `json:"body"`
	Checklists   checklistsJSON  `json:"checklists"`
	Comments     []entryJSON     `json:"comments"`
	Readiness    readinessJSON   `json:"readiness"`
}

// readinessJSON is derived from the whole store at read time and never stored,
// like revision and path. It carries the verdict `ready` filters on, plus what
// stands in the way, so a consumer drawing a list can grey out a blocked ticket
// and say why without a second call.
//
// isBlocked is about dependencies alone. A draft and a held ticket are unready
// and not blocked, because nothing is in their way but their own state.
type readinessJSON struct {
	IsReady   bool `json:"isReady"`
	IsBlocked bool `json:"isBlocked"`
	// BlockingDependencies resolve to a ticket that is not done.
	// MissingDependencies are the IDs no single ticket in the store claims,
	// which covers an ID nothing claims and one that two files claim. Neither
	// ever counts as satisfied.
	BlockingDependencies []string `json:"blockingDependencies"`
	MissingDependencies  []string `json:"missingDependencies"`
}

type referenceJSON struct {
	Ref  string  `json:"ref"`
	Path *string `json:"path"`
}

type actorJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type claimJSON struct {
	Actor     string  `json:"actor"`
	Branch    *string `json:"branch"`
	Worktree  *string `json:"worktree"`
	Commit    *string `json:"commit"`
	ClaimedAt *string `json:"claimedAt"`
	ExpiresAt *string `json:"expiresAt"`
}

type archiveJSON struct {
	ArchivedAt *string `json:"archivedAt"`
	FromStatus *string `json:"fromStatus"`
	Reason     *string `json:"reason"`
}

// bodyJSON is every section exactly as it appears in the file, one type for all
// of them, so nothing a person wrote by hand is dropped on the way out.
type bodyJSON struct {
	Preamble           string        `json:"preamble"`
	Description        string        `json:"description"`
	AcceptanceCriteria string        `json:"acceptanceCriteria"`
	DefinitionOfDone   string        `json:"definitionOfDone"`
	ImplementationPlan string        `json:"implementationPlan"`
	Notes              string        `json:"notes"`
	Comments           string        `json:"comments"`
	Summary            string        `json:"summary"`
	Extra              []sectionJSON `json:"extra"`
}

type sectionJSON struct {
	Heading string `json:"heading"`
	Text    string `json:"text"`
}

// checklistsJSON is derived from bodyJSON and never the other way round, so the
// two cannot disagree about what a ticket says.
type checklistsJSON struct {
	AcceptanceCriteria []checklistItemJSON `json:"acceptanceCriteria"`
	DefinitionOfDone   []checklistItemJSON `json:"definitionOfDone"`
}

// checklistItemJSON carries the index rather than leaving it to be counted from
// an array position, because the number `ac --check N` takes counts checkbox
// lines only and a section may hold other prose.
type checklistItemJSON struct {
	Index   int    `json:"index"`
	Checked bool   `json:"checked"`
	Text    string `json:"text"`
}

// entryJSON is one comment, split out of body.comments so a consumer can draw a
// thread without parsing Markdown. Like checklists it is derived and never the
// other way round: body.comments stays the whole section verbatim.
//
// actor and at are null for an entry somebody typed without the stamp `comment`
// writes. That entry is still a comment a person left, so it comes back with
// its text rather than being dropped.
type entryJSON struct {
	Index int     `json:"index"`
	Actor *string `json:"actor"`
	At    *string `json:"at"`
	Text  string  `json:"text"`
}

func newTicketJSON(s *ticket.Store, t *ticket.Ticket, r ticket.Readiness) *ticketJSON {
	out := &ticketJSON{
		ID:           t.ID,
		Revision:     t.Revision,
		Path:         optionalString(displayPath(s, t.Path)),
		Schema:       t.Schema,
		Title:        t.Title,
		Type:         t.Type,
		Status:       t.Status,
		StatusReason: copyString(t.StatusReason),
		Priority:     t.Priority,
		Labels:       stringSlice(t.Labels),
		Assignees:    stringSlice(t.Assignees),
		Milestone:    copyString(t.Milestone),
		Parent:       copyString(t.Parent),
		Dependencies: stringSlice(t.Dependencies),
		References:   make([]referenceJSON, 0, len(t.References)),
		CreatedAt:    timestamp(&t.CreatedAt),
		UpdatedAt:    timestamp(&t.UpdatedAt),
		CreatedBy:    actor(t.CreatedBy),
		UpdatedBy:    actor(t.UpdatedBy),
		Extensions:   t.ExtensionsMap(),
		Unknown:      t.UnknownMap(),
		Body: bodyJSON{
			Preamble:           t.Body.Preamble,
			Description:        t.Body.Description,
			AcceptanceCriteria: t.Body.AcceptanceCriteria,
			DefinitionOfDone:   t.Body.DefinitionOfDone,
			ImplementationPlan: t.Body.ImplementationPlan,
			Notes:              t.Body.Notes,
			Comments:           t.Body.Comments,
			Summary:            t.Body.Summary,
			Extra:              make([]sectionJSON, 0, len(t.Body.Extra)),
		},
		Checklists: checklistsJSON{
			AcceptanceCriteria: checklist(t.Body.AcceptanceCriteria),
			DefinitionOfDone:   checklist(t.Body.DefinitionOfDone),
		},
		Comments: entries(t.Body.Comments),
		Readiness: readinessJSON{
			IsReady:              r.Ready,
			IsBlocked:            r.Blocked,
			BlockingDependencies: stringSlice(r.Blocking),
			MissingDependencies:  stringSlice(r.Missing),
		},
	}
	for _, r := range t.References {
		out.References = append(out.References, referenceJSON{Ref: r.Ref, Path: copyString(r.Path)})
	}
	for _, s := range t.Body.Extra {
		out.Body.Extra = append(out.Body.Extra, sectionJSON{Heading: s.Heading, Text: s.Text})
	}
	if c := t.Claim; c != nil {
		out.Claim = &claimJSON{
			Actor:     c.Actor,
			Branch:    copyString(c.Branch),
			Worktree:  copyString(c.Worktree),
			Commit:    copyString(c.Commit),
			ClaimedAt: timestamp(c.ClaimedAt),
			ExpiresAt: timestamp(c.ExpiresAt),
		}
	}
	if a := t.Archive; a != nil {
		out.Archive = &archiveJSON{
			ArchivedAt: timestamp(a.ArchivedAt),
			FromStatus: copyString(a.FromStatus),
			Reason:     copyString(a.Reason),
		}
	}
	return out
}

// newCheckEnvelope converts a report for the wire. ok is the verdict rather
// than a count of errors: under --strict a warning fails the run, and plan 10.3
// makes ok true exactly when the command exits zero, so one field answers the
// question a caller is asking.
//
// --strict moves no finding between the arrays. The two arrays report severity
// as section 11 defines it, whatever the caller asked for.
func newCheckEnvelope(s *ticket.Store, r *ticket.Report, ok bool, repairs []ticket.Repair, dryRun bool) checkEnvelope {
	env := checkEnvelope{
		SchemaVersion: schemaVersion,
		Kind:          "check-report",
		OK:            ok,
		Errors:        findings(s, r.Errors),
		Warnings:      findings(s, r.Warnings),
		PathsChanged:  []string{},
		Repairs:       make([]repairJSON, 0, len(repairs)),
		DryRun:        dryRun,
	}
	for _, rep := range repairs {
		from := storePath(s, rep.From)
		to := storePath(s, rep.To)
		env.Repairs = append(env.Repairs, repairJSON{
			Codes: rep.Codes, Ticket: rep.Ticket, From: from, To: to,
		})
		if !dryRun {
			env.PathsChanged = append(env.PathsChanged, to, from)
		}
	}
	return env
}

// storePath turns a store-relative path into the repository-relative form every
// path in the contract uses, the same conversion findings makes.
func storePath(s *ticket.Store, rel string) string {
	return displayPath(s, filepath.Join(s.Path(), filepath.FromSlash(rel)))
}

// findings converts each file to the repository-relative form the rest of the
// contract uses. The library reports a finding's file relative to the store,
// because a report describes a store and may be made where no repository root
// is known, so the conversion belongs here. Plan section 10 says so.
func findings(s *ticket.Store, in []ticket.Finding) []findingJSON {
	out := make([]findingJSON, 0, len(in))
	for _, f := range in {
		out = append(out, findingJSON{
			Code:   f.Code,
			File:   displayPath(s, filepath.Join(s.Path(), f.File)),
			Ticket: optionalString(f.Ticket),
			Field:  optionalString(f.Field),
		})
	}
	return out
}

// checklist is the derived view: the checkbox lines of a section, numbered the
// way ac and dod number them.
func checklist(text string) []checklistItemJSON {
	items := ticket.Checklist(text)
	out := make([]checklistItemJSON, 0, len(items))
	for i, item := range items {
		out = append(out, checklistItemJSON{Index: i + 1, Checked: item.Checked, Text: item.Text})
	}
	return out
}

// entries is the derived view of the comments section: one element per stamped
// block, numbered from one the way a person would count them.
func entries(text string) []entryJSON {
	found := ticket.Entries(text)
	out := make([]entryJSON, 0, len(found))
	for _, e := range found {
		out = append(out, entryJSON{
			Index: e.Index,
			Actor: optionalString(e.Actor),
			At:    optionalString(e.At),
			Text:  e.Text,
		})
	}
	return out
}

func timestamp(t *ticket.Timestamp) *string {
	if t == nil || (t.Raw == "" && t.Time.IsZero()) {
		return nil
	}
	s := t.String()
	return &s
}

func actor(a *ticket.Actor) *actorJSON {
	if a == nil {
		return nil
	}
	return &actorJSON{ID: a.ID, Name: a.Name}
}

func copyString(s *string) *string {
	if s == nil {
		return nil
	}
	v := *s
	return &v
}

func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// stringSlice returns a slice that marshals as [] rather than null.
func stringSlice(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func writeJSON(w io.Writer, v any) {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	// The encoder writes to a stream this package owns. A failure here is a
	// broken stdout, which the caller finds out about anyway.
	if err := enc.Encode(v); err != nil {
		fmt.Fprintf(io.Discard, "%v", err)
	}
}
