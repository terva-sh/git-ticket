package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/terva-sh/git-ticket/ticket"
)

// The graded informational bucket doctor exits by under --strict, reserved in
// plan 10.2 beside self-update's 10 through 12.
//
// A grade and not a mask: the number is the worst level that fired, which is
// one ordered category, so `doctor --strict; test $? -ge 21` asks "did anything
// objective fail" with no parsing. The gap above self-update's range is left
// deliberately, so neither bucket has to move if either grows.
const (
	exitDoctorSoft = 20
	exitDoctorHard = 21
)

// doctorRules is the rule set the command runs.
//
// A variable rather than a direct call to ticket.DefaultRules so that a test
// can install rules of its own. That seam is not a convenience: DefaultRules is
// empty until TKT-01M2NJDAVHTXKEPJ0ZCAJP3QY0 lands the first two rules, so
// without it every grade, every level and every config override below would
// ship with no test that had ever run one.
var doctorRules = ticket.DefaultRules

// runDoctor reports hygiene: the things that are true of a ticket nobody can
// pick up and are not things check would call invalid.
//
// It is a separate command from check and not a flag on it, because check is
// what CI runs and what --fix repairs and it has to keep answering a yes-or-no
// question about validity. Hygiene is advice, it is opinionated on purpose, and
// an untidy store should not fail a build.
func runDoctor(ctx *cmdContext, args []string) error {
	var strict bool
	rest, err := ctx.parseFlags("doctor", args, func(fs *flag.FlagSet) {
		fs.BoolVar(&strict, "strict", false, "exit by the graded bucket rather than always zero")
	})
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usageErr("doctor takes no arguments; it reads the whole store")
	}

	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	report, err := s.Doctor(context.Background(), doctorRules(), time.Now().UTC())
	if err != nil {
		return err
	}

	if ctx.g.json {
		writeJSON(ctx.out, newDoctorEnvelope(s, report))
	} else {
		writeDoctorHuman(ctx.out, s, report, strict)
	}

	// Without --strict the status is always zero. Hygiene is advice, and advice
	// that fails a build is a rule.
	if !strict {
		return nil
	}
	switch report.Grade() {
	case ticket.GradeHard:
		return &exitStatusErr{code: exitDoctorHard}
	case ticket.GradeSoft:
		return &exitStatusErr{code: exitDoctorSoft}
	}
	return nil
}

// writeDoctorHuman prints the findings hard first, then a line saying what the
// grade was, because a reader who ran --strict and got a number needs to know
// which number without looking it up.
func writeDoctorHuman(w io.Writer, s *ticket.Store, r *ticket.DoctorReport, strict bool) {
	if len(r.Findings) == 0 {
		fmt.Fprintln(w, "Nothing to tidy.")
		return
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, f := range r.Findings {
		id := f.Ticket
		if id == "" {
			id = "-"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", f.Level, f.Rule, id, f.Message)
		// The remedy goes on its own line rather than a sixth column. A finding
		// that says what would resolve it is the whole point, and a column wide
		// enough for a sentence makes every other column unreadable.
		fmt.Fprintf(tw, "\t\t\t%s\n", f.Remedy)
	}
	tw.Flush()

	hard, soft := 0, 0
	for _, f := range r.Findings {
		if f.Level == ticket.LevelHard {
			hard++
			continue
		}
		soft++
	}
	fmt.Fprintf(w, "\n%s, %s\n", count(hard, "hard finding"), count(soft, "soft finding"))
	if !strict {
		return
	}
	switch r.Grade() {
	case ticket.GradeHard:
		fmt.Fprintf(w, "Exiting %d: hard findings.\n", exitDoctorHard)
	case ticket.GradeSoft:
		fmt.Fprintf(w, "Exiting %d: soft findings only, which are questions rather than failures.\n", exitDoctorSoft)
	}
}

// doctorEnvelope is the doctor-report kind of section 10.
type doctorEnvelope struct {
	SchemaVersion int    `json:"schemaVersion"`
	Kind          string `json:"kind"`
	// Grade is the worst level that fired, spelled rather than numbered. A
	// consumer reading JSON has no use for the exit number, and spelling it
	// keeps the envelope readable if the reserved numbers ever move.
	Grade    string              `json:"grade"`
	Findings []doctorFindingJSON `json:"findings"`
}

type doctorFindingJSON struct {
	Rule   string `json:"rule"`
	Level  string `json:"level"`
	Ticket string `json:"ticket"`
	// File is store-relative the way a check finding's path is, so the two
	// reports name the same file the same way.
	File    string `json:"file"`
	Message string `json:"message"`
	Remedy  string `json:"remedy"`
}

func newDoctorEnvelope(s *ticket.Store, r *ticket.DoctorReport) doctorEnvelope {
	env := doctorEnvelope{
		SchemaVersion: schemaVersion,
		Kind:          "doctor-report",
		Grade:         gradeName(r.Grade()),
		Findings:      make([]doctorFindingJSON, 0, len(r.Findings)),
	}
	for _, f := range r.Findings {
		out := doctorFindingJSON{
			Rule:    f.Rule,
			Level:   string(f.Level),
			Ticket:  f.Ticket,
			Message: f.Message,
			Remedy:  f.Remedy,
		}
		if f.File != "" {
			out.File = storePath(s, filepath.Join(s.Path(), f.File))
		}
		env.Findings = append(env.Findings, out)
	}
	return env
}

func gradeName(g ticket.Grade) string {
	switch g {
	case ticket.GradeHard:
		return "hard"
	case ticket.GradeSoft:
		return "soft"
	}
	return "clean"
}
