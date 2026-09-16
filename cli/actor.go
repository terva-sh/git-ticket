package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/terva-sh/git-ticket/ticket"
)

// runActor lists the roster or adds to it, per TKT-01M2NT7QS84SB5P7DR2GXB9DZ0.
//
// It exists for a bootstrapping problem. `init` with no --actor and no terminal
// leaves an empty roster, which is right for a script, and a write that names no
// actor is then refused because updated_by would say nothing. Until now the only
// way out was a text editor, and terva's TUI carries refusal text that exists
// only because of that.
//
// The bare form is a read and the add form writes config.yml rather than a
// ticket, which is what lets either run against a store with no actor at all.
// That is the same reason `series` is shaped this way, and it is load-bearing
// here rather than incidental: a mutation under section 9 would be refused by
// the emptiness it is trying to fix.
func runActor(ctx *cmdContext, args []string) error {
	var name string
	var makeDefault bool
	rest, err := ctx.parseFlags("actor", args, func(fs *flag.FlagSet) {
		fs.StringVar(&name, "name", "", "the display name to record beside the id")
		fs.BoolVar(&makeDefault, "default", false,
			"also make this the actor a write with no --actor records")
	})
	if err != nil {
		return err
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}

	var res *ticket.ActorResult
	switch {
	case len(rest) == 0:
		if name != "" || makeDefault {
			return usageErr("actor --name and --default describe an actor; name one, as in actor add human:you")
		}
		res = s.Actors()
	case len(rest) == 1:
		if rest[0] == "add" {
			return usageErr("actor add takes an id, as in actor add human:you")
		}
		return usageErr("actor takes no arguments, or add ID")
	case len(rest) == 2 && rest[0] == "add":
		res, err = s.AddActor(context.Background(), rest[1], name, makeDefault)
		if err != nil {
			return err
		}
	default:
		return usageErr("actor takes no arguments, or add ID")
	}

	if ctx.g.json {
		writeJSON(ctx.out, newActorEnvelope(s, res))
		return nil
	}
	writeActorHuman(ctx.out, s, res)
	return nil
}

// writeActorHuman prints the roster one per line, marking the declared default.
//
// An empty roster is said outright rather than printed as nothing, because
// nothing is what a store with a broken read looks like too, and this is the
// command somebody runs precisely when they suspect the roster is empty.
func writeActorHuman(w io.Writer, s *ticket.Store, r *ticket.ActorResult) {
	if len(r.Actors) == 0 {
		fmt.Fprintln(w, "No actors. A write that names no --actor will be refused.")
		fmt.Fprintln(w, "Add one with `git ticket actor add human:you --default`.")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, a := range r.Actors {
		mark := ""
		if a.ID == r.Default {
			mark = "default"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", a.ID, a.Name, mark)
	}
	tw.Flush()
	// The same rule the series writer follows: a write names the file it
	// touched, so "declared it" and "it was already there" differ on screen
	// without diffing config.yml.
	for _, p := range r.PathsChanged {
		fmt.Fprintf(w, "  %s\n", storePath(s, p))
	}
	if len(r.Actors) > 0 && r.Default == "" {
		fmt.Fprintln(w,
			"\nNo defaults.actor, so a write with no --actor is recorded as the first entry and warns.")
	}
}

// actorEnvelope is the actor kind of section 10.
type actorEnvelope struct {
	SchemaVersion int    `json:"schemaVersion"`
	Kind          string `json:"kind"`
	// actorJSON is the one a ticket's claim already publishes, reused rather
	// than redeclared: an actor is the same {id, name} pair wherever it appears.
	Actors []actorJSON `json:"actors"`
	// Default is null rather than empty when the store declares none, so a
	// consumer never has to tell missing from empty.
	Default      *string  `json:"default"`
	Changed      bool     `json:"changed"`
	PathsChanged []string `json:"pathsChanged"`
}

func newActorEnvelope(s *ticket.Store, r *ticket.ActorResult) actorEnvelope {
	env := actorEnvelope{
		SchemaVersion: schemaVersion,
		Kind:          "actor",
		Actors:        make([]actorJSON, 0, len(r.Actors)),
		Changed:       r.Changed,
		PathsChanged:  make([]string, 0, len(r.PathsChanged)),
	}
	for _, a := range r.Actors {
		env.Actors = append(env.Actors, actorJSON{ID: a.ID, Name: a.Name})
	}
	if r.Default != "" {
		d := r.Default
		env.Default = &d
	}
	for _, p := range r.PathsChanged {
		env.PathsChanged = append(env.PathsChanged, storePath(s, p))
	}
	return env
}
