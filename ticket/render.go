package ticket

import (
	"strconv"
	"strings"
)

// Render returns the canonical bytes for a ticket. It is a pure function of
// the parsed ticket, per plan 5.3: two writers holding the same logical ticket
// produce identical bytes, so a diff shows the fields a mutation changed and
// nothing else.
func Render(t *Ticket) []byte {
	var b strings.Builder
	b.WriteString("---\n")
	renderFrontmatter(&b, t)
	b.WriteString("---\n\n")
	renderBody(&b, t.Body)
	return []byte(b.String())
}

func renderFrontmatter(b *strings.Builder, t *Ticket) {
	m := &ymap{}
	schema := t.Schema
	if schema == 0 {
		schema = SchemaVersion
	}
	m.add("schema", yscalar{strconv.Itoa(schema)})
	m.addString("id", t.ID)
	m.addString("title", t.Title)
	m.addString("type", t.Type)
	m.addString("status", t.Status)
	m.addStringPtr("status_reason", t.StatusReason)
	m.addString("priority", t.Priority)
	m.addStringSeq("labels", t.Labels)
	m.addStringSeq("assignees", t.Assignees)
	m.addStringPtr("milestone", t.Milestone)
	m.addStringPtr("parent", t.Parent)
	m.addStringSeq("dependencies", t.Dependencies)
	m.add("references", referencesNode(t.References))
	m.add("claim", claimNode(t.Claim))
	m.add("archive", archiveNode(t.Archive))
	m.add("created_at", timestampNode(t.CreatedAt))
	m.add("updated_at", timestampNode(t.UpdatedAt))
	m.add("created_by", actorNode(t.CreatedBy))
	m.add("updated_by", actorNode(t.UpdatedBy))
	if t.Extensions == nil {
		m.add("extensions", &ymap{})
	} else {
		m.add("extensions", fromYAMLNode(t.Extensions))
	}
	// Unknown keys render after the known ones, in the order they arrived, so
	// a reader one minor version behind does not shuffle a newer reader's
	// fields on every write.
	for _, u := range t.Unknown {
		m.add(u.Key, fromYAMLNode(u.Value))
	}
	m.writeTo(b, 0)
}

// timestampNode renders a required timestamp, emitting null for the zero value
// so that a file written with an explicit null round-trips.
func timestampNode(ts Timestamp) ynode {
	if ts.Raw == "" && ts.Time.IsZero() {
		return yscalar{"null"}
	}
	return yscalar{ts.String()}
}

func referencesNode(refs []Reference) ynode {
	s := &yseq{}
	for _, r := range refs {
		m := &ymap{}
		m.addString("ref", r.Ref)
		m.addStringPtr("path", r.Path)
		s.items = append(s.items, m)
	}
	return s
}

func claimNode(c *Claim) ynode {
	if c == nil {
		return yscalar{"null"}
	}
	m := &ymap{}
	m.addString("actor", c.Actor)
	m.addStringPtr("branch", c.Branch)
	m.addStringPtr("worktree", c.Worktree)
	m.addStringPtr("commit", c.Commit)
	m.addTimestamp("claimed_at", c.ClaimedAt)
	m.addTimestamp("expires_at", c.ExpiresAt)
	return m
}

func archiveNode(a *Archive) ynode {
	if a == nil {
		return yscalar{"null"}
	}
	m := &ymap{}
	m.addTimestamp("archived_at", a.ArchivedAt)
	m.addStringPtr("from_status", a.FromStatus)
	m.addStringPtr("reason", a.Reason)
	return m
}

func actorNode(a *Actor) ynode {
	if a == nil {
		return yscalar{"null"}
	}
	m := &ymap{}
	m.addString("id", a.ID)
	m.addString("name", a.Name)
	return m
}

// renderBody writes the Markdown sections. Description is always emitted; the
// rest appear only when they have content, and unknown sections follow the
// known ones in their original relative order, per plan 5.2.
func renderBody(b *strings.Builder, body Body) {
	var parts []string
	if body.Preamble != "" {
		parts = append(parts, body.Preamble+"\n")
	}
	parts = append(parts, section("Description", body.Description))
	for _, s := range []Section{
		{"Acceptance criteria", body.AcceptanceCriteria},
		{"Definition of done", body.DefinitionOfDone},
		{"Implementation plan", body.ImplementationPlan},
		{"Notes", body.Notes},
		{"Comments", body.Comments},
		{"Summary", body.Summary},
	} {
		if s.Text == "" {
			continue
		}
		parts = append(parts, section(s.Heading, s.Text))
	}
	for _, s := range body.Extra {
		parts = append(parts, section(s.Heading, s.Text))
	}
	b.WriteString(strings.Join(parts, "\n"))
}

func section(heading, text string) string {
	if text == "" {
		return "## " + heading + "\n"
	}
	return "## " + heading + "\n\n" + text + "\n"
}
