package ticket

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// SeriesResult is what a read or a write of the series list answers with, per
// plan 5.6. Series is always the effective list after the operation, so a
// caller reads one shape whether it listed, added, or removed.
type SeriesResult struct {
	// Series is the effective list, which is never empty.
	Series []string
	// Changed is false when the operation wrote nothing. `series add` naming
	// one the store already declares is not an error, it is a no-op, so a
	// caller running it to be sure does not have to special-case success.
	Changed bool
	// PathsChanged is config.yml when the operation wrote, and empty when it
	// did not. It is the same field every mutation reports, per section 9.
	PathsChanged []string
}

// Series lists what the store declares, per plan 5.6. It is a read and takes no
// actor and no lock.
func (s *Store) Series() *SeriesResult {
	return &SeriesResult{Series: s.Config().EffectiveSeries()}
}

// AddSeries declares a prefix, per plan 5.6 and 12.1. Editing config.yml by
// hand does the same job; this exists so that adopting a series validates the
// grammar at the moment somebody asks for it, rather than at the next create.
//
// It writes config.yml rather than a ticket, which with RemoveSeries makes the
// two the only writes outside section 9. They still take the store lock,
// because a store whose series list is read by every create cannot have it
// changed underneath one.
func (s *Store) AddSeries(ctx context.Context, name string) (*SeriesResult, error) {
	name = strings.TrimSpace(name)
	if !ValidSeries(name) {
		return nil, seriesGrammarError(name)
	}

	lock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer lock.release()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Under the lock, like Migrate: this is one of the two operations that
	// change the declaration, so working from the copy taken at Open would let
	// two runs disagree about where the store started.
	cfg, err := s.readConfig()
	if err != nil {
		return nil, err
	}

	// A store that declares a series beyond TKT is at schema 2, per 5.6, and a
	// schema-1 store cannot render the field set that comes with it. Refusing
	// and naming the one command that fixes it beats writing a store an older
	// reader will silently misread.
	if !hasSeries(cfg.Schema) && name != DefaultSeries {
		return nil, &Error{
			Code: CodeValidationFailed,
			Message: fmt.Sprintf(
				"this store declares schema %s and a series needs schema %s; run git ticket migrate first",
				strconv.Itoa(cfg.Schema), strconv.Itoa(SchemaVersion)),
			Field: "series",
		}
	}

	if cfg.KnownSeries(name) {
		return &SeriesResult{Series: cfg.EffectiveSeries()}, nil
	}

	// From the effective list and not the literal one. A store that has written
	// no series key has [TKT] in effect, and starting from its empty literal
	// would write [IDEA] alone, turning every ticket already in the store into
	// an unknown_series error. This is the whole reason EffectiveSeries exists.
	cfg.Series = append(cfg.EffectiveSeries(), name)
	if err := writeFileAtomic(filepath.Join(s.path, configFile), RenderConfig(cfg)); err != nil {
		return nil, err
	}
	s.config = cfg
	return &SeriesResult{
		Series:       cfg.EffectiveSeries(),
		Changed:      true,
		PathsChanged: []string{configFile},
	}, nil
}

// RemoveSeries undeclares a prefix, per plan 5.6. It refuses while any ticket
// carries that prefix and names the count, because removing it turns every one
// of those tickets into an unknown_series error at the next check.
func (s *Store) RemoveSeries(ctx context.Context, name string) (*SeriesResult, error) {
	name = strings.TrimSpace(name)
	if !ValidSeries(name) {
		return nil, seriesGrammarError(name)
	}

	lock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer lock.release()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	cfg, err := s.readConfig()
	if err != nil {
		return nil, err
	}
	if !cfg.KnownSeries(name) {
		return &SeriesResult{Series: cfg.EffectiveSeries()}, nil
	}

	// An empty list means [TKT] rather than "none permitted", per 5.6, so
	// removing the last entry would not stick: the store would read back as
	// declaring exactly what it just dropped. Refuse rather than write
	// something that does not mean what it says.
	remaining := make([]string, 0, len(cfg.EffectiveSeries()))
	for _, existing := range cfg.EffectiveSeries() {
		if existing != name {
			remaining = append(remaining, existing)
		}
	}
	if len(remaining) == 0 {
		return nil, &Error{
			Code: CodeValidationFailed,
			Message: fmt.Sprintf(
				"%s is the only series this store declares, and an empty list means %s, so removing it would change nothing",
				name, DefaultSeries),
			Field: "series",
		}
	}

	held, err := s.countInSeries(name)
	if err != nil {
		return nil, err
	}
	if held > 0 {
		noun := "tickets"
		if held == 1 {
			noun = "ticket"
		}
		return nil, &Error{
			Code: CodeValidationFailed,
			Message: fmt.Sprintf(
				"%d %s in this store carry the series %s, and undeclaring it would report every one of them as unknown_series",
				held, noun, name),
			Field:   "series",
			Details: map[string]string{"count": strconv.Itoa(held), "series": name},
		}
	}

	cfg.Series = remaining
	if err := writeFileAtomic(filepath.Join(s.path, configFile), RenderConfig(cfg)); err != nil {
		return nil, err
	}
	s.config = cfg
	return &SeriesResult{
		Series:       cfg.EffectiveSeries(),
		Changed:      true,
		PathsChanged: []string{configFile},
	}, nil
}

// countInSeries counts every ticket carrying the prefix, unreadable ones
// included. A file too broken to parse still yields an ID from its name, per
// 5.6, and undeclaring a series out from under one is exactly as wrong as
// doing it to a ticket that parses.
func (s *Store) countInSeries(name string) (int, error) {
	files, err := s.load()
	if err != nil {
		return 0, err
	}
	n := 0
	for _, f := range files {
		id := f.id()
		if id == "" {
			continue
		}
		if series, _ := SplitID(id); series == name {
			n++
		}
	}
	return n, nil
}

// InUseSeries is every series some ticket in the store actually carries,
// sorted. It is what `check` compares against the declaration, and it is not
// the same as the declared list in either direction: a store may declare one
// nothing uses yet, and a hand-edited file may carry one nothing declares.
func (s *Store) InUseSeries() ([]string, error) {
	files, err := s.load()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []string
	for _, f := range files {
		id := f.id()
		if id == "" {
			continue
		}
		series, _ := SplitID(id)
		if series == "" || seen[series] {
			continue
		}
		seen[series] = true
		out = append(out, series)
	}
	sort.Strings(out)
	return out, nil
}

// seriesGrammarError is 5.6's grammar, said once. A malformed prefix is
// invalid_field rather than unknown_series, because the repair is to type a
// legal name and not to declare this one.
func seriesGrammarError(name string) error {
	return &Error{
		Code: CodeInvalidField,
		Message: fmt.Sprintf(
			"%q is not a series: %d to %d characters, uppercase letters and digits, and a letter first",
			name, SeriesMinLen, SeriesMaxLen),
		Field: "series",
	}
}
