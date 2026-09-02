package ticket

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Finding is one thing check has to say about the store. The code is the stable
// part a caller switches on; the message is for a person and may change.
type Finding struct {
	Code string
	File string
	// Ticket is empty when the file did not parse far enough to know its ID.
	Ticket string
	// Field is empty when the finding is about the file rather than one field.
	Field string
	// Message is not part of the recorded contract, so it stays out of the
	// JSON. The fixture sidecars carry exactly code, file, ticket, and field.
	Message string `json:"-"`
}

// MarshalJSON writes the four keys of the contract, with null for an absent
// ticket or field. Absent scalars are null and never omitted, so a consumer
// never has to tell missing from empty.
func (f Finding) MarshalJSON() ([]byte, error) {
	type wire struct {
		Code   string  `json:"code"`
		File   string  `json:"file"`
		Ticket *string `json:"ticket"`
		Field  *string `json:"field"`
	}
	w := wire{Code: f.Code, File: f.File}
	if f.Ticket != "" {
		w.Ticket = &f.Ticket
	}
	if f.Field != "" {
		w.Field = &f.Field
	}
	return json.Marshal(w)
}

// Report is the result of a check. Errors fail the run; warnings do not, unless
// the caller asked for strict.
type Report struct {
	Errors   []Finding `json:"errors"`
	Warnings []Finding `json:"warnings"`
}

// OK reports whether the store passed. A warning does not fail a check.
func (r *Report) OK() bool { return len(r.Errors) == 0 }

func (r *Report) addError(f Finding)   { r.Errors = append(r.Errors, f) }
func (r *Report) addWarning(f Finding) { r.Warnings = append(r.Warnings, f) }

// sortFindings orders findings by file, then code, then field, so a caller can
// compare two reports directly instead of treating them as sets.
func sortFindings(fs []Finding) {
	sort.SliceStable(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Field < b.Field
	})
}

// Unreadable reports the ticket files the store could not parse, as the same
// findings Check would report for them.
//
// A query leaves those files out, per plan section 8, so a listing can be short
// without saying so. This is how a caller learns that, without running the rest
// of Check. The fields are built exactly as the block in Check that reports the
// same files, so a host sees one shape whichever command it called.
func (s *Store) Unreadable(ctx context.Context) ([]Finding, error) {
	files, err := s.load()
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := make([]Finding, 0)
	for _, f := range files {
		if f.Err == nil {
			continue
		}
		out = append(out, Finding{
			Code:    f.Err.Code,
			File:    f.Rel,
			Ticket:  f.Err.Ticket,
			Field:   f.Err.Field,
			Message: f.Err.Message,
		})
	}
	return out, nil
}

// Check validates the whole store, per plan section 11. It runs offline, reads
// no clock but the store's, and never writes.
func (s *Store) Check(ctx context.Context) (*Report, error) {
	files, err := s.load()
	if err != nil {
		return nil, err
	}
	r := &Report{Errors: []Finding{}, Warnings: []Finding{}}
	now := s.now()
	cfg := s.config
	root := s.Root()

	// live holds the tickets that could be read, indexed by ID. A file that
	// failed to parse contributes one finding and nothing else, because
	// everything downstream of a parse failure would be noise.
	live := make(map[string]*Ticket, len(files))
	byID := make(map[string][]string, len(files))
	var parsed []file

	for _, f := range files {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if f.Err != nil {
			r.addError(Finding{
				Code:    f.Err.Code,
				File:    f.Rel,
				Ticket:  f.Err.Ticket,
				Field:   f.Err.Field,
				Message: f.Err.Message,
			})
			continue
		}
		parsed = append(parsed, f)
		byID[f.Ticket.ID] = append(byID[f.Ticket.ID], f.Rel)
		// The first file wins the index. A duplicate is reported separately,
		// and the checks that follow only need one ticket per ID.
		if _, seen := live[f.Ticket.ID]; !seen {
			live[f.Ticket.ID] = f.Ticket
		}
	}

	for _, f := range parsed {
		t := f.Ticket
		errs, warns := checkTicket(t, f.Rel, cfg, root, now)
		for _, e := range errs {
			r.addError(e)
		}
		for _, w := range warns {
			r.addWarning(w)
		}

		if len(byID[t.ID]) > 1 {
			r.addError(Finding{
				Code: CodeDuplicateID, File: f.Rel, Ticket: t.ID, Field: "id",
				Message: fmt.Sprintf("id %s also appears in %s", t.ID, otherThan(byID[t.ID], f.Rel)),
			})
		}
		if want := t.ID + ".md"; filepath.Base(f.Rel) != want {
			r.addError(Finding{
				Code: CodeFilenameIDMismatch, File: f.Rel, Ticket: t.ID, Field: "id",
				Message: fmt.Sprintf("the file should be named %s", want),
			})
		}
		// The status is authoritative when it disagrees with the directory,
		// per plan 6.3.
		if inArchive := filepath.Dir(f.Rel) == archiveDir; inArchive != t.Archived() {
			r.addError(Finding{
				Code: CodeArchiveLocationMismatch, File: f.Rel, Ticket: t.ID, Field: "status",
				Message: archiveMismatchMessage(t.Archived()),
			})
		}

		for _, dep := range t.Dependencies {
			other, ok := live[dep]
			if !ok {
				r.addError(Finding{
					Code: CodeDependencyMissing, File: f.Rel, Ticket: t.ID, Field: "dependencies",
					Message: fmt.Sprintf("no ticket %s in this store", dep),
				})
				continue
			}
			// Archiving a ticket that was never done does not satisfy
			// anything, so a live ticket waiting on one is stuck without
			// saying so.
			if !t.Archived() && other.Archived() && !other.SatisfiesDependency() {
				r.addWarning(Finding{
					Code: CodeDependencyArchivedIncomplete, File: f.Rel, Ticket: t.ID, Field: "dependencies",
					Message: fmt.Sprintf("%s was archived without reaching done", dep),
				})
			}
		}
		if t.Parent != nil && *t.Parent != "" {
			if _, ok := live[*t.Parent]; !ok {
				r.addError(Finding{
					Code: CodeParentMissing, File: f.Rel, Ticket: t.ID, Field: "parent",
					Message: fmt.Sprintf("no ticket %s in this store", *t.Parent),
				})
			}
		}
	}

	// Cycles are computed over the tickets that exist, since a missing edge is
	// already reported as dependency_missing or parent_missing.
	ids := make([]string, 0, len(live))
	for id := range live {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	deps := make(map[string][]string, len(ids))
	parents := make(map[string][]string, len(ids))
	for _, id := range ids {
		t := live[id]
		for _, d := range t.Dependencies {
			if _, ok := live[d]; ok {
				deps[id] = append(deps[id], d)
			}
		}
		if t.Parent != nil {
			if _, ok := live[*t.Parent]; ok {
				parents[id] = append(parents[id], *t.Parent)
			}
		}
	}

	inDepCycle := cycleMembers(ids, deps)
	inParentCycle := cycleMembers(ids, parents)
	for _, f := range parsed {
		id := f.Ticket.ID
		if inDepCycle[id] {
			r.addError(Finding{
				Code: CodeDependencyCycle, File: f.Rel, Ticket: id, Field: "dependencies",
				Message: "this ticket is part of a dependency cycle",
			})
		}
		if inParentCycle[id] {
			r.addError(Finding{
				Code: CodeParentCycle, File: f.Rel, Ticket: id, Field: "parent",
				Message: "this ticket is part of a parent cycle",
			})
		}
	}

	sortFindings(r.Errors)
	sortFindings(r.Warnings)
	return r, nil
}

// checkTicket runs the checks one file can answer on its own, given the store's
// config, repository root, and clock. Everything that needs another file lives
// in Check.
func checkTicket(t *Ticket, rel string, cfg Config, root string, now time.Time) (errs, warns []Finding) {
	at := func(code, field, msg string) Finding {
		return Finding{Code: code, File: rel, Ticket: t.ID, Field: field, Message: msg}
	}

	// An unknown top-level field is an error here and a warning on an ordinary
	// read, per plan 5.4. The field itself survives a write either way.
	for _, u := range t.Unknown {
		errs = append(errs, at(CodeUnknownField, u.Key,
			fmt.Sprintf("%q is not a field this version defines", u.Key)))
	}
	if !ValidStatus(t.Status) {
		errs = append(errs, at(CodeInvalidStatus, "status",
			fmt.Sprintf("%q is not one of %v", t.Status, Statuses)))
	}
	if !ValidType(t.Type) {
		errs = append(errs, at(CodeInvalidType, "type",
			fmt.Sprintf("%q is not one of %v", t.Type, Types)))
	}
	if !ValidPriority(t.Priority) {
		errs = append(errs, at(CodeInvalidPriority, "priority",
			fmt.Sprintf("%q is not one of %v", t.Priority, Priorities)))
	}

	if t.Claim.Expired(now) {
		warns = append(warns, at(CodeClaimExpired, "claim.expires_at",
			fmt.Sprintf("the claim by %s expired at %s", t.Claim.Actor, t.Claim.ExpiresAt)))
	}
	if t.Status == StatusInProgress && t.Claim == nil {
		warns = append(warns, at(CodeInProgressUnclaimed, "claim",
			"in-progress with nobody holding it"))
	}
	for _, label := range t.Labels {
		if !cfg.KnownLabel(label) {
			warns = append(warns, at(CodeLabelUnknown, "labels",
				fmt.Sprintf("%q is not in the config.yml allowlist", label)))
		}
	}
	// A store outside a repository has no root to resolve against, so the
	// check is skipped rather than measured against a guess, per plan 5.5.
	if root != "" {
		for _, ref := range t.References {
			if ref.Path == nil || *ref.Path == "" {
				continue
			}
			p := *ref.Path
			if !filepath.IsAbs(p) {
				p = filepath.Join(root, p)
			}
			if _, err := os.Stat(p); err != nil {
				warns = append(warns, at(CodeReferencePathUnresolved, "references.path",
					fmt.Sprintf("%s does not resolve against the repository root", *ref.Path)))
			}
		}
	}
	return errs, warns
}

func archiveMismatchMessage(archived bool) string {
	if archived {
		return "status is archived but the file is in tickets/"
	}
	return "the file is in archive/ but the status is not archived"
}

func otherThan(paths []string, self string) string {
	for _, p := range paths {
		if p != self {
			return p
		}
	}
	return "another file"
}

// cycleMembers returns the nodes that lie on a cycle, by Tarjan's strongly
// connected components. Peeling nodes with no outgoing edges would be shorter
// but wrong: it also flags a ticket that merely depends on a cycle, which is
// not itself in one.
func cycleMembers(nodes []string, edges map[string][]string) map[string]bool {
	index := make(map[string]int, len(nodes))
	low := make(map[string]int, len(nodes))
	onStack := make(map[string]bool, len(nodes))
	out := map[string]bool{}
	var stack []string
	next := 0

	var visit func(v string)
	visit = func(v string) {
		index[v], low[v] = next, next
		next++
		stack = append(stack, v)
		onStack[v] = true

		for _, w := range edges[v] {
			if _, seen := index[w]; !seen {
				visit(w)
				if low[w] < low[v] {
					low[v] = low[w]
				}
			} else if onStack[w] && index[w] < low[v] {
				low[v] = index[w]
			}
		}

		if low[v] != index[v] {
			return
		}
		var component []string
		for {
			w := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[w] = false
			component = append(component, w)
			if w == v {
				break
			}
		}
		if len(component) > 1 {
			for _, w := range component {
				out[w] = true
			}
			return
		}
		// A component of one is a cycle only when it points at itself.
		for _, w := range edges[component[0]] {
			if w == component[0] {
				out[component[0]] = true
			}
		}
	}

	for _, n := range nodes {
		if _, seen := index[n]; !seen {
			visit(n)
		}
	}
	return out
}
