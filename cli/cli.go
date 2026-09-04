// Package cli is the flag parsing and the human rendering for git-ticket.
// Every decision it makes about the format or the store belongs to the ticket
// package; this package only turns arguments into calls and results into text.
//
// It is exported so a host can embed the whole command surface instead of
// writing a second one. Run is the only entry point and Env is the only
// configuration, so `terva ticket list` and `git ticket list` are the same
// code reached two ways. A host that cannot import this has to reimplement
// flag parsing, rendering, and error mapping over the ticket package, which is
// the second parser plan 12.1 exists to prevent. See plan 12.2.
//
// For structured values rather than rendered text, import the ticket package
// directly. Run returns an exit status and writes to Env's streams, which is
// what a command wants and not what a tool wants.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/terva-sh/git-ticket/ticket"
)

// Env is what the process supplies. A test fills it in rather than changing the
// working directory or the real environment.
type Env struct {
	Dir    string
	Getenv func(string) string
	Stdout io.Writer
	Stderr io.Writer
	// Stdin is what a `-` path reads, per plan 12.1. Nil reads as empty, so a
	// host that supplies no input gets an empty section rather than a panic.
	Stdin io.Reader
	// Now overrides the clock the store writes with. Nil means the real one.
	Now func() time.Time
}

func (e Env) getenv(key string) string {
	if e.Getenv == nil {
		return ""
	}
	return e.Getenv(key)
}

// exit statuses, per plan 10.2. Zero when the command did what it was asked and
// one otherwise, with the code in the error envelope carrying the detail.
const (
	exitOK    = 0
	exitError = 1
)

// codeUsage is the code for an argument the CLI could not make sense of. It is
// the CLI's own, since the codes in section 10 are about the store.
const codeUsage = "usage"

// globals are the flags every subcommand accepts, per plan 12.1. They are
// registered on the top-level flag set and again on each subcommand's, so
// `git ticket --json list` and `git ticket list --json` both work.
type globals struct {
	json        bool
	store       string
	ifRevision  string
	actor       string
	lockTimeout time.Duration
}

func (g *globals) register(fs *flag.FlagSet) {
	fs.BoolVar(&g.json, "json", g.json, "emit the JSON envelope on stdout")
	fs.StringVar(&g.store, "store", g.store, "path to the .tickets store")
	fs.StringVar(&g.ifRevision, "if-revision", g.ifRevision, "refuse the write unless the ticket is still at this revision")
	fs.StringVar(&g.actor, "actor", g.actor, "who is making the change, such as human:sothr")
	fs.DurationVar(&g.lockTimeout, "lock-timeout", g.lockTimeout, "how long to wait for the store lock")
}

// command is one subcommand.
type command struct {
	name    string
	summary string
	// usage is the argument line shown after the command name.
	usage string
	run   func(*cmdContext, []string) error
}

// commands is the dispatch table. Phase 2 fills it in; these four are the
// scaffold.
func commands() []command {
	return []command{
		{"init", "create a ticket store in this repository", "", runInit},
		{"create", "write a new ticket", "--title T [flags]", runCreate},
		{"update", "change a ticket's fields", "ID [flags]", runUpdate},
		{"show", "print one ticket", "ID", runShow},
		{"list", "print the tickets that match", "[filters]", runList},
		{"ready", "print what could be picked up now", "", runReady},
		{"search", "search titles, body sections, and references", "QUERY [--regex]", runSearch},
		{"status", "move a ticket through the lifecycle", "ID STATUS [--reason R]", runStatus},
		{"claim", "record that you are working a ticket", "ID [--expires-in D] [--force]", runClaim},
		{"release", "drop your claim on a ticket", "ID", runRelease},
		{"link", "add a dependency or a reference", "ID --depends-on OTHER | --ref R", runLink},
		{"unlink", "remove a dependency or a reference", "ID --depends-on OTHER | --ref R", runUnlink},
		{"deps", "print what a ticket waits on", "ID [--transitive] [--dependents]", runDeps},
		{"files", "print tickets that recorded a reference to a path", "PATH", runFiles},
		{"refs", "print tickets carrying a reference, whole or by namespace", "REF", runRefs},
		{"ac", "edit the acceptance criteria", "ID [--add T] [--check N] [--uncheck N] [--remove N]", runAC},
		{"dod", "edit the definition of done", "ID [--add T] [--check N] [--uncheck N] [--remove N]", runDoD},
		{"plan", "set the implementation plan, replacing it", "ID TEXT", runPlan},
		{"note", "append a note", "ID TEXT", runNote},
		{"comment", "append a comment", "ID TEXT", runComment},
		{"summary", "set the summary, replacing it", "ID TEXT", runSummary},
		{"archive", "archive a ticket, moving its file", "ID [--reason R]", runArchive},
		{"unarchive", "restore an archived ticket to ready", "ID", runUnarchive},
		{"remove", "delete a ticket filed by mistake", "ID [--force]", runRemove},
		{"check", "validate every ticket in the store", "[--strict] [--fix [--dry-run]]", runCheck},
		{"schema", "print the values and codes this binary enforces", "", runSchema},
		{"instructions", "print the agent workflow block for an AGENTS.md", "[--write]", runInstructions},
		{"install-merge-driver", "configure this binary as Git's merge driver for ticket files", "", runInstallMergeDriver},
		{"merge-driver", "resolve a ticket file mid-merge, for git's merge.*.driver", "BASE OURS THEIRS", runMergeDriver},
	}
}

// cmdContext is what a subcommand gets: the parsed globals and somewhere to
// write. It is not a context.Context, and it is named so that a file importing
// the context package can still say what it means.
type cmdContext struct {
	g   *globals
	env Env
	out io.Writer
}

// Run executes one invocation and returns the process exit status. args
// excludes the program name.
func Run(args []string, env Env) int {
	if env.Stdout == nil {
		env.Stdout = io.Discard
	}
	if env.Stderr == nil {
		env.Stderr = io.Discard
	}
	if env.Stdin == nil {
		env.Stdin = strings.NewReader("")
	}

	g := &globals{}
	top := flag.NewFlagSet("git ticket", flag.ContinueOnError)
	top.SetOutput(io.Discard)
	top.Usage = func() {}
	g.register(top)
	// --version is top level only, and deliberately not one of the globals.
	// Registering it on every subcommand would let `git ticket list --version`
	// set a flag the command then ignores.
	var showVersion bool
	top.BoolVar(&showVersion, "version", false, "print the build version and exit")

	if err := top.Parse(args); err != nil {
		// --help is a question at this level too, and parseFlags already answers
		// it one level down. The standard library reports -h and --help as an
		// error like any other, so without this a bare `git ticket --help` exits
		// 1 with "flag: help requested" while `git ticket help` exits 0 with the
		// usage, which is the first thing anybody types against a new binary.
		//
		// The name check below cannot cover it. Parse consumes the flag before
		// the command name is read, so that branch sees a literal "--help" only
		// after a bare --, as in `git ticket -- --help`.
		if errors.Is(err, flag.ErrHelp) {
			writeUsage(env.Stdout)
			return exitOK
		}
		return fail(env, g, usageErr("%v", err))
	}
	if showVersion {
		writeVersion(env.Stdout, g.json)
		return exitOK
	}
	rest := top.Args()
	if len(rest) == 0 {
		writeUsage(env.Stderr)
		return exitError
	}

	name, argv := rest[0], rest[1:]
	if name == "help" || name == "-h" || name == "--help" {
		writeUsage(env.Stdout)
		return exitOK
	}
	for _, c := range commands() {
		if c.name != name {
			continue
		}
		ctx := &cmdContext{g: g, env: env, out: env.Stdout}
		if err := c.run(ctx, argv); err != nil {
			if errors.Is(err, errHelpShown) {
				return exitOK
			}
			// A command that already said its piece exits nonzero without
			// an error envelope on top of it.
			if errors.Is(err, errReported) {
				return exitError
			}
			return fail(env, g, err)
		}
		return exitOK
	}
	return fail(env, g, usageErr("unknown command %q; run `git ticket help`", name))
}

// parseFlags parses a subcommand's flags with the globals registered, and
// returns the positional arguments.
//
// A flag may come before or after a positional argument. The standard library
// does not do that on its own: its parse stops at the first word that is not a
// flag, so `git ticket show ID --json` would leave --json sitting in the
// positional arguments and never set it. Parsing in a loop, consuming one
// positional each time round, gets both orders. Everything after a bare -- is
// positional whatever it looks like, which is how a title or a note can start
// with a dash.
func (ctx *cmdContext) parseFlags(name string, args []string, register func(*flag.FlagSet)) ([]string, error) {
	fs := flag.NewFlagSet("git ticket "+name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	ctx.g.register(fs)
	if register != nil {
		register(fs)
	}

	var literal []string
	for i, a := range args {
		if a == "--" {
			literal = append(literal, args[i+1:]...)
			args = args[:i]
			break
		}
	}

	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			// --help is a question, not a mistake. The standard library
			// reports it as an error like any other, and answering it with
			// "flag: help requested" on stderr tells a caller who asked what
			// a command takes that they got it wrong.
			if errors.Is(err, flag.ErrHelp) {
				writeCommandUsage(ctx.env.Stdout, name, fs)
				return nil, errHelpShown
			}
			return nil, usageErr("%v", err)
		}
		rest := fs.Args()
		if len(rest) == 0 {
			break
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
	return append(positional, literal...), nil
}

// errHelpShown means the command printed its own usage because the caller
// asked for it. Like errReported it travels as an error so that parseFlags can
// stop the command, but the exit status is zero: asking is not failing.
var errHelpShown = errors.New("usage was printed")

// errReported means the command ran, wrote its own output, and the verdict is
// no. check returns it for a store with findings: the report is already on
// stdout and exiting one is that verdict, not a sign the command could not run.
// Plan 10.3 draws the line there, which is why such a store gets a check-report
// and never an error envelope.
var errReported = errors.New("the command reported its own failure")

// usageError is an argument the CLI could not make sense of, as opposed to a
// store or a ticket refusing the operation.
type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

func usageErr(format string, args ...any) error {
	return &usageError{msg: fmt.Sprintf(format, args...)}
}

// fail writes the error the way plan section 10 requires: the message on stderr
// in both modes, and the envelope on stdout in JSON mode only.
func fail(env Env, g *globals, err error) int {
	code := ticket.CodeOf(err)
	var details map[string]string
	var ue *usageError
	switch {
	case errors.As(err, &ue):
		code = codeUsage
	case code == "":
		code = ticket.CodeValidationFailed
	}
	var te *ticket.Error
	if errors.As(err, &te) {
		details = te.Details
	}

	fmt.Fprintf(env.Stderr, "git-ticket: %s\n", err.Error())
	if g.json {
		writeJSON(env.Stdout, errorEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "error",
			Error: errorBody{
				Code:    code,
				Message: err.Error(),
				Details: details,
			},
		})
	}
	return exitError
}

// openStore resolves the store, per plan 12.1: --store, then GIT_TICKET_STORE,
// then discovery up to the Git root. A path given explicitly is never a hint to
// search somewhere else.
func (ctx *cmdContext) openStore() (*ticket.Store, error) {
	opts := ticket.OpenOptions{Now: ctx.env.Now, LockTimeout: ctx.g.lockTimeout}
	if ctx.g.store != "" {
		return ticket.OpenWith(ctx.g.store, opts)
	}
	if fromEnv := ctx.env.getenv("GIT_TICKET_STORE"); fromEnv != "" {
		return ticket.OpenWith(fromEnv, opts)
	}
	return ticket.DiscoverWith(ctx.env.Dir, opts)
}

// applyTo runs one mutation against the ticket named by ref.
//
// Every mutation goes through here so that --if-revision reaches all of them.
// A flag that promises to refuse a stale write has to be wired to every write,
// because the one it misses is the one a caller trusted it for.
func (ctx *cmdContext) applyTo(s *ticket.Store, ref string, m ticket.Mutation) (*ticket.Result, error) {
	return s.Apply(context.Background(), ref, m, ticket.ApplyOptions{
		IfRevision: ctx.g.ifRevision,
		Actor:      ctx.actor(s),
	})
}

// actor is who the mutation is recorded as. An --actor that names somebody in
// config.yml picks up their name; anybody else is recorded by ID alone. With no
// flag the store falls back to the first actor in config.yml.
func (ctx *cmdContext) actor(s *ticket.Store) ticket.Actor {
	if ctx.g.actor == "" {
		return ticket.Actor{}
	}
	for _, a := range s.Config().Actors {
		if a.ID == ctx.g.actor {
			return a
		}
	}
	return ticket.Actor{ID: ctx.g.actor}
}

// displayPath makes a path relative to the repository root, per plan section
// 10. A store outside a repository has no root, so the path stays absolute.
func displayPath(s *ticket.Store, path string) string {
	root := s.Root()
	if root == "" || path == "" {
		return path
	}
	if rel, ok := relativeTo(root, path); ok {
		return rel
	}
	// A repository reached through a symlink reports a root in a different name
	// space than the store path, and every macOS temporary directory is one of
	// those: /var/folders is a link to /private/var/folders. Resolving both
	// sides puts them back in the same space.
	resolvedRoot, rerr := filepath.EvalSymlinks(root)
	resolvedPath, perr := evalExisting(path)
	if rerr == nil && perr == nil {
		if rel, ok := relativeTo(resolvedRoot, resolvedPath); ok {
			return rel
		}
	}
	return path
}

// evalExisting resolves symlinks in the deepest part of path that still exists
// and puts the rest back on the end.
//
// EvalSymlinks alone fails on a path that is not there, and a mutation that
// moves a file reports the old location after deleting it. Archiving did
// exactly that: the new path came back repository-relative and the old one
// absolute, in the same pathsChanged array.
func evalExisting(path string) (string, error) {
	cur, rest := path, ""
	for {
		resolved, err := filepath.EvalSymlinks(cur)
		if err == nil {
			return filepath.Join(resolved, rest), nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", err
		}
		rest = filepath.Join(filepath.Base(cur), rest)
		cur = parent
	}
}

// relativeTo reports path relative to root, and whether it stayed inside it. A
// prefix test on ".." alone would also reject a sibling named "..foo".
func relativeTo(root, path string) (string, bool) {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

func displayPaths(s *ticket.Store, paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, displayPath(s, p))
	}
	return out
}

// writeCommandUsage prints one subcommand's usage in answer to --help: the
// argument line from the dispatch table, and every flag the FlagSet carries.
//
// It walks the FlagSet rather than repeating the flags in prose, so a flag
// added to a command shows up here without anybody remembering to write it
// down. The globals are registered on every subcommand's FlagSet, so they
// appear too, which is true: each one is accepted here.
func writeCommandUsage(w io.Writer, name string, fs *flag.FlagSet) {
	line := name
	for _, c := range commands() {
		if c.name != name {
			continue
		}
		if c.usage != "" {
			line += " " + c.usage
		}
		fmt.Fprintf(w, "%s\n\n", c.summary)
		break
	}
	fmt.Fprintf(w, "usage: git ticket %s\n\nflags:\n", line)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fs.VisitAll(func(f *flag.Flag) {
		// Two dashes, because that is the form every other page of this
		// project writes. The flag package accepts one or two either way.
		spec := "--" + f.Name
		if placeholder, _ := flag.UnquoteUsage(f); placeholder != "" {
			spec += " " + placeholder
		}
		fmt.Fprintf(tw, "  %s\t%s\n", spec, f.Usage)
	})
	tw.Flush()
}

func writeUsage(w io.Writer) {
	fmt.Fprint(w, `git ticket keeps a work ledger in the repository it describes.

usage: git ticket [--json] [--store PATH] <command> [flags]

commands:
`)
	// A tabwriter rather than a fixed pad, because two of these argument
	// lines are wider than any column width worth hard-coding.
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, c := range commands() {
		line := c.name
		if c.usage != "" {
			line += " " + c.usage
		}
		fmt.Fprintf(tw, "  %s\t%s\n", line, c.summary)
	}
	tw.Flush()
	fmt.Fprint(w, `
Global flags are accepted before or after the command:
  --json                 emit the JSON envelope described in the plan
  --store PATH           the .tickets directory to use
  --actor ID             who to record the change as
  --if-revision R        refuse a write if the ticket moved since R
  --lock-timeout D       how long to wait for the store lock

An ID may be abbreviated to any unique prefix of at least four characters.

files reports the references that agents recorded, so it is only as complete as
they were. It is advisory and is not derived from Git history.

Text that starts with a dash goes after a bare --, which ends the flags.
`)
}
