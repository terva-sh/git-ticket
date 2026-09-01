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
	Schema   int        `yaml:"schema"`
	Actors   []Actor    `yaml:"actors"`
	Labels   []string   `yaml:"labels"`
	Defaults Defaults   `yaml:"defaults"`
	Lock     LockConfig `yaml:"lock"`
}

// Defaults are the values a create uses when the caller names none.
type Defaults struct {
	Type     string `yaml:"type"`
	Priority string `yaml:"priority"`
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
		Schema:   SchemaVersion,
		Actors:   []Actor{},
		Labels:   []string{},
		Defaults: Defaults{Type: "task", Priority: "normal"},
		Lock:     LockConfig{Timeout: Duration(DefaultLockTimeout)},
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

	d := &ymap{}
	d.addString("type", c.Defaults.Type)
	d.addString("priority", c.Defaults.Priority)
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

// KnownLabel reports whether the label is in the advisory allowlist. An empty
// allowlist permits everything, since a store that never listed a label has not
// expressed an opinion.
func (c Config) KnownLabel(label string) bool {
	if len(c.Labels) == 0 {
		return true
	}
	for _, l := range c.Labels {
		if l == label {
			return true
		}
	}
	return false
}
