package ticket

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
)

// MappingSelection is one explicit receiver decision. Action is adopt, alias,
// or decline. Local is a free local namespace for alias or a conflicting
// decline; an ordinary decline leaves it empty.
type MappingSelection struct {
	Namespace string
	Action    string
	Local     string
}

// MappingOffer reports what one incoming namespace means at this receiver.
type MappingOffer struct {
	Namespace   string
	Example     string
	Destination string
	Existing    string // absent, identical, or conflicting
	Action      string
	Local       string
}

// ReferenceMappingPlan is the read-only decision shared by preview and write.
// Rewrites maps incoming namespace names to their chosen local names.
type ReferenceMappingPlan struct {
	Offers     []MappingOffer
	Undeclared []string
	// Unresolved are conflicts that a preview can show but ticket adoption
	// must resolve with alias:LOCAL or decline:LOCAL.
	Unresolved  []string
	Rewrites    map[string]string
	MapRevision string
	Changed     bool
	HasSidecar  bool
	Registry    ReferenceRegistry
}

// ReferenceMappingResult reports the registry write independently of ticket
// filing, which can later stop after some tickets have landed.
type ReferenceMappingResult struct {
	Changed      bool
	MapRevision  string
	PathsChanged []string
}

func (s *Store) readRegistryForMapping() (ReferenceRegistry, string, bool, error) {
	registry, data, err := s.readReferenceRegistrySnapshot()
	if err != nil {
		return ReferenceRegistry{}, "", false, err
	}
	if registry == nil {
		return ReferenceRegistry{Version: referenceVersion, Stores: map[string]ReferenceStore{}, Namespaces: map[string]ReferenceNamespace{}}, Revision(nil), false, nil
	}
	if registry.Stores == nil {
		registry.Stores = map[string]ReferenceStore{}
	}
	if registry.Namespaces == nil {
		registry.Namespaces = map[string]ReferenceNamespace{}
	}
	return *registry, Revision(data), true, nil
}

func cloneRegistry(r ReferenceRegistry) ReferenceRegistry {
	copy := ReferenceRegistry{Version: r.Version, Stores: map[string]ReferenceStore{}, Namespaces: map[string]ReferenceNamespace{}}
	for key, value := range r.Stores {
		copy.Stores[key] = value
	}
	for key, value := range r.Namespaces {
		copy.Namespaces[key] = value
	}
	return copy
}

func sameMeaning(name string, offered ReferenceLookup, current ReferenceRegistry) bool {
	foreign := offered.Namespaces[name]
	local, ok := current.Namespaces[name]
	if !ok || local != foreign {
		return false
	}
	return foreign.Kind != "ticket-store" || current.Stores[foreign.Store] == offered.Stores[foreign.Store]
}

func offeredDestination(ref string, declaration ReferenceNamespace, registry ReferenceRegistry) string {
	_, identifier, _ := splitRef(ref)
	re, err := compileIdentifier(declaration.Identifier)
	if err != nil {
		return ""
	}
	match := re.FindStringSubmatch(identifier)
	if match == nil {
		return ""
	}
	if declaration.Kind == "ticket-store" {
		browse := registry.Stores[declaration.Store].Browse
		if strings.Contains(browse, "?") && strings.Contains(browse, "{id}") && strings.Contains(strings.SplitN(browse, "?", 2)[1], "{id}") {
			return strings.Replace(browse, "{id}", url.QueryEscape(identifier), 1)
		}
		return strings.Replace(browse, "{id}", url.PathEscape(identifier), 1)
	}
	values := map[string]string{}
	for i, name := range re.SubexpNames() {
		if name != "" {
			values[name] = match[i]
		}
	}
	return placeholder.ReplaceAllStringFunc(declaration.Template, func(token string) string {
		return url.PathEscape(values[token[1:len(token)-1]])
	})
}

// PlanReferenceMappings checks a sidecar and computes explicit receiver
// choices. Nil sidecar means an old export with no mapping offer.
func (s *Store) PlanReferenceMappings(patch, sidecar []byte, choices []MappingSelection) (*ReferenceMappingPlan, error) {
	current, revision, _, err := s.readRegistryForMapping()
	if err != nil {
		return nil, err
	}
	plan := &ReferenceMappingPlan{MapRevision: revision, Registry: cloneRegistry(current), Rewrites: map[string]string{}}
	if sidecar == nil {
		used, err := usedNamespacesFromPatch(patch)
		if err != nil {
			return nil, err
		}
		plan.Undeclared = sortedKeys(used)
		chosen := map[string]bool{}
		occupied := map[string]bool{}
		for name := range current.Namespaces {
			occupied[name] = true
		}
		for name := range used {
			occupied[name] = true
		}
		for _, choice := range choices {
			if _, ok := used[choice.Namespace]; !ok || chosen[choice.Namespace] {
				return nil, fmt.Errorf("namespace %q is not used by this export or was selected twice", choice.Namespace)
			}
			chosen[choice.Namespace] = true
			if choice.Action != "decline" || !namespaceName(choice.Local) || occupied[choice.Local] {
				return nil, fmt.Errorf("old exports only allow NAMESPACE=decline:FREE_OPAQUE_NAMESPACE")
			}
			occupied[choice.Local] = true
			plan.Rewrites[choice.Namespace] = choice.Local
		}
		for _, name := range plan.Undeclared {
			if _, local := current.Namespaces[name]; local && plan.Rewrites[name] == "" {
				plan.Unresolved = append(plan.Unresolved, name)
			}
		}
		return plan, nil
	}
	lookup, err := ParseReferenceLookup(patch, sidecar)
	if err != nil {
		return nil, err
	}
	plan.HasSidecar = true
	plan.Undeclared = append([]string(nil), lookup.Undeclared...)
	selection := map[string]MappingSelection{}
	for _, choice := range choices {
		if _, duplicate := selection[choice.Namespace]; duplicate {
			return nil, fmt.Errorf("mapping %q was selected twice", choice.Namespace)
		}
		_, offered := lookup.Namespaces[choice.Namespace]
		undeclared := false
		for _, name := range lookup.Undeclared {
			undeclared = undeclared || name == choice.Namespace
		}
		if !offered && !undeclared {
			return nil, fmt.Errorf("namespace %q is not used by this export", choice.Namespace)
		}
		selection[choice.Namespace] = choice
	}
	used, err := usedNamespacesFromPatch(patch)
	if err != nil {
		return nil, err
	}
	occupied := map[string]bool{}
	for name := range current.Namespaces {
		occupied[name] = true
	}
	for name := range lookup.Namespaces {
		occupied[name] = true
	}
	for _, name := range lookup.Undeclared {
		occupied[name] = true
	}
	for _, name := range sortedKeys(lookup.Namespaces) {
		declaration := lookup.Namespaces[name]
		state := "absent"
		if _, found := current.Namespaces[name]; found {
			state = "conflicting"
			if sameMeaning(name, *lookup, current) {
				state = "identical"
			}
		}
		choice, chosen := selection[name]
		if !chosen {
			choice = MappingSelection{Namespace: name, Action: "decline"}
		}
		switch choice.Action {
		case "adopt":
			if choice.Local != "" || state == "conflicting" {
				return nil, fmt.Errorf("mapping %q conflicts locally; alias it or decline to a free opaque namespace", name)
			}
			if state == "absent" {
				if err := addOfferedMapping(&plan.Registry, *lookup, name, name); err != nil {
					return nil, err
				}
			}
		case "alias":
			if !namespaceName(choice.Local) || occupied[choice.Local] {
				return nil, fmt.Errorf("alias for %q needs a free lower-case namespace", name)
			}
			occupied[choice.Local] = true
			if err := addOfferedMapping(&plan.Registry, *lookup, name, choice.Local); err != nil {
				return nil, err
			}
			plan.Rewrites[name] = choice.Local
		case "decline":
			if state == "conflicting" && choice.Local == "" {
				plan.Unresolved = append(plan.Unresolved, name)
			}
			if choice.Local != "" {
				if !namespaceName(choice.Local) || occupied[choice.Local] {
					return nil, fmt.Errorf("opaque target for %q needs a free lower-case namespace", name)
				}
				occupied[choice.Local] = true
				plan.Rewrites[name] = choice.Local
			}
		default:
			return nil, fmt.Errorf("mapping %q choice must be adopt, alias or decline", name)
		}
		plan.Offers = append(plan.Offers, MappingOffer{
			Namespace: name, Example: used[name], Destination: offeredDestination(used[name], declaration, ReferenceRegistry{Stores: lookup.Stores}),
			Existing: state, Action: choice.Action, Local: choice.Local,
		})
	}
	for _, name := range lookup.Undeclared {
		choice, chosen := selection[name]
		if !chosen {
			choice = MappingSelection{Namespace: name, Action: "decline"}
		}
		if choice.Action != "decline" {
			return nil, fmt.Errorf("undeclared namespace %q can only be declined to an opaque target", name)
		}
		if _, conflict := current.Namespaces[name]; conflict && choice.Local == "" {
			plan.Unresolved = append(plan.Unresolved, name)
		}
		if choice.Local != "" {
			if !namespaceName(choice.Local) || occupied[choice.Local] {
				return nil, fmt.Errorf("opaque target for %q needs a free lower-case namespace", name)
			}
			occupied[choice.Local] = true
			plan.Rewrites[name] = choice.Local
		}
	}
	if err := plan.Registry.validate(); err != nil {
		return nil, err
	}
	plan.Changed = !reflect.DeepEqual(plan.Registry, current)
	return plan, nil
}

func usedNamespacesFromPatch(patch []byte) (map[string]string, error) {
	files, err := ParseAddedFiles(string(patch))
	if err != nil {
		return nil, err
	}
	tickets := make([]*Ticket, 0, len(files))
	for _, file := range files {
		t, err := Parse([]byte(file.Body))
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return usedNamespaces(tickets)
}

func addOfferedMapping(target *ReferenceRegistry, lookup ReferenceLookup, incoming, local string) error {
	declaration := lookup.Namespaces[incoming]
	if declaration.Kind == "ticket-store" {
		store := lookup.Stores[declaration.Store]
		key := declaration.Store
		if existing, found := target.Stores[key]; found && existing != store {
			if incoming == local {
				return fmt.Errorf("store key %q has another meaning; alias namespace %q", key, incoming)
			}
			key = local + "_" + key
			if _, occupied := target.Stores[key]; occupied {
				return fmt.Errorf("aliased store key %q is already used", key)
			}
		}
		target.Stores[key] = store
		declaration.Store = key
	}
	target.Namespaces[local] = declaration
	return nil
}

// ApplyReferenceMappings writes accepted declarations under the store lock.
// The plan's revision is always checked, and ifRevision adds a caller's
// explicit precondition over the same raw registry bytes.
func (s *Store) ApplyReferenceMappings(ctx context.Context, plan *ReferenceMappingPlan, ifRevision string) (*ReferenceMappingResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("no reference mapping plan")
	}
	lock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer lock.release()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	_, current, _, err := s.readRegistryForMapping()
	if err != nil {
		return nil, err
	}
	if current != plan.MapRevision || ifRevision != "" && current != ifRevision {
		return nil, &Error{Code: CodeStaleRevision, Field: "references.yml", Message: "reference registry changed since the mapping preview"}
	}
	result := &ReferenceMappingResult{MapRevision: current}
	if !plan.Changed {
		return result, nil
	}
	data, err := RenderReferenceRegistry(plan.Registry)
	if err != nil {
		return nil, err
	}
	if err := writeFileAtomic(filepath.Join(s.path, referencesFile), data); err != nil {
		return nil, err
	}
	result.Changed = true
	result.MapRevision = Revision(data)
	result.PathsChanged = []string{referencesFile}
	return result, nil
}
