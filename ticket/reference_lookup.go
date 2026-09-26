package ticket

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// ReferenceLookup is the portable, optional mapping offer beside an export.
// It is bound to the ticket patch, not to any code patches composed beside it.
type ReferenceLookup struct {
	FormatVersion     int                           `json:"formatVersion"`
	TicketPatchSHA256 string                        `json:"ticketPatchSha256"`
	Stores            map[string]ReferenceStore     `json:"stores"`
	Namespaces        map[string]ReferenceNamespace `json:"namespaces"`
	Undeclared        []string                      `json:"undeclared"`
}

func patchDigest(patch []byte) string {
	sum := sha256.Sum256(patch)
	return hex.EncodeToString(sum[:])
}

func usedNamespaces(tickets []*Ticket) (map[string]string, error) {
	used := make(map[string]string)
	for _, t := range tickets {
		for _, ref := range t.References {
			name, _, typed := splitRef(ref.Ref)
			if !typed {
				continue
			}
			name = strings.ToLower(name)
			if !namespaceName(name) {
				return nil, fmt.Errorf("reference %q has an invalid namespace", ref.Ref)
			}
			if _, seen := used[name]; !seen {
				used[name] = ref.Ref
			}
		}
	}
	return used, nil
}

// BuildReferenceLookup offers only mappings used by the exported tickets.
// Machine-local checkout bindings never enter this JSON.
func (s *Store) BuildReferenceLookup(tickets []*Ticket, patch []byte) ([]byte, error) {
	registry, err := s.ReadReferenceRegistry()
	if err != nil {
		return nil, err
	}
	used, err := usedNamespaces(tickets)
	if err != nil {
		return nil, err
	}
	if registry != nil {
		for _, t := range tickets {
			for _, ref := range t.References {
				if !registry.validIdentifier(ref.Ref) {
					return nil, fmt.Errorf("ticket %s reference %q misses its declared grammar", t.ID, ref.Ref)
				}
			}
		}
	}
	lookup := ReferenceLookup{
		FormatVersion: 1, TicketPatchSHA256: patchDigest(patch),
		Stores: map[string]ReferenceStore{}, Namespaces: map[string]ReferenceNamespace{},
		Undeclared: []string{},
	}
	for _, name := range sortedKeys(used) {
		if registry == nil {
			lookup.Undeclared = append(lookup.Undeclared, name)
			continue
		}
		declaration, ok := registry.Namespaces[name]
		if !ok {
			lookup.Undeclared = append(lookup.Undeclared, name)
			continue
		}
		lookup.Namespaces[name] = declaration
		if declaration.Kind == "ticket-store" {
			lookup.Stores[declaration.Store] = registry.Stores[declaration.Store]
		}
	}
	return json.MarshalIndent(lookup, "", "  ")
}

// ParseReferenceLookup verifies the exact patch bytes and all offered syntax
// before a receiver previews or writes anything. A missing sidecar is handled
// by the caller as an older export, rather than passed here as empty JSON.
func ParseReferenceLookup(patch, data []byte) (*ReferenceLookup, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var lookup ReferenceLookup
	if err := decoder.Decode(&lookup); err != nil {
		return nil, fmt.Errorf("references.json: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, fmt.Errorf("references.json: expected one JSON object")
	}
	if lookup.FormatVersion != 1 {
		return nil, fmt.Errorf("references.json: unsupported formatVersion %d", lookup.FormatVersion)
	}
	if len(lookup.TicketPatchSHA256) != 64 {
		return nil, fmt.Errorf("references.json: ticket patch digest must be SHA-256")
	}
	if _, err := hex.DecodeString(lookup.TicketPatchSHA256); err != nil {
		return nil, fmt.Errorf("references.json: ticket patch digest must be hexadecimal")
	}
	if lookup.TicketPatchSHA256 != patchDigest(patch) {
		return nil, fmt.Errorf("references.json: ticket patch digest does not match 0001-tickets.patch")
	}
	if lookup.Stores == nil || lookup.Namespaces == nil || lookup.Undeclared == nil {
		return nil, fmt.Errorf("references.json: stores, namespaces and undeclared are required")
	}
	registry := ReferenceRegistry{Version: referenceVersion, Stores: lookup.Stores, Namespaces: lookup.Namespaces}
	if err := registry.validate(); err != nil {
		return nil, fmt.Errorf("references.json: %w", err)
	}
	for name := range lookup.Stores {
		used := false
		for _, declaration := range lookup.Namespaces {
			used = used || declaration.Store == name
		}
		if !used {
			return nil, fmt.Errorf("references.json: unused store %q", name)
		}
	}
	files, err := ParseAddedFiles(string(patch))
	if err != nil {
		return nil, err
	}
	tickets := make([]*Ticket, 0, len(files))
	for _, file := range files {
		ticket, err := Parse([]byte(file.Body))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file.Path, err)
		}
		tickets = append(tickets, ticket)
	}
	used, err := usedNamespaces(tickets)
	if err != nil {
		return nil, err
	}
	for _, t := range tickets {
		for _, ref := range t.References {
			if !registry.validIdentifier(ref.Ref) {
				return nil, fmt.Errorf("references.json: ticket %s reference %q misses its offered grammar", t.ID, ref.Ref)
			}
		}
	}
	listed := make(map[string]bool)
	for _, name := range lookup.Undeclared {
		if !namespaceName(name) || listed[name] || lookup.Namespaces[name].Kind != "" {
			return nil, fmt.Errorf("references.json: duplicate or invalid undeclared namespace %q", name)
		}
		listed[name] = true
	}
	for name := range lookup.Namespaces {
		if _, ok := used[name]; !ok {
			return nil, fmt.Errorf("references.json: namespace %q is not used by the ticket patch", name)
		}
		listed[name] = true
	}
	for name := range used {
		if !listed[name] {
			return nil, fmt.Errorf("references.json: namespace %q is missing from the lookup table", name)
		}
	}
	for name := range listed {
		if _, ok := used[name]; !ok {
			return nil, fmt.Errorf("references.json: namespace %q is not used by the ticket patch", name)
		}
	}
	if !sort.StringsAreSorted(lookup.Undeclared) {
		return nil, fmt.Errorf("references.json: undeclared namespaces must be sorted")
	}
	return &lookup, nil
}
