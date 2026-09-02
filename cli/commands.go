package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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

// intList collects a repeated numeric flag, such as --check. Without it the
// last value would win, and `--check 1 --check 3` would silently tick one box.
type intList []int

func (l *intList) String() string {
	parts := make([]string, len(*l))
	for i, v := range *l {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}

func (l *intList) Set(v string) error {
	n, err := strconv.Atoi(v)
	if err != nil {
		return fmt.Errorf("%q is not a number", v)
	}
	*l = append(*l, n)
	return nil
}

// abbrevLen is the fewest characters of a ULID a listing ever shows. Four is
// the minimum a prefix may be, per 5.5, and eight still looks like an ID.
const abbrevLen = 8

// shortestUnique maps each ID to the fewest characters that still resolve to
// it, never fewer than abbrevLen.
//
// A fixed width cannot do this. A ULID opens with ten characters of timestamp,
// so two tickets created in the same millisecond are identical that far in, and
// a listing printing eight shows one abbreviation on two rows. That tells the
// reader to type something that comes back ambiguous_id. Shortening to what is
// actually unique is git's rule for object hashes, which 5.5 already invokes
// for prefixes.
func shortestUnique(ids []string) map[string]string {
	bodies := make([]string, 0, len(ids))
	for _, id := range ids {
		bodies = append(bodies, ticket.NormalizeRef(id))
	}
	sorted := append([]string{}, bodies...)
	sort.Strings(sorted)

	// In sorted order the longest prefix an ID shares with any other is shared
	// with one of its two neighbours, so the pairs are enough.
	need := make(map[string]int, len(sorted))
	for i, b := range sorted {
		n := abbrevLen
		if i > 0 {
			n = max(n, commonPrefixLen(b, sorted[i-1])+1)
		}
		if i+1 < len(sorted) {
			n = max(n, commonPrefixLen(b, sorted[i+1])+1)
		}
		need[b] = n
	}

	out := make(map[string]string, len(ids))
	for i, id := range ids {
		b := bodies[i]
		n := min(need[b], len(b))
		out[id] = ticket.IDPrefix + b[:n]
	}
	return out
}

func commonPrefixLen(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}

// storeAbbreviations shortens against every ticket in the store, archived ones
// included, because what a listing prints gets pasted into a command that
// resolves against all of them.
//
// A failure here costs only brevity, so it falls back to full IDs rather than
// failing a read the caller asked for.
func storeAbbreviations(s *ticket.Store) map[string]string {
	all, err := s.List(context.Background(), ticket.Filter{IncludeArchived: true})
	if err != nil {
		return nil
	}
	ids := make([]string, 0, len(all))
	for _, t := range all {
		ids = append(ids, t.ID)
	}
	return shortestUnique(ids)
}

// runInit creates a store in the repository, or wherever --store points.
func runInit(ctx *cmdContext, args []string) error {
	var writeInstructions bool
	rest, err := ctx.parseFlags("init", args, func(fs *flag.FlagSet) {
		fs.BoolVar(&writeInstructions, "instructions", false,
			"also write the agent workflow block to "+instructionsFile)
	})
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

	// Checked before the store is made, so a refusal leaves no half-built
	// store behind for the user to clean up.
	if writeInstructions {
		if err := checkInstructionsFile(root); err != nil {
			return err
		}
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
	var instructions instructionsAction
	if writeInstructions {
		path, action, err := writeInstructionsFile(root)
		if err != nil {
			return err
		}
		instructions = action
		if action != instructionsCurrent {
			written = append(written, path)
		}
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
	if writeInstructions {
		fmt.Fprintf(ctx.out, "%s\n", instructions.sentence(displayPath(s, filepath.Join(root, instructionsFile))))
	}
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
		plan        string
		milestone   string
		parent      string
		labels      stringList
		assignees   stringList
		dependsOn   stringList
		ac          stringList
		dod         stringList
	)
	rest, err := ctx.parseFlags("create", args, func(fs *flag.FlagSet) {
		fs.StringVar(&title, "title", "", "what the ticket is about")
		fs.StringVar(&kind, "type", "", "task, bug, chore, spike, or epic")
		fs.StringVar(&priority, "priority", "", "low, normal, high, or urgent")
		fs.StringVar(&description, "description", "", "the Description section")
		fs.StringVar(&plan, "plan", "", "the Implementation plan section")
		fs.StringVar(&milestone, "milestone", "", "a milestone")
		fs.StringVar(&parent, "parent", "", "the epic or ticket this belongs to")
		fs.Var(&labels, "label", "a label, repeatable")
		fs.Var(&assignees, "assignee", "an assignee, repeatable")
		fs.Var(&dependsOn, "depends-on", "a ticket this waits on, repeatable")
		fs.Var(&ac, "ac", "an acceptance criterion, repeatable")
		fs.Var(&dod, "dod", "a definition of done item, repeatable")
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
	// A dependency or a parent may be typed as a prefix, like every other ID
	// this CLI takes.
	deps := make([]string, 0, len(dependsOn))
	for _, d := range dependsOn {
		id, err := resolveID(s, d)
		if err != nil {
			return err
		}
		deps = append(deps, id)
	}
	opts := ticket.CreateOptions{
		Title:              title,
		Type:               kind,
		Priority:           priority,
		Description:        description,
		ImplementationPlan: plan,
		AcceptanceCriteria: ac,
		DefinitionOfDone:   dod,
		Labels:             labels,
		Assignees:          assignees,
		Dependencies:       deps,
		Actor:              ctx.actor(s),
	}
	if milestone != "" {
		opts.Milestone = &milestone
	}
	if parent != "" {
		id, err := resolveID(s, parent)
		if err != nil {
			return err
		}
		opts.Parent = &id
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

	// Readiness is derived from the whole store rather than from this ticket,
	// so it costs one read whichever output mode follows.
	ready, err := s.Readiness(context.Background())
	if err != nil {
		return err
	}

	if ctx.g.json {
		writeJSON(ctx.out, ticketEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "ticket",
			Ticket:        newTicketJSON(s, t, ready[t.ID]),
		})
		return nil
	}
	writeTicketHuman(ctx.out, s, t, ready[t.ID])
	return nil
}

// parentNone is what a caller types to ask for the tickets that have no parent,
// which is what a board needs for its top level. It cannot be mistaken for a
// reference because every ID begins with TKT-. The library spells the same
// thing as an empty string, per plan section 8.
const parentNone = "none"

// runList prints the tickets that match the filters.
func runList(ctx *cmdContext, args []string) error {
	var (
		status    stringList
		kind      stringList
		priority  stringList
		labels    stringList
		assignees stringList
		milestone stringList
		parent    stringList
		archived  bool
	)
	rest, err := ctx.parseFlags("list", args, func(fs *flag.FlagSet) {
		fs.Var(&status, "status", "a status to include, repeatable")
		fs.Var(&kind, "type", "a type to include, repeatable")
		fs.Var(&priority, "priority", "a priority to include, repeatable")
		fs.Var(&labels, "label", "a label to match, repeatable")
		fs.Var(&assignees, "assignee", "an assignee to match, repeatable")
		fs.Var(&milestone, "milestone", "a milestone to match, repeatable")
		fs.Var(&parent, "parent", "a parent whose children to list, or "+parentNone+" for tickets with no parent, repeatable")
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
	// A parent resolves like every other ID, so a prefix works and one that
	// names nothing fails with ticket_not_found. Without this an unresolvable
	// parent would return an empty list, which reads exactly like an epic that
	// has no children.
	parents := make([]string, 0, len(parent))
	for _, p := range parent {
		if p == parentNone {
			parents = append(parents, "")
			continue
		}
		id, err := resolveID(s, p)
		if err != nil {
			return err
		}
		parents = append(parents, id)
	}

	tickets, err := s.List(context.Background(), ticket.Filter{
		Status:          status,
		Type:            kind,
		Priority:        priority,
		Labels:          labels,
		Assignees:       assignees,
		Milestone:       milestone,
		Parent:          parents,
		IncludeArchived: archived,
	})
	if err != nil {
		return err
	}

	return ctx.writeTicketList(s, tickets, "No tickets match.")
}

// resolveID turns a user-typed reference into the canonical ID the library
// takes.
//
// Plan 5.5 says any command taking an ID accepts a unique prefix, and the
// library's mutations take full IDs, so the CLI is where one becomes the other.
// Doing it here also means a prefix that names nothing fails with
// ticket_not_found before any write is attempted.
func resolveID(s *ticket.Store, ref string) (string, error) {
	t, err := s.Get(context.Background(), ref)
	if err != nil {
		return "", err
	}
	return t.ID, nil
}

// runUpdate changes the fields of a ticket, per plan 12.1.
//
// Everything the caller asked for lands as one write, or none of it does. A
// half-applied update would leave a ticket in a state nobody typed.
func runUpdate(ctx *cmdContext, args []string) error {
	var (
		fs          *flag.FlagSet
		title       string
		kind        string
		priority    string
		description string
		milestone   string
		parent      string
		addLabels   stringList
		rmLabels    stringList
		assign      stringList
		unassign    stringList
	)
	rest, err := ctx.parseFlags("update", args, func(f *flag.FlagSet) {
		fs = f
		f.StringVar(&title, "title", "", "a new title")
		f.StringVar(&kind, "type", "", "task, bug, chore, spike, or epic")
		f.StringVar(&priority, "priority", "", "low, normal, high, or urgent")
		f.StringVar(&description, "description", "", "a new Description section")
		f.StringVar(&milestone, "milestone", "", "a milestone, or empty to clear it")
		f.StringVar(&parent, "parent", "", "the epic or ticket this belongs to, or empty to clear it")
		f.Var(&addLabels, "add-label", "a label to add, repeatable")
		f.Var(&rmLabels, "remove-label", "a label to remove, repeatable")
		f.Var(&assign, "assign", "an assignee to add, repeatable")
		f.Var(&unassign, "unassign", "an assignee to remove, repeatable")
	})
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usageErr("update takes one ticket ID and at least one flag")
	}

	// --milestone "" clears the milestone and no --milestone at all leaves it
	// alone, so the zero value cannot stand in for absence.
	given := flagsGiven(fs)

	if priority != "" && !ticket.ValidPriority(priority) {
		return usageErr("%q is not one of %s", priority, strings.Join(ticket.Priorities, ", "))
	}
	if kind != "" && !ticket.ValidType(kind) {
		return usageErr("%q is not one of %s", kind, strings.Join(ticket.Types, ", "))
	}

	// Removals run before additions, so --remove-label x --add-label x ends
	// with the label present whichever order they were typed in.
	var ms ticket.Mutations
	if given["title"] {
		ms = append(ms, ticket.SetTitle{Title: title})
	}
	if given["type"] {
		ms = append(ms, ticket.SetType{Type: kind})
	}
	if given["priority"] {
		ms = append(ms, ticket.SetPriority{Priority: priority})
	}
	if given["description"] {
		ms = append(ms, ticket.SetDescription{Text: description})
	}
	if given["milestone"] {
		ms = append(ms, ticket.SetMilestone{Milestone: clearable(milestone)})
	}
	for _, l := range rmLabels {
		ms = append(ms, ticket.RemoveLabel{Label: l})
	}
	for _, l := range addLabels {
		ms = append(ms, ticket.AddLabel{Label: l})
	}
	for _, a := range unassign {
		ms = append(ms, ticket.Unassign{Actor: a})
	}
	for _, a := range assign {
		ms = append(ms, ticket.Assign{Actor: a})
	}
	if len(ms) == 0 && !given["parent"] {
		return usageErr("update needs something to change; run `git ticket help`")
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}

	// A parent is resolved last because it is the one field whose value has to
	// be looked up in the store, and a person types it as a prefix like any
	// other ID. An empty --parent clears it and resolves nothing.
	if given["parent"] {
		var to *string
		if parent != "" {
			id, err := resolveID(s, parent)
			if err != nil {
				return err
			}
			to = &id
		}
		ms = append(ms, ticket.SetParent{Parent: to})
	}

	res, err := ctx.applyTo(s, rest[0], ms)
	if err != nil {
		return err
	}
	return ctx.writeMutation(s, res, fmt.Sprintf("%s updated: %s",
		res.Ticket.ID, strings.Join(changedFields(given, rmLabels, addLabels, unassign, assign), ", ")))
}

// clearable turns an empty flag value into the nil that clears a field.
func clearable(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

// changedFields names what an update touched, so the one human line says what
// happened rather than only that something did.
func changedFields(given map[string]bool, rmLabels, addLabels, unassign, assign stringList) []string {
	var out []string
	for _, name := range []string{"title", "type", "priority", "description", "milestone", "parent"} {
		if given[name] {
			out = append(out, name)
		}
	}
	if len(rmLabels)+len(addLabels) > 0 {
		out = append(out, "labels")
	}
	if len(unassign)+len(assign) > 0 {
		out = append(out, "assignees")
	}
	return out
}

// runLink adds a dependency or a reference, per plan 12.1.
func runLink(ctx *cmdContext, args []string) error {
	var (
		fs        *flag.FlagSet
		dependsOn string
		ref       string
		path      string
	)
	rest, err := ctx.parseFlags("link", args, func(f *flag.FlagSet) {
		fs = f
		f.StringVar(&dependsOn, "depends-on", "", "a ticket this one waits on")
		f.StringVar(&ref, "ref", "", "a typed identifier, such as proposal:git-ticket")
		f.StringVar(&path, "path", "", "a repository-relative path, with --ref")
	})
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usageErr("link takes one ticket ID")
	}
	given := flagsGiven(fs)
	op, err := exactlyOne(given, "link", "depends-on", "ref")
	if err != nil {
		return err
	}
	if given["path"] && op != "ref" {
		return usageErr("--path goes with --ref, which names the thing the path points at")
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}

	var (
		m    ticket.Mutation
		what string
	)
	if op == "depends-on" {
		id, err := resolveID(s, dependsOn)
		if err != nil {
			return err
		}
		m = ticket.AddDependency{ID: id}
		what = "depends on " + id
	} else {
		m = ticket.AddReference{Ref: ref, Path: clearable(path)}
		what = "references " + ref
		if path != "" {
			what += " at " + path
		}
	}

	res, err := ctx.applyTo(s, rest[0], m)
	if err != nil {
		return err
	}
	return ctx.writeMutation(s, res, fmt.Sprintf("%s %s", res.Ticket.ID, what))
}

// runUnlink removes a dependency or a reference.
//
// Removing something that is not there succeeds and changes nothing but
// updated_at, because what the caller asked for is already true.
func runUnlink(ctx *cmdContext, args []string) error {
	var (
		fs        *flag.FlagSet
		dependsOn string
		ref       string
	)
	rest, err := ctx.parseFlags("unlink", args, func(f *flag.FlagSet) {
		fs = f
		f.StringVar(&dependsOn, "depends-on", "", "a dependency to drop")
		f.StringVar(&ref, "ref", "", "a reference to drop")
	})
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usageErr("unlink takes one ticket ID")
	}
	op, err := exactlyOne(flagsGiven(fs), "unlink", "depends-on", "ref")
	if err != nil {
		return err
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}

	var m ticket.Mutation
	what := "no longer references " + ref
	if op == "depends-on" {
		// A prefix resolves against what this ticket actually depends on,
		// rather than against the store. A dependency naming a ticket that no
		// longer exists is exactly the dependency_missing that check reports,
		// and unlink is how it gets repaired, so it has to stay removable.
		t, err := s.Get(context.Background(), rest[0])
		if err != nil {
			return err
		}
		id := dependsOn
		if resolved, err := ticket.ResolveRef(dependsOn, t.Dependencies); err == nil {
			id = resolved
		}
		m = ticket.RemoveDependency{ID: id}
		what = "no longer depends on " + id
	} else {
		m = ticket.RemoveReference{Ref: ref}
	}

	res, err := ctx.applyTo(s, rest[0], m)
	if err != nil {
		return err
	}
	return ctx.writeMutation(s, res, fmt.Sprintf("%s %s", res.Ticket.ID, what))
}

// runDeps prints what a ticket waits on, or what waits on it.
func runDeps(ctx *cmdContext, args []string) error {
	var transitive, dependents bool
	rest, err := ctx.parseFlags("deps", args, func(f *flag.FlagSet) {
		f.BoolVar(&transitive, "transitive", false, "follow the graph, not just the direct edges")
		f.BoolVar(&dependents, "dependents", false, "what waits on this ticket, rather than what it waits on")
	})
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usageErr("deps takes one ticket ID")
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	tickets, err := s.Deps(context.Background(), rest[0], ticket.DepsOptions{
		Transitive: transitive,
		Dependents: dependents,
	})
	if err != nil {
		return err
	}

	// An empty list does not say which way it looked, so the message does.
	empty := "It depends on nothing."
	if dependents {
		empty = "Nothing depends on it."
	}
	// An epic answering "nothing" is true and useless, per plan section 8. When
	// the answer is empty and the ticket has children, name them and the command
	// that lists them. deps still walks dependencies and nothing else: this is a
	// pointer to the other command, not a second edge kind in the result. The
	// lookups happen here rather than above so that they cost nothing on the
	// ordinary path, and --json never pays for them at all.
	if len(tickets) == 0 && !ctx.g.json {
		if hint := childHint(s, rest[0]); hint != "" {
			empty += " " + hint
		}
	}
	return ctx.writeTicketList(s, tickets, empty)
}

// childHint describes a ticket's children, to add to an otherwise empty deps
// answer.
//
// It returns an empty string when the ticket has none, and also when anything
// goes wrong. A hint is a courtesy and must never turn a working command into a
// failing one. The count comes from the same filter the message tells the
// caller to run, so the number they are given is the number they will see.
func childHint(s *ticket.Store, ref string) string {
	id, err := resolveID(s, ref)
	if err != nil {
		return ""
	}
	kids, err := s.List(context.Background(), ticket.Filter{Parent: []string{id}})
	if err != nil || len(kids) == 0 {
		return ""
	}
	noun, pronoun := "children", "them"
	if len(kids) == 1 {
		noun, pronoun = "child", "it"
	}
	return fmt.Sprintf("It has %d %s; run git ticket list --parent %s to see %s.",
		len(kids), noun, id, pronoun)
}

// runSearch looks through the title, the body sections, and the references,
// per plan section 8. The frontmatter beyond the title is deliberately not
// searched, so looking for "task" does not return every ticket of that type.
func runSearch(ctx *cmdContext, args []string) error {
	var regex bool
	rest, err := ctx.parseFlags("search", args, func(f *flag.FlagSet) {
		f.BoolVar(&regex, "regex", false, "read the query as an RE2 regular expression")
	})
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usageErr("search takes one query; quote it if it has spaces")
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	tickets, err := s.Search(context.Background(), ticket.Query{Text: rest[0], Regex: regex})
	if err != nil {
		return err
	}
	return ctx.writeTicketList(s, tickets, "Nothing matches.")
}

// runReady lists what could be started now: status ready, no live claim, and
// every dependency satisfied, per plan section 8.
func runReady(ctx *cmdContext, args []string) error {
	rest, err := ctx.parseFlags("ready", args, nil)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usageErr("ready takes no arguments")
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	tickets, err := s.Ready(context.Background())
	if err != nil {
		return err
	}
	return ctx.writeTicketList(s, tickets, "Nothing is ready to pick up.")
}

// runFiles finds the tickets that recorded a reference to a path.
//
// This reads what agents wrote and is only as complete as they were. It is
// advisory and is not derived from Git history, which the help text says too.
func runFiles(ctx *cmdContext, args []string) error {
	rest, err := ctx.parseFlags("files", args, nil)
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usageErr("files takes one repository-relative path")
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	tickets, err := s.Files(context.Background(), rest[0])
	if err != nil {
		return err
	}
	return ctx.writeTicketList(s, tickets, "No ticket recorded a reference to that path.")
}

// runNote, runComment, runPlan, and runSummary each take an ID and one piece
// of text. The first two append; a plan and a summary replace, per plan section
// 9, because each is one statement rather than a log and Notes is already the
// log. A plan says how the work will go and a summary says where it landed.
func runNote(ctx *cmdContext, args []string) error {
	return runTextEntry(ctx, "note", args, "noted on",
		func(text string) ticket.Mutation { return ticket.AppendNote{Text: text} })
}

func runComment(ctx *cmdContext, args []string) error {
	return runTextEntry(ctx, "comment", args, "commented on",
		func(text string) ticket.Mutation { return ticket.AppendComment{Text: text} })
}

func runPlan(ctx *cmdContext, args []string) error {
	return runTextEntry(ctx, "plan", args, "plan set on",
		func(text string) ticket.Mutation { return ticket.SetImplementationPlan{Text: text} })
}

func runSummary(ctx *cmdContext, args []string) error {
	return runTextEntry(ctx, "summary", args, "summary set on",
		func(text string) ticket.Mutation { return ticket.SetSummary{Text: text} })
}

func runTextEntry(ctx *cmdContext, name string, args []string, verb string,
	build func(string) ticket.Mutation) error {
	rest, err := ctx.parseFlags(name, args, nil)
	if err != nil {
		return err
	}
	if len(rest) != 2 {
		// Text starting with a dash looks like a flag, and -- is how a caller
		// says it is not.
		return usageErr("%s takes a ticket ID and the text; put text starting with a dash after --", name)
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	res, err := ctx.applyTo(s, rest[0], build(rest[1]))
	if err != nil {
		return err
	}
	return ctx.writeMutation(s, res, fmt.Sprintf("%s %s", verb, res.Ticket.ID))
}

// flagsGiven reports which flags were actually typed, as opposed to which hold
// a zero value. A zero value is a legal instruction in several places here, so
// emptiness cannot stand in for absence.
func flagsGiven(fs *flag.FlagSet) map[string]bool {
	given := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { given[f.Name] = true })
	return given
}

// exactlyOne picks the single operation a command was given, for a command
// whose flags are alternatives rather than a set.
//
// Two of them names both rather than resolving by precedence: a caller who
// typed two meant one, and silently picking one writes something nobody asked
// for.
func exactlyOne(given map[string]bool, command string, names ...string) (string, error) {
	var chosen []string
	for _, n := range names {
		if given[n] {
			chosen = append(chosen, n)
		}
	}
	switch len(chosen) {
	case 1:
		return chosen[0], nil
	case 0:
		return "", usageErr("%s needs one of %s", command, joinFlags(names, "or"))
	default:
		return "", usageErr("%s takes one of %s, not %s",
			command, joinFlags(names, "or"), joinFlags(chosen, "and"))
	}
}

// joinFlags renders a list of flag names as prose: "--a, --b, or --c".
func joinFlags(names []string, conjunction string) string {
	words := make([]string, 0, len(names))
	for _, n := range names {
		words = append(words, "--"+n)
	}
	switch len(words) {
	case 0:
		return ""
	case 1:
		return words[0]
	case 2:
		return words[0] + " " + conjunction + " " + words[1]
	default:
		return strings.Join(words[:len(words)-1], ", ") + ", " + conjunction + " " + words[len(words)-1]
	}
}

// runAC and runDoD are the same command over the two checklist sections of
// plan 5.2. They differ by which section they edit and by nothing else.
func runAC(ctx *cmdContext, args []string) error {
	return runChecklist(ctx, "ac", ticket.AcceptanceCriteria, args)
}

func runDoD(ctx *cmdContext, args []string) error {
	return runChecklist(ctx, "dod", ticket.DefinitionOfDone, args)
}

// runChecklist adds, checks, unchecks, and removes items, in any combination.
//
// The index counts checkbox lines from one, per plan 10.1, which is not the
// same as a line number or an array position: a section may hold prose above
// the list and still number its items 1, 2, 3.
//
// Every index means the item the caller read, and checklistOps orders the
// operations so that stays true. The whole set lands as one write or none of
// it does, which is the rule runUpdate already follows.
func runChecklist(ctx *cmdContext, name string, section ticket.ChecklistSection, args []string) error {
	var (
		add     stringList
		check   intList
		uncheck intList
		remove  intList
	)
	rest, err := ctx.parseFlags(name, args, func(f *flag.FlagSet) {
		f.Var(&add, "add", "append an unchecked item, repeatable")
		f.Var(&check, "check", "check item N, counting from 1, repeatable")
		f.Var(&uncheck, "uncheck", "uncheck item N, counting from 1, repeatable")
		f.Var(&remove, "remove", "remove item N, counting from 1, repeatable")
	})
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usageErr("%s takes one ticket ID", name)
	}
	if len(add)+len(check)+len(uncheck)+len(remove) == 0 {
		return usageErr("%s needs one of %s", name,
			joinFlags([]string{"add", "check", "uncheck", "remove"}, "or"))
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	res, err := ctx.applyTo(s, rest[0], checklistOps(section, add, check, uncheck, remove))
	if err != nil {
		return err
	}

	if ctx.g.json {
		return ctx.writeMutation(s, res, "")
	}
	// A person who just changed a checklist wants to see the checklist, with
	// the numbers the next command will take.
	fmt.Fprintf(ctx.out, "%s  %s\n", res.Ticket.ID, strings.ToLower(string(section)))
	writeChecklist(ctx.out, sectionText(res.Ticket, section))
	return nil
}

// checklistOps orders one invocation's operations so that every index means the
// item the caller read when they typed it.
//
// Checks and unchecks move nothing, so they go first and see the original
// numbering. Removals renumber everything below the item they drop, so they go
// highest first, which leaves every lower index still pointing at what it did.
// Adds append, so they cannot disturb an index and go last.
//
// A repeated removal index is applied once. Under the rule above, naming index
// 2 twice names the same item twice, so honouring both would delete an item the
// caller never pointed at.
func checklistOps(section ticket.ChecklistSection, add stringList, check, uncheck, remove intList) ticket.Mutations {
	ms := make(ticket.Mutations, 0, len(check)+len(uncheck)+len(remove)+len(add))
	for _, n := range check {
		ms = append(ms, ticket.SetChecklistItem{Section: section, Index: n, Checked: true})
	}
	for _, n := range uncheck {
		ms = append(ms, ticket.SetChecklistItem{Section: section, Index: n, Checked: false})
	}

	highestFirst := append([]int(nil), remove...)
	sort.Sort(sort.Reverse(sort.IntSlice(highestFirst)))
	seen := make(map[int]bool, len(highestFirst))
	for _, n := range highestFirst {
		if seen[n] {
			continue
		}
		seen[n] = true
		ms = append(ms, ticket.RemoveChecklistItem{Section: section, Index: n})
	}

	for _, text := range add {
		ms = append(ms, ticket.AddChecklistItem{Section: section, Text: text})
	}
	return ms
}

func sectionText(t *ticket.Ticket, section ticket.ChecklistSection) string {
	if section == ticket.DefinitionOfDone {
		return t.Body.DefinitionOfDone
	}
	return t.Body.AcceptanceCriteria
}

func writeChecklist(w io.Writer, text string) {
	items := ticket.Checklist(text)
	if len(items) == 0 {
		fmt.Fprintln(w, "  (empty)")
		return
	}
	tw := tabwriter.NewWriter(w, 0, 0, 1, ' ', 0)
	for i, item := range items {
		mark := " "
		if item.Checked {
			mark = "x"
		}
		fmt.Fprintf(tw, "  %d\t[%s]\t%s\n", i+1, mark, item.Text)
	}
	tw.Flush()
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
	var strict, fix, dryRun bool
	rest, err := ctx.parseFlags("check", args, func(fs *flag.FlagSet) {
		fs.BoolVar(&strict, "strict", false, "count a warning as a failure")
		fs.BoolVar(&fix, "fix", false, "repair the findings that have one correct repair")
		fs.BoolVar(&dryRun, "dry-run", false, "with --fix, report the repairs and make none")
	})
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usageErr("check takes no arguments; it validates the whole store")
	}
	// A dry run of a pass that was never asked for would report nothing and
	// look like a clean store, which is worse than saying so.
	if dryRun && !fix {
		return usageErr("--dry-run describes what --fix would do; pass --fix as well")
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}

	var report *ticket.Report
	var repairs []ticket.Repair
	if fix {
		res, err := s.Fix(context.Background(), ticket.FixOptions{DryRun: dryRun})
		if err != nil {
			return err
		}
		report, repairs = res.Report, res.Repairs
	} else {
		report, err = s.Check(context.Background())
		if err != nil {
			return err
		}
	}

	// --strict moves no finding between the arrays. It changes the verdict
	// only, per plan 10.3.
	ok := report.OK() && (!strict || len(report.Warnings) == 0)

	if ctx.g.json {
		writeJSON(ctx.out, newCheckEnvelope(s, report, ok, repairs, dryRun))
	} else {
		writeRepairsHuman(ctx.out, s, repairs, dryRun)
		writeCheckHuman(ctx.out, s, report, strict)
	}
	if !ok {
		return errReported
	}
	return nil
}

// writeRepairsHuman lists the moves ahead of the findings, so a reader sees
// what changed under them before they read what is left.
func writeRepairsHuman(w io.Writer, s *ticket.Store, repairs []ticket.Repair, dryRun bool) {
	if len(repairs) == 0 {
		return
	}
	verb := "moved"
	if dryRun {
		verb = "would move"
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, r := range repairs {
		fmt.Fprintf(tw, "%s\t%s\t-> %s\n", verb, storePath(s, r.From), storePath(s, r.To))
	}
	tw.Flush()
	fmt.Fprintln(w)
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

// envelopeKinds is the list section 10 defines. It is written out rather than
// derived, because the kinds are a contract with consumers and not a fact about
// any Go type in this package.
var envelopeKinds = []string{
	"ticket", "ticket-list", "mutation-result", "check-report", "error",
	"schema", "instructions", "version",
}

// runSchema prints the values a consumer would otherwise have to read the plan
// to learn: the enums, the transition table, and every code it may be handed.
// It reads no store, so it answers before `init` and outside a repository.
func runSchema(ctx *cmdContext, args []string) error {
	rest, err := ctx.parseFlags("schema", args, nil)
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return usageErr("schema takes no arguments")
	}

	// The transition table comes from the library rather than a copy here, so
	// what this prints is what SetStatus will actually allow.
	transitions := make(map[string][]string, len(ticket.Statuses))
	for _, s := range ticket.Statuses {
		to := ticket.PermittedTransitions(s)
		if to == nil {
			to = []string{}
		}
		transitions[s] = to
	}

	// usage is the CLI's own and never comes from the library, per section 10,
	// so it is appended here rather than living in ticket.OperationCodes.
	errorCodes := append(append([]string{}, ticket.OperationCodes...), codeUsage)

	findings := make([]findingCodeJSON, 0,
		len(ticket.CheckErrorCodes)+len(ticket.CheckWarningCodes))
	for _, c := range ticket.CheckErrorCodes {
		findings = append(findings, findingCodeJSON{Code: c, Severity: "error"})
	}
	for _, c := range ticket.CheckWarningCodes {
		findings = append(findings, findingCodeJSON{Code: c, Severity: "warning"})
	}

	if ctx.g.json {
		writeJSON(ctx.out, schemaEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "schema",
			TicketSchema:  ticket.SchemaVersion,
			Kinds:         envelopeKinds,
			Statuses:      ticket.Statuses,
			Types:         ticket.Types,
			Priorities:    ticket.Priorities,
			Transitions:   transitions,
			ErrorCodes:    errorCodes,
			FindingCodes:  findings,
		})
		return nil
	}

	fmt.Fprintf(ctx.out, "envelope schema %d, ticket schema %d\n\n", schemaVersion, ticket.SchemaVersion)
	fmt.Fprintf(ctx.out, "kinds       %s\n", strings.Join(envelopeKinds, " "))
	fmt.Fprintf(ctx.out, "statuses    %s\n", strings.Join(ticket.Statuses, " "))
	fmt.Fprintf(ctx.out, "types       %s\n", strings.Join(ticket.Types, " "))
	fmt.Fprintf(ctx.out, "priorities  %s\n", strings.Join(ticket.Priorities, " "))

	fmt.Fprintln(ctx.out, "\ntransitions")
	tw := tabwriter.NewWriter(ctx.out, 0, 0, 2, ' ', 0)
	for _, s := range ticket.Statuses {
		to := transitions[s]
		if len(to) == 0 {
			fmt.Fprintf(tw, "  %s\tnowhere\n", s)
			continue
		}
		fmt.Fprintf(tw, "  %s\t%s\n", s, strings.Join(to, " "))
	}
	tw.Flush()

	fmt.Fprintln(ctx.out, "\nerror codes")
	for _, c := range errorCodes {
		fmt.Fprintf(ctx.out, "  %s\n", c)
	}

	fmt.Fprintln(ctx.out, "\ncheck findings")
	tw = tabwriter.NewWriter(ctx.out, 0, 0, 2, ' ', 0)
	for _, f := range findings {
		fmt.Fprintf(tw, "  %s\t%s\n", f.Code, f.Severity)
	}
	return tw.Flush()
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

// writeTicketList reports a read that answers with tickets. Every such command
// emits the same kind, and each supplies its own line for an empty answer,
// because "nothing" means something different to search and to ready.
func (ctx *cmdContext) writeTicketList(s *ticket.Store, tickets []*ticket.Ticket, empty string) error {
	if ctx.g.json {
		// One call for the whole listing. Readiness reads the store to resolve
		// dependencies, so asking per row would re-read it per row.
		ready, err := s.Readiness(context.Background())
		if err != nil {
			return err
		}
		out := make([]*ticketJSON, 0, len(tickets))
		for _, t := range tickets {
			out = append(out, newTicketJSON(s, t, ready[t.ID]))
		}
		// A query leaves out a file it could not parse, per plan section 8, so
		// the envelope has to say which ones. A host building a board on this
		// cannot otherwise tell a short listing from a complete one.
		broken, err := s.Unreadable(context.Background())
		if err != nil {
			return err
		}
		writeJSON(ctx.out, ticketListEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "ticket-list",
			Tickets:       out,
			Unreadable:    findings(s, broken),
		})
		return nil
	}
	if len(tickets) == 0 {
		fmt.Fprintln(ctx.out, empty)
		return nil
	}
	writeListHuman(ctx.out, tickets, storeAbbreviations(s))
	return nil
}

func writeListHuman(w io.Writer, tickets []*ticket.Ticket, short map[string]string) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, t := range tickets {
		id, ok := short[t.ID]
		if !ok {
			id = t.ID
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", id, t.Status, t.Priority, t.Title)
	}
	tw.Flush()
}

func writeTicketHuman(w io.Writer, s *ticket.Store, t *ticket.Ticket, ready ticket.Readiness) {
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
	// Say what is in the way rather than leaving a reader to compare the
	// dependency list against the status of each one by hand.
	field("blocked by", strings.Join(ready.Blocking, ", "))
	// Named apart from the line above because the repair is different: a
	// blocking dependency needs finishing, and one nothing resolves needs
	// either the ticket it names or an unlink.
	field("unresolved", strings.Join(ready.Missing, ", "))
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
