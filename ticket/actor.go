package ticket

import (
	"context"
	"path/filepath"
	"strings"
)

// ActorResult is what an actor read or write answers with.
type ActorResult struct {
	// Actors is the roster the store now declares.
	Actors []Actor
	// Default is the ID of defaults.actor, empty when the store declares none.
	Default string
	// Changed is false when the operation wrote nothing. Adding an actor the
	// store already lists is a no-op rather than an error, so a caller running
	// it to be sure does not have to special-case success, which is the rule
	// SeriesResult.Changed follows.
	Changed bool
	// PathsChanged is config.yml when the operation wrote, and empty otherwise.
	PathsChanged []string
}

// Actors reads the roster. It writes nothing and needs no actor of its own,
// which matters here more than it does for Series: this is the read a caller
// makes when it does not yet know whether the store has an actor at all.
func (s *Store) Actors() *ActorResult {
	cfg := s.config
	return &ActorResult{
		Actors:  append([]Actor(nil), cfg.Actors...),
		Default: cfg.Defaults.Actor,
	}
}

// AddActor declares an actor in config.yml, and with makeDefault also names it
// defaults.actor.
//
// This exists because of a bootstrapping problem, per TKT-01M2NT7QS84SB5P7DR2GXB9DZ0.
// `init` with no --actor and no terminal leaves a store whose roster is empty,
// which is the right outcome for a script, and until now the only way to fill
// that roster was a text editor.
//
// Like AddSeries it writes config.yml rather than a ticket, which is what lets
// it run against a store that has no actor to record the write as. A mutation
// under section 9 could not: resolveActor refuses when the caller names nobody
// and the config declares nobody, so the one operation that fixes an empty
// roster would have been refused by the emptiness it fixes.
//
// makeDefault is not a convenience. Adding the first actor without it leaves a
// store that resolves writes to whoever happens to head the roster and warns on
// every one, and the warning's own advice is to set defaults.actor, which would
// send the caller back to the editor this command exists to replace.
func (s *Store) AddActor(ctx context.Context, id, name string, makeDefault bool) (*ActorResult, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, &Error{Code: CodeInvalidField, Message: "an actor needs an id", Field: "actor"}
	}

	lock, err := s.lock()
	if err != nil {
		return nil, err
	}
	defer lock.release()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// Under the lock and re-read, like AddSeries: this changes the declaration
	// every write resolves its actor against, so working from the copy taken at
	// Open would let two runs disagree about where the store started.
	cfg, err := s.readConfig()
	if err != nil {
		return nil, err
	}

	changed := false
	found := false
	for i, a := range cfg.Actors {
		if a.ID != id {
			continue
		}
		found = true
		// A name given for an actor already listed updates it. An absent name
		// leaves the one already there rather than clearing it, so re-running
		// the command to set a default cannot quietly drop a display name.
		if name != "" && a.Name != name {
			cfg.Actors[i].Name = name
			changed = true
		}
	}
	if !found {
		cfg.Actors = append(cfg.Actors, Actor{ID: id, Name: name})
		changed = true
	}
	if makeDefault && cfg.Defaults.Actor != id {
		cfg.Defaults.Actor = id
		changed = true
	}

	if !changed {
		return &ActorResult{Actors: append([]Actor(nil), cfg.Actors...), Default: cfg.Defaults.Actor}, nil
	}
	if err := writeFileAtomic(filepath.Join(s.path, configFile), RenderConfig(cfg)); err != nil {
		return nil, err
	}
	s.config = cfg
	return &ActorResult{
		Actors:       append([]Actor(nil), cfg.Actors...),
		Default:      cfg.Defaults.Actor,
		Changed:      true,
		PathsChanged: []string{configFile},
	}, nil
}

// There is deliberately no RemoveActor.
//
// Removing an actor is not the mirror of removing a series. A series lives in
// the ID of every ticket carrying it, so RemoveSeries can count them and refuse.
// An actor is recorded in created_by, updated_by, every note and comment, and
// every claim, and those are history rather than vocabulary: a ticket saying who
// wrote a note in March is still true after that person leaves. So removal
// cannot rewrite what it finds, and a roster entry removed while the history
// keeps the ID means display names silently empty out on the next write to any
// of those tickets, which is the behaviour TKT-01M2NT7VNBAM5KSAR0QKBT2JQD just
// documented.
//
// That is a decision about history rather than a missing symmetry, so it is left
// to its own ticket rather than guessed at here.
