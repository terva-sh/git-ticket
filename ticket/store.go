package ticket

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// StoreDirName is the directory a store lives in, inside the repository it
// describes.
const StoreDirName = ".tickets"

// Directory names inside a store.
const (
	ticketsDir = "tickets"
	archiveDir = "archive"
	configFile = "config.yml"
	readmeFile = "README.md"
)

// Store is one .tickets directory.
type Store struct {
	path   string
	config Config

	// now is the clock. Tests inject a fixed instant, because claim_expired is
	// the one finding that depends on the time and a corpus judged against the
	// real clock would start failing on its own.
	now func() time.Time

	// root is the directory a references path resolves against, per plan 5.5.
	// It is the Git repository root holding the store, and empty when the
	// store sits outside a repository.
	root     string
	rootOnce sync.Once
	rootSet  bool

	// lockFile is where the store lock lives. Finding it runs git, so the
	// answer is computed once.
	lockFile string
	lockOnce sync.Once

	// lockTimeout overrides config.yml when it is not zero.
	lockTimeout time.Duration
}

// OpenOptions carries what a caller may need to override. The zero value is
// what production uses: the real clock, and a repository root discovered with
// git.
type OpenOptions struct {
	// Now replaces the clock. A test must set this rather than let the store
	// read the system clock.
	Now func() time.Time
	// Root fixes the directory a references path resolves against, skipping
	// discovery. A test uses this so its expectations do not depend on where
	// the corpus happens to be checked out.
	Root string
	// NoRoot declares that there is no repository root, so reference paths are
	// not resolved at all.
	NoRoot bool
	// LockTimeout overrides how long acquisition waits before returning
	// lock_timeout. Zero means the value in config.yml.
	LockTimeout time.Duration
}

// Open opens the store at path, which is the .tickets directory itself.
func Open(path string) (*Store, error) { return OpenWith(path, OpenOptions{}) }

// OpenWith opens the store at path with the given overrides.
func OpenWith(path string, opts OpenOptions) (*Store, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, &Error{Code: CodeStoreNotFound, Message: err.Error(), Err: err}
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return nil, codedError(CodeStoreNotFound, "no ticket store at %s", abs)
	}

	s := &Store{path: abs, now: opts.Now}
	if s.now == nil {
		s.now = time.Now
	}
	if opts.NoRoot {
		s.rootSet, s.root = true, ""
		s.rootOnce.Do(func() {})
	} else if opts.Root != "" {
		s.rootSet, s.root = true, opts.Root
		s.rootOnce.Do(func() {})
	}

	s.lockTimeout = opts.LockTimeout
	s.config = DefaultConfig()
	data, err := os.ReadFile(filepath.Join(abs, configFile))
	switch {
	case err == nil:
		cfg, perr := ParseConfig(data)
		if perr != nil {
			return nil, perr
		}
		s.config = cfg
	case !os.IsNotExist(err):
		return nil, &Error{Code: CodeParseError, Message: err.Error(), Err: err}
	}
	return s, nil
}

// Discover walks up from dir looking for a .tickets directory, stopping at the
// Git root, per plan 4. A --store flag or GIT_TICKET_STORE overrides this and
// is the caller's business, not the library's.
func Discover(dir string) (*Store, error) { return DiscoverWith(dir, OpenOptions{}) }

// DiscoverWith is Discover with the overrides of OpenWith.
func DiscoverWith(dir string, opts OpenOptions) (*Store, error) {
	start, err := filepath.Abs(dir)
	if err != nil {
		return nil, &Error{Code: CodeStoreNotFound, Message: err.Error(), Err: err}
	}
	for cur := start; ; {
		candidate := filepath.Join(cur, StoreDirName)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return OpenWith(candidate, opts)
		}
		// The Git root is the last place worth looking: a store above it
		// belongs to a different repository.
		if isGitRoot(cur) {
			break
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			break
		}
		cur = parent
	}
	return nil, codedError(CodeStoreNotFound,
		"no %s directory in %s or any parent up to the repository root", StoreDirName, start)
}

// InitOptions configures a new store.
type InitOptions struct {
	// Actor is recorded in config.yml as the first known actor, when set.
	Actor Actor
	// Labels seeds the advisory label allowlist.
	Labels []string
	Now    func() time.Time
}

// Init creates a store under root and returns it. root is the repository root,
// so the store lands at root/.tickets.
func Init(root string, opts InitOptions) (*Store, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, &Error{Code: CodeStoreNotFound, Message: err.Error(), Err: err}
	}
	path := filepath.Join(abs, StoreDirName)
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return nil, codedError(CodeStoreExists, "a ticket store already exists at %s", path)
	}

	cfg := DefaultConfig()
	if opts.Actor.ID != "" {
		cfg.Actors = append(cfg.Actors, opts.Actor)
	}
	if len(opts.Labels) > 0 {
		cfg.Labels = append(cfg.Labels, opts.Labels...)
	}

	for _, d := range []string{path, filepath.Join(path, ticketsDir), filepath.Join(path, archiveDir)} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, &Error{Code: CodeValidationFailed, Message: err.Error(), Err: err}
		}
	}
	if err := os.WriteFile(filepath.Join(path, configFile), RenderConfig(cfg), 0o644); err != nil {
		return nil, &Error{Code: CodeValidationFailed, Message: err.Error(), Err: err}
	}
	// The README is for whoever finds this directory in a diff. The tool never
	// reads it, so it is written once and never rewritten.
	if err := os.WriteFile(filepath.Join(path, readmeFile), []byte(storeReadme), 0o644); err != nil {
		return nil, &Error{Code: CodeValidationFailed, Message: err.Error(), Err: err}
	}
	return OpenWith(path, OpenOptions{Now: opts.Now})
}

// Path is the store directory.
func (s *Store) Path() string { return s.path }

// Config is the parsed config.yml, or the defaults when the store has none.
func (s *Store) Config() Config { return s.config }

// Now returns the store's clock reading.
func (s *Store) Now() time.Time { return s.now() }

// TicketsDir holds the live tickets, and ArchiveDir the archived ones.
func (s *Store) TicketsDir() string { return filepath.Join(s.path, ticketsDir) }
func (s *Store) ArchiveDir() string { return filepath.Join(s.path, archiveDir) }

// Root is the directory a references path resolves against: the Git repository
// holding the store. It is empty when the store is outside a repository, and
// check skips path resolution entirely in that case, per plan 5.5.
func (s *Store) Root() string {
	s.rootOnce.Do(func() {
		if s.rootSet {
			return
		}
		s.root = gitToplevel(s.path)
		s.rootSet = true
	})
	return s.root
}

// file is one .md file in the store, with whatever reading it produced. A file
// that failed to parse carries the error and no ticket.
type file struct {
	// Path is absolute; Rel is relative to the store directory with forward
	// slashes, which is how a finding names a file.
	Path   string
	Rel    string
	Ticket *Ticket
	Err    *Error
}

// files lists every ticket file in the store, live then archived, each set
// sorted by name. Sorting makes findings and listings reproducible without a
// sort at every call site.
func (s *Store) files() ([]string, error) {
	var out []string
	for _, dir := range []string{s.TicketsDir(), s.ArchiveDir()} {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, &Error{Code: CodeStoreNotFound, Message: err.Error(), Err: err}
		}
		var names []string
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
				continue
			}
			names = append(names, e.Name())
		}
		sort.Strings(names)
		for _, n := range names {
			out = append(out, filepath.Join(dir, n))
		}
	}
	return out, nil
}

// load reads every ticket in the store. A file that cannot be read is returned
// with its error rather than dropped, because check has to report it.
func (s *Store) load() ([]file, error) {
	paths, err := s.files()
	if err != nil {
		return nil, err
	}
	out := make([]file, 0, len(paths))
	for _, p := range paths {
		f := file{Path: p, Rel: s.rel(p)}
		t, err := ParseFile(p)
		if err != nil {
			var e *Error
			if !errors.As(err, &e) {
				e = &Error{Code: CodeParseError, Message: err.Error(), Err: err}
			}
			f.Err = e
		} else {
			f.Ticket = t
		}
		out = append(out, f)
	}
	return out, nil
}

func (s *Store) rel(path string) string {
	r, err := filepath.Rel(s.path, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(r)
}

// isGitRoot reports whether dir holds a .git entry. A worktree has a .git file
// rather than a directory, so both count.
func isGitRoot(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// gitToplevel asks git for the repository root holding dir. It returns an empty
// string when dir is outside a repository or git is not installed.
func gitToplevel(dir string) string {
	out, err := runGit(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// runGit runs a read-only git command in dir. Every git call in this package
// goes through here, and every one of them only reads: the library runs no git
// command that writes, per the policy in plan 7.3.
func runGit(dir string, args ...string) (string, error) {
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

const storeReadme = `# Tickets

This directory is a git-ticket store. Each file under ` + "`tickets/`" + ` is one ticket:
Markdown with YAML frontmatter, meant to be read, edited, diffed, and merged
like any other file in the repository. Archived tickets move to ` + "`archive/`" + `.

A filename is the ticket ID and nothing else, so renaming a title does not break
` + "`git log`" + ` on the old path.

You can edit these files by hand. Run ` + "`git ticket check`" + ` afterwards, which
reports what a hand edit tends to break: a duplicate ID, a dependency on a
ticket that does not exist, a status outside the set, or leftover merge conflict
markers.

` + "`config.yml`" + ` sets defaults and the label vocabulary. Nothing here is generated
from anywhere else, and nothing outside this directory has to be in sync with
it.
`
