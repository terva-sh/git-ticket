package cli

import (
	"fmt"
	"io"

	"github.com/terva-sh/git-ticket/ticket"
)

// warnSectionHeadings reports text that will not land where the caller meant.
//
// A body section is written from one string, and the stored body is split on
// any line opening with "## ". A heading inside that string therefore ends the
// section early: the text above it stays, and everything below it becomes a
// section of its own.
//
// Nothing downstream notices. `check` passes, `show` prints every section in
// order, and the file reads correctly, which is how TKT-01M1HVMQ was filed that
// way and had to be filed again. The silence is the defect, not the splitting
// rule, which is right and is what makes a hand-written "## Risks" a section
// somebody wanted.
//
// This warns rather than refuses. Passing several sections in one string works
// today and somebody may be doing it deliberately, so refusing would break a
// path that functions in order to prevent a mistake. The caller keeps the
// choice and pays one line of stderr.
//
// It writes to stderr in both output modes. A warning is not an error, and
// putting it on stdout would corrupt the envelope that `--json` promised.
// Nothing here touches the exit status.
//
// source names where the text came from, either a flag such as "--description"
// or a command such as "summary".
func warnSectionHeadings(w io.Writer, source, text string) {
	headings := ticket.SectionHeadings(text)
	if len(headings) == 0 {
		return
	}
	// A heading naming a section the format owns is the one case where "### "
	// is the wrong advice, and following it is how a ticket ends up with
	// criteria no mutation can reach. "## Acceptance criteria" opens the real
	// section and its items tick; "### Acceptance criteria" is prose that
	// renders identically and cannot. So say what happened instead of proposing
	// a fix for something that is not broken.
	//
	// Only when it is the sole heading. With several, the text is being split
	// more ways than this sentence can describe, and the general warning below
	// is the honest one.
	if len(headings) == 1 && ticket.IsSectionName(headings[0]) {
		fmt.Fprintf(w, "git-ticket: note: the text for %s contains %q, which opened the real %s section.\n",
			source, "## "+headings[0], headings[0])
		fmt.Fprintf(w, "  Its content is live there rather than in the text you passed. Do not write %q, which would be prose no mutation reaches.\n",
			"### "+headings[0])
		return
	}
	// The first one is where the section actually ends, so it is the one worth
	// naming. Counting the rest says how much went with it without printing a
	// list nobody reads.
	first := "## " + headings[0]
	switch n := len(headings) - 1; {
	case n == 0:
		fmt.Fprintf(w, "git-ticket: warning: the text for %s contains %q, which opens a new section rather than a subheading.\n",
			source, first)
	case n == 1:
		fmt.Fprintf(w, "git-ticket: warning: the text for %s contains %q and one more heading below it, each opening a new section rather than a subheading.\n",
			source, first)
	default:
		fmt.Fprintf(w, "git-ticket: warning: the text for %s contains %q and %d more headings below it, each opening a new section rather than a subheading.\n",
			source, first, n)
	}
	fmt.Fprintf(w, "  Only the text above it stays in that section. Write %q instead.\n", "### "+headings[0])
}
