package ticket

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Building an export, per plan 12.8.
//
// Export returns the artifact as bytes. Writing it to a directory is the
// caller's decision: a command puts it on disk, another host may put it on a
// wire, and a return value is testable without a filesystem.
//
// It runs no git. The blob names in the diff are computed, per interchange.go,
// which is what lets an export be built where there is no repository and what
// keeps plan 7.4's command table short.

// exportSeriesPatches is how many patches the series this builds contains.
//
// It is one, and the cover letter says "0/1". A sender attaching code composes
// in from number 2 upward, per 12.8, which is why this is the count of what
// Export writes rather than the count of what the directory ends up holding.
const exportSeriesPatches = 1

// ExportOptions is what an export needs beyond the store.
type ExportOptions struct {
	// IDs are the tickets to carry, in the order they should appear. Each is
	// resolved the way any ref is, so a unique prefix works.
	IDs []string
	// From is the sender's identity for the From: header, as "Name <mail>".
	// It is the person sending rather than the actor of any ticket: an actor is
	// a session and a From: header is a person, and it is the person sending
	// who answers for what arrives.
	From string
	// Now is the instant the Date: headers carry.
	Now time.Time
}

// Export is the artifact, as bytes.
type Export struct {
	// Cover is the cover letter. It is not part of the series and has no diff,
	// which is why a caller should not name it *.patch.
	Cover []byte
	// Patch is the one patch of the series, adding the ticket files.
	Patch []byte
	// Tickets are the tickets it carries, resolved, for a caller that wants to
	// report on them without reading them again.
	Tickets []*Ticket
}

// Export builds the artifact that carries these tickets to another store.
func (s *Store) Export(ctx context.Context, o ExportOptions) (*Export, error) {
	if len(o.IDs) == 0 {
		return nil, fmt.Errorf("export takes at least one ticket ID")
	}
	tickets := make([]*Ticket, 0, len(o.IDs))
	for _, id := range o.IDs {
		t, err := s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}

	body, err := exportTicketFiles(s, tickets)
	if err != nil {
		return nil, err
	}
	return &Export{
		Cover:   []byte(exportCover(o.From, o.Now, tickets, exportSeriesPatches)),
		Patch:   []byte(MboxMessage(o.From, o.Now, exportSubject(tickets), exportCommitBody(tickets), body)),
		Tickets: tickets,
	}, nil
}

// exportSubject is the commit subject the receiving log will carry. One ticket
// lends its title, because that is what a person recognises; several cannot, so
// they are counted instead.
func exportSubject(tickets []*Ticket) string {
	if len(tickets) == 1 {
		return "Ticket: " + tickets[0].Title
	}
	return fmt.Sprintf("Tickets: %d from another store", len(tickets))
}

// exportCommitBody says what the commit adds. It stays short on purpose: the
// ticket text is in the diff directly below it, and repeating it there would
// double the size of every export to no end.
//
// Both leading columns are padded to the widest value in this set, rather than
// to a constant, so a person reading the commit in git log finds the titles on
// one column and a set of short statuses carries no trench of spaces.
//
// The ID is padded as well as the status, which is what the alignment actually
// needs. A status is 4 to 11 characters, and an ID is 29 to 35, because a
// series prefix is 2 to 8 per plan 5.6 and a store may declare several. So a
// multi-series export has two ragged columns and padding the status alone would
// leave the titles where they started.
func exportCommitBody(tickets []*Ticket) string {
	var b strings.Builder
	b.WriteString("Adds the ticket files below. The status of each decides the directory it\n")
	b.WriteString("lands in, so the store is consistent the moment this applies.\n\n")
	idWidth, statusWidth := 0, 0
	for _, t := range tickets {
		idWidth = max(idWidth, len(t.ID))
		statusWidth = max(statusWidth, len(string(t.Status)))
	}
	for _, t := range tickets {
		// The title is last and is never padded, so no line ends in spaces.
		fmt.Fprintf(&b, "  %-*s  %-*s  %s\n", idWidth, t.ID, statusWidth, t.Status, t.Title)
	}
	return b.String()
}

// exportTicketFiles builds the diff that adds every ticket file.
func exportTicketFiles(s *Store, tickets []*Ticket) (string, error) {
	root := s.Root()
	var diff, stat strings.Builder
	adds := 0
	for _, t := range tickets {
		data, err := os.ReadFile(t.Path)
		if err != nil {
			return "", err
		}
		rel := t.Path
		if r, ok := relativeToRoot(root, t.Path); ok && root != "" {
			rel = r
		}
		rel = filepath.ToSlash(rel)
		hunk, n := AddedFileHunk(rel, data)
		diff.WriteString(hunk)
		adds += n
		fmt.Fprintf(&stat, " %s | %d %s\n", rel, n, strings.Repeat("+", plusBar(n)))
	}
	var b strings.Builder
	b.WriteString(stat.String())
	fmt.Fprintf(&b, " %s changed, %s(+)\n", exportPlural(len(tickets), "file"), exportPlural(adds, "insertion"))
	for _, t := range tickets {
		rel := t.Path
		if r, ok := relativeToRoot(root, t.Path); ok && root != "" {
			rel = r
		}
		fmt.Fprintf(&b, " create mode 100644 %s\n", filepath.ToSlash(rel))
	}
	b.WriteString("\n")
	b.WriteString(diff.String())
	return b.String(), nil
}

// exportCover is the report half: what this is, how to apply it, and what it
// carries. It is written for somebody who has never run git-ticket, because the
// first person to receive one of these will not have it.
func exportCover(who string, when time.Time, tickets []*Ticket, patches int) string {
	var b strings.Builder
	b.WriteString(MboxFromLine + "\n")
	fmt.Fprintf(&b, "From: %s\n", who)
	fmt.Fprintf(&b, "Date: %s\n", when.Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Subject: [PATCH 0/%d] %s\n", patches, exportSubject(tickets))
	b.WriteString("\n")
	b.WriteString("This is a git-ticket export: one or more tickets, and the patches that\n")
	b.WriteString("carry them.\n\n")
	b.WriteString("    git am *.patch\n\n")
	b.WriteString("Nothing here needs git-ticket. The patches add ordinary Markdown files, so\n")
	b.WriteString("a repository that runs git-ticket gains real tickets and one that does not\n")
	b.WriteString("gains readable text in a directory.\n\n")
	b.WriteString("This file is not part of the series. It is named .txt so the glob above\n")
	b.WriteString("skips it, since a cover letter has no diff to apply.\n\n")
	b.WriteString("Numbers 0 and 1 are this export's. A sender attaching code uses\n")
	b.WriteString("`git format-patch --start-number 2 -o <this directory> <range>`, and the\n")
	b.WriteString("one command above still applies the whole of it.\n\n")
	for _, t := range tickets {
		b.WriteString(strings.Repeat("-", 72) + "\n")
		fmt.Fprintf(&b, "%s  [%s, %s, %s]\n\n", t.ID, t.Type, t.Status, t.Priority)
		fmt.Fprintf(&b, "%s\n", t.Title)
		if len(t.References) > 0 {
			b.WriteString("\nreferences:\n")
			for _, r := range t.References {
				fmt.Fprintf(&b, "  %s\n", r.Ref)
			}
		}
		b.WriteString("\n")
		if body, err := os.ReadFile(t.Path); err == nil {
			if raw, err := RawBody(body); err == nil {
				b.WriteString(strings.TrimSpace(raw) + "\n\n")
			}
		}
	}
	return b.String()
}

// plusBar is the width of a diffstat's bar. Real diffstat scales to the widest
// file in the set; this is cosmetic text in a patch nothing parses, so it is
// clamped instead of scaled.
//
// The ticket that moved the interchange here kept this in the CLI on the
// grounds that it is cosmetic. It came with the diffstat in the end, because
// the stat is part of the format the parser reads past, and splitting a
// formatter from the thing it formats buys nothing.
func plusBar(n int) int {
	if n > 40 {
		return 40
	}
	if n < 1 {
		return 1
	}
	return n
}

// relativeToRoot reports path relative to root, and whether it stayed inside
// it. A prefix test on ".." alone would also reject a sibling named "..foo".
func relativeToRoot(root, path string) (string, bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// exportPlural is the diffstat's counting, which is git's wording rather than
// this project's. It is here rather than shared with the CLI's own plural
// because a change to how a command counts things must not move the bytes of
// an artifact another store parses.
func exportPlural(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
