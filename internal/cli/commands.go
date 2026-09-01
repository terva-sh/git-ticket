package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/terva-sh/git-ticket/ticket"
)

// stringList collects a flag a caller may repeat, such as --label. Within one
// filter the values are alternatives, per plan section 8.
type stringList []string

func (l *stringList) String() string { return strings.Join(*l, ",") }

func (l *stringList) Set(v string) error {
	if v == "" {
		return fmt.Errorf("an empty value")
	}
	*l = append(*l, v)
	return nil
}

// abbrevLen is how much of a ULID a listing shows. It is well past the four
// characters a prefix needs, so what a person copies out of a listing resolves,
// and it is short enough that the title still fits on a line.
const abbrevLen = 8

func abbreviate(id string) string {
	if len(id) <= len(ticket.IDPrefix)+abbrevLen {
		return id
	}
	return id[:len(ticket.IDPrefix)+abbrevLen]
}

// runInit creates a store in the repository, or wherever --store points.
func runInit(ctx *cmdContext, args []string) error {
	rest, err := ctx.parseFlags("init", args, nil)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usageErr("init takes no arguments; use --store to choose where")
	}

	// init is the one command that cannot open the store first, since the
	// point is that there is not one yet. --store names the store directory
	// itself, so the root is its parent.
	root := ctx.env.Dir
	if ctx.g.store != "" {
		root = filepath.Dir(strings.TrimRight(ctx.g.store, "/"))
	} else if fromEnv := ctx.env.getenv("GIT_TICKET_STORE"); fromEnv != "" {
		root = filepath.Dir(strings.TrimRight(fromEnv, "/"))
	}

	s, err := ticket.Init(root, ticket.InitOptions{
		Actor: ticket.Actor{ID: ctx.g.actor},
		Now:   ctx.env.Now,
	})
	if err != nil {
		return err
	}

	written := []string{
		filepath.Join(s.Path(), "config.yml"),
		filepath.Join(s.Path(), "README.md"),
	}
	if ctx.g.json {
		writeJSON(ctx.out, mutationEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "mutation-result",
			Ticket:        nil,
			PathsChanged:  displayPaths(s, written),
		})
		return nil
	}
	fmt.Fprintf(ctx.out, "Initialized a ticket store at %s\n", displayPath(s, s.Path()))
	fmt.Fprintf(ctx.out, "Commit it, then run `git ticket create --title \"...\"`.\n")
	return nil
}

// runCreate writes a new ticket, which always starts in draft.
func runCreate(ctx *cmdContext, args []string) error {
	var (
		title       string
		kind        string
		priority    string
		description string
		parent      string
		labels      stringList
		assignees   stringList
		dependsOn   stringList
	)
	rest, err := ctx.parseFlags("create", args, func(fs *flag.FlagSet) {
		fs.StringVar(&title, "title", "", "what the ticket is about")
		fs.StringVar(&kind, "type", "", "task, bug, chore, spike, or epic")
		fs.StringVar(&priority, "priority", "", "low, normal, high, or urgent")
		fs.StringVar(&description, "description", "", "the Description section")
		fs.StringVar(&parent, "parent", "", "the epic or ticket this belongs to")
		fs.Var(&labels, "label", "a label, repeatable")
		fs.Var(&assignees, "assignee", "an assignee, repeatable")
		fs.Var(&dependsOn, "depends-on", "a ticket this waits on, repeatable")
	})
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usageErr("create takes flags, not positional arguments; did you mean --title %q?", rest[0])
	}
	if title == "" {
		return usageErr("create needs --title")
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	opts := ticket.CreateOptions{
		Title:        title,
		Type:         kind,
		Priority:     priority,
		Description:  description,
		Labels:       labels,
		Assignees:    assignees,
		Dependencies: dependsOn,
		Actor:        ctx.actor(s),
	}
	if parent != "" {
		opts.Parent = &parent
	}

	res, err := s.Create(context.Background(), opts)
	if err != nil {
		return err
	}
	return ctx.writeMutation(s, res, fmt.Sprintf("Created %s  %s", res.Ticket.ID, res.Ticket.Title))
}

// runShow prints one ticket, found by ID or by a unique prefix.
func runShow(ctx *cmdContext, args []string) error {
	rest, err := ctx.parseFlags("show", args, nil)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usageErr("show takes one ticket ID")
	}
	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	t, err := s.Get(context.Background(), rest[0])
	if err != nil {
		return err
	}

	if ctx.g.json {
		writeJSON(ctx.out, ticketEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "ticket",
			Ticket:        newTicketJSON(s, t),
		})
		return nil
	}
	writeTicketHuman(ctx.out, s, t)
	return nil
}

// runList prints the tickets that match the filters.
func runList(ctx *cmdContext, args []string) error {
	var (
		status    stringList
		kind      stringList
		priority  stringList
		labels    stringList
		assignees stringList
		milestone stringList
		archived  bool
	)
	rest, err := ctx.parseFlags("list", args, func(fs *flag.FlagSet) {
		fs.Var(&status, "status", "a status to include, repeatable")
		fs.Var(&kind, "type", "a type to include, repeatable")
		fs.Var(&priority, "priority", "a priority to include, repeatable")
		fs.Var(&labels, "label", "a label to match, repeatable")
		fs.Var(&assignees, "assignee", "an assignee to match, repeatable")
		fs.Var(&milestone, "milestone", "a milestone to match, repeatable")
		fs.BoolVar(&archived, "archived", false, "include archived tickets")
	})
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usageErr("list takes filters, not positional arguments")
	}
	for _, v := range status {
		if !ticket.ValidStatus(v) {
			return usageErr("%q is not one of %s", v, strings.Join(ticket.Statuses, ", "))
		}
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	tickets, err := s.List(context.Background(), ticket.Filter{
		Status:          status,
		Type:            kind,
		Priority:        priority,
		Labels:          labels,
		Assignees:       assignees,
		Milestone:       milestone,
		IncludeArchived: archived,
	})
	if err != nil {
		return err
	}

	if ctx.g.json {
		out := make([]*ticketJSON, 0, len(tickets))
		for _, t := range tickets {
			out = append(out, newTicketJSON(s, t))
		}
		writeJSON(ctx.out, ticketListEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "ticket-list",
			Tickets:       out,
		})
		return nil
	}
	writeListHuman(ctx.out, tickets)
	return nil
}

// runStatus moves a ticket through the lifecycle of plan 6.2.
//
// The transition table lives in the library, so a refusal here is
// invalid_transition naming where the ticket may go instead.
func runStatus(ctx *cmdContext, args []string) error {
	var reason string
	rest, err := ctx.parseFlags("status", args, func(fs *flag.FlagSet) {
		fs.StringVar(&reason, "reason", "", "why; required entering blocked and reopening from done")
	})
	if err != nil {
		return err
	}
	if len(rest) != 2 {
		return usageErr("status takes a ticket ID and a status, one of %s",
			strings.Join(ticket.Statuses, ", "))
	}
	ref, want := rest[0], rest[1]
	if !ticket.ValidStatus(want) {
		return usageErr("%q is not one of %s", want, strings.Join(ticket.Statuses, ", "))
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	res, err := ctx.applyTo(s, ref, ticket.SetStatus{Status: want, Reason: reason})
	if err != nil {
		return err
	}
	return ctx.writeMutation(s, res, fmt.Sprintf("%s is now %s", res.Ticket.ID, res.Ticket.Status))
}

// runClaim records that an actor is working this ticket, per plan 6.4. A claim
// is metadata and not a status, and it is advisory: it reserves nothing.
func runClaim(ctx *cmdContext, args []string) error {
	var (
		expiresIn time.Duration
		force     bool
	)
	rest, err := ctx.parseFlags("claim", args, func(fs *flag.FlagSet) {
		fs.DurationVar(&expiresIn, "expires-in", 0, "how long the claim stands; the default is no expiry")
		fs.BoolVar(&force, "force", false, "take a live claim held by another actor")
	})
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usageErr("claim takes one ticket ID")
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	// Plan 6.4: the branch and worktree when they can be determined, and the
	// commit the claim was based on.
	branch, worktree, commit := gitState(ctx.env.Dir)
	res, err := ctx.applyTo(s, rest[0], ticket.ClaimTicket{
		Branch:    branch,
		Worktree:  worktree,
		Commit:    commit,
		ExpiresIn: expiresIn,
		Force:     force,
	})
	if err != nil {
		return err
	}

	human := fmt.Sprintf("%s claimed by %s", res.Ticket.ID, claimant(res.Ticket))
	if branch != "" {
		human += " on " + branch
	}
	return ctx.writeMutation(s, res, human)
}

func claimant(t *ticket.Ticket) string {
	if t.Claim == nil {
		return "nobody"
	}
	return t.Claim.Actor
}

// runRelease drops a claim. Releasing an unclaimed ticket succeeds and changes
// nothing but updated_at, because what the caller asked for is already true.
func runRelease(ctx *cmdContext, args []string) error {
	rest, err := ctx.parseFlags("release", args, nil)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usageErr("release takes one ticket ID")
	}
	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	res, err := ctx.applyTo(s, rest[0], ticket.ReleaseClaim{})
	if err != nil {
		return err
	}
	return ctx.writeMutation(s, res, fmt.Sprintf("%s released", res.Ticket.ID))
}

// runArchive archives a ticket, which also moves its file to archive/. It is
// its own command rather than a status, because the status alone would leave
// the file where it was.
func runArchive(ctx *cmdContext, args []string) error {
	var reason string
	rest, err := ctx.parseFlags("archive", args, func(fs *flag.FlagSet) {
		fs.StringVar(&reason, "reason", "", "why this is being archived")
	})
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usageErr("archive takes one ticket ID")
	}
	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	res, err := ctx.applyTo(s, rest[0], ticket.ArchiveTicket{Reason: reason})
	if err != nil {
		return err
	}
	from := ""
	if a := res.Ticket.Archive; a != nil && a.FromStatus != nil {
		from = ", archived from " + *a.FromStatus
	}
	return ctx.writeMutation(s, res, fmt.Sprintf("%s archived%s", res.Ticket.ID, from))
}

// runUnarchive restores an archived ticket to ready and moves the file back.
func runUnarchive(ctx *cmdContext, args []string) error {
	rest, err := ctx.parseFlags("unarchive", args, nil)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usageErr("unarchive takes one ticket ID")
	}
	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	res, err := ctx.applyTo(s, rest[0], ticket.UnarchiveTicket{})
	if err != nil {
		return err
	}
	return ctx.writeMutation(s, res, fmt.Sprintf("%s is now %s", res.Ticket.ID, res.Ticket.Status))
}

// gitState is what a claim records about where the work is happening, per plan
// 6.4. Every part is best effort: a directory outside a repository has none of
// it, a repository with no commits yet has no HEAD, and a detached HEAD has no
// branch name. A claim records what it can and leaves the rest null rather than
// failing, because the claim is the point and the provenance is a courtesy.
//
// Every command here reads. The CLI runs no git command that writes, per the
// policy in plan 7.3.
func gitState(dir string) (branch, worktree, commit string) {
	worktree = readGit(dir, "rev-parse", "--show-toplevel")
	if worktree == "" {
		// Outside a repository the other two cannot mean anything either.
		return "", "", ""
	}
	commit = readGit(dir, "rev-parse", "HEAD")
	// symbolic-ref fails on a detached HEAD, which is what tells the two apart.
	branch = readGit(dir, "symbolic-ref", "--short", "HEAD")
	return branch, worktree, commit
}

func readGit(dir string, args ...string) string {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// runCheck validates the whole store, per plan section 11.
//
// A store with findings is a successful check whose answer is no. The report
// goes to stdout either way and the status is one, so CI gates on the status
// while a person still sees what is wrong.
func runCheck(ctx *cmdContext, args []string) error {
	var strict bool
	rest, err := ctx.parseFlags("check", args, func(fs *flag.FlagSet) {
		fs.BoolVar(&strict, "strict", false, "count a warning as a failure")
	})
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usageErr("check takes no arguments; it validates the whole store")
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	report, err := s.Check(context.Background())
	if err != nil {
		return err
	}

	// --strict moves no finding between the arrays. It changes the verdict
	// only, per plan 10.3.
	ok := report.OK() && (!strict || len(report.Warnings) == 0)

	if ctx.g.json {
		writeJSON(ctx.out, newCheckEnvelope(s, report, ok))
	} else {
		writeCheckHuman(ctx.out, s, report, strict)
	}
	if !ok {
		return errReported
	}
	return nil
}

func writeCheckHuman(w io.Writer, s *ticket.Store, r *ticket.Report, strict bool) {
	if len(r.Errors) == 0 && len(r.Warnings) == 0 {
		fmt.Fprintln(w, "No problems found.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	line := func(severity string, f ticket.Finding) {
		field := f.Field
		if field == "" {
			field = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			severity, f.Code, displayPath(s, filepath.Join(s.Path(), f.File)), field, f.Message)
	}
	for _, f := range r.Errors {
		line("error", f)
	}
	for _, f := range r.Warnings {
		line("warning", f)
	}
	tw.Flush()

	fmt.Fprintf(w, "\n%s, %s\n", count(len(r.Errors), "error"), count(len(r.Warnings), "warning"))
	if strict && len(r.Errors) == 0 && len(r.Warnings) > 0 {
		fmt.Fprintln(w, "Failing because --strict counts a warning as a failure.")
	}
}

func count(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// writeMutation reports a mutation as the envelope or as one human line.
func (ctx *cmdContext) writeMutation(s *ticket.Store, res *ticket.Result, human string) error {
	if ctx.g.json {
		writeJSON(ctx.out, mutationEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "mutation-result",
			Ticket:        &mutationTicket{ID: res.Ticket.ID, Revision: res.Ticket.Revision},
			PathsChanged:  displayPaths(s, res.PathsChanged),
		})
		return nil
	}
	fmt.Fprintln(ctx.out, human)
	for _, p := range displayPaths(s, res.PathsChanged) {
		fmt.Fprintf(ctx.out, "  %s\n", p)
	}
	return nil
}

func writeListHuman(w io.Writer, tickets []*ticket.Ticket) {
	if len(tickets) == 0 {
		fmt.Fprintln(w, "No tickets match.")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, t := range tickets {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", abbreviate(t.ID), t.Status, t.Priority, t.Title)
	}
	tw.Flush()
}

func writeTicketHuman(w io.Writer, s *ticket.Store, t *ticket.Ticket) {
	fmt.Fprintf(w, "%s  %s\n", t.ID, t.Title)
	fmt.Fprintf(w, "%s  %s  %s\n", t.Status, t.Type, t.Priority)
	if t.StatusReason != nil {
		fmt.Fprintf(w, "because: %s\n", *t.StatusReason)
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	field := func(name, value string) {
		if value != "" {
			fmt.Fprintf(tw, "%s:\t%s\n", name, value)
		}
	}
	field("labels", strings.Join(t.Labels, ", "))
	field("assignees", strings.Join(t.Assignees, ", "))
	if t.Milestone != nil {
		field("milestone", *t.Milestone)
	}
	if t.Parent != nil {
		field("parent", *t.Parent)
	}
	field("depends on", strings.Join(t.Dependencies, ", "))
	if c := t.Claim; c != nil {
		held := c.Actor
		if c.Branch != nil {
			held += " on " + *c.Branch
		}
		field("claimed by", held)
	}
	if a := t.Archive; a != nil && a.FromStatus != nil {
		field("archived from", *a.FromStatus)
	}
	field("revision", t.Revision)
	field("file", displayPath(s, t.Path))
	tw.Flush()

	section := func(heading, text string) {
		if strings.TrimSpace(text) == "" {
			return
		}
		fmt.Fprintf(w, "\n## %s\n\n%s\n", heading, text)
	}
	section("Description", t.Body.Description)
	section("Acceptance criteria", t.Body.AcceptanceCriteria)
	section("Definition of done", t.Body.DefinitionOfDone)
	section("Implementation plan", t.Body.ImplementationPlan)
	section("Notes", t.Body.Notes)
	section("Comments", t.Body.Comments)
	section("Summary", t.Body.Summary)
	for _, extra := range t.Body.Extra {
		section(extra.Heading, extra.Text)
	}
}
