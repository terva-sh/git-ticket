package ticket

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// Level is whether a rule can be settled mechanically. It is the only axis a
// rule has: there is no severity beside it, because check already spends
// "error" and "warning" on severity and one word cannot mean two things across
// two commands of one binary.
//
// A hard rule is checkable. "Every ticket carries at least one label" is true
// or false for a given ticket, and doctor may say so without qualification. A
// soft rule is a judgement doctor can prompt and cannot settle: nothing
// mechanical knows which of two labels describes a ticket better. The
// difference decides what the command is allowed to claim, which is why a soft
// finding reads as a question and may never fail a run.
type Level string

const (
	LevelHard Level = "hard"
	LevelSoft Level = "soft"
)

// Valid reports whether l is a level this binary knows. A config naming
// anything else is configuring nothing, and the caller says so rather than
// silently picking one.
func (l Level) Valid() bool { return l == LevelHard || l == LevelSoft }

// DoctorFinding is one thing doctor has to say about the store.
//
// It is its own type and not check's Finding. A doctor finding has to carry the
// rule that raised it and that rule's level, and Finding marshals exactly four
// keys that every fixture sidecar in testdata records. Growing that contract to
// carry two fields no check finding will ever have would rewrite the corpus for
// doctor's benefit, so doctor brings its own.
type DoctorFinding struct {
	// Rule is the stable identifier configuration refers to. It shares one
	// namespace with the codes in Finding, so a rule may name a check code when
	// it has to say where its own boundary is, and neither side may take a name
	// the other has spent.
	Rule  string `json:"rule"`
	Level Level  `json:"level"`
	// Ticket is the ticket this is about, and is empty only for a finding about
	// the store rather than about one of its tickets.
	Ticket string `json:"ticket"`
	File   string `json:"file"`
	// Message is what the rule found. A soft rule phrases it as a question,
	// because a tool that reports a judgement in the same voice as a fact
	// teaches people to skim both.
	Message string `json:"message"`
	// Remedy is what would resolve it. It is a field rather than a sentence
	// inside Message so that "every finding says what would resolve it" is a
	// thing a test can assert instead of a thing a reviewer has to read for.
	Remedy string `json:"remedy"`
}

// Grade is the worst level that fired, which is one ordered category and not a
// set of independent ones.
//
// That distinction is the test plan 10.2 sets for spending more of the exit
// status than one bit: it admits the reserved informational bucket of zypper
// and terraform's -detailed-exitcode and declines fsck's bitmask, because a
// grade stays readable in a shell comparison and a mask does not.
type Grade int

const (
	GradeClean Grade = iota
	GradeSoft
	GradeHard
)

// DoctorReport is what Doctor found.
type DoctorReport struct {
	// Findings are ordered hard first, then soft, so the first thing printed is
	// the thing most worth fixing.
	Findings []DoctorFinding `json:"findings"`
}

// Grade reports the worst level that fired.
func (r *DoctorReport) Grade() Grade {
	g := GradeClean
	for _, f := range r.Findings {
		if f.Level == LevelHard {
			return GradeHard
		}
		g = GradeSoft
	}
	return g
}

// RuleContext is what a rule is given. It holds the whole store rather than one
// ticket, so that a rule about a relation between tickets and a rule about one
// ticket are the same shape and the registry needs only one kind of entry.
type RuleContext struct {
	Tickets []*Ticket
	Config  Config
	// Params is what the store set under this rule's params key, empty when it
	// set none. A rule that takes no parameters ignores it.
	Params map[string]any
	// Level is the level this rule is running at, which is the level it ships
	// at unless the store moved it. A rule reads it rather than assuming its
	// own default, so that a rule moved to soft phrases itself as a question.
	Level Level
	// Now is the clock, injected so a rule about staleness is testable.
	Now time.Time
}

// Rule is one hygiene rule.
type Rule struct {
	// ID is the stable identifier. It is the compatibility surface of this
	// whole feature: it is what a store writes in config.yml and what schema
	// publishes, so renaming one breaks a file this tool does not own.
	ID string
	// Level is the level the rule ships at, which a store may override.
	Level Level
	// Summary is one line for `schema`, so a reader learns what a rule is for
	// without finding its source.
	Summary string
	// Check returns what the rule found. It returns findings rather than
	// appending to a report so that a rule cannot see, reorder, or suppress
	// what another rule said.
	Check func(RuleContext) []DoctorFinding
}

// RuleUnknown is the finding doctor raises for a rule identifier a store
// configured that this binary does not know.
//
// Doctor owns this and check does not, deliberately. check answers whether the
// store is valid, and a store whose validity depended on the rule registry
// would become invalid the day a rule was renamed, which would make validity a
// property of the binary rather than of the store. label_unknown and
// unknown_series each compare two things inside the store; a rule ID compares
// the store against this binary, which is a different relation and belongs to
// the command that owns the registry.
//
// It reports rather than refuses for a second reason. Refusing an ID this
// binary does not know would foreclose a store ever defining its own rules,
// which TKT-01M2NJDNMG5SBB6CEXY186HESF is left free to decide.
const RuleUnknown = "rule_unknown"

// Doctor reports what the hygiene rules have to say about the store.
//
// It never repairs and check never runs it. check is what CI runs and what
// --fix repairs, and it has to keep answering a yes-or-no question about
// validity; hygiene is advice and an untidy store is not a broken one.
func (s *Store) Doctor(ctx context.Context, rules []Rule, now time.Time) (*DoctorReport, error) {
	// The open set, which is what Filter{} means, per plan section 8: draft,
	// ready, in-progress, blocked and review, and not done or archived.
	//
	// Deliberate and not a default taken by accident. Hygiene is about a ticket
	// somebody might pick up, and nobody picks up an archived one. Measured on
	// this project's own store the difference is the whole feature: 11 findings
	// against the open set, 65 against every file, and the extra 54 are tickets
	// finished months ago that no one will label now. A report that large is one
	// a reader learns to skim, which is the failure this command exists to
	// avoid.
	tickets, err := s.List(ctx, Filter{})
	if err != nil {
		return nil, err
	}
	cfg := s.Config()

	byID := make(map[string]Rule, len(rules))
	for _, r := range rules {
		byID[r.ID] = r
	}

	report := &DoctorReport{Findings: []DoctorFinding{}}

	// A configured ID this binary does not know is reported before anything
	// runs, because it is the finding most likely to explain why the rest of
	// the report is not what the reader expected.
	for _, id := range sortedRuleIDs(cfg.Doctor.Rules) {
		if _, ok := byID[id]; ok {
			continue
		}
		report.Findings = append(report.Findings, DoctorFinding{
			Rule:    RuleUnknown,
			Level:   LevelHard,
			Message: fmt.Sprintf("config.yml configures %q, which this binary does not know", id),
			Remedy:  "correct the spelling, remove the entry, or upgrade git-ticket to a version that ships it",
			File:    "config.yml",
		})
	}

	for _, r := range rules {
		rc, on := ruleSettings(cfg, r)
		if !on {
			continue
		}
		report.Findings = append(report.Findings, r.Check(RuleContext{
			Tickets: tickets,
			Config:  cfg,
			Params:  rc.Params,
			Level:   rc.level(r),
			Now:     now,
		})...)
	}

	sortDoctorFindings(report.Findings)
	return report, nil
}

// ruleSettings resolves what the store said about one rule, and whether it runs
// at all.
func ruleSettings(cfg Config, r Rule) (RuleConfig, bool) {
	rc, ok := cfg.Doctor.Rules[r.ID]
	if !ok {
		// Shipped rules are on by default. A store that configures nothing gets
		// the house opinion, which is the only reason to have a hygiene command
		// rather than a linter everybody writes themselves.
		return RuleConfig{}, true
	}
	if rc.Enabled != nil && !*rc.Enabled {
		return rc, false
	}
	return rc, true
}

// level is the level a rule runs at: what the store said, or what the rule
// ships at. An unreadable level is ignored rather than guessed at, and
// config_test holds that to being the shipped level rather than a zero one.
func (rc RuleConfig) level(r Rule) Level {
	if rc.Level != nil && rc.Level.Valid() {
		return *rc.Level
	}
	return r.Level
}

// sortDoctorFindings puts every hard finding before every soft one, and orders
// within a level by rule then ticket, so two runs over an unchanged store
// compare directly.
func sortDoctorFindings(fs []DoctorFinding) {
	sort.SliceStable(fs, func(i, j int) bool {
		a, b := fs[i], fs[j]
		if a.Level != b.Level {
			return a.Level == LevelHard
		}
		if a.Rule != b.Rule {
			return a.Rule < b.Rule
		}
		return a.Ticket < b.Ticket
	})
}

func sortedRuleIDs(m map[string]RuleConfig) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
