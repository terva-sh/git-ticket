package ticket

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// MergeResult is what Merge produced: the file to write, and the names of the
// fields it could not settle.
type MergeResult struct {
	// Merged is a complete ticket file. When Conflicts is empty it parses; when
	// it is not, the disputed fields carry Git conflict markers and the file is
	// what parse reports as merge_conflict, per plan 11.
	Merged []byte
	// Conflicts names the frontmatter keys and body headings that both sides
	// changed differently, in the order plan 7.5 lists them.
	Conflicts []string
}

// Clean reports whether the merge settled everything.
func (r *MergeResult) Clean() bool { return len(r.Conflicts) == 0 }

// Merge is the three-way merge of plan 7.5. base is the common ancestor, ours
// is the version being merged into, and theirs is the version being merged in.
//
// It resolves what the format knows how to resolve and marks the rest. The
// point is not to settle every disagreement: most rows of the 7.5 table say
// conflict. It is that two edits which disagree about nothing should not need a
// person, and today they do, because every mutation rewrites updated_at and
// updated_by and Git sees two sides touch one region.
//
// A parse failure on any side is returned rather than papered over. Git falls
// back to its own driver, which is the right answer for a file this code cannot
// read.
func Merge(base, ours, theirs []byte) (*MergeResult, error) {
	b, err := Parse(base)
	if err != nil {
		return nil, fmt.Errorf("the merge base does not parse: %w", err)
	}
	o, err := Parse(ours)
	if err != nil {
		return nil, fmt.Errorf("our side does not parse: %w", err)
	}
	t, err := Parse(theirs)
	if err != nil {
		return nil, fmt.Errorf("their side does not parse: %w", err)
	}

	m := *o
	var conflicts []string
	// note records a conflict once, keeping 7.5's order rather than discovery
	// order, so two runs report the same list.
	note := func(name string) { conflicts = append(conflicts, name) }

	// Immutable identity. A difference here is corruption, not a merge.
	for _, f := range []struct {
		key     string
		b, o, t string
	}{
		{"schema", fmt.Sprint(b.Schema), fmt.Sprint(o.Schema), fmt.Sprint(t.Schema)},
		{"id", b.ID, o.ID, t.ID},
		{"created_at", b.CreatedAt.String(), o.CreatedAt.String(), t.CreatedAt.String()},
		{"created_by", actorKey(b.CreatedBy), actorKey(o.CreatedBy), actorKey(t.CreatedBy)},
	} {
		if f.o != f.t {
			note(f.key)
		}
	}

	// The provenance pair moves together: the later timestamp wins and carries
	// its actor, so updated_at and updated_by never describe two different
	// edits. This is the row that makes the driver worth having.
	if t.UpdatedAt.Time.After(o.UpdatedAt.Time) {
		m.UpdatedAt, m.UpdatedBy = t.UpdatedAt, t.UpdatedBy
	}

	// Scalars: take the side that changed, and conflict when both did.
	scalars := []struct {
		key     string
		b, o, t string
		set     func(string)
	}{
		{"title", b.Title, o.Title, t.Title, func(v string) { m.Title = v }},
		{"type", b.Type, o.Type, t.Type, func(v string) { m.Type = v }},
		{"status", b.Status, o.Status, t.Status, func(v string) { m.Status = v }},
		{"status_reason", deref(b.StatusReason), deref(o.StatusReason), deref(t.StatusReason), func(v string) { m.StatusReason = optional(v) }},
		{"priority", b.Priority, o.Priority, t.Priority, func(v string) { m.Priority = v }},
		{"due_on", deref(b.DueOn), deref(o.DueOn), deref(t.DueOn), func(v string) { m.DueOn = optional(v) }},
		{"milestone", deref(b.Milestone), deref(o.Milestone), deref(t.Milestone), func(v string) { m.Milestone = optional(v) }},
		{"parent", deref(b.Parent), deref(o.Parent), deref(t.Parent), func(v string) { m.Parent = optional(v) }},
		{"blocks_on", b.BlocksOn, o.BlocksOn, t.BlocksOn, func(v string) { m.BlocksOn = v }},
	}
	for _, s := range scalars {
		v, ok := pick3(s.b, s.o, s.t)
		if !ok {
			note(s.key)
			continue
		}
		s.set(v)
	}

	// Unordered sets. Two additions are compatible, and a removal on one side
	// survives an untouched other side.
	m.Labels = mergeSet(b.Labels, o.Labels, t.Labels, func(s string) string { return s })
	m.Assignees = mergeSet(b.Assignees, o.Assignees, t.Assignees, func(s string) string { return s })
	m.Dependencies = mergeSet(b.Dependencies, o.Dependencies, t.Dependencies, func(s string) string { return s })
	m.References = mergeSet(b.References, o.References, t.References, refKey)

	// A claim is a mutual exclusion record. Resolving it silently would hand
	// one ticket to two agents, each holding a file that says it is theirs.
	if claimKey(o.Claim) != claimKey(t.Claim) {
		note("claim")
	}
	// Two sides archiving out of different statuses disagree about history.
	if archiveKey(o.Archive) != archiveKey(t.Archive) {
		note("archive")
	}

	if ext, ok := mergeExtensions(b.Extensions, o.Extensions, t.Extensions); ok {
		m.Extensions = ext
	} else {
		note("extensions")
	}

	// Body sections. Prose conflicts, logs union, checklists union by item.
	prose := []struct {
		heading string
		b, o, t string
		set     func(string)
	}{
		{"Description", b.Body.Description, o.Body.Description, t.Body.Description, func(v string) { m.Body.Description = v }},
		{"Implementation plan", b.Body.ImplementationPlan, o.Body.ImplementationPlan, t.Body.ImplementationPlan, func(v string) { m.Body.ImplementationPlan = v }},
		{"Summary", b.Body.Summary, o.Body.Summary, t.Body.Summary, func(v string) { m.Body.Summary = v }},
	}
	for _, p := range prose {
		v, ok := pick3(p.b, p.o, p.t)
		if !ok {
			note(p.heading)
			continue
		}
		p.set(v)
	}

	m.Body.Notes = mergeEntries(o.Body.Notes, t.Body.Notes)
	m.Body.Comments = mergeEntries(o.Body.Comments, t.Body.Comments)

	if v, ok := mergeChecklist(b.Body.AcceptanceCriteria, o.Body.AcceptanceCriteria, t.Body.AcceptanceCriteria); ok {
		m.Body.AcceptanceCriteria = v
	} else {
		note("Acceptance criteria")
	}
	if v, ok := mergeChecklist(b.Body.DefinitionOfDone, o.Body.DefinitionOfDone, t.Body.DefinitionOfDone); ok {
		m.Body.DefinitionOfDone = v
	} else {
		note("Definition of done")
	}

	out := Render(&m)
	if len(conflicts) > 0 {
		out = markConflicts(out, Render(o), Render(t), conflicts)
	}
	return &MergeResult{Merged: out, Conflicts: conflicts}, nil
}

// pick3 resolves one value three ways. A side that did not move defers to the
// side that did, and two different moves conflict.
func pick3[T comparable](base, ours, theirs T) (T, bool) {
	switch {
	case ours == theirs:
		return ours, true
	case ours == base:
		return theirs, true
	case theirs == base:
		return ours, true
	}
	var zero T
	return zero, false
}

// mergeSet is a three-way merge of an unordered set. An element survives unless
// a side removed it, and an addition on either side is kept.
//
// Order follows ours and then theirs, so a merge does not reshuffle a list
// somebody is reading. The renderer does not sort these, and neither does this.
func mergeSet[T any](base, ours, theirs []T, key func(T) string) []T {
	inBase, inOurs, inTheirs := keySet(base, key), keySet(ours, key), keySet(theirs, key)
	keep := func(k string) bool {
		o, t := inOurs[k], inTheirs[k]
		switch {
		case o && t:
			return true // both have it
		case o:
			return !inBase[k] // ours added it, or theirs removed it
		case t:
			return !inBase[k]
		}
		return false
	}
	var out []T
	seen := map[string]bool{}
	for _, list := range [][]T{ours, theirs} {
		for _, v := range list {
			k := key(v)
			if seen[k] || !keep(k) {
				continue
			}
			seen[k] = true
			out = append(out, v)
		}
	}
	return out
}

func keySet[T any](list []T, key func(T) string) map[string]bool {
	out := make(map[string]bool, len(list))
	for _, v := range list {
		out[key(v)] = true
	}
	return out
}

func refKey(r Reference) string { return r.Ref + "\x00" + deref(r.Path) }

func actorKey(a *Actor) string {
	if a == nil {
		return ""
	}
	return a.ID + "\x00" + a.Name
}

func claimKey(c *Claim) string {
	if c == nil {
		return ""
	}
	return strings.Join([]string{
		c.Actor, deref(c.Branch), deref(c.Worktree), deref(c.Commit),
		stampKey(c.ClaimedAt), stampKey(c.ExpiresAt),
	}, "\x00")
}

func archiveKey(a *Archive) string {
	if a == nil {
		return ""
	}
	return strings.Join([]string{stampKey(a.ArchivedAt), deref(a.FromStatus), deref(a.Reason)}, "\x00")
}

func stampKey(t *Timestamp) string {
	if t == nil {
		return ""
	}
	return t.String()
}

// mergeExtensions merges the consumer-owned map key by key, per plan 7.5. A key
// only one side touched is taken, and a key both changed differently conflicts.
//
// 5.1 calls extensions the one place a consumer may write fields the core does
// not define, so the core has no business resolving a value it cannot read.
// Key-level is as far as this can honestly go.
func mergeExtensions(base, ours, theirs *yaml.Node) (*yaml.Node, bool) {
	b, o, t := nodeMap(base), nodeMap(ours), nodeMap(theirs)
	out := map[string]*yaml.Node{}
	for _, k := range unionKeys(o, t) {
		v, ok := pick3(nodeText(b[k]), nodeText(o[k]), nodeText(t[k]))
		if !ok {
			return nil, false
		}
		switch v {
		case nodeText(o[k]):
			if o[k] != nil {
				out[k] = o[k]
			}
		default:
			if t[k] != nil {
				out[k] = t[k]
			}
		}
	}
	return mappingNode(out), true
}

func nodeMap(n *yaml.Node) map[string]*yaml.Node {
	out := map[string]*yaml.Node{}
	if n == nil || n.Kind != yaml.MappingNode {
		return out
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		out[n.Content[i].Value] = n.Content[i+1]
	}
	return out
}

func nodeText(n *yaml.Node) string {
	if n == nil {
		return ""
	}
	b, err := yaml.Marshal(n)
	if err != nil {
		return ""
	}
	return string(b)
}

func unionKeys(a, b map[string]*yaml.Node) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range []map[string]*yaml.Node{a, b} {
		for k := range m {
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	sort.Strings(out)
	return out
}

func mappingNode(m map[string]*yaml.Node) *yaml.Node {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	n := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, k := range keys {
		n.Content = append(n.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k}, m[k])
	}
	return n
}

// mergeEntries unions two append-only logs, dropping an entry both sides carry.
//
// There is no base parameter because these sections only grow. Notes is where
// 6.2 puts the history of a blocked reason and Comments is a conversation, so a
// union is the whole answer and a three-way rule would only offer to delete
// somebody's entry. Appending at the same offset is what made Git call these a
// conflict in the first place.
//
// It reads through Entries, the same view mutation.go writes against, so the
// merge and the log cannot disagree about where one entry ends.
func mergeEntries(ours, theirs string) string {
	seen := map[string]bool{}
	var kept []Entry
	for _, side := range []string{ours, theirs} {
		for _, e := range Entries(side) {
			k := e.Actor + "\x00" + e.At + "\x00" + e.Text
			if seen[k] {
				continue
			}
			seen[k] = true
			kept = append(kept, e)
		}
	}
	// Stable and by stamp alone: an entry nobody stamped sorts first, because it
	// was in the section before anything stamped was appended to it.
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].At < kept[j].At })

	out := make([]string, 0, len(kept))
	for _, e := range kept {
		out = append(out, renderEntry(e))
	}
	return strings.Join(out, "\n\n")
}

// renderEntry writes an entry back in the shape appendEntry produced it.
func renderEntry(e Entry) string {
	if e.Actor == "" && e.At == "" {
		return e.Text
	}
	return fmt.Sprintf("**%s** at %s\n\n%s", e.Actor, e.At, e.Text)
}

// checkItem matches one checklist line, capturing its state and its text.
var checkItem = regexp.MustCompile(`^- \[([ xX])\] (.*)$`)

// mergeChecklist unions two checkbox lists by item text. Two sides ticking
// different boxes is the ordinary case and merges; two sides disagreeing about
// one box conflicts, because that is a disagreement about whether the work is
// done.
func mergeChecklist(base, ours, theirs string) (string, bool) {
	if ours == theirs {
		return ours, true
	}
	bItems, oItems, tItems := checklistMap(base), checklistMap(ours), checklistMap(theirs)

	var order []string
	seen := map[string]bool{}
	for _, side := range []string{ours, theirs} {
		for _, line := range strings.Split(side, "\n") {
			m := checkItem.FindStringSubmatch(strings.TrimSpace(line))
			if m == nil || seen[m[2]] {
				continue
			}
			seen[m[2]] = true
			order = append(order, m[2])
		}
	}

	var out []string
	for _, text := range order {
		o, inOurs := oItems[text]
		t, inTheirs := tItems[text]
		_, inBase := bItems[text]
		switch {
		case inOurs && inTheirs:
			state, ok := pick3(bItems[text], o, t)
			if !ok {
				return "", false
			}
			out = append(out, renderItem(state, text))
		case inOurs && !inBase:
			out = append(out, renderItem(o, text)) // ours added it
		case inTheirs && !inBase:
			out = append(out, renderItem(t, text)) // theirs added it
		}
		// An item present on one side and in base was removed by the other.
	}
	return strings.Join(out, "\n"), true
}

func checklistMap(s string) map[string]bool {
	out := map[string]bool{}
	for _, line := range strings.Split(s, "\n") {
		if m := checkItem.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			out[m[2]] = m[1] != " "
		}
	}
	return out
}

func renderItem(checked bool, text string) string {
	if checked {
		return "- [x] " + text
	}
	return "- [ ] " + text
}
