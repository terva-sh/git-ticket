package ticket

import (
	"context"
	"fmt"
	"os"
	"strings"
)

func validMoveDestination(ref string) bool {
	ns, id, typed := splitRef(ref)
	return typed && strings.TrimSpace(ns) != "" && strings.TrimSpace(id) != "" &&
		ref == strings.TrimSpace(ref)
}

// MoveTo marks the original ticket with its foreign destination. Its status
// stays as it was; a marker can never stand in for a completed dependency.
type MoveTo struct {
	Ref    string
	Reason string
}

func (m MoveTo) apply(t *Ticket, env mutEnv) error {
	if env.cfg.Schema < 4 || t.Schema < 4 {
		return &Error{Code: CodeSchemaUnsupported, Ticket: t.ID, Field: "schema", Message: "move requires schema 4; run git ticket migrate first"}
	}
	if !validMoveDestination(m.Ref) {
		return &Error{Code: CodeInvalidField, Ticket: t.ID, Field: "moved_to", Message: "destination needs a nonempty namespace and identifier"}
	}
	if strings.TrimSpace(m.Reason) == "" {
		return &Error{Code: CodeInvalidField, Ticket: t.ID, Field: "reason", Message: "move requires a reason"}
	}
	previous := "none"
	if t.MovedTo != nil {
		previous = *t.MovedTo
	}
	ref := m.Ref
	t.MovedTo = &ref
	return (AppendNote{Text: fmt.Sprintf("Moved destination from %s to %s: %s", previous, ref, m.Reason)}).apply(t, env)
}

// ResolveMoveOptions requires the revision observed on the original ticket.
// The dependent may also carry its ordinary revision precondition.
type ResolveMoveOptions struct {
	From             string
	IfSourceRevision string
	IfRevision       string
	WaitOn           string
	Reason           string
	Actor            Actor
}

// ResolveMove removes one moved dependency after a human checks its foreign
// destination. Both tickets and both preconditions are read under one lock.
func (s *Store) ResolveMove(ctx context.Context, dependent string, o ResolveMoveOptions) (*Result, error) {
	if o.IfSourceRevision == "" || strings.TrimSpace(o.Reason) == "" || o.From == "" {
		return nil, &Error{Code: CodeInvalidField, Field: "if-source-revision", Message: "resolve-move requires --from, --if-source-revision, and --reason"}
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
	ids := mergeIDs(index.ids(), broken.ids())
	depID, err := s.resolveRef(dependent, ids)
	if err != nil {
		return nil, err
	}
	sourceID, err := s.resolveRef(o.From, ids)
	if err != nil {
		return nil, err
	}
	if e := broken[depID]; e != nil {
		return nil, e
	}
	if e := broken[sourceID]; e != nil {
		return nil, e
	}
	var waitID string
	if o.WaitOn != "" {
		waitID, err = s.resolveRef(o.WaitOn, ids)
		if err != nil {
			return nil, err
		}
		if e := broken[waitID]; e != nil {
			return nil, e
		}
		if waitID == sourceID {
			return nil, &Error{Code: CodeInvalidField, Field: "wait_on", Message: "replacement dependency must differ from the moved source"}
		}
	}
	read := func(id string) (*Ticket, error) {
		data, err := os.ReadFile(index[id])
		if err != nil {
			return nil, &Error{Code: CodeTicketNotFound, Ticket: id, Message: err.Error(), Err: err}
		}
		t, err := Parse(data)
		if err != nil {
			return nil, err
		}
		t.Revision, t.Path = Revision(data), index[id]
		return t, nil
	}
	source, err := read(sourceID)
	if err != nil {
		return nil, err
	}
	dep, err := read(depID)
	if err != nil {
		return nil, err
	}
	if source.Revision != o.IfSourceRevision {
		return nil, &Error{Code: CodeStaleRevision, Ticket: source.ID, Title: source.Title, Message: "source changed since it was read", Details: map[string]string{"expected": o.IfSourceRevision, "actual": source.Revision}}
	}
	if o.IfRevision != "" && dep.Revision != o.IfRevision {
		return nil, &Error{Code: CodeStaleRevision, Ticket: dep.ID, Title: dep.Title, Message: "dependent changed since it was read", Details: map[string]string{"expected": o.IfRevision, "actual": dep.Revision}}
	}
	if s.config.Schema < 4 || source.Schema < 4 || dep.Schema < 4 {
		return nil, &Error{Code: CodeSchemaUnsupported, Ticket: dep.ID, Title: dep.Title, Field: "schema", Message: "resolve-move requires schema 4; run git ticket migrate first"}
	}
	if source.MovedTo == nil || !validMoveDestination(*source.MovedTo) {
		return nil, &Error{Code: CodeInvalidField, Ticket: source.ID, Title: source.Title, Field: "moved_to", Message: "source has no valid moved destination"}
	}
	found := false
	for _, id := range dep.Dependencies {
		if id == sourceID {
			found = true
			break
		}
	}
	if !found {
		return nil, &Error{Code: CodeInvalidField, Ticket: dep.ID, Title: dep.Title, Field: "dependencies", Message: "dependent does not wait on the moved source"}
	}
	env := mutEnv{now: s.now(), actor: actor, cfg: s.config, exists: func(id string) bool { _, ok := index[id]; return ok }}
	ms := Mutations{RemoveDependency{ID: sourceID}}
	alreadyReferenced := false
	for _, ref := range dep.References {
		if ref.Ref == *source.MovedTo {
			alreadyReferenced = true
			break
		}
	}
	if !alreadyReferenced {
		ms = append(ms, AddReference{Ref: *source.MovedTo})
	}
	ms = append(ms, AppendNote{Text: fmt.Sprintf("Resolved moved dependency %s (%s) at %s: %s", source.ID, source.Title, *source.MovedTo, o.Reason)})
	if waitID != "" {
		ms = append(ms, AddDependency{ID: waitID})
	}
	if err := ms.apply(dep, env); err != nil {
		return nil, withTitle(err, dep.Title)
	}
	dep.UpdatedAt = Now(env.now)
	dep.UpdatedBy = &Actor{ID: actor.ID, Name: actor.Name}
	return s.writeTicket(dep, index[depID])
}
