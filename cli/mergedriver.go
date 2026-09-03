package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/terva-sh/git-ticket/ticket"
)

// runMergeDriver is what Git invokes for a ticket file, per plan 7.5.
//
// The three arguments are Git's %O %A %B: the merge base, our version, and
// theirs. The result goes to the %A path, which is Git's contract and not a
// choice this makes. Exit zero means the file is clean and non-zero means it
// carries conflict markers for somebody to resolve.
//
// It opens no store. A driver runs on three temporary files Git wrote, which
// are not in the store and may not be in a repository this tool can find.
func runMergeDriver(ctx *cmdContext, args []string) error {
	rest, err := ctx.parseFlags("merge-driver", args, nil)
	if err != nil {
		return err
	}
	if len(rest) != 3 {
		return usageErr("merge-driver takes three paths: BASE OURS THEIRS")
	}
	basePath, oursPath, theirsPath := rest[0], rest[1], rest[2]

	base, err := os.ReadFile(basePath)
	if err != nil {
		return err
	}
	ours, err := os.ReadFile(oursPath)
	if err != nil {
		return err
	}
	theirs, err := os.ReadFile(theirsPath)
	if err != nil {
		return err
	}

	res, mergeErr := ticket.Merge(base, ours, theirs)
	if mergeErr != nil {
		// A file this tool cannot read must still not lose a side. Writing both
		// versions between markers is what Git's own driver would have done,
		// and it leaves the decision with the person rather than with a parser
		// that already said it does not understand the file.
		if err := writeResult(oursPath, wholeFileConflict(ours, theirs)); err != nil {
			return err
		}
		fmt.Fprintf(ctx.env.Stderr, "git-ticket: %s: %v\n", oursPath, mergeErr)
		return errReported
	}

	if err := writeResult(oursPath, res.Merged); err != nil {
		return err
	}
	if res.Clean() {
		return nil
	}
	fmt.Fprintf(ctx.env.Stderr, "git-ticket: %s: could not merge %s\n",
		oursPath, strings.Join(res.Conflicts, ", "))
	return errReported
}

// writeResult replaces the %A file, keeping the mode Git gave it.
func writeResult(path string, data []byte) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(path, data, mode)
}

// wholeFileConflict is the fallback for a file the parser refused: both
// versions, marked, with nothing dropped.
func wholeFileConflict(ours, theirs []byte) []byte {
	var b strings.Builder
	b.WriteString("<<<<<<< ours\n")
	b.Write(ours)
	if len(ours) > 0 && ours[len(ours)-1] != '\n' {
		b.WriteString("\n")
	}
	b.WriteString("=======\n")
	b.Write(theirs)
	if len(theirs) > 0 && theirs[len(theirs)-1] != '\n' {
		b.WriteString("\n")
	}
	b.WriteString(">>>>>>> theirs\n")
	return []byte(b.String())
}
