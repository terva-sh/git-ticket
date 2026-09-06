package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"github.com/terva-sh/git-ticket/ticket"
)

// runMigrate converts the store to a schema level, per plan 12.5, and answers
// with the migrate-result kind of 10.8.
//
// The exit status is 0 whenever the pass succeeded, and a pending migration
// under --dry-run is a success. 12.5 makes a store move only through a
// migration a person runs, so gating a job on this command would gate on a
// decision no job is allowed to take. check is where CI learns a store is
// behind, through migration_incomplete, and a second gate reporting the same
// fact through a different command is how two answers come to disagree.
func runMigrate(ctx *cmdContext, args []string) error {
	var to int
	var dryRun bool
	rest, err := ctx.parseFlags("migrate", args, func(fs *flag.FlagSet) {
		fs.IntVar(&to, "to", 0, "the schema level to convert to, newest this binary writes by default")
		fs.BoolVar(&dryRun, "dry-run", false, "report what the pass would do and write nothing")
	})
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usageErr("migrate takes no arguments; it converts the whole store")
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	res, err := s.Migrate(context.Background(), ticket.MigrateOptions{To: to, DryRun: dryRun})
	if err != nil {
		return err
	}

	if ctx.g.json {
		writeJSON(ctx.out, newMigrateEnvelope(s, res))
		return nil
	}
	writeMigrateHuman(ctx.out, s, res, dryRun)
	return nil
}

// writeMigrateHuman says what moved and what did not.
//
// A run with nothing to do is the ordinary outcome of the second one, so it
// gets a sentence rather than silence a reader has to interpret as success.
func writeMigrateHuman(w io.Writer, s *ticket.Store, r *ticket.MigrateResult, dryRun bool) {
	if !r.ConfigChanged && len(r.Tickets) == 0 {
		fmt.Fprintf(w, "already at schema %d, with %s at that level\n",
			r.To, plural(r.Skipped, "ticket"))
		writeMigrateUnreadable(w, s, r)
		return
	}

	verb := "migrated"
	if dryRun {
		verb = "would migrate"
	}
	fmt.Fprintf(w, "%s schema %d to %d\n", verb, r.From, r.To)
	if r.ConfigChanged {
		// Named first because it is written first, and because an interrupted
		// run leaves this done and the tickets not, per 12.5.
		fmt.Fprintf(w, "  %s\n", storePath(s, "config.yml"))
	}
	for _, rel := range r.Tickets {
		fmt.Fprintf(w, "  %s\n", storePath(s, rel))
	}
	if r.Skipped > 0 {
		fmt.Fprintf(w, "%s already at that level\n", plural(r.Skipped, "ticket"))
	}
	writeMigrateUnreadable(w, s, r)
}

// writeMigrateUnreadable names the files the pass could say nothing about. They
// are not failures of the migration and they are not successes either, so they
// are reported rather than counted silently, and check is where the reason is.
func writeMigrateUnreadable(w io.Writer, s *ticket.Store, r *ticket.MigrateResult) {
	if len(r.Unreadable) == 0 {
		return
	}
	fmt.Fprintf(w, "%s did not parse and %s left alone; run check\n",
		plural(len(r.Unreadable), "file"), were(len(r.Unreadable)))
	for _, rel := range r.Unreadable {
		fmt.Fprintf(w, "  %s\n", storePath(s, rel))
	}
}

func plural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func were(n int) string {
	if n == 1 {
		return "was"
	}
	return "were"
}
