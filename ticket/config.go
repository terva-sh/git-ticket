package ticket

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is .tickets/config.yml, per plan 4.1. It sets defaults and vocabulary.
// It cannot add a status, change a transition rule, or grant a consumer
// authority it does not otherwise have.
type Config struct {
	Schema     int        `yaml:"schema"`
	Actors     []Actor    `yaml:"actors"`
	Labels     []string   `yaml:"labels"`
	Milestones []string   `yaml:"milestones"`
	Defaults   Defaults   `yaml:"defaults"`
	Lock       LockConfig `yaml:"lock"`
}

// Defaults are the values a create uses when the caller names none.
type Defaults struct {
	Type     string `yaml:"type"`
	Priority string `yaml:"priority"`
	// Actor is who a write is recorded as when the caller names none.
	// Declaring one is how a store says that choice is deliberate. Leave it
	// empty and a write still resolves, to the first entry in Actors, but
	// nothing chose that actor and the CLI warns, per 4.1.
	Actor string `yaml:"actor"`
	// ClaimExpiry is how long a new claim lasts. There is no default expiry,
	// so nil means a claim does not expire on its own.
	ClaimExpiry *Duration `yaml:"claim_expiry"`
}

// LockConfig holds the store lock settings, per plan 7.2.
type LockConfig struct {
	Timeout Duration `yaml:"timeout"`
}

// DefaultLockTimeout is how long lock acquisition blocks before returning
// lock_timeout, when config.yml does not say.
const DefaultLockTimeout = 10 * time.Second

// Duration is a time.Duration written the way a person writes one, "10s".
type Duration time.Duration

func (d Duration) Duration() time.Duration { return time.Duration(d) }

func (d Duration) String() string { return time.Duration(d).String() }

func (d *Duration) UnmarshalYAML(n *yaml.Node) error {
	if n.Tag == "!!null" {
		*d = 0
		return nil
	}
	parsed, err := time.ParseDuration(n.Value)
	if err != nil {
		return fmt.Errorf("%q is not a duration such as 10s or 2h", n.Value)
	}
	*d = Duration(parsed)
	return nil
}

// DefaultConfig is what init writes and what a store with no readable config
// falls back to.
func DefaultConfig() Config {
	return Config{
		Schema:     SchemaVersion,
		Actors:     []Actor{},
		Labels:     []string{},
		Milestones: []string{},
		Defaults:   Defaults{Type: "task", Priority: "normal"},
		Lock:       LockConfig{Timeout: Duration(DefaultLockTimeout)},
	}
}

// ParseConfig reads config.yml.
func ParseConfig(data []byte) (Config, error) {
	c := DefaultConfig()
	if err := yaml.Unmarshal(data, &c); err != nil {
		return c, &Error{Code: CodeParseError, Message: "config.yml: " + yamlMessage(err), Err: err}
	}
	if c.Schema > SchemaVersion {
		return c, &Error{
			Code:    CodeSchemaUnsupported,
			Message: fmt.Sprintf("config.yml declares schema %d, this reader supports %d", c.Schema, SchemaVersion),
			Field:   "schema",
		}
	}
	if c.Lock.Timeout == 0 {
		c.Lock.Timeout = Duration(DefaultLockTimeout)
	}
	return c, nil
}

// RenderConfig writes config.yml through the same emitter the tickets use, so
// a store this tool creates has one formatting style.
func RenderConfig(c Config) []byte {
	m := &ymap{}
	m.add("schema", yscalar{fmt.Sprint(c.Schema)})
	actors := &yseq{}
	for _, a := range c.Actors {
		am := &ymap{}
		am.addString("id", a.ID)
		am.addString("name", a.Name)
		actors.items = append(actors.items, am)
	}
	m.add("actors", actors)
	m.addStringSeq("labels", c.Labels)
	m.addStringSeq("milestones", c.Milestones)

	d := &ymap{}
	d.addString("type", c.Defaults.Type)
	d.addString("priority", c.Defaults.Priority)
	if c.Defaults.Actor == "" {
		// Rendered as null rather than omitted, so a reader of a fresh
		// config.yml can see the field exists and fill it in.
		d.add("actor", yscalar{"null"})
	} else {
		d.addString("actor", c.Defaults.Actor)
	}
	if c.Defaults.ClaimExpiry == nil {
		d.add("claim_expiry", yscalar{"null"})
	} else {
		d.addString("claim_expiry", c.Defaults.ClaimExpiry.String())
	}
	m.add("defaults", d)

	l := &ymap{}
	l.addString("timeout", c.Lock.Timeout.String())
	m.add("lock", l)

	var b strings.Builder
	m.writeTo(&b, 0)
	return []byte(b.String())
}

// DefaultActor returns the actor a write records when the caller names none,
// per plan 4.1.
//
// declared is true when defaults.actor named it, which is a store saying the
// choice is deliberate. It is false when the actor is merely the first entry in
// actors: nothing chose that one, it is just first, and that is the case worth
// warning about. ok is false when the store neither declares a default nor
// lists an actor, which is the case that refuses the write outright.
func (c Config) DefaultActor() (a Actor, declared bool, ok bool) {
	if id := c.Defaults.Actor; id != "" {
		return c.actorByID(id), true, true
	}
	if len(c.Actors) > 0 {
		return c.Actors[0], false, true
	}
	return Actor{}, false, false
}

// actorByID gives a configured actor its display name, so a declared default
// carries the same name an explicit --actor naming the same ID would pick up.
// An ID that is not in the roster is still usable, exactly as --actor is.
func (c Config) actorByID(id string) Actor {
	for _, a := range c.Actors {
		if a.ID == id {
			return a
		}
	}
	return Actor{ID: id}
}

// KnownLabel reports whether the label is in the advisory allowlist. An empty
// allowlist permits everything, since a store that never listed a label has not
// expressed an opinion.
func (c Config) KnownLabel(label string) bool {
	return permitted(c.Labels, label)
}

// KnownMilestone reports whether the milestone is in the advisory allowlist. It
// carries the same strength as KnownLabel, and for the same reason: a typo is
// worth reporting, and naming a new milestone should not need a config edit
// before the ticket can be filed.
//
// The empty milestone means the ticket names none, which is always permitted.
// Only a store that listed its milestones has expressed an opinion at all.
func (c Config) KnownMilestone(milestone string) bool {
	if milestone == "" {
		return true
	}
	return permitted(c.Milestones, milestone)
}

// permitted is the advisory allowlist rule of plan 4.1: an empty list is not an
// empty vocabulary, it is a store that has not expressed an opinion.
func permitted(allowed []string, value string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if a == value {
			return true
		}
	}
	return false
}
