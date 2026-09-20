package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const testLayout = `schema: 3
board: "default"
cards:
  "%s": {x: 120, y: -40}
pens:
  "fe":
    title: "Frontend"
    x: 0
    y: 0
    w: 600
    h: 400
    color: "#759bcc"
    pin: {x: 20, y: 20}
    requiredLabels: ["frontend"]
  "fe-bugs":
    title: "Frontend bugs"
    x: 700
    y: 0
    w: 600
    h: 400
    color: "#b499be"
    pin: {x: 720, y: 20}
    requiredLabels: ["frontend", "bug"]
ruleOrder: ["fe", "fe-bugs"]
inbox: {x: -300, y: 0}
`

// canvasStore makes a store with three tickets and a layout that pins one of
// them. It returns the directory and the IDs in the order filed: a frontend
// bug (automatic), a backend ticket (automatic, unmatched), and a frontend
// ticket that is pinned.
func canvasStore(t *testing.T) (dir string, feBug, backend, pinned string) {
	t.Helper()
	dir = newStore(t)
	feBug = createTicket(t, dir, "--label", "frontend", "--label", "bug")["ticket"].(map[string]any)["id"].(string)
	backend = createTicket(t, dir, "--label", "backend")["ticket"].(map[string]any)["id"].(string)
	pinned = createTicket(t, dir, "--label", "frontend")["ticket"].(map[string]any)["id"].(string)
	writeLayout(t, dir, strings.ReplaceAll(testLayout, "%s", pinned))
	return dir, feBug, backend, pinned
}

func writeLayout(t *testing.T, dir, content string) {
	t.Helper()
	canvas := filepath.Join(dir, ".tickets", "canvas")
	if err := os.MkdirAll(canvas, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(canvas, "default.yml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// The contract of plan 12.10, seen through the CLI: first match in ruleOrder
// wins, so the frontend bug lands in the first pen and not the more specific
// second one; the backend ticket falls through to the inbox; the pinned card
// is listed apart with its coordinate.
func TestCanvasShowRoutesByFirstMatchAndListsPinsApart(t *testing.T) {
	dir, feBug, backend, pinned := canvasStore(t)
	got := runCLI(t, dir, nil, "canvas", "show")
	if got.code != exitOK {
		t.Fatalf("canvas show exited %d: %s%s", got.code, got.stdout, got.stderr)
	}
	for _, want := range []string{
		"1  fe  Frontend  labels: frontend  catches 1",
		"2  fe-bugs  Frontend bugs  labels: frontend, bug  catches 0",
		"inbox  (-300, 0)  catches 1",
		"pinned  1, placed by hand; routing does not apply",
		"(120, -40)",
	} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("show output lacks %q:\n%s", want, got.stdout)
		}
	}
	env := decode(t, runCLI(t, dir, nil, "--json", "canvas", "show").stdout)
	if env["kind"] != "canvas-board" || env["exists"] != true || env["applied"] != true {
		t.Fatalf("envelope kind/exists/applied = %v/%v/%v", env["kind"], env["exists"], env["applied"])
	}
	pens := env["pens"].([]any)
	if first := pens[0].(map[string]any); first["id"] != "fe" || first["tickets"].([]any)[0] != feBug {
		t.Errorf("first pen = %v, want fe catching %s", first, feBug)
	}
	if inbox := env["inbox"].(map[string]any); inbox["tickets"].([]any)[0] != backend {
		t.Errorf("inbox = %v, want %s", inbox, backend)
	}
	if pins := env["pinned"].([]any); pins[0].(map[string]any)["id"] != pinned {
		t.Errorf("pinned = %v, want %s", pins, pinned)
	}
}

func TestCanvasPensPrintsRulesInResolutionOrder(t *testing.T) {
	dir, _, _, _ := canvasStore(t)
	got := runCLI(t, dir, nil, "canvas", "pens")
	if got.code != exitOK {
		t.Fatalf("canvas pens exited %d: %s", got.code, got.stderr)
	}
	lines := strings.Split(strings.TrimSpace(got.stdout), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "1") || !strings.Contains(lines[0], "fe ") || !strings.HasPrefix(lines[1], "2") {
		t.Fatalf("pens printed:\n%s", got.stdout)
	}
	if env := decode(t, runCLI(t, dir, nil, "--json", "canvas", "pens").stdout); env["kind"] != "canvas-board" {
		t.Fatalf("pens kind = %v, want canvas-board", env["kind"])
	}
}

func TestCanvasExplainSaysWhyAndThatRoutingIsNotApplied(t *testing.T) {
	dir, feBug, backend, pinned := canvasStore(t)

	got := runCLI(t, dir, nil, "canvas", "explain", feBug)
	if got.code != exitOK {
		t.Fatalf("explain exited %d: %s", got.code, got.stderr)
	}
	for _, want := range []string{
		"automatic: the canvas places it by the rules below",
		"routing:",
		"goes to pen fe (Frontend): carries frontend",
		"not fe-bugs (rule 2): matches, but an earlier rule took it",
	} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("explain lacks %q:\n%s", want, got.stdout)
		}
	}

	got = runCLI(t, dir, nil, "canvas", "explain", backend)
	if !strings.Contains(got.stdout, "goes to the inbox (-300, 0): no rule matched") || !strings.Contains(got.stdout, "not fe (rule 1): missing frontend") {
		t.Errorf("unmatched explain:\n%s", got.stdout)
	}

	got = runCLI(t, dir, nil, "canvas", "explain", pinned)
	if !strings.Contains(got.stdout, "pinned at (120, -40); routing does not apply") {
		t.Errorf("pinned explain:\n%s", got.stdout)
	}
	if env := decode(t, runCLI(t, dir, nil, "--json", "canvas", "explain", feBug).stdout); env["placement"] != "rules" {
		t.Fatalf("placement on a board with pens = %v, want rules", env["placement"])
	}
	env := decode(t, runCLI(t, dir, nil, "--json", "canvas", "explain", pinned).stdout)
	if env["kind"] != "canvas-explain" || env["placement"] != "pinned" || env["applied"] != true {
		t.Fatalf("explain envelope = %v", env)
	}
	if routing := env["routing"].(map[string]any); routing["destination"] != "fe" {
		t.Fatalf("pinned card's routing = %v, want fe reported alongside the pin", routing)
	}
}

// A board with no layout file is a fact about the store, so it exits 0. show
// and pens say so in one line; explain answers about a ticket, so it names
// the ticket, then the absent file, then where the canvas puts the card. The
// envelope says the same with exists false.
func TestCanvasReportsAMissingLayoutFileWithoutFailing(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed, so there is no repository root to print the path against")
	}
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if got := runCLI(t, dir, nil, "init", "--actor", "human:sothr"); got.code != exitOK {
		t.Fatalf("init: %s", got.stderr)
	}
	id := createTicket(t, dir)["ticket"].(map[string]any)["id"].(string)
	for _, args := range [][]string{{"canvas", "show"}, {"canvas", "pens"}} {
		got := runCLI(t, dir, nil, args...)
		if got.code != exitOK || !strings.Contains(got.stdout, "has no layout file at .tickets/canvas/default.yml") {
			t.Errorf("%v exited %d:\n%s%s", args, got.code, got.stdout, got.stderr)
		}
	}
	got := runCLI(t, dir, nil, "canvas", "explain", id)
	if got.code != exitOK || !strings.Contains(got.stdout, "has no layout file") || !strings.Contains(got.stdout, "status lanes") {
		t.Errorf("explain exited %d:\n%s%s", got.code, got.stdout, got.stderr)
	}
	env := decode(t, runCLI(t, dir, nil, "--json", "canvas", "show").stdout)
	if env["exists"] != false || len(env["pens"].([]any)) != 0 {
		t.Fatalf("missing-file envelope = %v", env)
	}
}

func TestCanvasRefusesAnUnknownWord(t *testing.T) {
	dir := newStore(t)
	got := runCLI(t, dir, nil, "--json", "canvas", "draw")
	if got.code != exitError || decode(t, got.stdout)["error"].(map[string]any)["code"] != codeUsage {
		t.Fatalf("canvas draw exited %d: %s%s", got.code, got.stdout, got.stderr)
	}
}

// --board is documented after the word and the ID, and parseFlags reads flags
// on either side of positionals, so the documented form is the tested form.
func TestCanvasTakesBoardAfterTheWord(t *testing.T) {
	dir, feBug, _, _ := canvasStore(t)
	for _, args := range [][]string{
		{"canvas", "show", "--board", "default"},
		{"canvas", "pens", "--board", "default"},
		{"canvas", "explain", feBug, "--board", "default"},
		{"canvas", "--board", "default", "show"},
	} {
		if got := runCLI(t, dir, nil, args...); got.code != exitOK || strings.Contains(got.stdout, "no layout file") {
			t.Errorf("%v exited %d:\n%s%s", args, got.code, got.stdout, got.stderr)
		}
	}
	got := runCLI(t, dir, nil, "--json", "canvas", "show", "--board", "other")
	if env := decode(t, got.stdout); got.code != exitOK || env["board"] != "other" || env["exists"] != false {
		t.Fatalf("other board: exit %d, %v", got.code, env)
	}
}

// A board name is a file name under .tickets/canvas, so anything that could
// leave that directory is refused before a path is built, by the layout
// package's grammar rather than by a check here.
func TestCanvasRefusesABoardNameThatIsAPath(t *testing.T) {
	dir := newStore(t)
	for _, name := range []string{"../../other", "/etc/passwd", "a/b", ".", ".."} {
		got := runCLI(t, dir, nil, "canvas", "pens", "--board", name)
		if got.code == exitOK || !strings.Contains(got.stderr, "invalid board name") {
			t.Errorf("--board %q exited %d:\n%s%s", name, got.code, got.stdout, got.stderr)
		}
	}
}
