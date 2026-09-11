package cli

import (
	"context"
	"crypto/sha1"
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
// It is always written, even when --patch supplies code. A bare report and a
// code contribution are then the same artifact with different contents, and the
// receiving side has one thing to do rather than two.
const exportTicketPatch = "0001-tickets.patch"

// mboxFromLine opens every message. git's mailsplit needs it to find where one
// message ends and the next begins, and the value is conventional: format-patch
// writes the commit it came from, and a synthesised message has none, so this is
// zeroes. The date is the fixed one git itself writes, which is not a timestamp
// and is not read as one.
const mboxFromLine = "From 0000000000000000000000000000000000000000 Mon Sep 17 00:00:00 2001"

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

	tickets := make([]*ticket.Ticket, 0, len(rest))
	for _, id := range rest {
		t, err := s.Get(context.Background(), id)
		if err != nil {
			return err
		}
		tickets = append(tickets, t)
	}

	dir := out
	if dir == "" {
		dir = tickets[0].ID + "-export"
	}
	if err := exportDirIsFree(dir); err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// The sender's git identity, not the ticket's actor. An actor is a session
	// and a From: header is a person, and it is the person sending who answers
	// for what arrives.
	who := exportIdentity(ctx.env.Dir)
	when := time.Now()
	if ctx.env.Now != nil {
		when = ctx.env.Now()
	}

	written := make([]string, 0, 3)

	body, err := exportTicketFiles(s, tickets)
	if err != nil {
		return err
	}
	ticketPath := filepath.Join(dir, exportTicketPatch)
	if err := os.WriteFile(ticketPath, []byte(mboxMessage(who, when, exportSubject(tickets), exportCommitBody(tickets), body)), 0o644); err != nil {
		return err
	}
	written = append(written, ticketPath)

	coverPath := filepath.Join(dir, exportCoverName)
	if err := os.WriteFile(coverPath, []byte(exportCover(who, when, tickets, len(written))), 0o644); err != nil {
		return err
	}
	written = append([]string{coverPath}, written...)

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
	warnDanglingEdges(ctx, tickets)
	fmt.Fprintf(ctx.env.Stderr, "apply with: git am %s/*.patch\n", dir)
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

// exportSubject is the commit subject the receiving log will carry. One ticket
// lends its title, because that is what a person recognises; several cannot, so
// they are counted instead.
func exportSubject(tickets []*ticket.Ticket) string {
	if len(tickets) == 1 {
		return "Ticket: " + tickets[0].Title
	}
	return fmt.Sprintf("Tickets: %d from another store", len(tickets))
}

// exportCommitBody says what the commit adds. It stays short on purpose: the
// ticket text is in the diff directly below it, and repeating it there would
// double the size of every export to no end.
func exportCommitBody(tickets []*ticket.Ticket) string {
	var b strings.Builder
	b.WriteString("Adds the ticket files below. The status of each decides the directory it\n")
	b.WriteString("lands in, so the store is consistent the moment this applies.\n\n")
	for _, t := range tickets {
		fmt.Fprintf(&b, "  %s  %s  %s\n", t.ID, t.Status, t.Title)
	}
	return b.String()
}

// exportTicketFiles builds the diff that adds every ticket file.
func exportTicketFiles(s *ticket.Store, tickets []*ticket.Ticket) (string, error) {
	root := s.Root()
	var diff, stat strings.Builder
	adds := 0
	for _, t := range tickets {
		data, err := os.ReadFile(t.Path)
		if err != nil {
			return "", err
		}
		rel := t.Path
		if r, ok := relativeTo(root, t.Path); ok && root != "" {
			rel = r
		}
		rel = filepath.ToSlash(rel)
		n := addedFileDiff(&diff, rel, data)
		adds += n
		fmt.Fprintf(&stat, " %s | %d %s\n", rel, n, strings.Repeat("+", plusBar(n)))
	}
	var b strings.Builder
	b.WriteString(stat.String())
	fmt.Fprintf(&b, " %s changed, %s(+)\n", plural(len(tickets), "file"), plural(adds, "insertion"))
	for _, t := range tickets {
		rel := t.Path
		if r, ok := relativeTo(root, t.Path); ok && root != "" {
			rel = r
		}
		fmt.Fprintf(&b, " create mode 100644 %s\n", filepath.ToSlash(rel))
	}
	b.WriteString("\n")
	b.WriteString(diff.String())
	return b.String(), nil
}

// addedFileDiff writes one new-file hunk and returns the line count.
//
// A file with no trailing newline gets git's own marker, because a patch that
// silently adds one changes the blob and the index line then disagrees with what
// applies.
func addedFileDiff(b *strings.Builder, rel string, data []byte) int {
	lines := strings.Split(string(data), "\n")
	trailing := len(lines) > 0 && lines[len(lines)-1] == ""
	if trailing {
		lines = lines[:len(lines)-1]
	}
	fmt.Fprintf(b, "diff --git a/%s b/%s\n", rel, rel)
	b.WriteString("new file mode 100644\n")
	fmt.Fprintf(b, "index 0000000000000000000000000000000000000000..%s\n", blobSHA(data))
	b.WriteString("--- /dev/null\n")
	fmt.Fprintf(b, "+++ b/%s\n", rel)
	fmt.Fprintf(b, "@@ -0,0 +1,%d @@\n", len(lines))
	for _, l := range lines {
		b.WriteString("+" + l + "\n")
	}
	if !trailing {
		b.WriteString("\\ No newline at end of file\n")
	}
	return len(lines)
}

// blobSHA is the object name git would give this content. Computing it here
// keeps the index line honest without asking git to hash anything, and an honest
// index line is what lets `git apply --3way` fall back on the object store.
func blobSHA(data []byte) string {
	h := sha1.New()
	fmt.Fprintf(h, "blob %d", len(data))
	h.Write([]byte{0})
	h.Write(data)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// plusBar is the width of a diffstat's bar. Real diffstat scales to the widest
// file in the set; this is cosmetic text in a patch nothing parses, so it is
// clamped instead of scaled.
func plusBar(n int) int {
	if n > 40 {
		return 40
	}
	if n < 1 {
		return 1
	}
	return n
}

// mboxMessage wraps a subject, a body and a diff as one mail message.
func mboxMessage(who string, when time.Time, subject, body, diff string) string {
	var b strings.Builder
	b.WriteString(mboxFromLine + "\n")
	fmt.Fprintf(&b, "From: %s\n", who)
	fmt.Fprintf(&b, "Date: %s\n", when.Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Subject: [PATCH] %s\n", strings.ReplaceAll(subject, "\n", " "))
	b.WriteString("\n")
	b.WriteString(body)
	if !strings.HasSuffix(body, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("---\n")
	b.WriteString(diff)
	b.WriteString("-- \ngit-ticket\n\n")
	return b.String()
}

// exportCover is the report half: what this is, how to apply it, and what it
// carries. It is written for somebody who has never run git-ticket, because the
// first person to receive one of these will not have it.
func exportCover(who string, when time.Time, tickets []*ticket.Ticket, patches int) string {
	var b strings.Builder
	b.WriteString(mboxFromLine + "\n")
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
			if raw, err := ticket.RawBody(body); err == nil {
				b.WriteString(strings.TrimSpace(raw) + "\n\n")
			}
		}
	}
	return b.String()
}
