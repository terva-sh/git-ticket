package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/terva-sh/git-ticket/ticket"
)

// import is the half of the export pair that git cannot do alone.
//
// Where the two stores share a series, nothing here is needed: `git am
// *.patch` applies the export and the tickets are correct on arrival, which is
// what the export tests assert. This command exists for the case that fails —
// a ticket whose series the receiving store does not declare. `unknown_series`
// is an error rather than a warning, and the prefix is inside an immutable ID,
// so the only repair is to file the ticket again under a series this store
// knows. That is what importing means here: not applying a patch, but adopting
// its contents.
//
// It does not try to work out whether an export came from your own project.
// An earlier version decided that from the series, and it was wrong: `TKT` is
// the default every store has, so two strangers almost always share it and the
// guess said "same project" for the common case. The tool cannot know, so it
// asks. Without --adopt nothing is written and both routes are named; with it,
// every ticket is filed afresh under this store's own series.
//
// It runs no git at all, and moves no branch. Plan 7.3 is explicit
// that a sync helper "must never silently push, merge, switch branches, or
// rewrite a worktree", and an import that ran `git am` onto a scratch branch
// would do three of those four. Tickets are written through the same locked,
// atomic path every other write uses, and the result is staged for an ordinary
// `git commit` by the person who asked for it.

// runImport adopts the tickets an export carries into this store.
func runImport(ctx *cmdContext, args []string) error {
	var adopt bool
	var fromStore string
	rest, err := ctx.parseFlags("import", args, func(fs *flag.FlagSet) {
		fs.BoolVar(&adopt, "adopt", false, "file every ticket afresh under this store's series; without it nothing is written")
		fs.StringVar(&fromStore, "from-store", "", "the store this export came from, recorded as provenance")
	})
	if err != nil {
		return err
	}
	if len(rest) != 1 {
		return usageErr("import takes one export directory")
	}
	s, err := ctx.openStore()
	if err != nil {
		return err
	}

	incoming, err := readExportTickets(rest[0])
	if err != nil {
		return err
	}
	if len(incoming) == 0 {
		return fmt.Errorf("%s carries no tickets; it may be a patch series rather than an export", rest[0])
	}

	ordered, err := importOrder(incoming)
	if err != nil {
		return err
	}

	if !adopt {
		return importPreview(ctx, s, rest[0], ordered)
	}
	return importAdopt(ctx, s, ordered, fromStore)
}

// incomingTicket is one ticket read out of an export.
type incomingTicket struct {
	Ticket *ticket.Ticket
	// Path is where it sat in the sending store, which is the only thing that
	// says what its status was without trusting the frontmatter twice.
	Path string
}

// readExportTickets reads the ticket files an export carries.
//
// It parses the export's own patch rather than any patch: the format is this
// project's, produced by the command next door, and a general patch parser is a
// large thing to own for no gain. A file that is not in that shape is refused by
// name rather than guessed at.
func readExportTickets(dir string) ([]incomingTicket, error) {
	patch := filepath.Join(dir, exportTicketPatch)
	data, err := os.ReadFile(patch)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no %s in %s; import takes a directory written by export", exportTicketPatch, dir)
		}
		return nil, err
	}
	files, err := parseNewFileHunks(string(data))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", patch, err)
	}
	var out []incomingTicket
	for _, f := range files {
		t, err := ticket.Parse([]byte(f.body))
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", patch, f.path, err)
		}
		out = append(out, incomingTicket{Ticket: t, Path: f.path})
	}
	return out, nil
}

// newFile is one added file recovered from a patch.
type newFile struct {
	path string
	body string
}

// parseNewFileHunks recovers added files from the export's patch.
//
// Every hunk an export writes adds a whole new file, so there is no context to
// track and no deletion to apply: the body is the plus-prefixed lines with the
// prefix removed. The blob name on the index line is checked against the body
// that comes out, which is what turns a truncated or hand-edited patch into an
// error here rather than a puzzling ticket later.
func parseNewFileHunks(patch string) ([]newFile, error) {
	var out []newFile
	lines := strings.Split(patch, "\n")
	for i := 0; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "diff --git ") {
			continue
		}
		var path, want string
		var body strings.Builder
		newFileSeen := false
		for i++; i < len(lines); i++ {
			l := lines[i]
			switch {
			case l == "new file mode 100644":
				newFileSeen = true
			case strings.HasPrefix(l, "index ") && strings.Contains(l, ".."):
				want = strings.TrimSpace(l[strings.Index(l, "..")+2:])
			case strings.HasPrefix(l, "+++ b/"):
				path = strings.TrimPrefix(l, "+++ b/")
			case strings.HasPrefix(l, "@@"):
				// The hunk body runs to the next diff, the signature, or the end.
				for i++; i < len(lines); i++ {
					l := lines[i]
					if strings.HasPrefix(l, "diff --git ") || l == "-- " {
						i--
						break
					}
					if strings.HasPrefix(l, "+") {
						body.WriteString(l[1:] + "\n")
						continue
					}
					if strings.HasPrefix(l, "\\ No newline") {
						continue
					}
					if strings.TrimSpace(l) == "" {
						continue
					}
					i--
					break
				}
			}
			if path != "" && body.Len() > 0 {
				break
			}
		}
		if !newFileSeen || path == "" {
			continue
		}
		got := blobSHA([]byte(body.String()))
		if want != "" && got != want {
			return nil, fmt.Errorf("%s does not match its blob name (%s, expected %s); the patch has been altered or truncated", path, got[:12], want[:12])
		}
		out = append(out, newFile{path: path, body: body.String()})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no added files found")
	}
	return out, nil
}

// importOrder puts a ticket after everything it points at, so that a dependency
// or a parent inside the same export is already filed, and its new ID is known,
// by the time the ticket naming it is written.
//
// Edges to tickets outside the export are not ordering constraints; they are
// dropped at write time and reported, since a reminted store cannot resolve them.
func importOrder(in []incomingTicket) ([]incomingTicket, error) {
	byID := make(map[string]incomingTicket, len(in))
	for _, t := range in {
		byID[t.Ticket.ID] = t
	}
	var ordered []incomingTicket
	state := make(map[string]int) // 0 unseen, 1 visiting, 2 done
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case 2:
			return nil
		case 1:
			return fmt.Errorf("the export contains a dependency cycle at %s", id)
		}
		state[id] = 1
		cur := byID[id]
		edges := append([]string{}, cur.Ticket.Dependencies...)
		if cur.Ticket.Parent != nil && *cur.Ticket.Parent != "" {
			edges = append(edges, *cur.Ticket.Parent)
		}
		sort.Strings(edges)
		for _, e := range edges {
			if _, ok := byID[e]; ok {
				if err := visit(e); err != nil {
					return err
				}
			}
		}
		state[id] = 2
		ordered = append(ordered, cur)
		return nil
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := visit(id); err != nil {
			return nil, err
		}
	}
	return ordered, nil
}

// importPreview says what would happen and writes nothing.
//
// This is the default because an export is somebody else's content, and a
// command that ingests it without showing its hand first asks for trust it has
// not earned. --adopt is the sentence that grants it.
func importPreview(ctx *cmdContext, s *ticket.Store, dir string, in []incomingTicket) error {
	cfg := s.Config()
	shared := 0
	for _, t := range in {
		if series, _ := ticket.SplitID(t.Ticket.ID); cfg.KnownSeries(series) {
			shared++
		}
	}

	fmt.Fprintf(ctx.out, "%s carries %s\n\n", dir, plural(len(in), "ticket"))
	for _, t := range in {
		fmt.Fprintf(ctx.out, "  %s  %s\n", t.Ticket.ID, t.Ticket.Title)
		if _, dropped := keepKnownLabels(cfg, t.Ticket.Labels); len(dropped) > 0 {
			fmt.Fprintf(ctx.out, "    would drop labels: %s\n", strings.Join(dropped, ", "))
		}
		for _, r := range t.Ticket.References {
			if r.Path != nil && *r.Path != "" && !repoHasPath(s.Root(), *r.Path) {
				fmt.Fprintf(ctx.out, "    would drop the path on %s: not in this repository\n", r.Ref)
			}
		}
		for _, d := range importDroppedEdges(t, in) {
			fmt.Fprintf(ctx.out, "    would drop %s: not in this export\n", d)
		}
	}

	fmt.Fprintf(ctx.out, "\nNothing written. --adopt files all %s afresh under this store's series.\n", plural(len(in), "ticket"))
	// Advice, not a verdict. Whether this export is your own project's work is
	// something only you know: the series cannot say, because TKT is the default
	// every store has and two strangers share it by default.
	if shared > 0 {
		fmt.Fprintf(ctx.out, "\n%s already use a series this store declares. If this export is your own\nproject's work, `git am %s` keeps the original IDs and is the better route.\nIf it came from elsewhere, --adopt is correct even though the series matches.\n",
			plural(shared, "ticket"), filepath.Join(dir, "*.patch"))
	}
	if extra := importOtherPatches(dir); len(extra) > 0 {
		fmt.Fprintf(ctx.env.Stderr, "%s also carries %s of code, which import does not apply: git am %s\n",
			dir, plural(len(extra), "patch"), filepath.Join(dir, "*.patch"))
	}
	return nil
}

// importAdopt files each ticket afresh, rewriting the edges among them to the
// IDs this store just minted and dropping the vocabulary it does not share.
func importAdopt(ctx *cmdContext, s *ticket.Store, in []incomingTicket, fromStore string) error {
	cfg := s.Config()
	remap := make(map[string]string, len(in))
	var filed []string
	for _, t := range in {
		labels, droppedLabels := keepKnownLabels(cfg, t.Ticket.Labels)
		o := ticket.CreateOptions{
			Title:              t.Ticket.Title,
			Type:               t.Ticket.Type,
			Priority:           t.Ticket.Priority,
			Labels:             labels,
			Assignees:          t.Ticket.Assignees,
			Description:        t.Ticket.Body.Description,
			ImplementationPlan: t.Ticket.Body.ImplementationPlan,
			Actor:              ctx.actor(s),
		}
		for _, d := range t.Ticket.Dependencies {
			if to, ok := remap[d]; ok {
				o.Dependencies = append(o.Dependencies, to)
			}
		}
		if t.Ticket.Parent != nil {
			if to, ok := remap[*t.Ticket.Parent]; ok {
				p := to
				o.Parent = &p
			}
		}
		res, err := s.Create(context.Background(), o)
		if err != nil {
			return fmt.Errorf("filing %s (%s): %w", t.Ticket.Title, t.Ticket.ID, err)
		}
		remap[t.Ticket.ID] = res.Ticket.ID
		filed = append(filed, res.Ticket.ID)
		for _, l := range droppedLabels {
			fmt.Fprintf(ctx.env.Stderr, "  dropped label %q on %s: this store does not declare it\n", l, res.Ticket.ID)
		}

		// A reference whose path names a file this repository does not have is
		// reference_path_unresolved, a warning, and --strict fails on warnings.
		// The reference itself is still worth keeping - it says what the sender
		// pointed at - so the path is dropped and the ref survives.
		for _, r := range t.Ticket.References {
			keep := r
			if keep.Path != nil && *keep.Path != "" && !repoHasPath(s.Root(), *keep.Path) {
				fmt.Fprintf(ctx.env.Stderr, "  dropped the path on %s for %s: not in this repository\n", keep.Ref, res.Ticket.ID)
				keep.Path = nil
			}
			if _, err := ctx.applyTo(s, res.Ticket.ID, ticket.AddReference{Ref: keep.Ref, Path: keep.Path}); err != nil {
				return fmt.Errorf("carrying reference %s to %s: %w", keep.Ref, res.Ticket.ID, err)
			}
		}

		// Provenance is a reference and never origin. `origin` is checked
		// against this store and an unresolvable one is an error, which is the
		// guarantee it exists to carry; a reference is deliberately not checked,
		// which is exactly what a foreign ID needs.
		refs := []string{"origin-ticket:" + t.Ticket.ID}
		if fromStore != "" {
			refs = append(refs, "origin-store:"+fromStore)
		}
		for _, r := range refs {
			if _, err := ctx.applyTo(s, res.Ticket.ID, ticket.AddReference{Ref: r}); err != nil {
				return fmt.Errorf("recording provenance on %s: %w", res.Ticket.ID, err)
			}
		}
	}

	for i, t := range in {
		fmt.Fprintf(ctx.out, "%s  <- %s  %s\n", filed[i], t.Ticket.ID, t.Ticket.Title)
		for _, dropped := range importDroppedEdges(t, in) {
			fmt.Fprintf(ctx.env.Stderr, "  dropped %s on %s: not in this export\n", dropped, filed[i])
		}
	}
	fmt.Fprintf(ctx.env.Stderr, "%s filed as draft. Review, then commit.\n", plural(len(filed), "ticket"))
	return nil
}

// importDroppedEdges names the dependencies and parent an incoming ticket points
// at that the export does not carry.
//
// They cannot survive a remint: the ID they name belongs to another store, and
// a dependency on a ticket that does not exist here is dependency_missing, an
// error. Dropping them is the only thing that leaves a valid store, so the
// command's duty is to be loud about it rather than to avoid it.
func importDroppedEdges(t incomingTicket, in []incomingTicket) []string {
	present := make(map[string]bool, len(in))
	for _, o := range in {
		present[o.Ticket.ID] = true
	}
	var dropped []string
	for _, d := range t.Ticket.Dependencies {
		if !present[d] {
			dropped = append(dropped, "dependency "+d)
		}
	}
	if t.Ticket.Parent != nil && *t.Ticket.Parent != "" && !present[*t.Ticket.Parent] {
		dropped = append(dropped, "parent "+*t.Ticket.Parent)
	}
	return dropped
}

// importOtherPatches names the code patches an export carries beside its
// tickets, so the preview can say they are somebody else's job.
func importOtherPatches(dir string) []string {
	all, err := filepath.Glob(filepath.Join(dir, "*.patch"))
	if err != nil {
		return nil
	}
	var out []string
	for _, p := range all {
		if filepath.Base(p) != exportTicketPatch {
			out = append(out, p)
		}
	}
	return out
}

// keepKnownLabels splits a ticket's labels into the ones this store declares and
// the ones it does not.
//
// A label outside the allowlist is label_unknown, a warning, and a receiver
// whose CI runs check --strict fails on warnings. Carrying a sender's vocabulary
// into a store that never agreed to it is not a kindness, so the unknown ones
// are dropped and named. This is the reconciliation git am cannot do, and the
// clearest argument for import existing beside it.
func keepKnownLabels(cfg ticket.Config, labels []string) (keep, dropped []string) {
	for _, l := range labels {
		if cfg.KnownLabel(l) {
			keep = append(keep, l)
			continue
		}
		dropped = append(dropped, l)
	}
	return keep, dropped
}

// repoHasPath reports whether a reference's repository-relative path exists here.
func repoHasPath(root, rel string) bool {
	if root == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}
