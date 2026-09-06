package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/terva-sh/git-ticket/ticket"
)

// completionIDCap bounds what one tab press can produce. A ULID is unreadable
// and nobody scrolls a candidate list this far, so past here more candidates
// buy nothing and cost a wait. The filtering happens first, so the cap bites
// only on a store far larger than the prefix the user already typed.
const completionIDCap = 200

// The completion command generates shell completion scripts, and --dump is the
// single source of truth they read. Backlog.md's `completion install` is where
// the idea comes from; NOTICE records that.
//
// --dump reports what this binary accepts: every command, every flag of every
// command, and the enum values a flag takes. It is deliberately not a JSON
// kind under plan section 10 and is not covered by 12.4. It is an internal
// detail of how the scripts are built, and a generated script is free to stop
// using it. --json emits the same data for anything that would rather parse
// JSON than tab-separated lines.

// flagInfo is one flag of one command.
type flagInfo struct {
	Name string `json:"name"`
	// NeedsValue is false for a boolean flag, which a completion script must
	// not offer a value for.
	NeedsValue bool   `json:"needsValue"`
	Usage      string `json:"usage"`
}

// commandInfo is one subcommand and everything a script needs about it.
type commandInfo struct {
	Name    string `json:"name"`
	Usage   string `json:"usage"`
	Summary string `json:"summary"`
	// TakesID is derived from the usage line beginning with ID rather than
	// hand-listed, so a command added to the table offers ID completion with
	// no edit here. Plan 5.5 makes every such command accept a unique prefix.
	TakesID bool       `json:"takesId"`
	Flags   []flagInfo `json:"flags"`
}

// completionDump is the whole introspection payload.
type completionDump struct {
	Commands []commandInfo       `json:"commands"`
	Enums    map[string][]string `json:"enums"`
}

// completionShells is what `completion SHELL` accepts. fish and PowerShell are
// TKT-01M1W7AC8EW6TZXR6A8NYFFYYK, deferred until somebody asks for them.
var completionShells = []string{"bash", "zsh"}

// isBoolFlag reports whether a flag is a boolean, which the standard library
// exposes only through this optional interface on the value.
func isBoolFlag(f *flag.Flag) bool {
	bf, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && bf.IsBoolFlag()
}

// collectDump walks every command and captures its registered flags.
//
// It runs each command with captureFlags set, which stops it inside
// parseFlags once the flags exist and before it opens a store or writes
// anything. That is why this needs no store and no repository, and why adding
// a command or a flag needs no edit here.
func collectDump(env Env) completionDump {
	d := completionDump{Enums: map[string][]string{}}

	for _, c := range commands() {
		info := commandInfo{
			Name:    c.name,
			Usage:   c.usage,
			Summary: c.summary,
			TakesID: strings.HasPrefix(c.usage, "ID"),
		}
		sub := &cmdContext{g: &globals{}, env: env, out: io.Discard}
		sub.captureFlags = func(_ string, fs *flag.FlagSet) {
			fs.VisitAll(func(f *flag.Flag) {
				info.Flags = append(info.Flags, flagInfo{
					Name:       f.Name,
					NeedsValue: !isBoolFlag(f),
					Usage:      f.Usage,
				})
			})
		}
		// The error is always errFlagsCaptured. A command that somehow does
		// not call parseFlags simply reports no flags rather than failing the
		// dump, because a completion script with fewer candidates is better
		// than no completion script.
		_ = c.run(sub, nil)
		sort.Slice(info.Flags, func(i, j int) bool { return info.Flags[i].Name < info.Flags[j].Name })
		d.Commands = append(d.Commands, info)
	}

	// The enums a flag or a positional actually takes. These come from the
	// same package-level lists the validator uses, so a value added there
	// completes without an edit here.
	d.Enums["status"] = append([]string(nil), ticket.Statuses...)
	d.Enums["type"] = append([]string(nil), ticket.Types...)
	d.Enums["priority"] = append([]string(nil), ticket.Priorities...)
	d.Enums["sort"] = append([]string(nil), sortOrders...)
	d.Enums["ids"] = append([]string(nil), idModes...)
	d.Enums["shell"] = append([]string(nil), completionShells...)

	return d
}

// writeDumpLines emits the tab-separated form. A shell can read this with
// `IFS=$'\t' read -r`, with no jq on the box and no JSON parser in the script.
//
// Three record types, each with its kind in field one:
//
//	cmd  <name> <takesID 0|1> <usage> <summary>
//	flag <command> <flag> <needsValue 0|1>
//	enum <name> <space-separated values>
//
// The usage line is there so a script can complete positionals without
// restating them: a token like STATUS lowercases to an enum name, and a
// leading ID means a ticket goes there.
func writeDumpLines(w io.Writer, d completionDump) {
	for _, c := range d.Commands {
		fmt.Fprintf(w, "cmd\t%s\t%s\t%s\t%s\n", c.Name, boolDigit(c.TakesID), c.Usage, c.Summary)
	}
	for _, c := range d.Commands {
		for _, f := range c.Flags {
			fmt.Fprintf(w, "flag\t%s\t%s\t%s\n", c.Name, f.Name, boolDigit(f.NeedsValue))
		}
	}
	names := make([]string, 0, len(d.Enums))
	for k := range d.Enums {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		fmt.Fprintf(w, "enum\t%s\t%s\n", k, strings.Join(d.Enums[k], " "))
	}
}

func boolDigit(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// writeCompletionIDs prints candidate ticket IDs, one per line, as
// abbreviation then tab then title. bash takes the first field and zsh shows
// both, because zsh puts a description beside a candidate and bash does not.
//
// It never fails. A tab press outside a store, or in a broken one, offers
// nothing, and that is the right answer: a diagnostic printed into somebody's
// command line is worse than silence.
func writeCompletionIDs(ctx *cmdContext, prefix string) error {
	s, err := ctx.openStore()
	if err != nil {
		return nil
	}
	all, err := s.List(context.Background(), ticket.Filter{All: true})
	if err != nil {
		return nil
	}

	// idsStore is the ShortestUnique abbreviation across the whole store,
	// which is what plan 5.5 says a command will accept back. Offering a
	// candidate the CLI would then reject is the one thing completion must
	// never do.
	short := storeAbbreviations(s, all, idsStore)
	prefix = strings.ToUpper(prefix)

	var n int
	for _, t := range all {
		id := t.ID
		if a := short[t.ID]; a != "" {
			id = a
		}
		// Match on either form. Somebody who pasted a full ID should still see
		// it, and somebody typing gets the short one.
		if prefix != "" && !strings.HasPrefix(id, prefix) && !strings.HasPrefix(t.ID, prefix) {
			continue
		}
		fmt.Fprintf(ctx.out, "%s\t%s\n", id, t.Title)
		n++
		if n >= completionIDCap {
			break
		}
	}
	return nil
}

// completionScript is the script for one shell. The two differ more than a
// port: bash needs a shim function plus its own registration plus argument
// normalization, and zsh needs none of those because its _git has already
// consumed the git word before calling the handler.
func completionScript(shell string) string {
	switch shell {
	case "bash":
		return bashScript
	case "zsh":
		return zshScript
	}
	return ""
}

// completionInstallPath decides where a script belongs.
//
// The file names are not decoration. bash's loader asks for a file named after
// the binary, and zsh's _git scans $fpath for _git-* and calls the function of
// the same name, so getting either name wrong silently disables the
// subcommand form while the direct form keeps working.
func completionInstallPath(shell, dir string, env Env) (string, error) {
	switch shell {
	case "bash":
		if dir == "" {
			base := env.Getenv("XDG_DATA_HOME")
			if base == "" {
				home := env.Getenv("HOME")
				if home == "" {
					return "", usageErr("neither XDG_DATA_HOME nor HOME is set, so pass --dir")
				}
				base = filepath.Join(home, ".local", "share")
			}
			dir = filepath.Join(base, "bash-completion", "completions")
		}
		return filepath.Join(dir, "git-ticket"), nil
	case "zsh":
		// zsh has no standard user completion directory and no XDG
		// convention, and the directory has to be on $fpath before compinit
		// runs. Guessing would write a file that silently does nothing, so
		// this asks instead. Every tool surveyed does the same.
		if dir == "" {
			return "", usageErr("zsh has no standard completion directory, so --dir is required.\n" +
				"It must name a directory on your $fpath, set before compinit runs.")
		}
		return filepath.Join(dir, "_git-ticket"), nil
	}
	return "", usageErr("unknown shell %q", shell)
}

// shadowingGit returns the path of a git-shipped _git that may sit ahead of
// zsh's own on $fpath, or "" when it finds none.
//
// git ships its own zsh completion, and installing it ahead of zsh's replaces
// the _git that dispatches to _git-ticket. The subcommand form then stops
// working with no error anywhere. Homebrew's git does this on macOS, and
// git-lfs and git-extras both document hitting it.
//
// Reading the real $fpath would mean executing zsh, and plan 7.4 allows one
// non-git exec, the clipboard tools of 12.7. A warning is not worth a second
// exception, so this looks in the places such a file actually lives instead.
// That cannot prove the ordering, which is why the caller says "may shadow"
// rather than asserting it.
func shadowingGit(dir string, env Env) string {
	candidates := []string{filepath.Join(dir, "_git")}
	if home := env.Getenv("HOME"); home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".zsh", "completions", "_git"),
			filepath.Join(home, ".zfunc", "_git"),
		)
	}
	candidates = append(candidates,
		"/opt/homebrew/share/zsh/site-functions/_git",
		"/usr/local/share/zsh/site-functions/_git",
	)

	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		// git's version sources git-completion.bash. zsh's own does not, and
		// zsh's is the one that has to win.
		if strings.Contains(string(b), "git-completion.bash") {
			return p
		}
	}
	return ""
}

// installCompletion writes the script where the shell will find it.
func installCompletion(ctx *cmdContext, shell, dir string, force bool) error {
	path, err := completionInstallPath(shell, dir, ctx.env)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil && !force {
		return usageErr("%s already exists.\n"+
			"Pass --force to overwrite it, or print the script and redirect it yourself.", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(completionScript(shell)), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(ctx.out, "wrote %s\n", path)

	if shell == "zsh" {
		fmt.Fprintf(ctx.out, "\nIf %s is not on your $fpath already, add this to .zshrc above compinit:\n\n"+
			"  fpath=(%s $fpath)\n  autoload -Uz compinit && compinit\n\n"+
			"compinit caches its scan, so a new file may need: rm -f ~/.zcompdump && compinit\n",
			filepath.Dir(path), filepath.Dir(path))
		if shadow := shadowingGit(filepath.Dir(path), ctx.env); shadow != "" {
			fmt.Fprintf(ctx.env.Stderr,
				"warning: %s is git's own zsh completion rather than zsh's.\n"+
					"  If it comes before zsh's on your $fpath it replaces the _git that dispatches\n"+
					"  to _git-ticket, and `git ticket <TAB>` stops working with no error anywhere.\n"+
					"  `git-ticket <TAB>` is unaffected either way.\n", shadow)
		}
	}
	return nil
}

// runCompletion prints a completion script, or the dump the scripts read.
func runCompletion(ctx *cmdContext, args []string) error {
	var dump, ids, install, force bool
	var dir string
	rest, err := ctx.parseFlags("completion", args, func(fs *flag.FlagSet) {
		fs.BoolVar(&dump, "dump", false, "print what the scripts read: every command, its flags, and the enum values")
		fs.BoolVar(&ids, "ids", false, "print candidate ticket IDs, filtered by an optional prefix")
		fs.BoolVar(&install, "install", false, "write the script where the shell will find it")
		fs.StringVar(&dir, "dir", "", "directory to install into, required for zsh")
		fs.BoolVar(&force, "force", false, "overwrite an existing completion file")
	})
	if err != nil {
		return err
	}

	if ids {
		if len(rest) > 1 {
			return usageErr("completion --ids takes at most one prefix")
		}
		var prefix string
		if len(rest) == 1 {
			prefix = rest[0]
		}
		return writeCompletionIDs(ctx, prefix)
	}

	if dump {
		if len(rest) != 0 {
			return usageErr("completion --dump takes no shell")
		}
		d := collectDump(ctx.env)
		// A bare object rather than an envelope, deliberately. Every kind in
		// plan section 10 is a published contract under 12.4, and this is not
		// one: it is how the scripts are built and it may change with them.
		if ctx.g.json {
			writeJSON(ctx.out, d)
			return nil
		}
		writeDumpLines(ctx.out, d)
		return nil
	}

	if len(rest) != 1 {
		return usageErr("completion takes one shell: %s", strings.Join(completionShells, " or "))
	}
	shell := rest[0]
	if !slices.Contains(completionShells, shell) {
		return usageErr("unknown shell %q: completion supports %s", shell, strings.Join(completionShells, " and "))
	}
	if install {
		return installCompletion(ctx, shell, dir, force)
	}
	fmt.Fprint(ctx.out, completionScript(shell))
	return nil
}
