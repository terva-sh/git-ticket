package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/terva-sh/git-ticket/ticket"
)

// The two halves of an installed driver, per plan 7.5. The name is what a
// `.gitattributes` entry points at and what the config keys hang off, so the
// three strings have to agree and are written once here.
const (
	mergeDriverName = "gitticket"
	mergeDriverDesc = "git-ticket three-way merge"
	attributesFile  = ".gitattributes"
)

// runMergeDriver is what Git invokes for a ticket file, per plan 7.5.
//
// The three arguments are Git's %O %A %B: the merge base, our version, and
// theirs. The result goes to the %A path, which is Git's contract and not a
// choice this makes. Exit zero means the file is clean and non-zero means it
// carries conflict markers for somebody to resolve.
//
// It opens no store. A driver runs on three temporary files Git wrote, which
// are not in the store and may not be in a repository this tool can find.
func runMergeDriver(ctx *cmdContext, args []string) error {
	rest, err := ctx.parseFlags("merge-driver", args, nil)
	if err != nil {
		return err
	}
	if len(rest) != 3 {
		return usageErr("merge-driver takes three paths: BASE OURS THEIRS")
	}
	basePath, oursPath, theirsPath := rest[0], rest[1], rest[2]

	base, err := os.ReadFile(basePath)
	if err != nil {
		return err
	}
	ours, err := os.ReadFile(oursPath)
	if err != nil {
		return err
	}
	theirs, err := os.ReadFile(theirsPath)
	if err != nil {
		return err
	}

	res, mergeErr := ticket.Merge(base, ours, theirs)
	if mergeErr != nil {
		// A file this tool cannot read must still not lose a side. Writing both
		// versions between markers is what Git's own driver would have done,
		// and it leaves the decision with the person rather than with a parser
		// that already said it does not understand the file.
		if err := writeResult(oursPath, wholeFileConflict(ours, theirs)); err != nil {
			return err
		}
		fmt.Fprintf(ctx.env.Stderr, "git-ticket: %s: %v\n", oursPath, mergeErr)
		return errReported
	}

	if err := writeResult(oursPath, res.Merged); err != nil {
		return err
	}
	if res.Clean() {
		return nil
	}
	fmt.Fprintf(ctx.env.Stderr, "git-ticket: %s: could not merge %s\n",
		oursPath, strings.Join(res.Conflicts, ", "))
	return errReported
}

// runInstallMergeDriver wires this executable in as the merge driver for the
// store's ticket files, per plan 7.5.
//
// Two pieces make an installed driver and a repository can supply one of them.
// The `.gitattributes` entry is tracked, so `init` writes it and a clone gets
// it. The config keys are not tracked, because Git refuses on purpose to take
// an executable name from a repository. That refusal is the reason this command
// exists: it makes the untracked half one command rather than a paragraph to
// transcribe.
//
// Running it twice is not an error. The second run finds both keys already set
// and says so, because a person who cannot tell an install from a re-install
// will run it again to be sure.
func runInstallMergeDriver(ctx *cmdContext, args []string) error {
	rest, err := ctx.parseFlags("install-merge-driver", args, nil)
	if err != nil {
		return err
	}
	if len(rest) > 0 {
		return usageErr("install-merge-driver takes no arguments")
	}
	s, err := ctx.openStore()
	if err != nil {
		return err
	}
	root := s.Root()
	if root == "" {
		return fmt.Errorf("a merge driver needs a Git repository, and %s is not in one",
			displayPath(s, s.Path()))
	}

	exe, err := driverExecutable()
	if err != nil {
		return err
	}
	driver := driverCommand(exe)

	// Reported rather than silently reapplied. Somebody who installed a
	// different build wants to see that this run replaced the path.
	var changed, already []string
	for _, kv := range []struct{ key, value string }{
		{"merge." + mergeDriverName + ".name", mergeDriverDesc},
		{"merge." + mergeDriverName + ".driver", driver},
	} {
		if readGit(root, "config", "--local", "--get", kv.key) == kv.value {
			already = append(already, kv.key)
			continue
		}
		if err := writeGit(root, "config", "--local", kv.key, kv.value); err != nil {
			return err
		}
		changed = append(changed, kv.key)
	}

	attrPath, added, err := ensureMergeAttribute(s)
	if err != nil {
		return err
	}

	var written []string
	if added {
		written = append(written, attrPath)
	}
	if ctx.g.json {
		writeJSON(ctx.out, mutationEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "mutation-result",
			Ticket:        nil,
			PathsChanged:  displayPaths(s, written),
		})
		return nil
	}

	fmt.Fprintf(ctx.out, "merge driver %q installed for %s\n", mergeDriverName, mergeAttrPattern(s))
	for _, k := range changed {
		fmt.Fprintf(ctx.out, "  set %s\n", k)
	}
	for _, k := range already {
		fmt.Fprintf(ctx.out, "  already set %s\n", k)
	}
	if added {
		fmt.Fprintf(ctx.out, "  wrote %s\n", displayPath(s, attrPath))
	} else {
		fmt.Fprintf(ctx.out, "  already in %s\n", displayPath(s, attrPath))
	}
	fmt.Fprintf(ctx.out, "  driver: %s\n", driver)
	return nil
}

// driverExecutable is the absolute path to the running binary.
//
// A driver line reading `git-ticket` fails the moment Git runs it from a
// context where that is not on PATH, and the failure surfaces as a merge
// conflict rather than as a missing command, so it is worth the resolve here.
func driverExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("cannot find this executable to name it as the driver: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return exe, nil
}

// driverCommand is the config value Git runs. Git hands the line to a shell,
// so a path with a space in it has to arrive quoted.
func driverCommand(exe string) string {
	if strings.ContainsAny(exe, " \t'\"\\$") {
		exe = "'" + strings.ReplaceAll(exe, "'", `'\''`) + "'"
	}
	return exe + " merge-driver %O %A %B"
}

// mergeAttrPattern matches every ticket file in this store, from the repository
// root. It is computed rather than written down, because a store reached with
// --store need not be named `.tickets` and need not sit at the root.
func mergeAttrPattern(s *ticket.Store) string {
	return filepath.ToSlash(displayPath(s, s.Path())) + "/**/*.md"
}

// ensureMergeAttribute appends the driver's `.gitattributes` entry if it is not
// already there, and reports the path either way.
//
// It appends rather than rewrites. The file belongs to the repository and not
// to this tool, so whatever else is in it survives untouched.
func ensureMergeAttribute(s *ticket.Store) (path string, added bool, err error) {
	root := s.Root()
	if root == "" {
		return "", false, nil
	}
	path = filepath.Join(root, attributesFile)
	want := mergeAttrPattern(s) + " merge=" + mergeDriverName

	current, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return "", false, err
	}
	for _, line := range strings.Split(string(current), "\n") {
		if strings.TrimSpace(line) == want {
			return path, false, nil
		}
	}

	var b strings.Builder
	b.Write(current)
	if len(current) > 0 && current[len(current)-1] != '\n' {
		b.WriteString("\n")
	}
	b.WriteString(want + "\n")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return "", false, err
	}
	return path, true, nil
}

// writeGit runs the one Git command in plan 7.4 that writes.
//
// It is separate from readGit because readGit answers with a string and treats
// a failure as an empty answer. A write that failed silently is the worst of
// the outcomes here, so this one returns the error and the command's output
// with it.
func writeGit(dir string, args ...string) error {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	if err != nil {
		if msg := strings.TrimSpace(string(out)); msg != "" {
			return fmt.Errorf("git %s: %s", args[0], msg)
		}
		return fmt.Errorf("git %s: %w", args[0], err)
	}
	return nil
}

// writeResult replaces the %A file, keeping the mode Git gave it.
func writeResult(path string, data []byte) error {
	mode := os.FileMode(0o644)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(path, data, mode)
}

// wholeFileConflict is the fallback for a file the parser refused: both
// versions, marked, with nothing dropped.
func wholeFileConflict(ours, theirs []byte) []byte {
	var b strings.Builder
	b.WriteString("<<<<<<< ours\n")
	b.Write(ours)
	if len(ours) > 0 && ours[len(ours)-1] != '\n' {
		b.WriteString("\n")
	}
	b.WriteString("=======\n")
	b.Write(theirs)
	if len(theirs) > 0 && theirs[len(theirs)-1] != '\n' {
		b.WriteString("\n")
	}
	b.WriteString(">>>>>>> theirs\n")
	return []byte(b.String())
}
