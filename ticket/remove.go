package ticket

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
)

// RemoveOptions controls one removal, per plan section 9.1.
//
// There is no Actor. Every other write records who made it, but a removal
// leaves no file to record it in, and section 9.1 has remove write no history
// at all. An Actor field here would be accepted and then ignored, which is a
// worse answer than not offering one.
type RemoveOptions struct {
	// IfRevision refuses the removal when the ticket has changed since it was
	// read, exactly as it does for a mutation.
	IfRevision string
	// Force removes a ticket that either refusal would otherwise stop. The
	// dangling references it creates come back in RemoveResult.Dangling.
	Force bool
}

// Dangling is a reference left pointing at a ticket that no longer exists.
// Remove reports these only under Force, because without it the reference is
// the reason the removal was refused.
type Dangling struct {
	// Ticket and Title name the ticket still holding the reference.
	Ticket string
	Title  string
	// Field is "parent" or "dependencies", which are the two places a ticket
	// names another, per 5.1.
	Field string
}

// RemoveResult is what Remove returns.
//
// It is not a Result. A Result carries the ticket a write produced, and remove
// produces none: Ticket here is the last state the ticket had rather than a new
// one, per 9.1. Dangling has no place in a Result either, since no other
// operation can create one.
type RemoveResult struct {
	// Ticket is the removed ticket in full, read before the file was deleted.
	// A caller that wants it back writes it out again, which is the undo for a
	// removal that was never committed.
	Ticket *Ticket
	// PathsChanged holds the one path the ticket no longer occupies. It is
	// absolute, like every other PathsChanged.
	PathsChanged []string
	// Dangling is empty unless Force removed a referenced ticket.
	Dangling []Dangling
}

// Remove deletes one ticket file, per plan section 9.1.
//
// It refuses a ticket another names in dependencies or parent, because the
// store fails check afterwards and check --fix declines that repair. It refuses
// a ticket carrying notes, comments, a summary, a claim, or an archive record,
// because somebody worked it after filing and archive is the operation for
// that. Force overrides both.
//
// It writes no history and touches no Git index. The file leaves the working
// tree and a person stages the deletion like any other change, which is also
// the undo while the file is still in Git.
func (s *Store) Remove(ctx context.Context, ref string, o RemoveOptions) (*RemoveResult, error) {
	lock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer lock.release()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	files, err := s.load()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(files))
	for _, f := range files {
		if f.Ticket != nil {
			ids = append(ids, f.Ticket.ID)
		} else if f.Err != nil && f.Err.Ticket != "" {
			ids = append(ids, f.Err.Ticket)
		}
	}
	id, err := ResolveRef(ref, ids)
	if err != nil {
		return nil, err
	}

	var target *file
	for i := range files {
		f := &files[i]
		if (f.Ticket != nil && f.Ticket.ID == id) || (f.Err != nil && f.Err.Ticket == id) {
			target = f
			break
		}
	}
	if target == nil || target.Ticket == nil {
		// A file that does not parse cannot be checked against either refusal,
		// so removing it would be a guess. It is also the case rm handles
		// perfectly well, having no opinion to lose.
		if target != nil {
			return nil, target.Err
		}
		return nil, &Error{Code: CodeTicketNotFound, Message: "no such ticket in this store", Ticket: id}
	}

	// The precondition is read under the lock, from the bytes on disk rather
	// than from the parse above, so nothing can slip in between the check and
	// the delete.
	data, err := os.ReadFile(target.Path)
	if err != nil {
		return nil, &Error{Code: CodeTicketNotFound, Message: err.Error(), Ticket: id, Err: err}
	}
	t := target.Ticket
	if actual := Revision(data); o.IfRevision != "" && o.IfRevision != actual {
		return nil, &Error{
			Code:    CodeStaleRevision,
			Message: "ticket changed since it was read",
			Ticket:  id,
			Title:   t.Title,
			Details: map[string]string{"expected": o.IfRevision, "actual": actual},
		}
	}

	dangling := referencesTo(files, id)
	if !o.Force {
		if len(dangling) > 0 {
			return nil, referencedError(t, dangling)
		}
		if why := touched(t); why != "" {
			return nil, &Error{
				Code:    CodeTicketTouched,
				Message: why + ", so this is work rather than a mistake: archive it, or pass --force",
				Ticket:  id,
				Title:   t.Title,
			}
		}
	}

	if err := os.Remove(target.Path); err != nil {
		return nil, &Error{Code: CodeValidationFailed, Message: err.Error(), Ticket: id, Title: t.Title, Err: err}
	}
	t.Revision = Revision(data)
	return &RemoveResult{Ticket: t, PathsChanged: []string{target.Path}, Dangling: dangling}, nil
}

// referencesTo finds every ticket naming id as a parent or a dependency.
//
// Deps with Dependents walks dependencies alone, so this cannot reuse it: a
// child naming the ticket as its parent is just as dangling and is the case an
// epic hits.
func referencesTo(files []file, id string) []Dangling {
	out := make([]Dangling, 0)
	for _, f := range files {
		t := f.Ticket
		if t == nil || t.ID == id {
			continue
		}
		if t.Parent != nil && *t.Parent == id {
			out = append(out, Dangling{Ticket: t.ID, Title: t.Title, Field: "parent"})
		}
		for _, d := range t.Dependencies {
			if d == id {
				out = append(out, Dangling{Ticket: t.ID, Title: t.Title, Field: "dependencies"})
				break
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Ticket != out[j].Ticket {
			return out[i].Ticket < out[j].Ticket
		}
		return out[i].Field < out[j].Field
	})
	return out
}

// referencedError names the tickets pointing at the one being removed, because
// finding that out by hand means grepping the store for an ID, and that is the
// whole of what this refusal buys over rm.
//
// The message names each with its title, per the convention in 12.1: a reader
// sent to unlink a reference should not have to look up which ticket they are
// about to edit. Details keeps the bare IDs, which is what a caller switching
// on the code wants.
func referencedError(t *Ticket, dangling []Dangling) *Error {
	ids := make([]string, 0, len(dangling))
	named := make([]string, 0, len(dangling))
	var byDep, byParent bool
	for _, d := range dangling {
		if d.Field == "parent" {
			byParent = true
		} else {
			byDep = true
		}
		if len(ids) > 0 && ids[len(ids)-1] == d.Ticket {
			// One ticket naming it as both parent and dependency is one
			// referrer, listed once.
			continue
		}
		ids = append(ids, d.Ticket)
		named = append(named, d.Ticket+" ("+d.Title+")")
	}

	// The repair depends on which field points here, because unlink drops a
	// dependency and does nothing to a parent. Naming the wrong one sends a
	// reader to a command that reports success and changes nothing.
	repair := "unlink it there first"
	switch {
	case byParent && byDep:
		repair = "unlink it and clear the parent there first"
	case byParent:
		repair = "clear the parent there first"
	}

	// A store where twenty tickets hang off one epic would otherwise print a
	// paragraph. Two named referrers is enough to start on, and the count says
	// how much is left.
	list := strings.Join(named, ", ")
	if len(named) > 3 {
		list = fmt.Sprintf("%s, and %d more", strings.Join(named[:2], ", "), len(named)-2)
	}
	return &Error{
		Code:    CodeTicketReferenced,
		Message: "still named by " + list + ": " + repair + ", or pass --force",
		Ticket:  t.ID,
		Title:   t.Title,
		Details: map[string]string{"referencedBy": strings.Join(ids, ", ")},
	}
}

// touched says what marks a ticket as worked rather than merely filed, or the
// empty string when nothing does.
//
// A plan, acceptance criteria, and a definition of done are not on this list.
// create seeds all three, per 12.1, so a carefully filed mistake carries them
// from its first revision and refusing on them would refuse the best-filed
// mistakes.
func touched(t *Ticket) string {
	switch {
	case strings.TrimSpace(t.Body.Notes) != "":
		return "it carries notes"
	case strings.TrimSpace(t.Body.Comments) != "":
		return "it carries comments"
	case strings.TrimSpace(t.Body.Summary) != "":
		return "it carries a summary"
	case t.Claim != nil:
		return "it is claimed"
	case t.Archive != nil:
		return "it is archived"
	}
	return ""
}
