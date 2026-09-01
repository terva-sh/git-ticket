package ticket

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// knownFields are the top-level frontmatter keys this version defines, in the
// order plan 5.1 renders them. Anything else is preserved as an unknown field.
var knownFields = []string{
	"schema", "id", "title", "type", "status", "priority",
	"labels", "assignees", "milestone", "parent", "dependencies",
	"references", "claim", "archive",
	"created_at", "updated_at", "created_by", "updated_by", "extensions",
}

// knownSections are the body sections this version defines, in the order plan
// 5.2 renders them.
var knownSections = []string{
	"Description", "Acceptance criteria", "Definition of done",
	"Implementation plan", "Notes", "Comments", "Summary",
}

// Revision returns the revision token for a ticket file's bytes: "sha256:"
// followed by 64 lowercase hex characters. It is computed on read and never
// stored in the file, per plan 7.1.
func Revision(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// ParseFile reads and parses the ticket at path, recording its path and
// revision.
func ParseFile(path string) (*Ticket, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &Error{Code: CodeParseError, Message: err.Error(), Err: err}
	}
	t, err := Parse(data)
	if t != nil {
		t.Path = path
	}
	if err != nil {
		return t, err
	}
	return t, nil
}

// Parse reads one ticket file. It returns a coded error for a file that cannot
// be read as a ticket: merge_conflict, parse_error, or schema_unsupported. An
// unknown top-level field is not an error here; it is preserved on the ticket
// and reported by Check, per plan 5.4.
func Parse(data []byte) (*Ticket, error) {
	text := string(data)
	if line, ok := findConflictMarker(text); ok {
		return nil, &Error{
			Code:    CodeMergeConflict,
			Message: fmt.Sprintf("git conflict markers at line %d", line),
		}
	}

	front, body, err := splitFrontmatter(text)
	if err != nil {
		return nil, err
	}

	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(front), &doc); err != nil {
		return nil, &Error{Code: CodeParseError, Message: yamlMessage(err), Err: err}
	}
	root, err := mappingRoot(&doc)
	if err != nil {
		return nil, err
	}

	t := &Ticket{Revision: Revision(data)}
	// The ID and schema come first, so that a schema refusal can name the
	// ticket it refused.
	for i := 0; i+1 < len(root.Content); i += 2 {
		switch root.Content[i].Value {
		case "id":
			t.ID = root.Content[i+1].Value
		}
	}
	schemaNode := fieldNode(root, "schema")
	if schemaNode == nil {
		return nil, &Error{Code: CodeParseError, Message: "frontmatter has no schema field", Ticket: t.ID, Field: "schema"}
	}
	n, convErr := strconv.Atoi(schemaNode.Value)
	if convErr != nil {
		return nil, &Error{Code: CodeParseError, Message: "schema is not an integer", Ticket: t.ID, Field: "schema"}
	}
	t.Schema = n
	if n > SchemaVersion {
		return nil, &Error{
			Code:    CodeSchemaUnsupported,
			Message: fmt.Sprintf("ticket declares schema %d, this reader supports %d", n, SchemaVersion),
			Ticket:  t.ID,
			Field:   "schema",
			Details: map[string]string{"found": strconv.Itoa(n), "supported": strconv.Itoa(SchemaVersion)},
		}
	}

	if err := decodeFields(t, root); err != nil {
		return nil, err
	}
	t.Body = parseBody(body)
	return t, nil
}

func decodeFields(t *Ticket, root *yaml.Node) error {
	known := make(map[string]bool, len(knownFields))
	for _, k := range knownFields {
		known[k] = true
	}

	fail := func(field, format string, args ...any) error {
		return &Error{
			Code:    CodeParseError,
			Message: fmt.Sprintf(format, args...),
			Ticket:  t.ID,
			Field:   field,
		}
	}

	for i := 0; i+1 < len(root.Content); i += 2 {
		key := root.Content[i].Value
		val := root.Content[i+1]
		if !known[key] {
			t.Unknown = append(t.Unknown, UnknownField{Key: key, Value: val})
			continue
		}
		var err error
		switch key {
		case "schema", "id":
			// Already read.
		case "title":
			t.Title, err = scalarString(val, key)
		case "type":
			t.Type, err = scalarString(val, key)
		case "status":
			t.Status, err = scalarString(val, key)
		case "priority":
			t.Priority, err = scalarString(val, key)
		case "labels":
			t.Labels, err = stringSeq(val, key)
		case "assignees":
			t.Assignees, err = stringSeq(val, key)
		case "milestone":
			t.Milestone, err = optionalString(val, key)
		case "parent":
			t.Parent, err = optionalString(val, key)
		case "dependencies":
			t.Dependencies, err = stringSeq(val, key)
		case "references":
			t.References, err = decodeReferences(val)
		case "claim":
			t.Claim, err = decodeClaim(val)
		case "archive":
			t.Archive, err = decodeArchive(val)
		case "created_at":
			var ts *Timestamp
			if ts, err = decodeTimestamp(val, key); err == nil && ts != nil {
				t.CreatedAt = *ts
			}
		case "updated_at":
			var ts *Timestamp
			if ts, err = decodeTimestamp(val, key); err == nil && ts != nil {
				t.UpdatedAt = *ts
			}
		case "created_by":
			t.CreatedBy, err = decodeActor(val, key)
		case "updated_by":
			t.UpdatedBy, err = decodeActor(val, key)
		case "extensions":
			// The node is kept as it was written, null included, so that a
			// hand-written `extensions: null` renders back as null rather than
			// as {}.
			if !isNull(val) && val.Kind != yaml.MappingNode {
				err = fmt.Errorf("extensions must be a mapping")
			} else {
				t.Extensions = val
			}
		}
		if err != nil {
			if e, ok := err.(*Error); ok {
				e.Ticket = t.ID
				return e
			}
			return fail(key, "%s", err.Error())
		}
	}
	return nil
}

func fieldNode(root *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			return root.Content[i+1]
		}
	}
	return nil
}

func isNull(n *yaml.Node) bool {
	return n == nil || n.Tag == "!!null" || (n.Kind == yaml.ScalarNode && n.Value == "" && n.Style == 0)
}

func scalarString(n *yaml.Node, field string) (string, error) {
	if isNull(n) {
		return "", nil
	}
	if n.Kind != yaml.ScalarNode {
		return "", &Error{Code: CodeParseError, Message: field + " must be a scalar", Field: field}
	}
	return n.Value, nil
}

func optionalString(n *yaml.Node, field string) (*string, error) {
	if isNull(n) {
		return nil, nil
	}
	s, err := scalarString(n, field)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func stringSeq(n *yaml.Node, field string) ([]string, error) {
	if isNull(n) {
		return nil, nil
	}
	if n.Kind != yaml.SequenceNode {
		return nil, &Error{Code: CodeParseError, Message: field + " must be a sequence", Field: field}
	}
	out := make([]string, 0, len(n.Content))
	for _, c := range n.Content {
		if c.Kind != yaml.ScalarNode {
			return nil, &Error{Code: CodeParseError, Message: field + " must hold scalars", Field: field}
		}
		out = append(out, c.Value)
	}
	return out, nil
}

func decodeTimestamp(n *yaml.Node, field string) (*Timestamp, error) {
	if isNull(n) {
		return nil, nil
	}
	if n.Kind != yaml.ScalarNode {
		return nil, &Error{Code: CodeParseError, Message: field + " must be a timestamp", Field: field}
	}
	parsed, err := time.Parse(time.RFC3339, n.Value)
	if err != nil {
		return nil, &Error{
			Code:    CodeParseError,
			Message: fmt.Sprintf("%s is not an RFC 3339 timestamp: %s", field, n.Value),
			Field:   field,
			Err:     err,
		}
	}
	return &Timestamp{Time: parsed, Raw: n.Value}, nil
}

func decodeActor(n *yaml.Node, field string) (*Actor, error) {
	if isNull(n) {
		return nil, nil
	}
	if n.Kind != yaml.MappingNode {
		return nil, &Error{Code: CodeParseError, Message: field + " must be a mapping", Field: field}
	}
	a := &Actor{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		switch n.Content[i].Value {
		case "id":
			a.ID = n.Content[i+1].Value
		case "name":
			a.Name = n.Content[i+1].Value
		}
	}
	return a, nil
}

func decodeReferences(n *yaml.Node) ([]Reference, error) {
	if isNull(n) {
		return nil, nil
	}
	if n.Kind != yaml.SequenceNode {
		return nil, &Error{Code: CodeParseError, Message: "references must be a sequence", Field: "references"}
	}
	out := make([]Reference, 0, len(n.Content))
	for _, item := range n.Content {
		if item.Kind != yaml.MappingNode {
			return nil, &Error{Code: CodeParseError, Message: "each reference must be a mapping", Field: "references"}
		}
		var r Reference
		for i := 0; i+1 < len(item.Content); i += 2 {
			key := item.Content[i].Value
			val := item.Content[i+1]
			switch key {
			case "ref":
				r.Ref = val.Value
			case "path":
				p, err := optionalString(val, "references.path")
				if err != nil {
					return nil, err
				}
				r.Path = p
			}
		}
		out = append(out, r)
	}
	return out, nil
}

func decodeClaim(n *yaml.Node) (*Claim, error) {
	if isNull(n) {
		return nil, nil
	}
	if n.Kind != yaml.MappingNode {
		return nil, &Error{Code: CodeParseError, Message: "claim must be a mapping", Field: "claim"}
	}
	c := &Claim{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		key := n.Content[i].Value
		val := n.Content[i+1]
		var err error
		switch key {
		case "actor":
			c.Actor, err = scalarString(val, "claim.actor")
		case "branch":
			c.Branch, err = optionalString(val, "claim.branch")
		case "worktree":
			c.Worktree, err = optionalString(val, "claim.worktree")
		case "commit":
			c.Commit, err = optionalString(val, "claim.commit")
		case "claimed_at":
			c.ClaimedAt, err = decodeTimestamp(val, "claim.claimed_at")
		case "expires_at":
			c.ExpiresAt, err = decodeTimestamp(val, "claim.expires_at")
		}
		if err != nil {
			return nil, err
		}
	}
	return c, nil
}

func decodeArchive(n *yaml.Node) (*Archive, error) {
	if isNull(n) {
		return nil, nil
	}
	if n.Kind != yaml.MappingNode {
		return nil, &Error{Code: CodeParseError, Message: "archive must be a mapping", Field: "archive"}
	}
	a := &Archive{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		key := n.Content[i].Value
		val := n.Content[i+1]
		var err error
		switch key {
		case "archived_at":
			a.ArchivedAt, err = decodeTimestamp(val, "archive.archived_at")
		case "from_status":
			a.FromStatus, err = optionalString(val, "archive.from_status")
		case "reason":
			a.Reason, err = optionalString(val, "archive.reason")
		}
		if err != nil {
			return nil, err
		}
	}
	return a, nil
}

// splitFrontmatter separates the YAML frontmatter from the Markdown body. The
// file must open with a --- line and close the block with another.
func splitFrontmatter(text string) (front, body string, err error) {
	if !strings.HasPrefix(text, "---\n") {
		return "", "", codedError(CodeParseError, "file does not start with a --- frontmatter fence")
	}
	rest := text[len("---\n"):]
	idx := strings.Index(rest, "\n---\n")
	switch {
	case strings.HasPrefix(rest, "---\n"):
		return "", rest[len("---\n"):], nil
	case idx >= 0:
		return rest[:idx+1], rest[idx+len("\n---\n"):], nil
	case strings.HasSuffix(rest, "\n---"):
		return rest[:len(rest)-len("---")], "", nil
	default:
		return "", "", codedError(CodeParseError, "frontmatter is not closed by a --- line")
	}
}

func mappingRoot(doc *yaml.Node) (*yaml.Node, error) {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, codedError(CodeParseError, "frontmatter is empty")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, codedError(CodeParseError, "frontmatter is not a mapping")
	}
	return root, nil
}

// yamlMessage trims the "yaml: " prefix yaml.v3 puts on its errors, since the
// caller already knows which parser spoke.
func yamlMessage(err error) string {
	msg := err.Error()
	msg = strings.TrimPrefix(msg, "yaml: ")
	return strings.ReplaceAll(msg, "\n", "; ")
}

// findConflictMarker reports the first Git conflict marker in the text, with
// its one-based line number. A file mid-merge is reported as merge_conflict
// rather than as a YAML failure, because a mapping error on line 6 sends the
// user looking for a syntax mistake they did not make.
//
// The opening marker alone is the test: a ======= line by itself is a Markdown
// setext heading and appears in ordinary prose.
func findConflictMarker(text string) (int, bool) {
	var opened int
	for i, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "<<<<<<<") && (len(line) == 7 || line[7] == ' ') {
			opened = i + 1
		}
		if opened > 0 && strings.HasPrefix(line, ">>>>>>>") && (len(line) == 7 || line[7] == ' ') {
			return opened, true
		}
	}
	return 0, false
}

// parseBody splits the Markdown below the frontmatter into sections. A heading
// inside a fenced code block is text, not a heading.
func parseBody(body string) Body {
	var b Body
	type rawSection struct {
		heading string
		lines   []string
	}
	var preamble []string
	var sections []rawSection
	cur := -1
	fence := ""

	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case fence != "":
			if strings.HasPrefix(trimmed, fence) {
				fence = ""
			}
		case strings.HasPrefix(trimmed, "```"):
			fence = "```"
		case strings.HasPrefix(trimmed, "~~~"):
			fence = "~~~"
		}
		if fence == "" && strings.HasPrefix(line, "## ") {
			sections = append(sections, rawSection{heading: strings.TrimSpace(line[3:])})
			cur = len(sections) - 1
			continue
		}
		if cur < 0 {
			preamble = append(preamble, line)
			continue
		}
		sections[cur].lines = append(sections[cur].lines, line)
	}

	b.Preamble = trimBlankLines(preamble)
	for _, s := range sections {
		text := trimBlankLines(s.lines)
		switch s.heading {
		case "Description":
			b.Description = text
		case "Acceptance criteria":
			b.AcceptanceCriteria = text
		case "Definition of done":
			b.DefinitionOfDone = text
		case "Implementation plan":
			b.ImplementationPlan = text
		case "Notes":
			b.Notes = text
		case "Comments":
			b.Comments = text
		case "Summary":
			b.Summary = text
		default:
			b.Extra = append(b.Extra, Section{Heading: s.heading, Text: text})
		}
	}
	return b
}

// trimBlankLines joins lines and removes the blank lines at each end, which
// the renderer puts back in canonical positions.
func trimBlankLines(lines []string) string {
	start, end := 0, len(lines)
	for start < end && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	if start >= end {
		return ""
	}
	return strings.Join(lines[start:end], "\n")
}
