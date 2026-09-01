package ticket

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// The emitter here exists because yaml.v3's encoder cannot produce the bytes
// plan 5.3 requires. It line-wraps a plain scalar at 80 columns with no way to
// turn that off, and it quotes any string that would resolve as a timestamp or
// a number, so `created_at: 2026-08-31T12:00:00Z` comes back quoted. Both break
// the round-trip property. This emitter writes block style only, at two-space
// indent, and quotes a scalar only when YAML would otherwise read it as
// something else.

// ynode is a value ready to render: a pre-rendered scalar, a sequence, or an
// ordered mapping. Ordered, because frontmatter key order is part of the
// format.
type ynode interface {
	writeTo(b *strings.Builder, indent int)
}

// yscalar holds text that is already in its final rendered form, quoting
// included.
type yscalar struct{ text string }

type yseq struct{ items []ynode }

type ymap struct {
	keys []string
	vals []ynode
}

func (m *ymap) add(key string, v ynode) { m.keys = append(m.keys, key); m.vals = append(m.vals, v) }

func (m *ymap) addString(key, v string) { m.add(key, yscalar{quoteScalar(v)}) }

func (m *ymap) addStringPtr(key string, v *string) {
	if v == nil {
		m.add(key, yscalar{"null"})
		return
	}
	m.addString(key, *v)
}

func (m *ymap) addStringSeq(key string, vs []string) {
	items := make([]ynode, 0, len(vs))
	for _, v := range vs {
		items = append(items, yscalar{quoteScalar(v)})
	}
	m.add(key, &yseq{items})
}

func (m *ymap) addTimestamp(key string, t *Timestamp) {
	if t == nil {
		m.add(key, yscalar{"null"})
		return
	}
	// A timestamp is a plain scalar: YAML resolves it as a timestamp and any
	// reader gets the same instant back.
	m.add(key, yscalar{t.String()})
}

func (m *ymap) empty() bool { return len(m.keys) == 0 }

func pad(n int) string { return strings.Repeat(" ", n) }

func (s yscalar) writeTo(b *strings.Builder, indent int) { b.WriteString(s.text) }

func (s *yseq) writeTo(b *strings.Builder, indent int) {
	// Never called for the inline empty form; writeField handles that.
	for _, it := range s.items {
		writeSeqItem(b, indent, it)
	}
}

func (m *ymap) writeTo(b *strings.Builder, indent int) {
	for i, k := range m.keys {
		writeField(b, indent, k, m.vals[i])
	}
}

// writeField renders `key: value` at the given indent, choosing the inline or
// block form for the value.
func writeField(b *strings.Builder, indent int, key string, v ynode) {
	switch n := v.(type) {
	case yscalar:
		fmt.Fprintf(b, "%s%s: %s\n", pad(indent), key, n.text)
	case *yseq:
		if len(n.items) == 0 {
			fmt.Fprintf(b, "%s%s: []\n", pad(indent), key)
			return
		}
		fmt.Fprintf(b, "%s%s:\n", pad(indent), key)
		n.writeTo(b, indent+2)
	case *ymap:
		if n.empty() {
			fmt.Fprintf(b, "%s%s: {}\n", pad(indent), key)
			return
		}
		fmt.Fprintf(b, "%s%s:\n", pad(indent), key)
		n.writeTo(b, indent+2)
	default:
		panic("ticket: unknown ynode type")
	}
}

// writeSeqItem renders one `- item` at the given indent. A collection item is
// rendered at indent+2 and then has its first two spaces replaced by "- ",
// which is how YAML writes a block collection inside a sequence.
func writeSeqItem(b *strings.Builder, indent int, v ynode) {
	switch n := v.(type) {
	case yscalar:
		fmt.Fprintf(b, "%s- %s\n", pad(indent), n.text)
	case *yseq:
		if len(n.items) == 0 {
			fmt.Fprintf(b, "%s- []\n", pad(indent))
			return
		}
		var inner strings.Builder
		n.writeTo(&inner, indent+2)
		b.WriteString(dashFirstLine(inner.String(), indent))
	case *ymap:
		if n.empty() {
			fmt.Fprintf(b, "%s- {}\n", pad(indent))
			return
		}
		var inner strings.Builder
		n.writeTo(&inner, indent+2)
		b.WriteString(dashFirstLine(inner.String(), indent))
	default:
		panic("ticket: unknown ynode type")
	}
}

// dashFirstLine turns the leading indent of the first line into a "- " bullet.
func dashFirstLine(block string, indent int) string {
	prefix := pad(indent + 2)
	if !strings.HasPrefix(block, prefix) {
		return block
	}
	return pad(indent) + "- " + block[len(prefix):]
}

// quoteScalar renders a Go string as a YAML scalar, quoting only when a plain
// scalar would not read back as the same string.
func quoteScalar(s string) string {
	if needsQuote(s) {
		return doubleQuote(s)
	}
	return s
}

func needsQuote(s string) bool {
	if s == "" {
		return true
	}
	if strings.TrimSpace(s) != s {
		return true
	}
	if resolvesToNonString(s) {
		return true
	}
	// An indicator at the start of a plain scalar.
	switch r, _ := utf8.DecodeRuneInString(s); r {
	case ',', '[', ']', '{', '}', '#', '&', '*', '!', '|', '>', '\'', '"', '%', '@', '`':
		return true
	case '-', '?', ':':
		// Only an indicator when followed by a space or nothing.
		if len(s) == 1 || s[1] == ' ' {
			return true
		}
	}
	if strings.Contains(s, ": ") || strings.HasSuffix(s, ":") {
		return true
	}
	if strings.Contains(s, " #") {
		return true
	}
	for _, r := range s {
		if r == '\n' || r == '\t' || unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// yaml11Bools are not booleans in YAML 1.2, which yaml.v3 follows, but enough
// readers treat them as booleans that writing one unquoted is a trap.
var yaml11Bools = map[string]bool{
	"y": true, "Y": true, "n": true, "N": true,
	"yes": true, "Yes": true, "YES": true,
	"no": true, "No": true, "NO": true,
	"on": true, "On": true, "ON": true,
	"off": true, "Off": true, "OFF": true,
}

func resolvesToNonString(s string) bool {
	switch s {
	case "null", "Null", "NULL", "~", "true", "True", "TRUE", "false", "False", "FALSE":
		return true
	}
	if yaml11Bools[s] {
		return true
	}
	if _, err := strconv.ParseInt(s, 0, 64); err == nil {
		return true
	}
	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return true
	}
	return looksLikeTimestamp(s)
}

// looksLikeTimestamp reports whether YAML would resolve s as a date or a
// timestamp rather than a string.
func looksLikeTimestamp(s string) bool {
	if len(s) < 8 {
		return false
	}
	digits := func(part string) bool {
		for _, r := range part {
			if r < '0' || r > '9' {
				return false
			}
		}
		return len(part) > 0
	}
	// YYYY-MM-DD is the shortest form YAML resolves.
	if len(s) >= 10 && s[4] == '-' && s[7] == '-' &&
		digits(s[0:4]) && digits(s[5:7]) && digits(s[8:10]) {
		return true
	}
	return false
}

func doubleQuote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		default:
			if unicode.IsControl(r) {
				fmt.Fprintf(&b, `\u%04X`, r)
				continue
			}
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// fromYAMLNode converts a preserved yaml.v3 node into a renderable one. Style
// is deliberately not preserved: plan 5.3 makes the renderer a pure function of
// the parsed ticket, so a value written in flow style comes back in block
// style.
func fromYAMLNode(n *yaml.Node) ynode {
	if n == nil {
		return yscalar{"null"}
	}
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return yscalar{"null"}
		}
		return fromYAMLNode(n.Content[0])
	case yaml.AliasNode:
		// No anchors or aliases in the output format, so an alias is expanded
		// into the value it names.
		return fromYAMLNode(n.Alias)
	case yaml.MappingNode:
		m := &ymap{}
		for i := 0; i+1 < len(n.Content); i += 2 {
			m.add(n.Content[i].Value, fromYAMLNode(n.Content[i+1]))
		}
		return m
	case yaml.SequenceNode:
		s := &yseq{}
		for _, c := range n.Content {
			s.items = append(s.items, fromYAMLNode(c))
		}
		return s
	default:
		return yscalar{scalarText(n)}
	}
}

// scalarText renders a scalar node in its canonical form for its resolved tag.
func scalarText(n *yaml.Node) string {
	switch n.Tag {
	case "!!null":
		return "null"
	case "!!bool":
		if strings.EqualFold(n.Value, "true") {
			return "true"
		}
		return "false"
	case "!!int", "!!float":
		return n.Value
	case "!!timestamp":
		return n.Value
	case "!!str", "":
		return quoteScalar(n.Value)
	default:
		return quoteScalar(n.Value)
	}
}
