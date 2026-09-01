// Command git-ticket is the standalone CLI for a git-ticket store. Git finds it
// on PATH, so `git ticket list` and `git-ticket list` are the same command.
//
// Everything it does lives in internal/cli and in the ticket package. This file
// only supplies the process: the working directory, the environment, the
// streams, and the exit status.
package main

import (
	"os"

	"github.com/terva-sh/git-ticket/internal/cli"
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
	}))
}
