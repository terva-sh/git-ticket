package view

import (
	"context"
	"os"

	"github.com/terva-sh/git-ticket/ticket"
)

// Actions is the view's write surface: three closures, each taking the
// ref and the revision the view last read. The revision is the whole
// concurrency story, per the TKT-01M1QBS9 spike: every write carries
// the precondition, and a stale_revision answer reloads and
// re-presents rather than overwriting.
//
// A nil closure means the action is not wired, and the view says so in
// the footer instead of doing nothing silently. That is the same
// degradation rule terva's InteractiveConfig uses, where nil-ness is
// the feature detection.
type Actions struct {
	SetStatus func(ref, revision, status, reason string) error
	Claim     func(ref, revision string) error
	Release   func(ref, revision string) error
	// Create files a new ticket and returns its ID. The store fills
	// every field the form does not ask for from its own defaults,
	// which is the same shape `git ticket create --title` has.
	// template names a store template to seed from, per plan 4.2, and
	// empty means a bare create.
	Create func(title, description, template string) (id string, err error)
	// Edit replaces the title and the description as one write, so a
	// half-applied edit cannot exist. The revision precondition rides
	// on it like every other write.
	Edit func(ref, revision, title, description string) error
	// Copy puts the ticket's stored body on the system clipboard and
	// reports which path took it and how many bytes, so the footer can
	// tell the truth. It reads the file rather than the view's styled
	// render, per plan 12.7: a paste carries what a reader of the file
	// sees.
	Copy func(ref string) (via string, bytes int, err error)
	// Links answers what a ticket is connected to, for the t picker:
	// its parent, its children, what it needs, and what needs it. The
	// snapshot behind it must include done and archived work, because
	// an epic's done children are exactly what a person checks an epic
	// for.
	Links func(ref string) ([]Linked, error)
	// Templates lists the store's templates for the create flow, per
	// plan 4.2, with each description along so the form can prefill
	// its editor: the person edits the skeleton instead of typing over
	// an invisible one. Nil or empty means n opens the bare form.
	Templates func() ([]TemplateChoice, error)
}

// TemplateChoice is one template the create flow can start from.
type TemplateChoice struct {
	Name        string
	Description string
}

// Linked is one edge from a ticket, ready for the picker: the role the
// other ticket plays from the viewed ticket's side, and the ticket
// itself.
type Linked struct {
	Role   string
	Ticket *ticket.Ticket
}

// The roles a Linked can carry, phrased from the viewed ticket's side
// so the picker reads as a sentence: this ticket's parent, its child,
// what it needs, what needs it.
const (
	RoleParent   = "parent"
	RoleChild    = "child"
	RoleNeeds    = "needs"
	RoleNeededBy = "needed by"
)

// StoreParams is everything a store-backed TUI needs that the view
// cannot find for itself: the store, who the writes are recorded as,
// and the git provenance a claim carries per plan 6.4. The cli package
// computes these; its UIParams mirrors this struct field for field so
// the composition root converts one to the other in a cast.
type StoreParams struct {
	Store    *ticket.Store
	Actor    ticket.Actor
	Branch   string
	Worktree string
	Commit   string
	// Clipboard writes body to the system clipboard and names the path
	// it took, per plan 12.7. The cli package binds its probing helper
	// here; nil leaves Copy unwired and the view says so instead of
	// doing nothing silently.
	Clipboard func(body []byte) (via string, err error)
}

// RunProcStore runs the TUI on the process terminal over a store. It
// is the one-line binding cmd/git-ticket wants: lister and actions
// built here, so the composition root does not grow logic an
// embedding host would silently miss.
func RunProcStore(p StoreParams) error {
	return RunProc(StoreLister(p.Store), StoreActions(p))
}

// StoreLister is open work from the store, the list view's contract.
func StoreLister(s *ticket.Store) Lister {
	return func() ([]*ticket.Ticket, error) {
		return s.List(context.Background(), ticket.Filter{})
	}
}

// StoreActions binds Actions to Store.Apply, each write carrying the
// caller's revision and the actor.
func StoreActions(p StoreParams) Actions {
	apply := func(ref, revision string, m ticket.Mutation) error {
		_, err := p.Store.Apply(context.Background(), ref, m, ticket.ApplyOptions{
			IfRevision: revision,
			Actor:      p.Actor,
		})
		return err
	}
	return Actions{
		SetStatus: func(ref, revision, status, reason string) error {
			return apply(ref, revision, ticket.SetStatus{Status: status, Reason: reason})
		},
		Create: func(title, description, template string) (string, error) {
			res, err := p.Store.Create(context.Background(), ticket.CreateOptions{
				Title:       title,
				Description: description,
				Template:    template,
				Actor:       p.Actor,
			})
			if err != nil {
				return "", err
			}
			return res.Ticket.ID, nil
		},
		Templates: func() ([]TemplateChoice, error) {
			names, err := p.Store.Templates()
			if err != nil {
				return nil, err
			}
			choices := make([]TemplateChoice, 0, len(names))
			for _, name := range names {
				tpl, err := p.Store.Template(name)
				if err != nil {
					return nil, err
				}
				choices = append(choices, TemplateChoice{Name: name, Description: tpl.Description})
			}
			return choices, nil
		},
		Edit: func(ref, revision, title, description string) error {
			return apply(ref, revision, ticket.Mutations{
				ticket.SetTitle{Title: title},
				ticket.SetDescription{Text: description},
			})
		},
		Claim: func(ref, revision string) error {
			return apply(ref, revision, ticket.ClaimTicket{
				Branch:   p.Branch,
				Worktree: p.Worktree,
				Commit:   p.Commit,
			})
		},
		Release: func(ref, revision string) error {
			return apply(ref, revision, ticket.ReleaseClaim{})
		},
		Copy:  storeCopy(p),
		Links: storeLinks(p),
	}
}

// storeLinks binds Actions.Links to the store. The snapshot is
// Filter{All: true}, per the TKT-01M1RPZ0 decision: an epic's done
// children are exactly what a person checks an epic for, and the
// open-work default would silently hide them. A dependency that names
// a missing ticket is skipped rather than invented, because
// dependency_missing is check's finding to report, not the picker's.
func storeLinks(p StoreParams) func(string) ([]Linked, error) {
	return func(ref string) ([]Linked, error) {
		t, err := p.Store.Get(context.Background(), ref)
		if err != nil {
			return nil, err
		}
		all, err := p.Store.List(context.Background(), ticket.Filter{All: true})
		if err != nil {
			return nil, err
		}
		byID := make(map[string]*ticket.Ticket, len(all))
		for _, o := range all {
			byID[o.ID] = o
		}

		var links []Linked
		if t.Parent != nil && *t.Parent != "" {
			if parent, ok := byID[*t.Parent]; ok {
				links = append(links, Linked{Role: RoleParent, Ticket: parent})
			}
		}
		for _, o := range all {
			if o.Parent != nil && *o.Parent == t.ID {
				links = append(links, Linked{Role: RoleChild, Ticket: o})
			}
		}
		for _, dep := range t.Dependencies {
			if d, ok := byID[dep]; ok {
				links = append(links, Linked{Role: RoleNeeds, Ticket: d})
			}
		}
		for _, o := range all {
			for _, dep := range o.Dependencies {
				if dep == t.ID {
					links = append(links, Linked{Role: RoleNeededBy, Ticket: o})
					break
				}
			}
		}
		return links, nil
	}
}

// storeCopy binds Actions.Copy to the store and the clipboard writer,
// or to nil when no writer is wired, which is the same nil-ness
// degradation every other action uses.
func storeCopy(p StoreParams) func(string) (string, int, error) {
	if p.Clipboard == nil {
		return nil
	}
	return func(ref string) (string, int, error) {
		t, err := p.Store.Get(context.Background(), ref)
		if err != nil {
			return "", 0, err
		}
		data, err := os.ReadFile(t.Path)
		if err != nil {
			return "", 0, err
		}
		body, err := ticket.RawBody(data)
		if err != nil {
			return "", 0, err
		}
		via, err := p.Clipboard([]byte(body))
		if err != nil {
			return "", 0, err
		}
		return via, len(body), nil
	}
}
