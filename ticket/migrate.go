package ticket

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
)

// MigrateOptions controls a schema migration.
type MigrateOptions struct {
	// To is the level to migrate to. Zero means SchemaVersion, which is what
	// somebody running the command with no argument is asking for.
	To int
	// DryRun plans the work and writes nothing.
	DryRun bool
}

// MigrateResult is what a migration did, or would have done under DryRun.
type MigrateResult struct {
	// From is the level config.yml declared when the pass started, and To is
	// the level it declares when the pass returns.
	From int
	To   int
	// ConfigChanged reports whether config.yml was rewritten. It is false on a
	// run that finishes an interrupted one, because that run's config already
	// arrived at the target.
	ConfigChanged bool
	// Tickets are the ticket files rewritten, relative to the store with
	// forward slashes, sorted, which is how a finding names a file.
	Tickets []string
	// Skipped counts the tickets already at the target. A run that finishes an
	// interrupted one reports the rest here, which makes idempotence visible
	// rather than merely true.
	Skipped int
	// Unreadable are the files that did not parse, so nothing could be said
	// about their level and nothing was written to them. check reports why.
	Unreadable []string
}

// Migrate converts the whole store to a schema level in one pass, per plan
// 12.5. A person runs the command and a host embedding this library calls this,
// because both need it and neither can drive the other.
//
// It is idempotent. A ticket already at the target is skipped rather than
// rewritten, so a run interrupted by a crash or a full disk is finished by
// running it again.
//
// It writes files and does not commit. Publishing stays the user's ordinary Git
// workflow, per 7.4 and the sync-helper decision in section 15.
//
// There is no downgrade. A field added in a later schema has nowhere to go in an
// earlier one, and a migration that quietly dropped it would lose work.
func (s *Store) Migrate(ctx context.Context, o MigrateOptions) (*MigrateResult, error) {
	target := o.To
	if target == 0 {
		target = SchemaVersion
	}
	if target < 1 {
		return nil, &Error{
			Code:    CodeInvalidField,
			Field:   "to",
			Message: fmt.Sprintf("schema %d is not a level: the first schema is 1", target),
		}
	}
	if target > SchemaVersion {
		return nil, &Error{
			Code:    CodeSchemaUnsupported,
			Field:   "to",
			Message: fmt.Sprintf("cannot migrate to schema %d, this binary supports %d", target, SchemaVersion),
			Details: map[string]string{"requested": strconv.Itoa(target), "supported": strconv.Itoa(SchemaVersion)},
		}
	}

	lock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer lock.release()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Read the declaration under the lock. This is the one operation that
	// changes it, so working from the copy taken at Open would let a second run
	// disagree with the first about where the store started.
	cfg, err := s.readConfig()
	if err != nil {
		return nil, err
	}
	from := cfg.Schema
	if from <= 0 {
		from = SchemaVersion
	}
	if target < from {
		return nil, &Error{
			Code:  CodeInvalidField,
			Field: "to",
			Message: fmt.Sprintf(
				"cannot migrate from schema %d down to %d: a field a later schema added has nowhere to go in an earlier one, so a downgrade would lose work",
				from, target),
		}
	}

	files, err := s.load()
	if err != nil {
		return nil, err
	}

	res := &MigrateResult{From: from, To: target, ConfigChanged: cfg.Schema != target}
	var pending []file
	for _, f := range files {
		switch {
		case f.Ticket == nil:
			res.Unreadable = append(res.Unreadable, f.Rel)
		case f.Ticket.Schema >= target:
			// Already there, or above the target because somebody edited it by
			// hand. Neither is rewritten: the first is the idempotence rule and
			// the second would be a downgrade of one file.
			res.Skipped++
		default:
			pending = append(pending, f)
			res.Tickets = append(res.Tickets, f.Rel)
		}
	}
	sort.Strings(res.Tickets)
	sort.Strings(res.Unreadable)

	if o.DryRun {
		return res, nil
	}

	// config.yml first, before any ticket. The two failure modes are not
	// symmetric: a config a reader does not understand refuses the whole store
	// loudly and on every command, while a ticket it does not understand drops
	// out of queries and is reported only by check. An interrupted migration
	// should leave a store an old reader refuses outright rather than one it
	// reads with tickets missing.
	if res.ConfigChanged {
		cfg.Schema = target
		if err := writeFileAtomic(filepath.Join(s.path, configFile), RenderConfig(cfg)); err != nil {
			return nil, err
		}
		s.config = cfg
	}

	for _, f := range pending {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		t := f.Ticket
		if err := promoteUnknown(t, target, f.Rel); err != nil {
			return nil, err
		}
		t.Schema = target
		// Rewritten in place, at the path it already occupies, with updated_at
		// and updated_by left alone. A migration is not a mutation by an actor:
		// stamping every ticket would claim the whole store was touched at one
		// instant, which destroys the recency order section 8 sorts by and puts
		// somebody's name on work they did not do.
		if err := writeFileAtomic(f.Path, Render(t)); err != nil {
			return nil, err
		}
	}
	return res, nil
}

// promoteUnknown moves a preserved key the target level defines into the struct
// field that now owns it.
//
// This is not tidying. A schema-1 file may legitimately carry origin, per 5.4,
// where it is an unknown field that round-trips after extensions. At schema 2
// that key is defined, so leaving it in Unknown would render it twice, once as
// the known null and once as the preserved copy, and the result is a file with
// a duplicate key that no longer parses the way it was written.
func promoteUnknown(t *Ticket, target int, rel string) error {
	if len(t.Unknown) == 0 {
		return nil
	}
	known := knownFieldsAt(target)
	kept := make([]UnknownField, 0, len(t.Unknown))
	for _, u := range t.Unknown {
		if !known[u.Key] {
			kept = append(kept, u)
			continue
		}
		var err error
		switch u.Key {
		case "origin":
			t.Origin, err = optionalString(u.Value, u.Key)
		default:
			// A key the target defines with nothing here to promote it into
			// means this function fell behind knownFieldsAt. Refusing beats
			// dropping the value or writing it twice.
			err = fmt.Errorf("no rule for promoting it to schema %d", target)
		}
		if err != nil {
			return &Error{
				Code:    CodeParseError,
				Ticket:  t.ID,
				Field:   u.Key,
				Message: fmt.Sprintf("%s: cannot migrate %s: %s", rel, u.Key, err),
			}
		}
	}
	t.Unknown = kept
	return nil
}

// readConfig re-reads config.yml from disk. Open parses it once, which is right
// for every other operation, and wrong for the one that rewrites it.
func (s *Store) readConfig() (Config, error) {
	data, err := os.ReadFile(filepath.Join(s.path, configFile))
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return Config{}, &Error{Code: CodeParseError, Message: err.Error(), Err: err}
	}
	return ParseConfig(data)
}
