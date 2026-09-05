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
	Create func(title, description string) (id string, err error)
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
}

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
		Create: func(title, description string) (string, error) {
			res, err := p.Store.Create(context.Background(), ticket.CreateOptions{
				Title:       title,
				Description: description,
				Actor:       p.Actor,
			})
			if err != nil {
				return "", err
			}
			return res.Ticket.ID, nil
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
		Copy: storeCopy(p),
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
