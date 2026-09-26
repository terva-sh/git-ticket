package cli

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/terva-sh/git-ticket/ticket"
)

func runMove(ctx *cmdContext, args []string) error {
	var to, reason string
	rest, err := ctx.parseFlags("move", args, func(fs *flag.FlagSet) {
		fs.StringVar(&to, "to-ref", "", "typed destination reference")
		fs.StringVar(&reason, "reason", "", "why the work moved")
	})
	if err != nil {
		return err
	}
	if len(rest) != 1 || to == "" || strings.TrimSpace(reason) == "" {
		return usageErr("move takes one ticket ID, --to-ref, and --reason")
	}
	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	res, err := ctx.applyTo(s, rest[0], ticket.MoveTo{Ref: to, Reason: reason})
	if err != nil {
		return err
	}
	if err := ctx.writeMutation(s, res, fmt.Sprintf("%s (%s) moved to %s", res.Ticket.ID, res.Ticket.Title, to)); err != nil {
		return err
	}
	if ctx.g.json {
		return nil
	}
	dependents, err := s.Deps(context.Background(), res.Ticket.ID, ticket.DepsOptions{Dependents: true})
	if err != nil {
		return err
	}
	if len(dependents) == 0 {
		fmt.Fprintln(ctx.out, "No local dependents need resolution.")
		return nil
	}
	fmt.Fprintln(ctx.out, "Affected local dependents (resolve each open dependency manually):")
	for _, d := range dependents {
		fmt.Fprintf(ctx.out, "  %s  %s\n", d.ID, d.Title)
	}
	return nil
}

func runResolveMove(ctx *cmdContext, args []string) error {
	var from, sourceRevision, reason, waitOn string
	rest, err := ctx.parseFlags("resolve-move", args, func(fs *flag.FlagSet) {
		fs.StringVar(&from, "from", "", "moved original ticket")
		fs.StringVar(&sourceRevision, "if-source-revision", "", "revision read from the original")
		fs.StringVar(&reason, "reason", "", "manual resolution decision")
		fs.StringVar(&waitOn, "wait-on", "", "replacement local prerequisite")
	})
	if err != nil {
		return err
	}
	if len(rest) != 1 || from == "" || sourceRevision == "" || strings.TrimSpace(reason) == "" {
		return usageErr("resolve-move takes one dependent ID, --from, --if-source-revision, and --reason")
	}
	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	res, err := s.ResolveMove(context.Background(), rest[0], ticket.ResolveMoveOptions{
		From: from, IfSourceRevision: sourceRevision, IfRevision: ctx.g.ifRevision,
		WaitOn: waitOn, Reason: reason, Actor: ctx.actor(s),
	})
	if err != nil {
		return err
	}
	return ctx.writeMutation(s, res, fmt.Sprintf("%s (%s) resolved moved dependency on %s", res.Ticket.ID, res.Ticket.Title, from))
}
