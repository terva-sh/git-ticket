package view

import (
	"context"

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
	}
}
