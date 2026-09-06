package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// dumpLines runs `completion --dump` and splits it into records.
func dumpLines(t *testing.T, dir string, args ...string) [][]string {
	t.Helper()
	got := runCLI(t, dir, nil, append([]string{"completion", "--dump"}, args...)...)
	if got.code != exitOK {
		t.Fatalf("completion --dump: %s", got.stderr)
	}
	var out [][]string
	for _, line := range strings.Split(strings.TrimRight(got.stdout, "\n"), "\n") {
		if line == "" {
			continue
		}
		out = append(out, strings.Split(line, "\t"))
	}
	return out
}

// TestCompletionDumpNeedsNoStore is the property the whole design rests on.
// The dump walks every command with captureFlags set, which stops each one
// inside parseFlags before it opens anything, so a completion script works in
// a directory that is not a repository and has no store. A bogus --store is
// the sharper half: if any command reached its store, this would fail.
func TestCompletionDumpNeedsNoStore(t *testing.T) {
	dir := t.TempDir()

	records := dumpLines(t, dir, "--store", "/nonexistent-store-abc")

	var commandCount int
	for _, r := range records {
		if r[0] == "cmd" {
			commandCount++
		}
	}
	if commandCount != len(commands()) {
		t.Errorf("dump reported %d commands, want %d", commandCount, len(commands()))
	}
}

// TestCompletionDumpNamesEveryCommand holds the dump to the dispatch table. A
// command added to commands() must appear here with no edit to the generator,
// which is the point of reading the table rather than restating it.
func TestCompletionDumpNamesEveryCommand(t *testing.T) {
	dir := t.TempDir()

	named := map[string]bool{}
	for _, r := range dumpLines(t, dir) {
		if r[0] == "cmd" {
			named[r[1]] = true
		}
	}
	for _, c := range commands() {
		if !named[c.name] {
			t.Errorf("commands() has %q but the dump does not name it", c.name)
		}
	}
}

// TestCompletionDumpDerivesTakesIDFromUsage. Which commands offer ID
// completion is derived from the usage line, not hand-listed, so a new command
// taking an ID inherits it. This holds the derivation to the table itself
// rather than to a copy of the answer.
func TestCompletionDumpDerivesTakesIDFromUsage(t *testing.T) {
	dir := t.TempDir()

	got := map[string]string{}
	for _, r := range dumpLines(t, dir) {
		if r[0] == "cmd" {
			got[r[1]] = r[2]
		}
	}
	for _, c := range commands() {
		want := "0"
		if strings.HasPrefix(c.usage, "ID") {
			want = "1"
		}
		if got[c.name] != want {
			t.Errorf("%s takesID = %s, want %s (usage %q)", c.name, got[c.name], want, c.usage)
		}
	}
}

// TestCompletionDumpMarksBoolFlagsAsTakingNoValue. A script that offers a
// value after --json produces a completion nobody can use, so the boolean-ness
// has to survive into the dump. --json is a global bool on every command and
// --actor is a global string, which makes them the pair to check.
func TestCompletionDumpMarksBoolFlagsAsTakingNoValue(t *testing.T) {
	dir := t.TempDir()

	type key struct{ cmd, flag string }
	needsValue := map[key]string{}
	for _, r := range dumpLines(t, dir) {
		if r[0] == "flag" {
			needsValue[key{r[1], r[2]}] = r[3]
		}
	}

	if got := needsValue[key{"archive", "json"}]; got != "0" {
		t.Errorf("archive --json needsValue = %q, want 0: it is a bool", got)
	}
	if got := needsValue[key{"archive", "actor"}]; got != "1" {
		t.Errorf("archive --actor needsValue = %q, want 1: it takes a string", got)
	}
	// A flag the command itself registers, rather than a global, so this also
	// proves the per-command register closure ran.
	if got := needsValue[key{"archive", "reason"}]; got != "1" {
		t.Errorf("archive --reason needsValue = %q, want 1", got)
	}
}

// TestCompletionDumpJSONCarriesTheSameCommands. The JSON form exists for
// callers that would rather not split on tabs. It is the same data or it is a
// second source of truth, which is the thing this design set out to avoid.
func TestCompletionDumpJSONCarriesTheSameCommands(t *testing.T) {
	dir := t.TempDir()

	fromLines := map[string]bool{}
	for _, r := range dumpLines(t, dir) {
		if r[0] == "cmd" {
			fromLines[r[1]] = true
		}
	}

	got := runCLI(t, dir, nil, "completion", "--dump", "--json")
	if got.code != exitOK {
		t.Fatalf("completion --dump --json: %s", got.stderr)
	}
	decoded := decode(t, got.stdout)
	list, ok := decoded["commands"].([]any)
	if !ok {
		t.Fatalf("commands is not a list in %v", decoded)
	}
	if len(list) != len(fromLines) {
		t.Errorf("JSON has %d commands, the lines have %d", len(list), len(fromLines))
	}
	for _, item := range list {
		m, _ := item.(map[string]any)
		name, _ := m["name"].(string)
		if !fromLines[name] {
			t.Errorf("JSON names %q but the tab-separated form does not", name)
		}
	}
}

// TestCompletionDumpIsNotAnEnvelope. Every kind in plan section 10 is a
// published contract that 12.4 then covers. This dump is deliberately not one,
// so it must not grow the envelope keys that would make it look like one.
func TestCompletionDumpIsNotAnEnvelope(t *testing.T) {
	dir := t.TempDir()

	decoded := decode(t, runCLI(t, dir, nil, "completion", "--dump", "--json").stdout)
	for _, key := range []string{"schemaVersion", "kind"} {
		if _, ok := decoded[key]; ok {
			t.Errorf("the dump carries %q, which would make it look like a published kind", key)
		}
	}
}

// idLines runs `completion --ids` and returns abbreviation/title pairs.
func idLines(t *testing.T, dir string, args ...string) [][2]string {
	t.Helper()
	got := runCLI(t, dir, nil, append([]string{"completion", "--ids"}, args...)...)
	if got.code != exitOK {
		t.Fatalf("completion --ids: %s", got.stderr)
	}
	var out [][2]string
	for _, line := range strings.Split(strings.TrimRight(got.stdout, "\n"), "\n") {
		if line == "" {
			continue
		}
		id, title, _ := strings.Cut(line, "\t")
		out = append(out, [2]string{id, title})
	}
	return out
}

// TestCompletionIDsDegradesOutsideAStore. A tab press is not a command a
// person is waiting on an answer from, so failing there means printing a
// diagnostic into their command line. Offering nothing is the right answer.
func TestCompletionIDsDegradesOutsideAStore(t *testing.T) {
	dir := t.TempDir()

	got := runCLI(t, dir, nil, "completion", "--ids")
	if got.code != exitOK {
		t.Errorf("exit = %d outside a store, want %d: completion must degrade, not fail", got.code, exitOK)
	}
	if got.stdout != "" {
		t.Errorf("stdout = %q outside a store, want nothing", got.stdout)
	}
}

// TestCompletionIDsOffersWhatTheCLIAccepts is the criterion this feature turns
// on. Plan 5.5 says a command taking an ID accepts a unique prefix, and
// storeAbbreviations shortens to exactly what resolves. Offering a candidate
// the CLI would then reject is the one failure completion must never have, so
// every abbreviation is fed back to show.
func TestCompletionIDsOffersWhatTheCLIAccepts(t *testing.T) {
	dir := newStore(t)
	for _, title := range []string{"Rotate the signing key", "Retire the old runner", "Cache the parsed store"} {
		got := runCLI(t, dir, nil, "create", "--title", title, "--actor", "human:sothr")
		if got.code != exitOK {
			t.Fatalf("create %q: %s", title, got.stderr)
		}
	}

	candidates := idLines(t, dir)
	if len(candidates) != 3 {
		t.Fatalf("got %d candidates, want 3", len(candidates))
	}
	for _, c := range candidates {
		got := runCLI(t, dir, nil, "show", c[0])
		if got.code != exitOK {
			t.Errorf("completion offered %q but show rejected it: %s", c[0], got.stderr)
		}
	}
}

// TestCompletionIDsCarriesTitles. zsh shows a description beside a candidate
// and bash does not, so one format serves both: bash takes field one and zsh
// takes both. A bare ULID with no title is close to unpickable.
func TestCompletionIDsCarriesTitles(t *testing.T) {
	dir := newStore(t)
	got := runCLI(t, dir, nil, "create", "--title", "Rotate the signing key", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create: %s", got.stderr)
	}

	candidates := idLines(t, dir)
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
	if candidates[0][1] != "Rotate the signing key" {
		t.Errorf("title = %q, want the ticket's title", candidates[0][1])
	}
}

// TestCompletionIDsFiltersByPrefix. The filter runs before the cap, so a
// prefix the user already typed narrows the set rather than being applied to
// an already-truncated list.
func TestCompletionIDsFiltersByPrefix(t *testing.T) {
	dir := newStore(t)
	got := runCLI(t, dir, nil, "create", "--title", "Rotate the signing key", "--actor", "human:sothr")
	if got.code != exitOK {
		t.Fatalf("create: %s", got.stderr)
	}
	all := idLines(t, dir)
	if len(all) != 1 {
		t.Fatalf("got %d candidates, want 1", len(all))
	}

	if matched := idLines(t, dir, all[0][0]); len(matched) != 1 {
		t.Errorf("its own abbreviation matched %d candidates, want 1", len(matched))
	}
	if matched := idLines(t, dir, "ZZZ-NOPE"); len(matched) != 0 {
		t.Errorf("a prefix matching nothing returned %d candidates, want 0", len(matched))
	}
}

// TestCompletionInstallUsesTheNamesTheShellsLookFor. These two file names are
// the whole reason the subcommand form works. bash's loader asks for a file
// named after the binary, and zsh's _git scans $fpath for _git-* and calls the
// function of the same name. Get either wrong and `git ticket <TAB>` quietly
// does nothing while `git-ticket <TAB>` keeps working, which is the hardest
// kind of bug to notice.
func TestCompletionInstallUsesTheNamesTheShellsLookFor(t *testing.T) {
	dir := t.TempDir()
	xdg := filepath.Join(dir, "xdg")
	zdir := filepath.Join(dir, "zfuncs")

	got := runCLI(t, dir, map[string]string{"XDG_DATA_HOME": xdg}, "completion", "bash", "--install")
	if got.code != exitOK {
		t.Fatalf("bash install: %s", got.stderr)
	}
	if _, err := os.Stat(filepath.Join(xdg, "bash-completion", "completions", "git-ticket")); err != nil {
		t.Errorf("bash script is not at the name the loader asks for: %v", err)
	}

	got = runCLI(t, dir, nil, "completion", "zsh", "--install", "--dir", zdir)
	if got.code != exitOK {
		t.Fatalf("zsh install: %s", got.stderr)
	}
	if _, err := os.Stat(filepath.Join(zdir, "_git-ticket")); err != nil {
		t.Errorf("zsh script is not at the name _git scans for: %v", err)
	}
}

// TestCompletionInstallRefusesAnExistingFile follows the init --instructions
// precedent: refuse, name the file, and say what to do instead.
func TestCompletionInstallRefusesAnExistingFile(t *testing.T) {
	dir := t.TempDir()
	env := map[string]string{"XDG_DATA_HOME": filepath.Join(dir, "xdg")}

	if got := runCLI(t, dir, env, "completion", "bash", "--install"); got.code != exitOK {
		t.Fatalf("first install: %s", got.stderr)
	}

	got := runCLI(t, dir, env, "completion", "bash", "--install")
	if got.code == exitOK {
		t.Fatal("a second install succeeded, want a refusal")
	}
	if !strings.Contains(got.stderr, "--force") {
		t.Errorf("stderr = %q, want it to name --force", got.stderr)
	}

	if got := runCLI(t, dir, env, "completion", "bash", "--install", "--force"); got.code != exitOK {
		t.Errorf("--force did not overwrite: %s", got.stderr)
	}
}

// TestCompletionInstallZshRequiresADir. zsh has no standard completion
// directory and no XDG convention, and the directory must already be on
// $fpath. Guessing would write a file that silently does nothing, so this
// refuses and says what the directory has to be.
func TestCompletionInstallZshRequiresADir(t *testing.T) {
	dir := t.TempDir()

	got := runCLI(t, dir, nil, "completion", "zsh", "--install")
	if got.code == exitOK {
		t.Fatal("zsh --install with no --dir succeeded, want a refusal")
	}
	for _, want := range []string{"--dir", "fpath"} {
		if !strings.Contains(got.stderr, want) {
			t.Errorf("stderr = %q, want it to mention %s", got.stderr, want)
		}
	}
}

// TestCompletionScriptsCarryTheirEntryPoints. Each shell reaches the script by
// a different name, and neither is derivable from the other: bash computes
// _git_${command//-/_} and zsh calls _git-$words[1].
func TestCompletionScriptsCarryTheirEntryPoints(t *testing.T) {
	dir := t.TempDir()

	bash := runCLI(t, dir, nil, "completion", "bash").stdout
	for _, want := range []string{"_git_ticket()", "complete ", "-o nospace", "completion --dump"} {
		if !strings.Contains(bash, want) {
			t.Errorf("bash script is missing %q", want)
		}
	}

	zsh := runCLI(t, dir, nil, "completion", "zsh").stdout
	if !strings.HasPrefix(zsh, "#compdef git-ticket\n") {
		t.Error("zsh script must open with #compdef git-ticket, or compinit will not bind it")
	}
	for _, want := range []string{"_git-ticket()", "#description ", "completion --dump"} {
		if !strings.Contains(zsh, want) {
			t.Errorf("zsh script is missing %q", want)
		}
	}
}

// TestCompletionInstallWarnsAboutAShadowingGit. git ships its own zsh
// completion, and ahead of zsh's on $fpath it replaces the _git that
// dispatches to _git-ticket, so `git ticket <TAB>` stops working with no error
// anywhere. Reading the real $fpath would mean executing zsh, which plan 7.4
// does not allow for a warning, so this looks where such a file lives and says
// "if it comes before" rather than asserting an order it cannot see.
func TestCompletionInstallWarnsAboutAShadowingGit(t *testing.T) {
	dir := t.TempDir()
	zdir := filepath.Join(dir, "zfuncs")
	if err := os.MkdirAll(zdir, 0o755); err != nil {
		t.Fatal(err)
	}
	// git's version sources git-completion.bash; zsh's own does not.
	if err := os.WriteFile(filepath.Join(zdir, "_git"), []byte("#compdef git\n. git-completion.bash\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := runCLI(t, dir, nil, "completion", "zsh", "--install", "--dir", zdir)
	if got.code != exitOK {
		t.Fatalf("install: %s", got.stderr)
	}
	if !strings.Contains(got.stderr, "_git") || !strings.Contains(got.stderr, "fpath") {
		t.Errorf("stderr = %q, want a warning naming the file and $fpath", got.stderr)
	}

	// zsh's own _git must not trip it, or the warning becomes noise everybody
	// learns to ignore.
	clean := filepath.Join(dir, "clean")
	if err := os.MkdirAll(clean, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(clean, "_git"), []byte("#compdef git git-cvsserver\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := runCLI(t, dir, nil, "completion", "zsh", "--install", "--dir", clean); strings.Contains(got.stderr, "warning") {
		t.Errorf("zsh's own _git triggered a warning: %q", got.stderr)
	}
}

// TestCompletionRefusesAnUnknownShell. fish and PowerShell are deferred to
// TKT-01M1W7AC8EW6TZXR6A8NYFFYYK, and a caller asking for one should be told
// so rather than handed an empty script.
func TestCompletionRefusesAnUnknownShell(t *testing.T) {
	dir := t.TempDir()

	got := runCLI(t, dir, nil, "completion", "fish")
	if got.code == exitOK {
		t.Fatal("completion fish succeeded, want a refusal naming the supported shells")
	}
	for _, want := range []string{"bash", "zsh"} {
		if !strings.Contains(got.stderr, want) {
			t.Errorf("stderr = %q, want it to name %s", got.stderr, want)
		}
	}
}
