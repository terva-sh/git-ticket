package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/terva-sh/git-ticket/ticket"
)

// An export is a directory of mail-formatted patches: a cover letter that reads
// as a report, and the patches that carry the work. It exists so a ticket can
// reach another store as a ticket rather than as prose somebody re-types.
//
// Everything here is generated rather than asked of git, because the CLI runs no
// git command that writes, per the policy in plan 7.3. Synthesising a commit to
// hand to format-patch would break that, and a patch that adds a new text file
// is the one diff simple enough to write out correctly: no rename detection, no
// context, and a hunk header that is always the same shape.
//
// Nothing here runs git at all. Code patches compose in from the outside:
// `git format-patch --start-number 2 -o <export-dir> <range>` drops its numbered
// files alongside these, and `git am *.patch` then applies the tickets and the
// code in one go. Export owns numbers 0 and 1 so that composition has room.

// exportCoverName is the cover letter, and the extension is load-bearing.
//
// Named .patch it would be swept up by the `git am *.patch` a reader will type,
// and a cover letter has no diff, so am stops on it and wants --empty=drop,
// which is git 2.34 or newer. Named .txt the glob never sees it, plain `git am
// *.patch` works on any git, and the cover still opens in any editor. One
// character, one fewer version floor.
const exportCoverName = "0000-cover-letter.txt"

// exportTicketPatch is the patch that carries the ticket files themselves.
//
// It is always written, even when code patches are composed in beside it. A bare
// report and a code contribution are then the same artifact with different
// contents, and the receiving side has one thing to do rather than two.
const exportTicketPatch = "0001-tickets.patch"

// runExport writes an export directory for one or more tickets.
func runExport(ctx *cmdContext, args []string) error {
	var out string
	rest, err := ctx.parseFlags("export", args, func(fs *flag.FlagSet) {
		fs.StringVar(&out, "out", "", "the directory to write, defaulting to <first-id>-export")
	})
	if err != nil {
		return err
	}
	if len(rest) == 0 {
		return usageErr("export takes at least one ticket ID")
	}
	s, err := ctx.openStore()
	if err != nil {
		return err
	}

	when := time.Now()
	if ctx.env.Now != nil {
		when = ctx.env.Now()
	}

	// The identity is the sender's git config, not the ticket's actor. An actor
	// is a session and a From: header is a person, and it is the person sending
	// who answers for what arrives. It is read here rather than in the library
	// because it comes from git, which the library does not run.
	art, err := s.Export(context.Background(), ticket.ExportOptions{
		IDs:  rest,
		From: exportIdentity(ctx.env.Dir),
		Now:  when,
	})
	if err != nil {
		return err
	}

	// Where it lands is the command's decision, which is why Export hands back
	// bytes rather than filling a directory.
	dir := out
	if dir == "" {
		dir = art.Tickets[0].ID + "-export"
	}
	if err := exportDirIsFree(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	written := make([]string, 0, 3)
	ticketPath := filepath.Join(dir, exportTicketPatch)
	if err := os.WriteFile(ticketPath, art.Patch, 0o644); err != nil {
		return err
	}
	written = append(written, ticketPath)

	coverPath := filepath.Join(dir, exportCoverName)
	if err := os.WriteFile(coverPath, art.Cover, 0o644); err != nil {
		return err
	}
	written = append([]string{coverPath}, written...)

	// Before the --json branch, because an edge that will not travel breaks the
	// receiving store and a caller passing --json is the one least likely to be
	// reading the artifact by eye. Warnings go to stderr in both modes, as the
	// actor and heading warnings already do.
	warnDanglingEdges(ctx, art.Tickets)

	if ctx.g.json {
		writeJSON(ctx.out, mutationEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "mutation-result",
			Ticket:        nil,
			PathsChanged:  written,
		})
		return nil
	}
	fmt.Fprintf(ctx.out, "wrote %s\n", dir)
	for _, p := range written {
		fmt.Fprintf(ctx.out, "  %s\n", filepath.Base(p))
	}
	fmt.Fprintf(ctx.env.Stderr, "apply with: git am %s/*.patch\n", dir)
	// The composition, with the directory already filled in. There is no --patch
	// flag, per 12.8, so a sender who does not know to compose simply ships an
	// export with no code in it and finds out from the receiver.
	fmt.Fprintf(ctx.env.Stderr, "add code with: git format-patch --start-number 2 -o %s RANGE\n", dir)
	return nil
}

// warnDanglingEdges names the parents and dependencies that will not travel.
//
// `git am` applies a ticket file verbatim, so an edge naming a ticket outside
// the export arrives pointing at nothing, and parent_missing and
// dependency_missing are errors rather than warnings. The receiving store is
// then broken by an artifact that applied without complaint, which is the worst
// shape a failure can take.
//
// Export cannot fix this. Including the missing tickets is the sender's call,
// and stripping the edges would quietly change what the ticket says. So it says
// so, at the moment the sender can still do something about it.
func warnDanglingEdges(ctx *cmdContext, tickets []*ticket.Ticket) {
	inSet := make(map[string]bool, len(tickets))
	for _, t := range tickets {
		inSet[t.ID] = true
	}
	var lines []string
	for _, t := range tickets {
		for _, d := range t.Dependencies {
			if !inSet[d] {
				lines = append(lines, fmt.Sprintf("  %s depends on %s", t.ID, d))
			}
		}
		if t.Parent != nil && *t.Parent != "" && !inSet[*t.Parent] {
			lines = append(lines, fmt.Sprintf("  %s is parented to %s", t.ID, *t.Parent))
		}
	}
	if len(lines) == 0 {
		return
	}
	fmt.Fprintf(ctx.env.Stderr,
		"warning: %s naming a ticket this export does not carry:\n%s\n",
		plural(len(lines), "edge"), strings.Join(lines, "\n"))
	fmt.Fprintf(ctx.env.Stderr,
		"  Applied with `git am` these are parent_missing and dependency_missing, which are errors.\n"+
			"  Export those tickets too, or have the receiver use `git ticket import`, which drops what it cannot resolve.\n")
}

// exportDirIsFree refuses a directory that already holds something.
//
// Writing into one would leave a mixture of two exports numbered the same way,
// and `git am *.patch` would apply both without any sign that it had.
func exportDirIsFree(dir string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(entries) > 0 {
		return fmt.Errorf("%s is not empty; name an empty directory with --out", dir)
	}
	return nil
}

// exportIdentity reads the sender out of git config. Both halves are optional
// as far as git is concerned, so a store used without an identity still exports;
// the message simply says less about who sent it.
func exportIdentity(dir string) string {
	name := readGit(dir, "config", "user.name")
	mail := readGit(dir, "config", "user.email")
	switch {
	case name != "" && mail != "":
		return fmt.Sprintf("%s <%s>", name, mail)
	case mail != "":
		return fmt.Sprintf("unknown <%s>", mail)
	case name != "":
		return fmt.Sprintf("%s <unknown@localhost>", name)
	}
	return "unknown <unknown@localhost>"
}
