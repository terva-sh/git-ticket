package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/terva-sh/git-ticket/ticket"
)

// Reading a ticket should not cost its whole history.
//
// `show` printed every note in full, so a worked ticket answered a question
// about its current state with everything anybody had ever written on it. In
// this repository's own store that is 463 lines and 3,626 words from one
// command on the worst ticket, and 32,835 words across the 75 that carry notes.
//
// A person pays that in scrolling. An agent pays it in context, which is the
// cost that prompted this: `show` is the ordinary way to read a ticket, and an
// agent reading a worked one spent thousands of tokens on history to learn the
// current state. The notes are worth keeping and were never worth printing all
// at once.
//
// So this is display and nothing else. No file changes, nothing is deleted, and
// `--json` is untouched: `body.notes` is a string a consumer already reads as
// one, and truncating it would be a break under 12.4 for no gain, since a caller
// that asked for the envelope can slice it itself.

// notesShownInFull is how many of the most recent notes `show` prints whole.
//
// One, and the rule is a count rather than a size threshold on purpose. A
// threshold is a magic number that makes the same ticket print differently in
// two stores and gives a reader nothing to predict from. The newest note is
// nearly always the live one, and every older note is one command away.
const notesShownInFull = 1

// compactNotes renders the Notes section for a person: the most recent note in
// full, and one line standing in for the rest.
//
// A ticket with one note prints exactly what it printed before. That is the
// case worth protecting, because most tickets have one note and making them
// cost a second command to read it would trade a real improvement on the worst
// tickets for a small tax on the common one.
func compactNotes(text, id string) string {
	entries := ticket.Entries(text)
	if len(entries) <= notesShownInFull {
		return text
	}
	hidden := entries[:len(entries)-notesShownInFull]
	shown := entries[len(entries)-notesShownInFull:]

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", notesElidedLine(hidden, id))
	for i, e := range shown {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(renderEntry(e))
	}
	return b.String()
}

// notesElidedLine is the stand-in for the notes that were not printed.
//
// It carries the count, the numbers, and the command that gets them back,
// because a reader who has to work out how to retrieve something has been given
// a puzzle rather than a summary. The range is contiguous and starts at one, so
// naming its bounds says everything a list of numbers would.
func notesElidedLine(hidden []ticket.Entry, id string) string {
	span := fmt.Sprintf("%d", hidden[0].Index)
	if len(hidden) > 1 {
		span = fmt.Sprintf("%d-%d", hidden[0].Index, hidden[len(hidden)-1].Index)
	}
	return fmt.Sprintf("_%s, %s hidden. Read them with `git ticket note %s --show %s`, or `--list` for an index._",
		plural(len(hidden), "earlier note"), span, id, span)
}

// renderEntry writes one entry back in the shape the file carries it, so a note
// printed by `show` and the same note in the Markdown read identically.
func renderEntry(e ticket.Entry) string {
	if e.Actor == "" && e.At == "" {
		return e.Text
	}
	return fmt.Sprintf("**%s** at %s\n\n%s", e.Actor, e.At, e.Text)
}

// noteRange is a parsed `--show` or `--fold` argument: one entry, a contiguous
// span, or everything.
type noteRange struct{ from, to int }

// parseNoteRange reads "3", "2-5", or "all".
//
// "all" is spelled out rather than offered as an empty value, because a flag
// whose empty form means everything is one typo away from doing the most it
// could do. The user asked for "a range of notes or all of them" and those are
// exactly the two forms.
func parseNoteRange(arg string, count int) (noteRange, error) {
	if count == 0 {
		return noteRange{}, usageErr("this ticket has no notes")
	}
	if strings.EqualFold(strings.TrimSpace(arg), "all") {
		return noteRange{1, count}, nil
	}
	from, to, err := splitRange(arg)
	if err != nil {
		return noteRange{}, err
	}
	// Both bounds are reported against the count the reader can see, because
	// "out of range" without saying the range is a second question.
	if from < 1 || to > count || from > to {
		return noteRange{}, usageErr("%s is not a range of this ticket's %s; they are numbered 1-%d",
			arg, plural(count, "note"), count)
	}
	return noteRange{from, to}, nil
}

// splitRange parses "N" or "N-M" into its bounds.
func splitRange(arg string) (int, int, error) {
	arg = strings.TrimSpace(arg)
	lo, hi, found := strings.Cut(arg, "-")
	from, err := strconv.Atoi(strings.TrimSpace(lo))
	if err != nil {
		return 0, 0, usageErr("%q is not a note number or a range like 2-5", arg)
	}
	if !found {
		return from, from, nil
	}
	to, err := strconv.Atoi(strings.TrimSpace(hi))
	if err != nil {
		return 0, 0, usageErr("%q is not a note number or a range like 2-5", arg)
	}
	return from, to, nil
}

// writeNoteIndex prints one line per note: number, actor, instant, and opening
// line. It is what makes a range aimable, since a reader cannot ask for note 3
// without a way to see which one that is.
func writeNoteIndex(w *strings.Builder, entries []ticket.Entry) {
	for _, e := range entries {
		actor := e.Actor
		if actor == "" {
			actor = "unattributed"
		}
		at := e.At
		if at == "" {
			at = "no instant"
		}
		fmt.Fprintf(w, "%3d  %s  %s\n     %s\n", e.Index, at, actor, firstLine(e.Text))
	}
}

// firstLine is the opening line of an entry, for the index. An entry whose first
// line is long is cut, because the index earns its keep by being scannable.
func firstLine(text string) string {
	line := text
	if i := strings.IndexByte(line, '\n'); i >= 0 {
		line = line[:i]
	}
	line = strings.TrimSpace(line)
	const width = 72
	if len(line) > width {
		return line[:width-1] + "…"
	}
	if line == "" {
		return "(empty)"
	}
	return line
}
