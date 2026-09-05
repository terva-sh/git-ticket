package ticket

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Mutation is one change to a ticket. It is a typed set of operations rather
// than a struct of pointers, so "set the title to empty" and "do not touch the
// title" cannot be confused. Full-file replacement is deliberately not an
// operation this API offers.
type Mutation interface {
	apply(t *Ticket, env mutEnv) error
}

// Mutations applies several changes as one write, in order. Either all of them
// land or none do, because the file is only written after the last one
// succeeds.
type Mutations []Mutation

func (ms Mutations) apply(t *Ticket, env mutEnv) error {
	for _, m := range ms {
		if err := m.apply(t, env); err != nil {
			return err
		}
	}
	return nil
}

// transitions is the table in plan 6.2. Anything not listed returns
// invalid_transition naming the permitted targets.
var transitions = map[string][]string{
	StatusDraft:      {StatusReady, StatusDone, StatusArchived},
	StatusReady:      {StatusDraft, StatusInProgress, StatusBlocked, StatusArchived},
	StatusInProgress: {StatusReady, StatusBlocked, StatusReview, StatusDone, StatusArchived},
	StatusBlocked:    {StatusReady, StatusInProgress, StatusArchived},
	StatusReview:     {StatusInProgress, StatusBlocked, StatusDone, StatusArchived},
	StatusDone:       {StatusInProgress, StatusArchived},
	StatusArchived:   {StatusReady},
}

// PermittedTransitions lists where a ticket in this status may go.
func PermittedTransitions(from string) []string { return transitions[from] }

func mayTransition(from, to string) bool {
	for _, s := range transitions[from] {
		if s == to {
			return true
		}
	}
	return false
}

// SetStatus moves a ticket through the lifecycle of plan 6.2.
//
// Reason is required entering blocked and reopening from done, and accepted
// anywhere else. It lands in two places: status_reason, which a query reads
// back, and a Notes entry, which survives the transition that clears the field.
type SetStatus struct {
	Status string
	Reason string
}

func (m SetStatus) apply(t *Ticket, env mutEnv) error {
	if !ValidStatus(m.Status) {
		return &Error{
			Code:    CodeInvalidField,
			Message: fmt.Sprintf("%q is not one of %v", m.Status, Statuses),
			Ticket:  t.ID, Field: "status",
		}
	}
	// Archiving also moves the file, so it has its own operation and is not
	// reachable here.
	if m.Status == StatusArchived {
		return &Error{
			Code:    CodeInvalidTransition,
			Message: "archiving also moves the file; use archive",
			Ticket:  t.ID, Field: "status",
		}
	}
	if m.Status == t.Status {
		return nil
	}
	if !mayTransition(t.Status, m.Status) {
		return &Error{
			Code:    CodeInvalidTransition,
			Message: fmt.Sprintf("%s cannot go to %s", t.Status, m.Status),
			Ticket:  t.ID, Field: "status",
			Details: map[string]string{
				"from":      t.Status,
				"permitted": strings.Join(permittedThroughStatus(t.Status), " "),
			},
		}
	}

	if m.Reason == "" && ReasonRequired(t.Status, m.Status) {
		return &Error{
			Code:    CodeInvalidField,
			Message: reasonRequiredMessage(t.Status, m.Status),
			Ticket:  t.ID, Field: "status_reason",
		}
	}

	from := t.Status
	t.Status = m.Status
	if m.Reason != "" {
		reason := m.Reason
		t.StatusReason = &reason
		appendNote(&t.Body, env, fmt.Sprintf("%s to %s: %s", from, m.Status, m.Reason))
	} else {
		// The field carries the current reason only. The history stays in
		// Notes.
		t.StatusReason = nil
	}
	return nil
}

// permittedThroughStatus is what the status operation itself allows, which is
// the transition table less archived.
func permittedThroughStatus(from string) []string {
	var out []string
	for _, s := range transitions[from] {
		if s != StatusArchived {
			out = append(out, s)
		}
	}
	return out
}

// ReasonRequired reports whether the transition needs a --reason, per plan
// 6.2: into blocked, reopening from done, and closing a draft straight to
// done. One function, because the status picker asks the same question
// before it applies, and two copies of this list would drift.
func ReasonRequired(from, to string) bool {
	switch {
	case to == StatusBlocked:
		return true
	case from == StatusDone && to == StatusInProgress:
		return true
	case from == StatusDraft && to == StatusDone:
		return true
	}
	return false
}

func reasonRequiredMessage(from, to string) string {
	switch {
	case from == StatusDone && to == StatusInProgress:
		return "reopening from done needs a reason"
	case from == StatusDraft && to == StatusDone:
		return "closing a draft as done needs a reason saying where the work happened"
	}
	return "moving to " + to + " needs a reason"
}

// SetTitle changes the title, which may not be empty.
type SetTitle struct{ Title string }

func (m SetTitle) apply(t *Ticket, env mutEnv) error {
	if strings.TrimSpace(m.Title) == "" {
		return &Error{Code: CodeInvalidField, Message: "a ticket needs a title", Ticket: t.ID, Field: "title"}
	}
	if err := checkTitleLength(m.Title, t.ID); err != nil {
		return err
	}
	t.Title = m.Title
	return nil
}

// checkTitleLength refuses a title past the length plan 5.1 allows a write to
// store. It is refused here rather than only reported by check, the way an
// invalid priority is, because a caller that just typed it can fix it now and a
// finding discovered later has to be chased back to whoever wrote it.
//
// The message carries both numbers, since a caller who is 3 over needs to know
// by how much and not merely that it failed.
func checkTitleLength(title, id string) error {
	if !TitleTooLong(title) {
		return nil
	}
	return &Error{
		Code: CodeInvalidField,
		Message: fmt.Sprintf("the title is %d characters, over the %d a ticket may hold",
			TitleLength(title), TitleMax),
		Ticket: id,
		Field:  "title",
	}
}

// SetType and SetPriority change one enum field each.
type SetType struct{ Type string }

func (m SetType) apply(t *Ticket, env mutEnv) error {
	if !ValidType(m.Type) {
		return &Error{Code: CodeInvalidField, Message: fmt.Sprintf("%q is not one of %v", m.Type, Types), Ticket: t.ID, Field: "type"}
	}
	t.Type = m.Type
	return nil
}

// SetBlocksOn selects which edges gate a ticket beyond its dependencies, per
// plan 5.1. It never switches dependency gating off, because BlocksOn is
// additive.
type SetBlocksOn struct{ BlocksOn string }

func (m SetBlocksOn) apply(t *Ticket, env mutEnv) error {
	if !ValidBlocksOn(m.BlocksOn) {
		return &Error{Code: CodeInvalidField, Message: fmt.Sprintf("%q is not one of %v", m.BlocksOn, BlocksOnValues), Ticket: t.ID, Field: "blocks_on"}
	}
	t.BlocksOn = m.BlocksOn
	return nil
}

type SetPriority struct{ Priority string }

func (m SetPriority) apply(t *Ticket, env mutEnv) error {
	if !ValidPriority(m.Priority) {
		return &Error{Code: CodeInvalidField, Message: fmt.Sprintf("%q is not one of %v", m.Priority, Priorities), Ticket: t.ID, Field: "priority"}
	}
	t.Priority = m.Priority
	return nil
}

// SetMilestone sets or clears the milestone. A nil Milestone clears it, which
// is why this carries a pointer and the others do not.
type SetMilestone struct{ Milestone *string }

func (m SetMilestone) apply(t *Ticket, env mutEnv) error {
	t.Milestone = m.Milestone
	return nil
}

// SetDueOn sets or clears the deadline. Nil and the empty string both clear it.
//
// The value has to be a YYYY-MM-DD date. An RFC3339 instant is refused rather
// than truncated to its date, per plan 5.1: truncating throws away a
// distinction the author can be seen making, and a reader could not then tell
// an expanded date from one somebody chose.
type SetDueOn struct{ DueOn *string }

func (m SetDueOn) apply(t *Ticket, env mutEnv) error {
	if m.DueOn == nil || *m.DueOn == "" {
		t.DueOn = nil
		return nil
	}
	if !ValidDueOn(*m.DueOn) {
		return &Error{
			Code:    CodeInvalidField,
			Message: fmt.Sprintf("%q is not a YYYY-MM-DD date", *m.DueOn),
			Ticket:  t.ID,
			Field:   "due_on",
		}
	}
	t.DueOn = m.DueOn
	return nil
}

// SetParent sets or clears the parent. The parent must exist.
type SetParent struct{ Parent *string }

func (m SetParent) apply(t *Ticket, env mutEnv) error {
	if m.Parent == nil || *m.Parent == "" {
		t.Parent = nil
		return nil
	}
	if *m.Parent == t.ID {
		return &Error{Code: CodeParentCycle, Message: "a ticket cannot be its own parent", Ticket: t.ID, Field: "parent"}
	}
	if !env.exists(*m.Parent) {
		return &Error{Code: CodeInvalidField, Message: "no ticket " + *m.Parent + " in this store", Ticket: t.ID, Field: "parent"}
	}
	parent := *m.Parent
	t.Parent = &parent
	return nil
}

// AddLabel and RemoveLabel edit the label set. A label outside the config
// allowlist is accepted here and warned about by check, because the allowlist
// is advisory.
type AddLabel struct{ Label string }

func (m AddLabel) apply(t *Ticket, env mutEnv) error {
	if m.Label == "" {
		return &Error{Code: CodeInvalidField, Message: "an empty label", Ticket: t.ID, Field: "labels"}
	}
	t.Labels = addUnique(t.Labels, m.Label)
	return nil
}

type RemoveLabel struct{ Label string }

func (m RemoveLabel) apply(t *Ticket, env mutEnv) error {
	t.Labels = remove(t.Labels, m.Label)
	return nil
}

// Assign and Unassign edit the assignee set.
type Assign struct{ Actor string }

func (m Assign) apply(t *Ticket, env mutEnv) error {
	if m.Actor == "" {
		return &Error{Code: CodeInvalidField, Message: "an empty assignee", Ticket: t.ID, Field: "assignees"}
	}
	t.Assignees = addUnique(t.Assignees, m.Actor)
	return nil
}

type Unassign struct{ Actor string }

func (m Unassign) apply(t *Ticket, env mutEnv) error {
	t.Assignees = remove(t.Assignees, m.Actor)
	return nil
}

// AddDependency records that this ticket waits on another.
type AddDependency struct{ ID string }

func (m AddDependency) apply(t *Ticket, env mutEnv) error {
	if m.ID == t.ID {
		return &Error{Code: CodeDependencyCycle, Message: "a ticket cannot depend on itself", Ticket: t.ID, Field: "dependencies"}
	}
	if !env.exists(m.ID) {
		return &Error{Code: CodeDependencyMissing, Message: "no ticket " + m.ID + " in this store", Ticket: t.ID, Field: "dependencies"}
	}
	t.Dependencies = addUnique(t.Dependencies, m.ID)
	return nil
}

type RemoveDependency struct{ ID string }

func (m RemoveDependency) apply(t *Ticket, env mutEnv) error {
	t.Dependencies = remove(t.Dependencies, m.ID)
	return nil
}

// AddReference records a typed identifier and an optional repository-relative
// path. Adding a reference that is already there replaces its path, so link is
// idempotent.
type AddReference struct {
	Ref  string
	Path *string
}

func (m AddReference) apply(t *Ticket, env mutEnv) error {
	if m.Ref == "" {
		return &Error{Code: CodeInvalidField, Message: "a reference needs a ref", Ticket: t.ID, Field: "references"}
	}
	for i := range t.References {
		if t.References[i].Ref == m.Ref {
			t.References[i].Path = m.Path
			return nil
		}
	}
	t.References = append(t.References, Reference{Ref: m.Ref, Path: m.Path})
	return nil
}

type RemoveReference struct{ Ref string }

func (m RemoveReference) apply(t *Ticket, env mutEnv) error {
	out := t.References[:0]
	for _, r := range t.References {
		if r.Ref != m.Ref {
			out = append(out, r)
		}
	}
	t.References = out
	return nil
}

// ClaimTicket records that an actor is working this ticket. A claim is
// metadata and not a status, per plan 6.4.
//
// Claiming a ticket the same actor already holds renews it rather than
// replacing it. claimed_at survives, and so does an expiry that nothing else
// supplied. See the renewal rules in plan 6.4.
type ClaimTicket struct {
	Branch   string
	Worktree string
	Commit   string
	// ExpiresIn overrides the store default. Zero means the store default,
	// and there is no default expiry, so a claim usually does not expire.
	// Zero on a renewal keeps the expiry the live claim already carried.
	ExpiresIn time.Duration
	// Force takes a live claim from another actor and records the displaced
	// claim in Notes, because taking work from another agent should leave a
	// trace.
	Force bool
}

func (m ClaimTicket) apply(t *Ticket, env mutEnv) error {
	switch t.Status {
	case StatusReady, StatusInProgress, StatusBlocked, StatusReview:
	default:
		return &Error{
			Code:    CodeValidationFailed,
			Message: "a ticket in " + t.Status + " cannot be claimed",
			Ticket:  t.ID, Field: "claim",
		}
	}

	held := t.Claim
	if held != nil && held.Actor != env.actor.ID && !held.Expired(env.now) {
		if !m.Force {
			return &Error{
				Code:    CodeClaimConflict,
				Message: "already claimed by " + held.Actor,
				Ticket:  t.ID, Field: "claim",
				Details: map[string]string{"actor": held.Actor},
			}
		}
		appendNote(&t.Body, env, fmt.Sprintf("claim taken from %s by %s", held.Actor, env.actor.ID))
	}

	// Plan 6.4: a re-claim by the actor already named on the claim renews it.
	renewal := held != nil && held.Actor == env.actor.ID

	claim := &Claim{Actor: env.actor.ID}
	// Branch, worktree, and commit describe the claim being recorded now, so a
	// renewal from somewhere else updates them.
	claim.Branch = optional(m.Branch)
	claim.Worktree = optional(m.Worktree)
	claim.Commit = optional(m.Commit)

	// claimed_at is the only record of when the work started, and renewing is
	// not restarting.
	claimedAt := Now(env.now)
	if renewal && held.ClaimedAt != nil {
		claimedAt = *held.ClaimedAt
	}
	claim.ClaimedAt = &claimedAt

	expiry := m.ExpiresIn
	if expiry == 0 && env.cfg.Defaults.ClaimExpiry != nil {
		expiry = env.cfg.Defaults.ClaimExpiry.Duration()
	}
	switch {
	case expiry > 0:
		expiresAt := Now(env.now.Add(expiry))
		claim.ExpiresAt = &expiresAt
	case renewal && !held.Expired(env.now):
		// Nothing supplied an expiry, so keep the bound the live claim already
		// carried. Clearing it would widen a bounded claim into an unbounded
		// one on the routine gesture for staying alive.
		claim.ExpiresAt = held.ExpiresAt
	}

	t.Claim = claim
	return nil
}

// ReleaseClaim drops the claim. Releasing an unclaimed ticket succeeds and
// changes nothing but updated_at, because the caller's intent is already true.
type ReleaseClaim struct{}

func (m ReleaseClaim) apply(t *Ticket, env mutEnv) error {
	t.Claim = nil
	return nil
}

// ArchiveTicket sets the status to archived, records the archive block
// including from_status, and lets the write path move the file.
//
// from_status is what keeps archiving from silently blocking dependents: a
// dependency is satisfied by a ticket archived out of done, and by nothing
// else.
//
// A reason lands in the archive block and in Notes, per plan 6.3, for the same
// reason a status reason does: UnarchiveTicket deletes the block, and without
// the note nothing would say why the ticket was ever closed out.
type ArchiveTicket struct{ Reason string }

func (m ArchiveTicket) apply(t *Ticket, env mutEnv) error {
	if t.Archived() {
		return &Error{Code: CodeInvalidTransition, Message: "already archived", Ticket: t.ID, Field: "status"}
	}
	from := t.Status
	archivedAt := Now(env.now)
	t.Archive = &Archive{
		ArchivedAt: &archivedAt,
		FromStatus: &from,
		Reason:     optional(m.Reason),
	}
	t.Status = StatusArchived
	t.StatusReason = nil
	if m.Reason != "" {
		appendNote(&t.Body, env, fmt.Sprintf("archived from %s: %s", from, m.Reason))
	}
	return nil
}

// UnarchiveTicket restores a ticket to ready and moves the file back.
type UnarchiveTicket struct{}

func (m UnarchiveTicket) apply(t *Ticket, env mutEnv) error {
	if !t.Archived() {
		return &Error{Code: CodeInvalidTransition, Message: "not archived", Ticket: t.ID, Field: "status"}
	}
	// 5.1 defines the archive block as what a ticket carries while it is
	// archived, so it goes when the archive does. The Notes entry is what keeps
	// the history.
	var was string
	if t.Archive != nil && t.Archive.FromStatus != nil {
		was = *t.Archive.FromStatus
	}
	t.Archive = nil
	t.Status = StatusReady
	if was != "" {
		appendNote(&t.Body, env, "unarchived to ready, archived from "+was)
	} else {
		appendNote(&t.Body, env, "unarchived to ready")
	}
	return nil
}

// AppendNote adds an entry to Notes, and AppendComment to Comments. Both stamp
// the actor and the time, because a bare line of prose in a shared file does
// not say who wrote it.
type AppendNote struct{ Text string }

func (m AppendNote) apply(t *Ticket, env mutEnv) error {
	if strings.TrimSpace(m.Text) == "" {
		return &Error{Code: CodeInvalidField, Message: "an empty note", Ticket: t.ID, Field: "notes"}
	}
	appendNote(&t.Body, env, m.Text)
	return nil
}

type AppendComment struct{ Text string }

func (m AppendComment) apply(t *Ticket, env mutEnv) error {
	if strings.TrimSpace(m.Text) == "" {
		return &Error{Code: CodeInvalidField, Message: "an empty comment", Ticket: t.ID, Field: "comments"}
	}
	t.Body.Comments = appendEntry(t.Body.Comments, env, m.Text)
	return nil
}

// SetSummary replaces the summary, which is one statement of where the ticket
// landed rather than a log.
type SetSummary struct{ Text string }

func (m SetSummary) apply(t *Ticket, env mutEnv) error {
	t.Body.Summary = m.Text
	return nil
}

// SetDescription replaces the description.
type SetDescription struct{ Text string }

func (m SetDescription) apply(t *Ticket, env mutEnv) error {
	t.Body.Description = m.Text
	return nil
}

// SetImplementationPlan replaces the plan. It replaces rather than appends for
// the reason SetSummary does: a plan is one statement of how the work will go,
// and a log of what happened on the way is what Notes already is. An agent that
// rewrites its plan halfway through means the first one to be gone, not to be
// read alongside the second.
type SetImplementationPlan struct{ Text string }

func (m SetImplementationPlan) apply(t *Ticket, env mutEnv) error {
	t.Body.ImplementationPlan = m.Text
	return nil
}

// ChecklistSection names one of the two checkbox sections.
type ChecklistSection string

const (
	AcceptanceCriteria ChecklistSection = "Acceptance criteria"
	DefinitionOfDone   ChecklistSection = "Definition of done"
)

func (c ChecklistSection) field(b *Body) (*string, error) {
	switch c {
	case AcceptanceCriteria:
		return &b.AcceptanceCriteria, nil
	case DefinitionOfDone:
		return &b.DefinitionOfDone, nil
	default:
		return nil, codedError(CodeInvalidField, "%q is not a checklist section", string(c))
	}
}

// AddChecklistItem appends an unchecked item to a checklist section.
type AddChecklistItem struct {
	Section ChecklistSection
	Text    string
}

func (m AddChecklistItem) apply(t *Ticket, env mutEnv) error {
	target, err := m.Section.field(&t.Body)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(m.Text)
	if text == "" {
		return &Error{Code: CodeInvalidField, Message: "an empty checklist item", Ticket: t.ID, Field: string(m.Section)}
	}
	line := uncheckedItem(text)
	if *target == "" {
		*target = line
		return nil
	}
	*target += "\n" + line
	return nil
}

// uncheckedItem renders one new checklist line. Seeding a section at create and
// building one by repeated adds go through here, so the two cannot drift into
// producing different bytes for the same list.
func uncheckedItem(text string) string { return "- [ ] " + text }

// checklistSection renders seed items as a section body. An empty item is the
// same mistake AddChecklistItem refuses, and it is refused here for the same
// reason: a blank checkbox is a criterion nobody can meet or check off.
func checklistSection(items []string) (string, error) {
	lines := make([]string, 0, len(items))
	for _, item := range items {
		text := strings.TrimSpace(item)
		if text == "" {
			return "", codedError(CodeInvalidField, "an empty checklist item")
		}
		lines = append(lines, uncheckedItem(text))
	}
	return strings.Join(lines, "\n"), nil
}

// SetChecklistItem checks or unchecks item Index, counting from one in the
// order the items appear.
type SetChecklistItem struct {
	Section ChecklistSection
	Index   int
	Checked bool
}

func (m SetChecklistItem) apply(t *Ticket, env mutEnv) error {
	target, err := m.Section.field(&t.Body)
	if err != nil {
		return err
	}
	updated, err := setChecklistItem(*target, m.Index, m.Checked)
	if err != nil {
		if e, ok := err.(*Error); ok {
			e.Ticket = t.ID
			e.Field = string(m.Section)
		}
		return err
	}
	*target = updated
	return nil
}

// RemoveChecklistItem deletes item Index, counting from one in the order the
// items appear.
//
// Removing an item renumbers every item after it. A caller removing more than
// one therefore has to apply them highest first, or the second index means a
// different item than the one it read. runChecklist does that ordering.
type RemoveChecklistItem struct {
	Section ChecklistSection
	Index   int
}

func (m RemoveChecklistItem) apply(t *Ticket, env mutEnv) error {
	target, err := m.Section.field(&t.Body)
	if err != nil {
		return err
	}
	updated, err := removeChecklistItem(*target, m.Index)
	if err != nil {
		if e, ok := err.(*Error); ok {
			e.Ticket = t.ID
			e.Field = string(m.Section)
		}
		return err
	}
	*target = updated
	return nil
}

// checkboxLine matches a checklist item and captures its marker and its text.
var checkboxLine = regexp.MustCompile(`^(\s*- \[)([ xX])(\] )(.*)$`)

// ChecklistItem is one checkbox in a section.
type ChecklistItem struct {
	Checked bool
	Text    string
}

// Checklist reads the items of a section. It is a view over the raw text, so an
// item a person typed by hand counts the same as one this tool wrote.
func Checklist(text string) []ChecklistItem {
	var out []ChecklistItem
	for _, line := range strings.Split(text, "\n") {
		m := checkboxLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		out = append(out, ChecklistItem{Checked: m[2] != " ", Text: m[4]})
	}
	return out
}

// entryHeader matches the stamp appendEntry writes: the actor in bold, then the
// instant. A line a person wrote by hand does not match, and the text under it
// is kept as an entry carrying neither.
var entryHeader = regexp.MustCompile(`^\*\*(.+?)\*\* at (\S+)\s*$`)

// Entry is one stamped item in a log section. Notes and Comments are both
// written by appendEntry, so both read back with Entries.
type Entry struct {
	// Index counts entries from one, in the order they appear.
	Index int

	// Actor and At come from the stamp, and are empty for an entry somebody
	// wrote by hand without one. The format is meant to be hand-edited, so that
	// is an ordinary case rather than a broken file, and dropping such an entry
	// would lose a comment a person actually left.
	Actor string
	At    string

	Text string
}

// Entries reads a log section back as records.
//
// It is a view over the raw text and runs one way only, like Checklist: the
// section is the document and this is a reading of it, so the two cannot
// disagree and there is no question which one wins.
//
// An entry runs from its stamp to the next stamp. A blank line inside one does
// not end it, because a comment may have several paragraphs. The consequence is
// that prose somebody appends by hand below a stamped entry joins that entry
// and reads as its author's. Splitting on the blank line instead would not help:
// the fragment still sits under that stamp and would still carry that actor,
// and every multi-paragraph comment would come apart. This way the reading
// matches what a person sees in the file, which is the same conclusion they
// would draw from the Markdown.
func Entries(text string) []Entry {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	var out []Entry
	var cur *Entry
	var body []string
	flush := func() {
		if cur == nil {
			return
		}
		cur.Text = trimBlankLines(body)
		cur.Index = len(out) + 1
		out = append(out, *cur)
		cur, body = nil, nil
	}

	for _, line := range strings.Split(text, "\n") {
		if m := entryHeader.FindStringSubmatch(line); m != nil {
			flush()
			cur = &Entry{Actor: m[1], At: m[2]}
			continue
		}
		if cur == nil {
			// Prose ahead of any stamp is its own entry. A blank line is not,
			// or the separator between two entries would start a third.
			if strings.TrimSpace(line) == "" {
				continue
			}
			cur = &Entry{}
		}
		body = append(body, line)
	}
	flush()
	return out
}

func setChecklistItem(text string, index int, checked bool) (string, error) {
	if index < 1 {
		return "", codedError(CodeInvalidField, "checklist items count from 1")
	}
	lines := strings.Split(text, "\n")
	seen := 0
	for i, line := range lines {
		m := checkboxLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		seen++
		if seen != index {
			continue
		}
		mark := " "
		if checked {
			mark = "x"
		}
		lines[i] = m[1] + mark + m[3] + m[4]
		return strings.Join(lines, "\n"), nil
	}
	return "", codedError(CodeInvalidField, "there is no item %d; the section has %d", index, seen)
}

func removeChecklistItem(text string, index int) (string, error) {
	if index < 1 {
		return "", codedError(CodeInvalidField, "checklist items count from 1")
	}
	lines := strings.Split(text, "\n")
	seen := 0
	for i, line := range lines {
		if checkboxLine.FindStringSubmatch(line) == nil {
			continue
		}
		seen++
		if seen != index {
			continue
		}
		// The full slice expression forces a copy, so this does not write
		// through into the caller's backing array.
		kept := append(lines[:i:i], lines[i+1:]...)
		// Dropping the last item under a section that also holds prose would
		// otherwise leave the blank line that separated them, and the parser
		// strips those, so the bytes would not survive a round trip.
		return trimBlankLines(kept), nil
	}
	return "", codedError(CodeInvalidField, "there is no item %d; the section has %d", index, seen)
}

// appendNote adds a stamped entry to Notes.
func appendNote(b *Body, env mutEnv, text string) {
	b.Notes = appendEntry(b.Notes, env, text)
}

// appendEntry adds a stamped entry to a section, separated from what is already
// there by a blank line.
func appendEntry(existing string, env mutEnv, text string) string {
	entry := fmt.Sprintf("**%s** at %s\n\n%s", env.actor.ID, Now(env.now).String(), strings.TrimSpace(text))
	if strings.TrimSpace(existing) == "" {
		return entry
	}
	return strings.TrimRight(existing, "\n") + "\n\n" + entry
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func addUnique(list []string, v string) []string {
	for _, x := range list {
		if x == v {
			return list
		}
	}
	return append(list, v)
}

func remove(list []string, v string) []string {
	out := make([]string, 0, len(list))
	for _, x := range list {
		if x != v {
			out = append(out, x)
		}
	}
	return out
}
