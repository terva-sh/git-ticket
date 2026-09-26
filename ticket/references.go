package ticket

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"regexp/syntax"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	referencesFile      = "references.yml"
	localReferencesFile = "references.local.yml"
	referenceVersion    = 1
)

// ReferenceRegistry is the portable, versioned declaration in
// .tickets/references.yml. It is deliberately independent of config.yml and
// the ticket schema: older binaries leave it alone when rewriting either.
type ReferenceRegistry struct {
	Version    int                           `yaml:"version"`
	Stores     map[string]ReferenceStore     `yaml:"stores"`
	Namespaces map[string]ReferenceNamespace `yaml:"namespaces"`
}

// ReferenceStore names a foreign repository and how to browse its tickets.
type ReferenceStore struct {
	Repository string `yaml:"repository"`
	Path       string `yaml:"path"`
	Browse     string `yaml:"browse"`
}

// ReferenceNamespace declares one identifier grammar and destination kind.
type ReferenceNamespace struct {
	Kind       string `yaml:"kind"`
	Identifier string `yaml:"identifier"`
	Template   string `yaml:"template"`
	Store      string `yaml:"store"`
}

// LocalReferenceBindings names checkout roots for portable store keys. This
// file is ignored by Git and never consulted by Check.
type LocalReferenceBindings struct {
	Version int               `yaml:"version"`
	Stores  map[string]string `yaml:"stores"`
}

// ResolvedReference keeps the stored reference intact. URL is the portable
// navigation target; LocalPath is set only when a safe local file exists.
type ResolvedReference struct {
	Ref       string
	URL       string
	LocalPath string
}

func registryError(file, message string, err error) error {
	return &Error{Code: CodeReferenceRegistryInvalid, Field: file, Message: message, Err: err}
}

func decodeRegistryYAML(data []byte, into any, file string) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(into); err != nil {
		return registryError(file, yamlMessage(err), err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return registryError(file, "only one YAML document is permitted", nil)
		}
		return registryError(file, yamlMessage(err), err)
	}
	return nil
}

// ParseReferenceRegistry validates the tracked version-1 registry without
// contacting a forge or a foreign ticket store.
func ParseReferenceRegistry(data []byte) (*ReferenceRegistry, error) {
	var r ReferenceRegistry
	if err := decodeRegistryYAML(data, &r, referencesFile); err != nil {
		return nil, err
	}
	if err := r.validate(); err != nil {
		return nil, registryError(referencesFile, err.Error(), err)
	}
	return &r, nil
}

func namespaceName(s string) bool {
	if s == "" || strings.ToLower(s) != s {
		return false
	}
	for i, c := range s {
		if c >= 'a' && c <= 'z' || i > 0 && (c >= '0' && c <= '9' || c == '-' || c == '_') {
			continue
		}
		return false
	}
	return true
}

func (r ReferenceRegistry) validate() error {
	if r.Version != referenceVersion {
		return fmt.Errorf("references.yml version %d is unsupported; expected %d", r.Version, referenceVersion)
	}
	for key, store := range r.Stores {
		if !namespaceName(key) {
			return fmt.Errorf("store key %q must be lower-case and path-safe", key)
		}
		if err := validRepositoryURL(store.Repository); err != nil {
			return fmt.Errorf("store %s repository: %w", key, err)
		}
		if !safeRelativeStorePath(store.Path) {
			return fmt.Errorf("store %s path must be a repository-relative path without traversal", key)
		}
		if err := validTemplate(store.Browse, map[string]bool{"id": true}, true); err != nil {
			return fmt.Errorf("store %s browse: %w", key, err)
		}
	}
	for name, declaration := range r.Namespaces {
		if !namespaceName(name) {
			return fmt.Errorf("namespace %q must be lower-case", name)
		}
		re, err := compileIdentifier(declaration.Identifier)
		if err != nil {
			return fmt.Errorf("namespace %s identifier: %w", name, err)
		}
		switch declaration.Kind {
		case "url":
			if declaration.Store != "" || declaration.Template == "" {
				return fmt.Errorf("namespace %s: url requires template and forbids store", name)
			}
			allowed := map[string]bool{}
			for _, capture := range re.SubexpNames() {
				if capture != "" {
					allowed[capture] = true
				}
			}
			if err := validTemplate(declaration.Template, allowed, false); err != nil {
				return fmt.Errorf("namespace %s template: %w", name, err)
			}
		case "ticket-store":
			if declaration.Template != "" || r.Stores[declaration.Store].Repository == "" {
				return fmt.Errorf("namespace %s: ticket-store requires a declared store and forbids template", name)
			}
			id := re.SubexpIndex("id")
			if id < 1 || !wholeIDCapture(declaration.Identifier) {
				return fmt.Errorf("namespace %s: a named id capture must cover the whole ticket-store identifier", name)
			}
		default:
			return fmt.Errorf("namespace %s: kind %q is unsupported", name, declaration.Kind)
		}
	}
	return nil
}

// A ticket-store's id is the entire identifier, not a substring a template
// might navigate to by accident. Anchors around the one capture are allowed.
func wholeIDCapture(pattern string) bool {
	parsed, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return false
	}
	if parsed.Op == syntax.OpConcat {
		parts := parsed.Sub
		if len(parts) > 0 && (parts[0].Op == syntax.OpBeginText || parts[0].Op == syntax.OpBeginLine) {
			parts = parts[1:]
		}
		if len(parts) > 0 && (parts[len(parts)-1].Op == syntax.OpEndText || parts[len(parts)-1].Op == syntax.OpEndLine) {
			parts = parts[:len(parts)-1]
		}
		if len(parts) != 1 {
			return false
		}
		parsed = parts[0]
	}
	return parsed.Op == syntax.OpCapture && parsed.Name == "id"
}

func compileIdentifier(pattern string) (*regexp.Regexp, error) {
	if pattern == "" {
		return nil, fmt.Errorf("an identifier pattern is required")
	}
	re, err := regexp.Compile("^(?:" + pattern + ")$")
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for _, name := range re.SubexpNames() {
		if name == "" {
			continue
		}
		if names[name] {
			return nil, fmt.Errorf("capture %q occurs more than once", name)
		}
		names[name] = true
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("an identifier needs a named capture")
	}
	return re, nil
}

func validRepositoryURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u == nil || (u.Scheme != "https" && u.Scheme != "ssh") || u.Hostname() == "" || u.Path == "" || u.Path == "/" || u.RawQuery != "" || u.Fragment != "" || (u.User != nil && u.User.Username() == "") {
		return fmt.Errorf("expected an HTTPS or SSH repository URL without query or fragment")
	}
	if u.User != nil {
		if u.Scheme == "https" {
			return fmt.Errorf("repository URL must not contain credentials")
		}
		if _, password := u.User.Password(); password {
			return fmt.Errorf("repository URL must not contain a password")
		}
	}
	if strings.ContainsAny(u.Host, "{}") {
		return fmt.Errorf("repository host must be literal")
	}
	return nil
}

func safeRelativeStorePath(p string) bool {
	return p != "" && p != "." && !strings.ContainsAny(p, "\\:\x00") && !path.IsAbs(p) && path.Clean(p) == p && p != ".." && !strings.HasPrefix(p, "../")
}

var placeholder = regexp.MustCompile(`\{([A-Za-z][A-Za-z0-9_]*)\}`)

func validTemplate(raw string, allowed map[string]bool, browse bool) error {
	u, err := url.Parse(raw)
	if err != nil || u == nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" || strings.ContainsAny(u.Host, "{}") {
		return fmt.Errorf("expected an HTTPS URL with a literal host, no credentials or fragment")
	}
	if !browse && (u.RawQuery != "" || u.ForceQuery) {
		return fmt.Errorf("URL namespace templates cannot have a query")
	}
	count := 0
	for _, segment := range strings.Split(u.Path, "/") {
		if !strings.ContainsAny(segment, "{}") {
			if segment == "." || segment == ".." {
				return fmt.Errorf("dot segments are not permitted")
			}
			continue
		}
		match := placeholder.FindStringSubmatch(segment)
		if match == nil || match[0] != segment || !allowed[match[1]] {
			return fmt.Errorf("each placeholder must name a capture and occupy a complete path segment")
		}
		count++
	}
	if u.RawQuery != "" {
		query, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return err
		}
		for key, values := range query {
			if strings.ContainsAny(key, "{}") {
				return fmt.Errorf("placeholders cannot occupy query keys")
			}
			for _, value := range values {
				if !strings.ContainsAny(value, "{}") {
					continue
				}
				if value != "{id}" || !browse {
					return fmt.Errorf("browse placeholder must occupy a complete query value")
				}
				count++
			}
		}
	}
	if browse && count != 1 {
		return fmt.Errorf("browse template needs exactly one {id} placeholder")
	}
	return nil
}

// RenderReferenceRegistry emits deterministic bytes for a validated portable
// registry. Unknown or unsupported declarations are refused, never discarded.
func RenderReferenceRegistry(r ReferenceRegistry) ([]byte, error) {
	if err := r.validate(); err != nil {
		return nil, registryError(referencesFile, err.Error(), err)
	}
	m := &ymap{}
	m.add("version", yscalar{"1"})
	stores := &ymap{}
	for _, key := range sortedKeys(r.Stores) {
		v := r.Stores[key]
		item := &ymap{}
		item.addString("repository", v.Repository)
		item.addString("path", v.Path)
		item.addString("browse", v.Browse)
		stores.add(key, item)
	}
	m.add("stores", stores)
	namespaces := &ymap{}
	for _, key := range sortedKeys(r.Namespaces) {
		v := r.Namespaces[key]
		item := &ymap{}
		item.addString("kind", v.Kind)
		item.addString("identifier", v.Identifier)
		if v.Kind == "url" {
			item.addString("template", v.Template)
		} else {
			item.addString("store", v.Store)
		}
		namespaces.add(key, item)
	}
	m.add("namespaces", namespaces)
	var b strings.Builder
	m.writeTo(&b, 0)
	return []byte(b.String()), nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// ParseLocalReferenceBindings validates machine-specific checkout roots.
func ParseLocalReferenceBindings(data []byte) (*LocalReferenceBindings, error) {
	var local LocalReferenceBindings
	if err := decodeRegistryYAML(data, &local, localReferencesFile); err != nil {
		return nil, err
	}
	if local.Version != referenceVersion {
		return nil, registryError(localReferencesFile, fmt.Sprintf("version %d is unsupported", local.Version), nil)
	}
	for key, checkout := range local.Stores {
		if !namespaceName(key) || !filepath.IsAbs(checkout) || filepath.Clean(checkout) != checkout {
			return nil, registryError(localReferencesFile, fmt.Sprintf("store %q needs a clean absolute checkout path", key), nil)
		}
	}
	return &local, nil
}

// RenderLocalReferenceBindings emits deterministic local binding bytes.
func RenderLocalReferenceBindings(local LocalReferenceBindings) ([]byte, error) {
	if local.Version != referenceVersion {
		return nil, registryError(localReferencesFile, "unsupported local binding version", nil)
	}
	m := &ymap{}
	m.add("version", yscalar{"1"})
	stores := &ymap{}
	for _, key := range sortedKeys(local.Stores) {
		checkout := local.Stores[key]
		if !namespaceName(key) || !filepath.IsAbs(checkout) || filepath.Clean(checkout) != checkout {
			return nil, registryError(localReferencesFile, fmt.Sprintf("store %q needs a clean absolute checkout path", key), nil)
		}
		stores.addString(key, checkout)
	}
	m.add("stores", stores)
	var b strings.Builder
	m.writeTo(&b, 0)
	return []byte(b.String()), nil
}

// ReadReferenceRegistry reads only the tracked registry. Nil means the store
// has not declared any namespace; old and legacy references stay opaque.
func (s *Store) ReadReferenceRegistry() (*ReferenceRegistry, error) {
	data, err := os.ReadFile(filepath.Join(s.path, referencesFile))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, registryError(referencesFile, err.Error(), err)
	}
	return ParseReferenceRegistry(data)
}

func (s *Store) readLocalReferenceBindings() (*LocalReferenceBindings, error) {
	data, err := os.ReadFile(filepath.Join(s.path, localReferencesFile))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, registryError(localReferencesFile, err.Error(), err)
	}
	return ParseLocalReferenceBindings(data)
}

func (r *ReferenceRegistry) validIdentifier(ref string) bool {
	namespace, identifier, typed := splitRef(ref)
	if !typed {
		return true
	}
	declaration, declared := r.Namespaces[strings.ToLower(namespace)]
	if !declared {
		return true
	}
	re, err := compileIdentifier(declaration.Identifier)
	if err != nil {
		return false
	}
	match := re.FindStringSubmatchIndex(identifier)
	if match == nil {
		return false
	}
	if declaration.Kind == "ticket-store" {
		i := re.SubexpIndex("id")
		return i > 0 && match[2*i] == 0 && match[2*i+1] == len(identifier) && safeTicketName(identifier)
	}
	for _, token := range placeholder.FindAllString(declaration.Template, -1) {
		i := re.SubexpIndex(token[1 : len(token)-1])
		if i < 1 || match[2*i] < 0 || match[2*i] == match[2*i+1] {
			return false
		}
		value := identifier[match[2*i]:match[2*i+1]]
		if value == "." || value == ".." {
			return false
		}
	}
	return true
}

// ResolveReference resolves one stored reference without a network read.
// Undeclared namespaces return nil; a declared identifier that misses its
// grammar returns reference_identifier_invalid and keeps its original bytes.
func (s *Store) ResolveReference(ctx context.Context, ref Reference) (*ResolvedReference, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r, err := s.ReadReferenceRegistry()
	if err != nil || r == nil {
		return nil, err
	}
	namespace, identifier, typed := splitRef(ref.Ref)
	if !typed {
		return nil, nil
	}
	decl, ok := r.Namespaces[strings.ToLower(namespace)]
	if !ok {
		return nil, nil
	}
	if !r.validIdentifier(ref.Ref) {
		return nil, &Error{Code: CodeReferenceIdentifierInvalid, Field: "references.ref", Message: fmt.Sprintf("%q misses the declared %s identifier grammar", ref.Ref, namespace)}
	}
	re, _ := compileIdentifier(decl.Identifier) // validated by ReadReferenceRegistry
	match := re.FindStringSubmatchIndex(identifier)
	values := map[string]string{}
	for i, name := range re.SubexpNames() {
		if name != "" && match[2*i] >= 0 {
			values[name] = identifier[match[2*i]:match[2*i+1]]
		}
	}
	resolved := &ResolvedReference{Ref: ref.Ref}
	if decl.Kind == "url" {
		resolved.URL = placeholder.ReplaceAllStringFunc(decl.Template, func(token string) string {
			return url.PathEscape(values[token[1:len(token)-1]])
		})
	} else {
		if values["id"] != identifier || !safeTicketName(identifier) {
			return nil, &Error{Code: CodeReferenceIdentifierInvalid, Field: "references.ref", Message: "ticket-store id capture must cover a safe whole identifier"}
		}
		store := r.Stores[decl.Store]
		encoded := url.PathEscape(identifier)
		browse, _ := url.Parse(store.Browse) // validated by ReadReferenceRegistry
		if strings.Contains(browse.RawQuery, "{id}") {
			encoded = url.QueryEscape(identifier)
		}
		resolved.URL = strings.Replace(store.Browse, "{id}", encoded, 1)
		local, err := s.readLocalReferenceBindings()
		if err != nil {
			return nil, err
		}
		if local != nil {
			resolved.LocalPath = localTicketPath(local.Stores[decl.Store], store.Path, identifier)
		}
	}
	if ref.Path != nil {
		if own := safeExistingPath(s.Root(), *ref.Path); own != "" {
			resolved.LocalPath = own
		}
	}
	return resolved, nil
}

func safeTicketName(id string) bool {
	return id != "" && id != "." && id != ".." && !strings.ContainsAny(id, "/\\\x00")
}

func safeExistingPath(root, relative string) string {
	if root == "" || relative == "" || filepath.IsAbs(relative) {
		return ""
	}
	base, err := filepath.EvalSymlinks(root)
	if err != nil {
		return ""
	}
	candidate := filepath.Join(base, relative)
	real, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(base, real)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ""
	}
	return real
}

func localTicketPath(checkout, storePath, id string) string {
	if checkout == "" || !safeTicketName(id) {
		return ""
	}
	var found string
	for _, dir := range storeDirs {
		if candidate := safeExistingPath(checkout, filepath.Join(storePath, dir, id+".md")); candidate != "" {
			info, err := os.Stat(candidate)
			if err != nil || !info.Mode().IsRegular() {
				continue
			}
			if found != "" {
				return "" // duplicate ID in the checkout; do not choose one
			}
			found = candidate
		}
	}
	return found
}
