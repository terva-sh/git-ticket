package ticket

import (
	"context"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// sortByID orders tickets oldest first. A ULID sorts by creation time, so this
// is chronological for free, per plan 5.5.
func sortByID(ts []*Ticket) {
	sort.Slice(ts, func(i, j int) bool { return ts[i].ID < ts[j].ID })
}

// crossBranchWindow bounds the ref scan, per plan section 8. A ref whose last
// commit is older than this is skipped, which keeps the cost proportional to
// what is actually in flight rather than to how long the repository has run.
const crossBranchWindow = 30 * 24 * time.Hour

// branchRef is a ref worth reading, and when it was last committed to.
type branchRef struct {
	Name string
	When time.Time
}

// branchRefs lists the local and remote-tracking refs recent enough to read.
//
// Both sets, per plan section 8. An agent working a repository pushes to a
// remote, so in every other worktree their work is a remote-tracking ref and
// never a local head. Reading refs/heads/ alone would miss the exact case
// cross-branch reads exist for.
//
// A symbolic ref is skipped. refs/remotes/origin/HEAD points at another ref
// already in this list, so reading it would parse every ticket a second time
// and report provenance of "origin" for a copy that came from origin/main.
func (s *Store) branchRefs(now time.Time) []branchRef {
	root := s.Root()
	if root == "" {
		return nil
	}
	out, err := runGit(root, "for-each-ref",
		"--format=%(refname:short)\t%(committerdate:unix)\t%(symref)",
		"refs/heads/", "refs/remotes/")
	if err != nil {
		return nil
	}
	var refs []branchRef
	for _, line := range strings.Split(out, "\n") {
		parts := strings.Split(strings.TrimRight(line, "\r"), "\t")
		if len(parts) < 3 || parts[0] == "" {
			continue
		}
		if parts[2] != "" {
			continue
		}
		secs, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}
		when := time.Unix(secs, 0)
		if now.Sub(when) > crossBranchWindow {
			continue
		}
		refs = append(refs, branchRef{Name: parts[0], When: when})
	}
	return refs
}

// treeEntry is one ticket file as it stands on one ref.
type treeEntry struct {
	// Path is relative to the repository root, with forward slashes, which is
	// the form ls-tree prints.
	Path string
	Blob string
}

// storePathspec is the store directory relative to the repository root, which
// is the form ls-tree takes as a pathspec.
func (s *Store) storePathspec() string {
	root := s.Root()
	if root == "" {
		return ""
	}
	rel, err := filepath.Rel(root, s.path)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

// treeAt lists the ticket files on one ref.
//
// It reads the whole store directory rather than the working set alone, per
// plan section 8. The store partitions by status, so a file's directory is its
// status: reading tickets/ only would miss a draft, and would read a ticket as
// unchanged when the thing that changed was the directory it sits in.
func (s *Store) treeAt(ref string) []treeEntry {
	spec := s.storePathspec()
	if spec == "" {
		return nil
	}
	out, err := runGit(s.Root(), "ls-tree", "-r", ref, "--", spec)
	if err != nil {
		return nil
	}
	var entries []treeEntry
	for _, line := range strings.Split(out, "\n") {
		// Each line is "<mode> <type> <object>\t<path>".
		tab := strings.IndexByte(line, '\t')
		if tab < 0 {
			continue
		}
		fields := strings.Fields(line[:tab])
		if len(fields) != 3 || fields[1] != "blob" {
			continue
		}
		path := strings.TrimRight(line[tab+1:], "\r")
		if !strings.HasSuffix(path, ".md") {
			continue
		}
		entries = append(entries, treeEntry{Path: path, Blob: fields[2]})
	}
	return entries
}

// blobReader parses a ticket out of a blob, reading each distinct blob once
// however many refs carry it.
//
// The cache is what keeps this affordable. A ticket nobody has touched is the
// same blob on every branch, so without it a scan costs one process per ticket
// per ref, and with it one per distinct version. A blob that does not parse is
// cached as nil so a broken ticket is not re-read for every ref either.
type blobReader struct {
	root string
	seen map[string]*Ticket
}

func newBlobReader(root string) *blobReader {
	return &blobReader{root: root, seen: make(map[string]*Ticket)}
}

func (b *blobReader) ticket(blob string) *Ticket {
	if t, ok := b.seen[blob]; ok {
		return t
	}
	b.seen[blob] = nil
	out, err := runGit(b.root, "cat-file", "blob", blob)
	if err != nil {
		return nil
	}
	t, err := Parse([]byte(out))
	if err != nil {
		return nil
	}
	b.seen[blob] = t
	return t
}

// crossView is what a cross-branch query answers with: one winning copy of
// each ticket, and the IDs carrying a live claim somewhere.
type crossView struct {
	Tickets []*Ticket
	// ClaimedElsewhere holds the IDs a live claim was found for on any scanned
	// ref. It is deliberately a set of IDs and not a claim: per plan section 8
	// a query never picks between two claims, it only reports that one exists.
	ClaimedElsewhere map[string]bool
}

// crossBranchView merges the working tree with every recent ref.
//
// Resolution follows 7.5 rather than inventing a second rule. The copy with the
// later updated_at wins, which is what the merge driver does when Git merges two
// versions of one ticket, so a listing cannot predict the wrong outcome of the
// merge it is warning about. Every mutation rewrites updated_at, so an
// uncommitted edit already carries the newest timestamp and the working tree
// needs no privilege of its own.
//
// A tie goes to the working tree, because that is the copy the caller can act
// on. Ties between two refs go to the one for-each-ref listed first, which is
// alphabetical, so the answer does not depend on map ordering.
func (s *Store) crossBranchView(ctx context.Context, local []*Ticket) (*crossView, error) {
	now := s.now()
	view := &crossView{
		Tickets:          local,
		ClaimedElsewhere: make(map[string]bool),
	}
	root := s.Root()
	if root == "" {
		return view, nil
	}

	// The working tree is seeded first and only a strictly later updated_at
	// displaces it, which is what makes a tie fall its way.
	winner := make(map[string]*Ticket, len(local))
	order := make([]string, 0, len(local))
	for _, t := range local {
		if _, ok := winner[t.ID]; !ok {
			order = append(order, t.ID)
		}
		winner[t.ID] = t
	}

	reader := newBlobReader(root)
	for _, ref := range s.branchRefs(now) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for _, entry := range s.treeAt(ref.Name) {
			t := reader.ticket(entry.Blob)
			if t == nil || t.ID == "" {
				continue
			}
			if t.Claim != nil && !t.Claim.Expired(now) {
				view.ClaimedElsewhere[t.ID] = true
			}
			cur, ok := winner[t.ID]
			if ok && !t.UpdatedAt.Time.After(cur.UpdatedAt.Time) {
				continue
			}
			// A shallow copy, because the blob cache hands the same Ticket to
			// every ref that carries it and provenance differs per ref.
			c := *t
			c.Branch = ref.Name
			c.Path = filepath.Join(root, filepath.FromSlash(entry.Path))
			if !ok {
				order = append(order, c.ID)
			}
			winner[c.ID] = &c
		}
	}

	out := make([]*Ticket, 0, len(order))
	for _, id := range order {
		out = append(out, winner[id])
	}
	// ULIDs sort by creation time, so this is chronological, matching what the
	// working-tree read already answers with.
	sortByID(out)
	view.Tickets = out
	return view, nil
}
