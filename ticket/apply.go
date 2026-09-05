package ticket

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ApplyOptions carries the precondition and the actor for one mutation.
type ApplyOptions struct {
	// IfRevision is the revision the caller last read. Empty means no
	// precondition, and the last write wins under the store lock.
	//
	// It is optional here so that a person typing one command at a terminal is
	// not forced into a read-then-write dance where no concurrency exists.
	// Terva's tool schema marks it required, because agents read before they
	// write anyway and multi-agent is where the races happen.
	IfRevision string
	// Actor is who is making the change. It is recorded in updated_by. When it
	// is empty the first actor in config.yml is used.
	Actor Actor
}

// Result is what a mutation produced.
type Result struct {
	Ticket *Ticket
	// PathsChanged are absolute. A caller that shows them to a person makes
	// them relative to the repository root itself, because the library does not
	// assume the store is inside one.
	PathsChanged []string
}

// Apply changes one ticket under the store lock, per plan section 7. The write
// path is: acquire, read, verify the precondition, write a temporary file in
// the same directory, fsync, rename over the target, release. A failure at any
// step leaves the original file untouched.
func (s *Store) Apply(ctx context.Context, ref string, m Mutation, o ApplyOptions) (*Result, error) {
	if m == nil {
		return nil, codedError(CodeValidationFailed, "no mutation given")
	}
	actor, err := s.resolveActor(o.Actor)
	if err != nil {
		return nil, err
	}

	lock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer lock.release()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	index, broken, err := s.index()
	if err != nil {
		return nil, err
	}
	id, err := ResolveRef(ref, mergeIDs(index.ids(), broken.ids()))
	if err != nil {
		return nil, err
	}
	if e, ok := broken[id]; ok {
		return nil, e
	}
	path := index[id]

	// The precondition is checked after the lock is held, so nothing can slip
	// in between the check and the write.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &Error{Code: CodeTicketNotFound, Message: err.Error(), Ticket: id, Err: err}
	}
	actualRevision := Revision(data)
	if o.IfRevision != "" && o.IfRevision != actualRevision {
		e := &Error{
			Code:    CodeStaleRevision,
			Message: "ticket changed since it was read",
			Ticket:  id,
			Details: map[string]string{"expected": o.IfRevision, "actual": actualRevision},
		}
		// This returns before the parse below, so the title is taken here from
		// bytes already in hand. Losing a race is the failure an agent hits most,
		// and it is worth naming the ticket it lost. A file that no longer parses
		// still reports the staleness, without a title.
		if parsed, perr := Parse(data); perr == nil {
			e.Title = parsed.Title
		}
		return nil, e
	}

	t, err := Parse(data)
	if err != nil {
		return nil, err
	}
	t.Path = path
	t.Revision = actualRevision

	env := mutEnv{
		now:    s.now(),
		actor:  actor,
		cfg:    s.config,
		exists: func(other string) bool { _, ok := index[other]; return ok },
	}
	if err := m.apply(t, env); err != nil {
		// The one place holding both the failure and the parsed ticket, so a
		// mutation never has to carry the title itself.
		return nil, withTitle(err, t.Title)
	}

	// Every mutation records who touched the ticket and when, so the diff says
	// so even when nothing else changed.
	t.UpdatedAt = Now(env.now)
	t.UpdatedBy = &Actor{ID: actor.ID, Name: actor.Name}

	return s.writeTicket(t, path)
}

// CreateOptions describes a new ticket. Type and priority fall back to the
// store defaults.
type CreateOptions struct {
	Title        string
	Type         string
	Priority     string
	Labels       []string
	Assignees    []string
	Milestone    *string
	Parent       *string
	Dependencies []string
	// BlocksOn names the edges that gate the ticket beyond its dependencies.
	// Empty means BlocksOnNone, which is what every ticket did before the field
	// existed.
	BlocksOn string
	// DueOn is a deadline as a YYYY-MM-DD date. Nil and the empty string both
	// mean no deadline, and an instant is refused rather than truncated.
	DueOn       *string
	Description string
	// ImplementationPlan seeds the plan section. Writing it after the fact is
	// SetImplementationPlan.
	ImplementationPlan string
	// AcceptanceCriteria and DefinitionOfDone seed the two checkbox sections,
	// unchecked and in the order given. Seeding them here rather than through
	// repeated AddChecklistItem calls keeps a filed ticket to one write and one
	// revision.
	AcceptanceCriteria []string
	DefinitionOfDone   []string
	Actor              Actor
	// Entropy is the source for the ID's random half. Nil means crypto/rand.
	Entropy io.Reader
	// Status files the ticket directly as done or archived, per plan 6.2.1,
	// which is how a backport records finished or abandoned work without
	// walking the lifecycle. Empty means draft. Every other value is refused,
	// because promotion out of draft is a human call and a create that lands
	// in ready would quietly bypass that gate.
	Status string
	// Created backdates the ticket, per plan 6.2.1: the ULID takes its time
	// part from it, so backported history sorts chronologically, and
	// created_at and updated_at agree with it. The zero value means now. An
	// instant after now is refused.
	Created time.Time
	// Reason is accepted with Status archived only and lands in the archive
	// block and Notes, the same two places ArchiveTicket writes. With any
	// other status it is refused, because there is nothing for it to mean.
	Reason string
	// Template names a file in the store's templates directory to seed
	// from, per plan 4.2. Every explicit field here wins over the template,
	// because a template is a starting point and not an argument. A name
	// that resolves to no file refuses the create.
	Template string
}

// Create writes a new ticket and returns it.
func (s *Store) Create(ctx context.Context, o CreateOptions) (*Result, error) {
	actor, err := s.resolveActor(o.Actor)
	if err != nil {
		return nil, err
	}
	if o.Title == "" {
		return nil, &Error{Code: CodeInvalidField, Message: "a ticket needs a title", Field: "title"}
	}
	// The ID does not exist yet, so the refusal names no ticket. There is
	// nothing to look up and the caller is holding the title it just typed.
	if err := checkTitleLength(o.Title, ""); err != nil {
		return nil, err
	}
	// The template merges before the defaults resolve, so the precedence
	// reads option, then template, then store default, per plan 4.2. The
	// template's own values pass through the same validation as typed ones,
	// so a template carrying a type the schema does not know refuses here
	// like any other bad field.
	var tpl *Template
	if o.Template != "" {
		loaded, err := s.loadTemplate(o.Template)
		if err != nil {
			return nil, err
		}
		tpl = loaded
		if o.Type == "" {
			o.Type = tpl.Type
		}
		if o.Priority == "" {
			o.Priority = tpl.Priority
		}
		if len(o.Labels) == 0 {
			o.Labels = tpl.Labels
		}
		if len(o.Assignees) == 0 {
			o.Assignees = tpl.Assignees
		}
		if o.Milestone == nil {
			o.Milestone = tpl.Milestone
		}
		if o.Description == "" {
			o.Description = tpl.Description
		}
		if o.ImplementationPlan == "" {
			o.ImplementationPlan = tpl.ImplementationPlan
		}
	}
	kind := o.Type
	if kind == "" {
		kind = s.config.Defaults.Type
	}
	priority := o.Priority
	if priority == "" {
		priority = s.config.Defaults.Priority
	}
	if !ValidType(kind) {
		return nil, &Error{Code: CodeInvalidField, Message: fmt.Sprintf("%q is not one of %v", kind, Types), Field: "type"}
	}
	if !ValidPriority(priority) {
		return nil, &Error{Code: CodeInvalidField, Message: fmt.Sprintf("%q is not one of %v", priority, Priorities), Field: "priority"}
	}
	blocksOn := o.BlocksOn
	if blocksOn == "" {
		blocksOn = BlocksOnNone
	}
	if !ValidBlocksOn(blocksOn) {
		return nil, &Error{Code: CodeInvalidField, Message: fmt.Sprintf("%q is not one of %v", blocksOn, BlocksOnValues), Field: "blocks_on"}
	}
	// Refused rather than truncated to its date, per plan 5.1, so an RFC3339
	// instant never reaches the store. Checked here with the other fields, so a
	// bad date costs no ID and writes no file.
	dueOn := o.DueOn
	if dueOn != nil && *dueOn == "" {
		dueOn = nil
	}
	if dueOn != nil && !ValidDueOn(*dueOn) {
		return nil, &Error{Code: CodeInvalidField, Message: fmt.Sprintf("%q is not a YYYY-MM-DD date", *dueOn), Field: "due_on"}
	}
	status := o.Status
	if status == "" {
		status = StatusDraft
	}
	switch status {
	case StatusDraft, StatusDone, StatusArchived:
	default:
		return nil, &Error{
			Code:    CodeInvalidField,
			Message: fmt.Sprintf("create files a draft; --status accepts done and archived only, per plan 6.2.1, not %q", status),
			Field:   "status",
		}
	}
	if o.Reason != "" && status != StatusArchived {
		return nil, &Error{
			Code:    CodeInvalidField,
			Message: "a create reason belongs to --status archived; nothing else has a place for it",
			Field:   "status_reason",
		}
	}

	lock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer lock.release()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	index, broken, err := s.index()
	if err != nil {
		return nil, err
	}
	for _, dep := range o.Dependencies {
		if _, ok := index[dep]; !ok {
			return nil, &Error{Code: CodeDependencyMissing, Message: "no ticket " + dep + " in this store", Field: "dependencies"}
		}
	}
	if o.Parent != nil && *o.Parent != "" {
		if _, ok := index[*o.Parent]; !ok {
			return nil, &Error{Code: CodeInvalidField, Message: "no ticket " + *o.Parent + " in this store", Field: "parent"}
		}
	}

	now := s.now()
	if !o.Created.IsZero() {
		if o.Created.After(now) {
			return nil, &Error{
				Code:    CodeInvalidField,
				Message: fmt.Sprintf("--created %s is in the future; a backport is about the past", o.Created.UTC().Format(time.RFC3339)),
				Field:   "created_at",
			}
		}
		now = o.Created
	}
	id, err := NewID(now, o.Entropy)
	if err != nil {
		return nil, &Error{Code: CodeValidationFailed, Message: err.Error(), Err: err}
	}
	_, taken := index[id]
	_, takenByBroken := broken[id]
	if taken || takenByBroken {
		return nil, &Error{Code: CodeValidationFailed, Message: "generated an ID that already exists: " + id}
	}

	// Plan 12.5: a new ticket is written at the store's declared schema, not at
	// this binary's maximum. Otherwise a newer binary working in an older store
	// drifts it upward with no migration run, and the files a colleague's older
	// binary cannot read are the newest ones. A store that declares nothing
	// falls back to this reader's version, which is what Init writes anyway.
	schema := s.config.Schema
	if schema <= 0 {
		schema = SchemaVersion
	}

	// Rendered before the ticket exists, so an empty criterion refuses the
	// whole create rather than leaving a filed ticket with half its checklist.
	ac, err := checklistSection(o.AcceptanceCriteria)
	if err != nil {
		return nil, err
	}
	dod, err := checklistSection(o.DefinitionOfDone)
	if err != nil {
		return nil, err
	}
	// The checklists merge at the section level: a template carries them as
	// rendered "- [ ]" lines, and explicit --ac and --dod items win
	// wholesale, the same per-field rule as everything else.
	if tpl != nil {
		if ac == "" {
			ac = tpl.AcceptanceCriteria
		}
		if dod == "" {
			dod = tpl.DefinitionOfDone
		}
	}

	t := &Ticket{
		Schema:       schema,
		ID:           id,
		Title:        o.Title,
		Type:         kind,
		Status:       status,
		Priority:     priority,
		DueOn:        dueOn,
		Labels:       append([]string{}, o.Labels...),
		Assignees:    append([]string{}, o.Assignees...),
		Milestone:    o.Milestone,
		Parent:       o.Parent,
		Dependencies: append([]string{}, o.Dependencies...),
		BlocksOn:     blocksOn,
		CreatedAt:    Now(now),
		UpdatedAt:    Now(now),
		CreatedBy:    &Actor{ID: actor.ID, Name: actor.Name},
		UpdatedBy:    &Actor{ID: actor.ID, Name: actor.Name},
		// Seeded verbatim. writeTicket normalizes the body on the way to disk,
		// so a caller's padding is settled in one place rather than by every
		// writer remembering to trim.
		Body: Body{
			Description:        o.Description,
			AcceptanceCriteria: ac,
			DefinitionOfDone:   dod,
			ImplementationPlan: o.ImplementationPlan,
		},
	}
	// A ticket created as archived carries the block ArchiveTicket would have
	// written, per plan 6.2.1. from_status is draft, because the ticket never
	// lived in this store's working set, and per 6.3 that is what keeps an
	// abandoned backport from satisfying a dependency.
	if status == StatusArchived {
		archivedAt := Now(now)
		from := StatusDraft
		t.Archive = &Archive{
			ArchivedAt: &archivedAt,
			FromStatus: &from,
			Reason:     optional(o.Reason),
		}
		if o.Reason != "" {
			appendNote(&t.Body, mutEnv{now: now, actor: actor}, "created archived: "+o.Reason)
		}
	}
	return s.writeTicket(t, "")
}

// writeTicket renders the ticket and puts it where its status says it belongs.
// oldPath is empty for a new ticket.
//
// The destination follows the status rather than the current directory, because
// the status is authoritative when the two disagree, per plan 6.3. A write to a
// ticket sitting in the wrong directory therefore also puts it back.
func (s *Store) writeTicket(t *Ticket, oldPath string) (*Result, error) {
	target := filepath.Join(s.StatusDir(t.Status), t.ID+".md")

	// Every write funnels through here, so this is the one place the body has
	// to be put in the shape parse returns. Render is a faithful echo of the
	// struct, per plan 5.3, and normalizing there instead would leave the
	// Ticket this returns disagreeing with the bytes on disk. Callers read that
	// struct, and the CLI serializes it, so the divergence would be invisible
	// where the byte instability at least shows up in a diff.
	t.Body.normalize()

	data := Render(t)
	if err := writeFileAtomic(target, data); err != nil {
		return nil, err
	}
	changed := []string{target}
	if oldPath != "" && oldPath != target {
		if err := os.Remove(oldPath); err != nil {
			return nil, &Error{Code: CodeValidationFailed, Message: err.Error(), Err: err}
		}
		changed = append(changed, oldPath)
	}

	t.Path = target
	t.Revision = Revision(data)
	return &Result{Ticket: t, PathsChanged: changed}, nil
}

// writeFileAtomic writes through a temporary file in the same directory, syncs
// it, and renames it over the target. A rename within a directory is atomic, so
// a reader sees either the old file or the new one and never a half-written
// one.
func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return &Error{Code: CodeValidationFailed, Message: err.Error(), Err: err}
	}
	tmp, err := os.CreateTemp(dir, ".git-ticket-*.tmp")
	if err != nil {
		return &Error{Code: CodeValidationFailed, Message: err.Error(), Err: err}
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return &Error{Code: CodeValidationFailed, Message: err.Error(), Err: err}
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return &Error{Code: CodeValidationFailed, Message: err.Error(), Err: err}
	}
	if err := tmp.Close(); err != nil {
		return &Error{Code: CodeValidationFailed, Message: err.Error(), Err: err}
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return &Error{Code: CodeValidationFailed, Message: err.Error(), Err: err}
	}
	if err := os.Rename(tmpName, path); err != nil {
		return &Error{Code: CodeValidationFailed, Message: err.Error(), Err: err}
	}
	return nil
}

// ticketIndex maps a ticket ID to the file holding it.
type ticketIndex map[string]string

func (ix ticketIndex) ids() []string {
	out := make([]string, 0, len(ix))
	for id := range ix {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// unreadableIndex maps a ticket ID to the error that stopped its file parsing.
type unreadableIndex map[string]*Error

func (ix unreadableIndex) ids() []string {
	out := make([]string, 0, len(ix))
	for id := range ix {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// mergeIDs returns the sorted union of two ID lists.
func mergeIDs(a, b []string) []string {
	if len(b) == 0 {
		return a
	}
	out := append(append(make([]string, 0, len(a)+len(b)), a...), b...)
	sort.Strings(out)
	return out
}

// index reads every ticket file to learn which IDs exist and where they live.
// A file that does not parse cannot be mutated, so it goes into a second map
// rather than being dropped. Resolution still has to see it, or a full ID
// answers ticket_not_found for a file on disk and a shared prefix resolves
// silently to a readable neighbour. Plan section 8 holds the rule.
func (s *Store) index() (ticketIndex, unreadableIndex, error) {
	files, err := s.load()
	if err != nil {
		return nil, nil, err
	}
	ix := make(ticketIndex, len(files))
	broken := make(unreadableIndex)
	for _, f := range files {
		if f.Ticket == nil || f.Ticket.ID == "" {
			if id := f.id(); id != "" {
				broken[id] = unreadable(id, f.Err)
			}
			continue
		}
		if _, seen := ix[f.Ticket.ID]; seen {
			// A duplicate ID is a check error. Resolving to whichever file was
			// read first is arbitrary, so refuse instead of guessing.
			return nil, nil, &Error{
				Code:    CodeValidationFailed,
				Message: "id " + f.Ticket.ID + " appears in more than one file; run check",
				Ticket:  f.Ticket.ID,
			}
		}
		ix[f.Ticket.ID] = f.Path
	}
	return ix, broken, nil
}

// resolveActor picks who is making a change: the caller's actor, or the first
// one in config.yml. A mutation with no actor is refused, because updated_by
// would then say nothing.
func (s *Store) resolveActor(a Actor) (Actor, error) {
	if a.ID != "" {
		return a, nil
	}
	if d, _, ok := s.config.DefaultActor(); ok {
		return d, nil
	}
	return Actor{}, &Error{
		Code:    CodeInvalidField,
		Message: "no actor given, and config.yml neither declares defaults.actor nor lists an actor",
		Field:   "actor",
	}
}

// Get reads one ticket by ID or unique prefix.
func (s *Store) Get(ctx context.Context, ref string) (*Ticket, error) {
	files, err := s.load()
	if err != nil {
		return nil, err
	}
	// A file that did not parse takes part in resolution too. Leaving it out
	// makes a full ID answer ticket_not_found for a file on disk, and makes a
	// shared prefix resolve silently to a readable neighbour, which is the
	// worse of the two. Plan section 8 holds the rule.
	ids := make([]string, 0, len(files))
	byID := make(map[string]file, len(files))
	for _, f := range files {
		id := f.id()
		if id == "" {
			continue
		}
		ids = append(ids, id)
		byID[id] = f
	}
	sort.Strings(ids)
	id, err := ResolveRef(ref, ids)
	if err != nil {
		return nil, err
	}
	if f := byID[id]; f.Ticket == nil {
		return nil, unreadable(id, f.Err)
	}
	return byID[id].Ticket, nil
}

// unreadable reports why a present ticket could not be read. load always
// records an error beside a file it could not parse, and the fallback is here
// so a nil can never turn back into silence.
func unreadable(id string, err *Error) *Error {
	if err != nil {
		return err
	}
	// Error() supplies the ID from Ticket, so the message does not repeat it.
	return &Error{Code: CodeParseError, Message: "could not be read", Ticket: id}
}

// mutEnv is what a mutation needs beyond the ticket itself.
type mutEnv struct {
	now    time.Time
	actor  Actor
	cfg    Config
	exists func(id string) bool
}
