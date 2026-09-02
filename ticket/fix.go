package ticket

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
)

// Repair is one move Fix made, or would have made under DryRun.
//
// Codes names the findings the move clears. A file can be wrong in both ways at
// once, sitting in the wrong directory under the wrong name, and one move
// settles both, so this is a list rather than a single code.
type Repair struct {
	Codes  []string
	Ticket string
	// From and To are relative to the store directory, with forward slashes,
	// which is how a finding names a file.
	From string
	To   string
}

// FixOptions controls a repair pass.
type FixOptions struct {
	// DryRun plans the repairs and applies none of them.
	DryRun bool
}

// FixResult is what a repair pass did and what the store looks like after.
type FixResult struct {
	Repairs []Repair
	// Report describes the store as it stands when the pass returns. Under
	// DryRun nothing was written, so it still shows the findings the planned
	// repairs would have cleared.
	Report *Report
}

// Fix repairs the two findings of plan section 11 that have exactly one correct
// repair, and touches nothing else.
//
// `filename_id_mismatch` is a rename to <id>.md, because plan 4 fixes the
// target name and leaves no second reading. `archive_location_mismatch` is a
// move, because 6.3 already rules that the status wins when the status and the
// directory disagree. Both are the destination rule writeTicket already
// follows.
//
// Every other finding needs a judgement this cannot make. `duplicate_id` has to
// choose which file keeps the ID, `dependency_missing` is repaired either by
// dropping the edge or by creating the ticket, and `label_unknown` is either a
// typo or a gap in the allowlist. A tool that guessed at those would be wrong
// about half of them and silent about it.
//
// The repair moves the bytes rather than re-rendering the ticket. Both findings
// are about where a file sits, so a pass that also rewrote its contents would
// be doing something the caller did not ask for.
//
// The lock is held across planning, moving, and the re-check, so the report
// describes the store the repairs left rather than one somebody else has since
// changed.
func (s *Store) Fix(ctx context.Context, o FixOptions) (*FixResult, error) {
	lock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer lock.release()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repairs, err := s.planRepairs()
	if err != nil {
		return nil, err
	}

	if !o.DryRun {
		for _, r := range repairs {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			from := filepath.Join(s.path, filepath.FromSlash(r.From))
			to := filepath.Join(s.path, filepath.FromSlash(r.To))
			if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
				return nil, &Error{Code: CodeValidationFailed, Message: err.Error(), Err: err}
			}
			if err := os.Rename(from, to); err != nil {
				return nil, &Error{
					Code:    CodeValidationFailed,
					Message: fmt.Sprintf("moving %s to %s: %s", r.From, r.To, err),
					Ticket:  r.Ticket,
					Err:     err,
				}
			}
		}
	}

	report, err := s.Check(ctx)
	if err != nil {
		return nil, err
	}
	return &FixResult{Repairs: repairs, Report: report}, nil
}

// planRepairs works out which files are in the wrong place and where each one
// belongs, without writing anything.
//
// A repair is dropped when its destination is already taken, or when two files
// want the same one. Both mean a second ticket is involved and only a person
// knows which file is the real one, which is the `duplicate_id` judgement this
// deliberately does not make. Dropping the repair leaves the finding, so the
// store still says what is wrong.
func (s *Store) planRepairs() ([]Repair, error) {
	files, err := s.load()
	if err != nil {
		return nil, err
	}

	occupied := make(map[string]bool, len(files))
	for _, f := range files {
		occupied[f.Rel] = true
	}

	claims := make(map[string]int)
	var candidates []Repair
	for _, f := range files {
		// A file that did not parse says nothing trustworthy about where it
		// belongs, and one finding is all it contributes anyway.
		if f.Ticket == nil {
			continue
		}
		want := s.relTarget(f.Ticket)
		if want == f.Rel {
			continue
		}

		var codes []string
		if path.Base(f.Rel) != f.Ticket.ID+".md" {
			codes = append(codes, CodeFilenameIDMismatch)
		}
		if path.Dir(f.Rel) != path.Dir(want) {
			codes = append(codes, CodeArchiveLocationMismatch)
		}
		candidates = append(candidates, Repair{
			Codes: codes, Ticket: f.Ticket.ID, From: f.Rel, To: want,
		})
		claims[want]++
	}

	out := make([]Repair, 0, len(candidates))
	for _, c := range candidates {
		if occupied[c.To] || claims[c.To] > 1 {
			continue
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].From < out[j].From })
	return out, nil
}

// relTarget is where a ticket belongs, relative to the store. The directory
// follows the status rather than where the file currently sits, per plan 6.3.
func (s *Store) relTarget(t *Ticket) string {
	dir := ticketsDir
	if t.Archived() {
		dir = archiveDir
	}
	return dir + "/" + t.ID + ".md"
}
