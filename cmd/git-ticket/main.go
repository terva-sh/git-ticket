// Command git-ticket is the standalone CLI for a git-ticket store. Git finds it
// on PATH, so `git ticket list` and `git-ticket list` are the same command.
//
// Everything it does lives in the cli and ticket packages. This file only
// supplies the process: the working directory, the environment, the streams,
// and the exit status.
//
// It is also the reference caller for cli.Run, which is exported so a host can
// embed the whole command surface. Terva does exactly this for `terva ticket`.
// Keep this function trivial: anything it grows is something an embedding host
// silently does not get.
package main

import (
	"context"
	"os"

	"github.com/terva-sh/git-ticket/cli"
	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui/view"
)

func main() {
	dir, err := os.Getwd()
	if err != nil {
		// Without a working directory there is nothing to discover a store
		// from, and no store means no useful work.
		os.Stderr.WriteString("git-ticket: " + err.Error() + "\n")
		os.Exit(1)
	}
	os.Exit(cli.Run(os.Args[1:], cli.Env{
		Dir:    dir,
		Getenv: os.Getenv,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
		// The one binding an embedding host does not inherit, by design:
		// a host that wants the TUI wires its own terminal, and a host
		// that only wants the command surface never builds this one.
		RunUI: func(s *ticket.Store) error {
			return view.RunProc(func() ([]*ticket.Ticket, error) {
				return s.List(context.Background(), ticket.Filter{})
			})
		},
	}))
}
