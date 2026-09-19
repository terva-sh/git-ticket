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

	"gopkg.in/yaml.v3"
)

// Point is an absolute board coordinate.
type Point struct {
	X float64 `yaml:"x" json:"x"`
	Y float64 `yaml:"y" json:"y"`
}

// Pen defines a label conjunction and its automatic placement region.
// Membership and manual card placements belong to separate records.
type Pen struct {
	Title          string   `yaml:"title" json:"title"`
	X              float64  `yaml:"x" json:"x"`
	Y              float64  `yaml:"y" json:"y"`
	W              float64  `yaml:"w" json:"w"`
	H              float64  `yaml:"h" json:"h"`
	Color          string   `yaml:"color" json:"color"`
	Pin            *Point   `yaml:"pin" json:"pin"`
	RequiredLabels []string `yaml:"requiredLabels" json:"requiredLabels"`
}

// Routing is replaced as one conditional record, including explicit tie order.
type Routing struct {
	Pens      map[string]Pen `yaml:"pens" json:"pens"`
	RuleOrder []string       `yaml:"ruleOrder" json:"ruleOrder"`
	Inbox     *Point         `yaml:"inbox" json:"inbox"`
}

// Required fields distinguish a complete authored record from a partial edit.
// In particular, an omitted or null coordinate must not become zero in a CAS.
var pointFields = []string{"x", "y"}
var penFields = []string{"title", "x", "y", "w", "h", "color", "pin", "requiredLabels"}

func decodeJSONRecord(data []byte, target any, fields []string) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for _, key := range fields {
		value, ok := raw[key]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("missing or null %s", key)
		}
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(target)
}
func decodeYAMLRecord(node *yaml.Node, target any, fields []string) error {
	if node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	if node.Kind != yaml.MappingNode {
		return errors.New("expected a mapping")
	}
	seen := map[string]bool{}
	for i := 0; i < len(node.Content); i += 2 {
		key, value := node.Content[i].Value, node.Content[i+1]
		if !slices.Contains(fields, key) || seen[key] {
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
	for _, key := range fields {
		if !seen[key] {
			return fmt.Errorf("missing %s", key)
		}
	}
	return node.Decode(target)
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
	return decodeJSONRecord(data, (*plain)(p), penFields)
}
func (p *Pen) UnmarshalYAML(node *yaml.Node) error {
	type plain Pen
	return decodeYAMLRecord(node, (*plain)(p), penFields)
}

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
		switch p.Color {
		case "#759bcc", "#b499be", "#89ad97":
		default:
			return fmt.Errorf("invalid pen color for %s", id)
		}
		if len(p.RequiredLabels) == 0 {
			return fmt.Errorf("pen %s requires a nonempty rule", id)
		}
		for _, label := range p.RequiredLabels {
			if strings.TrimSpace(label) == "" || !utf8.ValidString(label) || strings.ContainsFunc(label, unicode.IsControl) {
				return fmt.Errorf("invalid required label for %s", id)
			}
		}
	}
	return nil
}

// canonicalRouting copies caller-owned records and deduplicates exact labels.
// Reads preserve coordinates; explicit writes round them like cards and frames.
func canonicalRouting(r Routing, round bool) Routing {
	out := Routing{Pens: make(map[string]Pen, len(r.Pens)), RuleOrder: append([]string{}, r.RuleOrder...), Inbox: &Point{X: r.Inbox.X, Y: r.Inbox.Y}}
	if round {
		out.Inbox.X, out.Inbox.Y = round2(out.Inbox.X), round2(out.Inbox.Y)
	}
	for id, p := range r.Pens {
		pin := *p.Pin
		p.Pin = &pin
		labels := []string{}
		for _, label := range p.RequiredLabels {
			if !slices.Contains(labels, label) {
				labels = append(labels, label)
			}
		}
		p.RequiredLabels = labels
		if round {
			p.X, p.Y, p.W, p.H = round2(p.X), round2(p.Y), round2(p.W), round2(p.H)
			p.Pin.X, p.Pin.Y = round2(p.Pin.X), round2(p.Pin.Y)
		}
		out.Pens[id] = p
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
		pl, ql := slices.Clone(p.RequiredLabels), slices.Clone(q.RequiredLabels)
		sort.Strings(pl)
		sort.Strings(ql)
		if !slices.Equal(pl, ql) {
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
		fmt.Fprintf(sb, "  %s: {title: %s, x: %s, y: %s, w: %s, h: %s, color: %s, pin: {x: %s, y: %s}, requiredLabels: [", strconv.Quote(id), strconv.Quote(p.Title), num(p.X), num(p.Y), num(p.W), num(p.H), strconv.Quote(p.Color), num(p.Pin.X), num(p.Pin.Y))
		renderStrings(sb, p.RequiredLabels)
		sb.WriteString("]}\n")
	}
	sb.WriteString("ruleOrder: [")
	renderStrings(sb, r.RuleOrder)
	sb.WriteString("]\n")
	fmt.Fprintf(sb, "inbox: {x: %s, y: %s}\n", num(r.Inbox.X), num(r.Inbox.Y))
}
func renderStrings(sb *strings.Builder, values []string) {
	for i, v := range values {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(strconv.Quote(v))
	}
}
