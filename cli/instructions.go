package cli

import (
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// instructionsText is the agent workflow block of plan 12.1. It lives in a
// Markdown file rather than a Go string so a wording change reviews as a prose
// diff, and so nobody has to escape a backtick to edit it.
//
//go:embed instructions.md
var instructionsText string

// instructionsCore is the short form, and it is what --write installs.
//
// The two exist because the written block is loaded in every session of every
// project that adopts this tool, which makes it the largest standing context
// cost here: the full text is 2,434 words and most of them are the reason for a
// rule rather than the rule. An agent needs the rule every time and the reason
// when a rule surprises it, so the summary goes in the file and the argument
// stays one command away. That is the same split `show` makes for note history.
//
// The core names `git ticket instructions` so the long form is reachable from
// the short one. A summary nobody can get behind is worse than the volume it
// replaced.
//
//go:embed instructions-core.md
var instructionsCore string

// instructionsFile is where the block is written. It is the file terva and most
// agent harnesses already read.
const instructionsFile = "AGENTS.md"

// The block is fenced by these markers so it can be replaced in place later,
// leaving everything a person wrote around it untouched. They are in
// instructions.md rather than added by the writer, so every way the block
// leaves this binary carries them: stdout for a hand-paste, the JSON text, and
// the file. A block somebody pasted by hand is refreshable for that reason.
//
// An HTML comment is the form because it renders as nothing, so it does not
// clutter a file a person reads and edits. The worry in section F of
// docs/review-backlog-md.md is real but does not bite here. Backlog.md parses
// its checklists out of markers, so losing one loses data; these bound a block
// this tool regenerates, so losing them costs a refresh.
const (
	instructionsBegin = "<!-- git-ticket:begin -->"
	instructionsEnd   = "<!-- git-ticket:end -->"
)

// runInstructions prints the block for a person to paste or redirect, and with
// --write puts it in AGENTS.md and keeps it current there. It reads no store,
// so it answers outside a repository and before init.
func runInstructions(ctx *cmdContext, args []string) error {
	var write, core, full bool
	rest, err := ctx.parseFlags("instructions", args, func(fs *flag.FlagSet) {
		fs.BoolVar(&write, "write", false,
			"write the short form to "+instructionsFile+", replacing an earlier one in place")
		fs.BoolVar(&core, "core", false,
			"print the short form, which is what --write installs")
		fs.BoolVar(&full, "full", false,
			"the long form, with the reason behind each rule; the default when printing")
	})
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return usageErr("instructions takes no arguments")
	}
	if core && full {
		return usageErr("instructions --core and --full ask for different texts; pass one")
	}
	// Printing and writing default to different forms, because the callers want
	// different things. Somebody typing the command is asking how to work here
	// and wants the reasoning; a setup step is filling a file that is read in
	// every session afterwards and wants it short. Either default is overridable
	// and the help text says which is which.
	if write {
		return writeInstructions(ctx, full)
	}
	if core {
		return printInstructions(ctx, instructionsCore)
	}
	return printInstructions(ctx, instructionsText)
}

// printInstructions writes one form to stdout, or into the envelope.
func printInstructions(ctx *cmdContext, text string) error {
	if ctx.g.json {
		writeJSON(ctx.out, instructionsEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "instructions",
			Text:          text,
		})
		return nil
	}
	_, err := fmt.Fprint(ctx.out, text)
	return err
}

// writeInstructions puts the block in AGENTS.md at the repository root, or in
// the working directory when there is no repository, since the command answers
// anywhere.
func writeInstructions(ctx *cmdContext, full bool) error {
	root := readGit(ctx.env.Dir, "rev-parse", "--show-toplevel")
	if root == "" {
		root = ctx.env.Dir
	}

	block := instructionsCore
	if full {
		block = instructionsText
	}
	path, action, err := writeInstructionsFile(root, block)
	if err != nil {
		return err
	}

	shown := path
	if rel, ok := relativeTo(root, path); ok {
		shown = rel
	}
	if ctx.g.json {
		changed := []string{shown}
		if action == instructionsCurrent {
			changed = []string{}
		}
		writeJSON(ctx.out, mutationEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "mutation-result",
			Ticket:        nil,
			PathsChanged:  changed,
		})
		return nil
	}
	_, err = fmt.Fprintf(ctx.out, "%s\n", action.sentence(shown))
	return err
}

// spliceInstructions returns what the file should hold once the block is
// current, and reports what it did so the caller can say so.
//
// A file carrying both markers has the text between them replaced and every
// byte outside them left alone. A file carrying neither is appended to, which
// is the ordinary case for a project that already has an AGENTS.md.
//
// Anything else refuses. One marker without its partner, a pair out of order,
// or a second copy of either leaves no honest reading of where the block ends,
// and the wrong guess would delete prose somebody wrote. A refusal costs a
// person one edit; guessing costs them work they cannot get back.
func spliceInstructions(existing, block string) (string, instructionsAction, error) {
	begins := strings.Count(existing, instructionsBegin)
	ends := strings.Count(existing, instructionsEnd)

	switch {
	case begins == 0 && ends == 0:
		return appendBlock(existing, block), instructionsAppended, nil
	case begins == 1 && ends == 1:
		start := strings.Index(existing, instructionsBegin)
		stop := strings.Index(existing, instructionsEnd)
		if stop < start {
			return "", "", usageErr("%s has %s before %s; put them back in order and run this again",
				instructionsFile, instructionsEnd, instructionsBegin)
		}
		stop += len(instructionsEnd)
		return existing[:start] + strings.TrimRight(block, "\n") + existing[stop:], instructionsRefreshed, nil
	default:
		return "", "", usageErr("%s carries %d %s and %d %s; leave exactly one of each around the block and run this again",
			instructionsFile, begins, instructionsBegin, ends, instructionsEnd)
	}
}

// instructionsAction is what the write did, so the caller reports it in a
// sentence rather than assembling one out of fragments.
type instructionsAction string

const (
	instructionsWrote     instructionsAction = "wrote"
	instructionsAppended  instructionsAction = "appended"
	instructionsRefreshed instructionsAction = "refreshed"
	instructionsCurrent   instructionsAction = "current"
)

func (a instructionsAction) sentence(path string) string {
	switch a {
	case instructionsAppended:
		return fmt.Sprintf("Appended the ticket workflow block to %s", path)
	case instructionsRefreshed:
		return fmt.Sprintf("Refreshed the ticket workflow block in %s", path)
	case instructionsCurrent:
		return fmt.Sprintf("%s is already current", path)
	default:
		return fmt.Sprintf("Wrote %s", path)
	}
}

// appendBlock puts the block at the end of what is already there, with one
// blank line between, and leaves a file that ends in exactly one newline.
func appendBlock(existing, block string) string {
	if !strings.HasSuffix(block, "\n") {
		block += "\n"
	}
	if strings.TrimSpace(existing) == "" {
		return block
	}
	return strings.TrimRight(existing, "\n") + "\n\n" + block
}

// checkInstructionsFile reports whether AGENTS.md is in a state the writer can
// act on, without changing anything.
//
// runInit calls this before creating the store. The write itself has to come
// after, because the store is what the block talks about, and plan 12.1 wants a
// refusal to leave nothing half-built. Splicing twice costs nothing next to
// making a user clean up a store they did not get.
func checkInstructionsFile(root string) error {
	existing, err := os.ReadFile(filepath.Join(root, instructionsFile))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	_, _, err = spliceInstructions(string(existing), instructionsText)
	return err
}

// writeInstructionsFile brings AGENTS.md at the repository root up to date and
// reports the path and what it did. A file that is not there is created.
func writeInstructionsFile(root, block string) (string, instructionsAction, error) {
	path := filepath.Join(root, instructionsFile)

	existing, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := os.WriteFile(path, []byte(appendBlock("", block)), 0o644); err != nil {
			return "", "", err
		}
		return path, instructionsWrote, nil
	}
	if err != nil {
		return "", "", err
	}

	next, action, err := spliceInstructions(string(existing), block)
	if err != nil {
		return "", "", err
	}
	// Rewriting identical bytes would touch the mtime and show up as a change
	// in a diff that has nothing in it.
	if next == string(existing) {
		return path, instructionsCurrent, nil
	}
	if err := os.WriteFile(path, []byte(next), 0o644); err != nil {
		return "", "", err
	}
	return path, action, nil
}
