package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/terva-sh/git-ticket/ticket"
)

// runSeries lists, declares, or undeclares an ID prefix, per plan 5.6 and 12.1.
// All three forms answer with the series kind of 10.9.
//
// One kind for a read and two writes, because what a caller wants back from any
// of the three is the same thing: the list the store now declares. A
// mutation-result would answer a question about a ticket for an operation that
// touches no ticket.
//
// The bare form is a read and needs no actor. add and remove write config.yml
// rather than a ticket, which makes them the only writes outside section 9, and
// they still take the store lock, because a store whose series list is read by
// every create cannot have it changed underneath one.
func runSeries(ctx *cmdContext, args []string) error {
	rest, err := ctx.parseFlags("series", args, nil)
	if err != nil {
		return err
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}

	var res *ticket.SeriesResult
	switch {
	case len(rest) == 0:
		res = s.Series()
	case len(rest) == 1:
		return usageErr("series %s takes a name, as in series %s IDEA", rest[0], rest[0])
	case len(rest) != 2:
		return usageErr("series takes no arguments, or add NAME, or remove NAME")
	default:
		// Uppercased here rather than refused, because the grammar is
		// uppercase and a person typing `series add idea` meant IDEA. A
		// reference resolves case-insensitively for the same reason, per 5.5.
		name := strings.ToUpper(strings.TrimSpace(rest[1]))
		switch rest[0] {
		case "add":
			res, err = s.AddSeries(context.Background(), name)
		case "remove":
			res, err = s.RemoveSeries(context.Background(), name)
		default:
			return usageErr("%q is not a series subcommand; use add or remove", rest[0])
		}
		if err != nil {
			return err
		}
	}

	if ctx.g.json {
		writeJSON(ctx.out, newSeriesEnvelope(s, res))
		return nil
	}
	writeSeriesHuman(ctx.out, s, res)
	return nil
}

// writeSeriesHuman prints the list one per line, which is what the bare read
// answers with and what a caller pipes into something else.
//
// A write adds the file it touched, the way every mutation names its paths. A
// no-op add prints the list alone, so the difference between "declared it" and
// "it was already there" is visible without diffing config.yml.
func writeSeriesHuman(w io.Writer, s *ticket.Store, r *ticket.SeriesResult) {
	for _, name := range r.Series {
		fmt.Fprintln(w, name)
	}
	if !r.Changed {
		return
	}
	fmt.Fprintln(w)
	for _, rel := range r.PathsChanged {
		fmt.Fprintf(w, "  %s\n", storePath(s, rel))
	}
}
