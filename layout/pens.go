package layout

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/terva-sh/git-ticket/internal/idgrammar"
	"gopkg.in/yaml.v3"
)

// Point is an absolute board coordinate.
type Point struct {
	X float64 `yaml:"x" json:"x"`
	Y float64 `yaml:"y" json:"y"`
}

// Match is a pen's rule, per plan 12.10: which tickets it catches. Every
// field is optional and an absent or empty one matches everything, so a rule
// is the fields it names and nothing else. The fields that are present are
// conjoined.
//
// Labels conjoins within itself and the other three do not, which is the
// shape of what they read rather than an inconsistency. A ticket carries many
// labels at once, so a list of them can only mean all of them, which is what
// requiredLabels meant before schema 4 and is why a schema 3 board routes
// unchanged. A ticket has one status, one type and one parent, so a list
// there can only mean any of them.
type Match struct {
	Labels []string `yaml:"labels" json:"labels"`
	Status []string `yaml:"status" json:"status"`
	Type   []string `yaml:"type" json:"type"`
	Parent []string `yaml:"parent" json:"parent"`
}

// MarshalJSON publishes all four fields as arrays and never as null, per plan
// 10.10. A field the board file leaves out is a field the rule does not test,
// which a consumer should read as an empty list rather than infer from a
// missing key.
func (m Match) MarshalJSON() ([]byte, error) {
	type plain Match
	return json.Marshal(plain{Labels: orEmpty(m.Labels), Status: orEmpty(m.Status), Type: orEmpty(m.Type), Parent: orEmpty(m.Parent)})
}

func (m *Match) UnmarshalJSON(data []byte) error {
	type plain Match
	return decodeJSONRecord(data, (*plain)(m), matchFields)
}
func (m *Match) UnmarshalYAML(node *yaml.Node) error {
	type plain Match
	return decodeYAMLRecord(node, (*plain)(m), matchFields)
}

func orEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// matchField is one field of a rule with its name, so that validation,
// rendering, comparison and the failure report all walk the rule in one
// order: the order plan 10.10 publishes in failed.
type matchField struct {
	name   string
	values []string
}

func (m Match) fields() []matchField {
	return []matchField{{"labels", m.Labels}, {"status", m.Status}, {"type", m.Type}, {"parent", m.Parent}}
}

// empty reports a rule that tests nothing, which validateRouting refuses: a
// pen with no rule at all would catch the whole store from wherever it sat in
// ruleOrder and leave every rule below it dead.
func (m Match) empty() bool {
	return len(m.Labels) == 0 && len(m.Status) == 0 && len(m.Type) == 0 && len(m.Parent) == 0
}

// Pen defines a rule and its automatic placement region. Membership and
// manual card placements belong to separate records.
type Pen struct {
	Title string  `yaml:"title" json:"title"`
	X     float64 `yaml:"x" json:"x"`
	Y     float64 `yaml:"y" json:"y"`
	W     float64 `yaml:"w" json:"w"`
	H     float64 `yaml:"h" json:"h"`
	Color string  `yaml:"color" json:"color"`
	Pin   *Point  `yaml:"pin" json:"pin"`
	Match Match   `yaml:"match" json:"match"`
}

// Routing is replaced as one conditional record, including explicit tie order.
type Routing struct {
	Pens      map[string]Pen `yaml:"pens" json:"pens"`
	RuleOrder []string       `yaml:"ruleOrder" json:"ruleOrder"`
	Inbox     *Point         `yaml:"inbox" json:"inbox"`
}

// recordFields is what keys one record may carry. Required fields distinguish
// a complete authored record from a partial edit: in particular, an omitted
// or null coordinate must not become zero in a CAS. Optional fields may be
// absent, oneOf is a choice exactly one key must make, and any other key is
// refused.
type recordFields struct {
	required []string
	optional []string
	oneOf    []string
}

func (f recordFields) allowed() []string {
	out := append([]string{}, f.required...)
	out = append(out, f.optional...)
	return append(out, f.oneOf...)
}

var pointFields = recordFields{required: []string{"x", "y"}}
var matchFields = recordFields{optional: []string{"labels", "status", "type", "parent"}}

// A pen spells its rule requiredLabels at schema 3 and match at schema 4, and
// the per-record decoder cannot see the file's schema, so it accepts exactly
// one of the two and Parse decides which one that file may carry. The JSON
// form takes match alone: the only JSON writer is the canvas server replacing
// a whole Routing, which is built against this package and writes what it
// renders.
var penBaseFields = []string{"title", "x", "y", "w", "h", "color", "pin"}
var penYAMLFields = recordFields{required: penBaseFields, oneOf: []string{"match", "requiredLabels"}}
var penJSONFields = recordFields{required: append(append([]string{}, penBaseFields...), "match")}

func decodeJSONRecord(data []byte, target any, fields recordFields) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	null := func(value json.RawMessage) bool { return bytes.Equal(bytes.TrimSpace(value), []byte("null")) }
	for _, key := range fields.required {
		value, ok := raw[key]
		if !ok || null(value) {
			return fmt.Errorf("missing or null %s", key)
		}
	}
	for _, key := range fields.optional {
		if value, ok := raw[key]; ok && null(value) {
			return fmt.Errorf("null %s", key)
		}
	}
	if err := chose(fields.oneOf, func(key string) bool {
		value, ok := raw[key]
		return ok && !null(value)
	}); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(target)
}
func decodeYAMLRecord(node *yaml.Node, target any, fields recordFields) error {
	if node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	if node.Kind != yaml.MappingNode {
		return errors.New("expected a mapping")
	}
	allowed := fields.allowed()
	seen := map[string]bool{}
	for i := 0; i < len(node.Content); i += 2 {
		key, value := node.Content[i].Value, node.Content[i+1]
		if !slices.Contains(allowed, key) || seen[key] {
			return fmt.Errorf("unknown or duplicate field %s", key)
		}
		if value.Kind == yaml.AliasNode {
			value = value.Alias
		}
		if value.Tag == "!!null" {
			return fmt.Errorf("null %s", key)
		}
		seen[key] = true
	}
	for _, key := range fields.required {
		if !seen[key] {
			return fmt.Errorf("missing %s", key)
		}
	}
	if err := chose(fields.oneOf, func(key string) bool { return seen[key] }); err != nil {
		return err
	}
	return node.Decode(target)
}

// chose reports whether exactly one of a set of alternative keys is present.
func chose(oneOf []string, present func(string) bool) error {
	if len(oneOf) == 0 {
		return nil
	}
	var found []string
	for _, key := range oneOf {
		if present(key) {
			found = append(found, key)
		}
	}
	if len(found) != 1 {
		return fmt.Errorf("exactly one of %s is required, got %d", strings.Join(oneOf, " or "), len(found))
	}
	return nil
}

func (p *Point) UnmarshalJSON(data []byte) error {
	type plain Point
	return decodeJSONRecord(data, (*plain)(p), pointFields)
}
func (p *Point) UnmarshalYAML(node *yaml.Node) error {
	type plain Point
	return decodeYAMLRecord(node, (*plain)(p), pointFields)
}
func (p *Pen) UnmarshalJSON(data []byte) error {
	type plain Pen
	return decodeJSONRecord(data, (*plain)(p), penJSONFields)
}

// penYAML is a pen as either schema spells its rule. The schema 3 form reads
// into the same record, because requiredLabels is match.labels and always
// was: every board written before schema 4 routes exactly as it did.
type penYAML struct {
	Title          string   `yaml:"title"`
	X              float64  `yaml:"x"`
	Y              float64  `yaml:"y"`
	W              float64  `yaml:"w"`
	H              float64  `yaml:"h"`
	Color          string   `yaml:"color"`
	Pin            *Point   `yaml:"pin"`
	Match          Match    `yaml:"match"`
	RequiredLabels []string `yaml:"requiredLabels"`
}

func (p *Pen) UnmarshalYAML(node *yaml.Node) error {
	var w penYAML
	if err := decodeYAMLRecord(node, &w, penYAMLFields); err != nil {
		return err
	}
	*p = Pen{Title: w.Title, X: w.X, Y: w.Y, W: w.W, H: w.H, Color: w.Color, Pin: w.Pin, Match: w.Match}
	if w.RequiredLabels != nil {
		p.Match = Match{Labels: w.RequiredLabels}
	}
	return nil
}

// penRuleSpelling holds a file to the spelling its schema declares: match
// requires schema 4, the way the routing fields already require schema 3, and
// a schema 4 file may not carry the older key. The two are the same record,
// so accepting both in one file would leave two readings of one rule.
func penRuleSpelling(data []byte, schema int) error {
	var doc struct {
		Pens map[string]map[string]yaml.Node `yaml:"pens"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}
	ids := make([]string, 0, len(doc.Pens))
	for id := range doc.Pens {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if _, ok := doc.Pens[id]["match"]; ok && schema < 4 {
			return fmt.Errorf("pen %s: match requires layout schema 4", id)
		}
		if _, ok := doc.Pens[id]["requiredLabels"]; ok && schema >= 4 {
			return fmt.Errorf("pen %s: requiredLabels is the schema 3 spelling; schema %d writes match", id, schema)
		}
	}
	return nil
}

// Colors are the three a pen or a frame may carry, and the canvas draws no
// others. They are published so a writer offers the same three.
var Colors = []string{"#759bcc", "#b499be", "#89ad97"}

func emptyRouting() Routing {
	return Routing{Pens: map[string]Pen{}, RuleOrder: []string{}, Inbox: &Point{}}
}

func validateRouting(r Routing) error {
	if r.Pens == nil || r.RuleOrder == nil || r.Inbox == nil {
		return errors.New("routing requires pens, ruleOrder and inbox")
	}
	if !finite(r.Inbox.X) || !finite(r.Inbox.Y) {
		return errors.New("invalid Inbox coordinate")
	}
	seen := map[string]bool{}
	for _, id := range r.RuleOrder {
		if _, ok := r.Pens[id]; !ok || seen[id] {
			return errors.New("ruleOrder must contain each pen exactly once")
		}
		seen[id] = true
	}
	if len(seen) != len(r.Pens) {
		return errors.New("ruleOrder must contain each pen exactly once")
	}
	for id, p := range r.Pens {
		if !validRecordID(id) {
			return fmt.Errorf("invalid pen ID %q", id)
		}
		if strings.TrimSpace(p.Title) == "" || !utf8.ValidString(p.Title) || utf8.RuneCountInString(p.Title) > 80 || strings.ContainsFunc(p.Title, unicode.IsControl) {
			return fmt.Errorf("invalid pen title for %s", id)
		}
		if !finite(p.X) || !finite(p.Y) || !finite(p.W) || !finite(p.H) || round2(p.W) <= 0 || round2(p.H) <= 0 || p.Pin == nil || !finite(p.Pin.X) || !finite(p.Pin.Y) {
			return fmt.Errorf("invalid pen geometry for %s", id)
		}
		if !slices.Contains(Colors, p.Color) {
			return fmt.Errorf("invalid pen color for %s", id)
		}
		if err := validateMatch(p.Match); err != nil {
			return fmt.Errorf("pen %s %w", id, err)
		}
	}
	return nil
}

// validateMatch holds a rule to what a rule can be. A parent is a ticket ID,
// checked by the grammar of plan 5.6 the way a frame member is. A status or a
// type is not checked against the sets of 6.1 and 5.1: this package cannot
// import ticket, and a status nobody uses catches nothing, which a person can
// see in `canvas show`.
func validateMatch(m Match) error {
	if m.empty() {
		return errors.New("requires a nonempty rule: labels, status, type or parent")
	}
	for _, f := range m.fields() {
		for _, v := range f.values {
			if strings.TrimSpace(v) == "" || !utf8.ValidString(v) || strings.ContainsFunc(v, unicode.IsControl) {
				return fmt.Errorf("has an invalid %s value %q", f.name, v)
			}
			if f.name == "parent" && !idgrammar.Valid(v) {
				return fmt.Errorf("names a parent %q that is not a ticket ID", v)
			}
		}
	}
	return nil
}

// canonicalRouting copies caller-owned records and deduplicates exact rule
// values. Reads preserve coordinates; explicit writes round them like cards
// and frames.
func canonicalRouting(r Routing, round bool) Routing {
	out := Routing{Pens: make(map[string]Pen, len(r.Pens)), RuleOrder: append([]string{}, r.RuleOrder...), Inbox: &Point{X: r.Inbox.X, Y: r.Inbox.Y}}
	if round {
		out.Inbox.X, out.Inbox.Y = round2(out.Inbox.X), round2(out.Inbox.Y)
	}
	for id, p := range r.Pens {
		pin := *p.Pin
		p.Pin = &pin
		p.Match = canonicalMatch(p.Match)
		if round {
			p.X, p.Y, p.W, p.H = round2(p.X), round2(p.Y), round2(p.W), round2(p.H)
			p.Pin.X, p.Pin.Y = round2(p.Pin.X), round2(p.Pin.Y)
		}
		out.Pens[id] = p
	}
	return out
}

// canonicalMatch deduplicates every field and keeps the order each was
// written in, so a rule reads back the way its author wrote it. A value
// repeated in a conjunction or a disjunction says nothing the first one did
// not, and leaving it would rewrite the line on the next save.
func canonicalMatch(m Match) Match {
	return Match{Labels: dedupe(m.Labels), Status: dedupe(m.Status), Type: dedupe(m.Type), Parent: dedupe(m.Parent)}
}

func dedupe(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := []string{}
	for _, v := range values {
		if !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	return out
}

func sameRouting(a, b Routing) bool {
	if *a.Inbox != *b.Inbox || !slices.Equal(a.RuleOrder, b.RuleOrder) || len(a.Pens) != len(b.Pens) {
		return false
	}
	for id, p := range a.Pens {
		q, ok := b.Pens[id]
		if !ok || p.Title != q.Title || p.X != q.X || p.Y != q.Y || p.W != q.W || p.H != q.H || p.Color != q.Color || *p.Pin != *q.Pin {
			return false
		}
		if !sameMatch(p.Match, q.Match) {
			return false
		}
	}
	return true
}

func sameMatch(a, b Match) bool {
	af, bf := a.fields(), b.fields()
	for i := range af {
		x, y := slices.Clone(af[i].values), slices.Clone(bf[i].values)
		sort.Strings(x)
		sort.Strings(y)
		if !slices.Equal(x, y) {
			return false
		}
	}
	return true
}

func renderRouting(sb *strings.Builder, r Routing) {
	if r.Pens == nil && r.RuleOrder == nil && r.Inbox == nil {
		r = emptyRouting()
	}
	ids := make([]string, 0, len(r.Pens))
	for id := range r.Pens {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		sb.WriteString("pens: {}\n")
	} else {
		sb.WriteString("pens:\n")
	}
	for _, id := range ids {
		p := r.Pens[id]
		fmt.Fprintf(sb, "  %s: {title: %s, x: %s, y: %s, w: %s, h: %s, color: %s, pin: {x: %s, y: %s}, match: {", strconv.Quote(id), strconv.Quote(p.Title), num(p.X), num(p.Y), num(p.W), num(p.H), strconv.Quote(p.Color), num(p.Pin.X), num(p.Pin.Y))
		renderMatch(sb, p.Match)
		sb.WriteString("}}\n")
	}
	sb.WriteString("ruleOrder: [")
	renderStrings(sb, r.RuleOrder)
	sb.WriteString("]\n")
	fmt.Fprintf(sb, "inbox: {x: %s, y: %s}\n", num(r.Inbox.X), num(r.Inbox.Y))
}

// renderMatch writes the rule's fields in one order, omitting the ones it
// does not test, so a labels-only pen reads much as it did at schema 3.
func renderMatch(sb *strings.Builder, m Match) {
	first := true
	for _, f := range m.fields() {
		if len(f.values) == 0 {
			continue
		}
		if !first {
			sb.WriteString(", ")
		}
		first = false
		fmt.Fprintf(sb, "%s: [", f.name)
		renderStrings(sb, f.values)
		sb.WriteString("]")
	}
}

func renderStrings(sb *strings.Builder, values []string) {
	for i, v := range values {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(strconv.Quote(v))
	}
}
