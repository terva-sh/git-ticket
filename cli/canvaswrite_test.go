package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readLayout(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, ".tickets", "canvas", "default.yml"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func mustCanvas(t *testing.T, dir string, args ...string) result {
	t.Helper()
	got := runCLI(t, dir, nil, append([]string{"canvas"}, args...)...)
	if got.code != exitOK {
		t.Fatalf("canvas %s exited %d: %s%s", strings.Join(args, " "), got.code, got.stdout, got.stderr)
	}
	return got
}

// The design's own example, run end to end: six writes, no coordinate
// computed, and a board the read words and check both accept. A pen lands
// last in ruleOrder, pen order rewrites it, and the file is the canonical
// form, which is what lets check stay silent about it.
func TestCanvasWritesBuildABoardTheReadWordsAndCheckAccept(t *testing.T) {
	dir := newStore(t)
	auth := ticketID(t, createTicket(t, dir, "--label", "authentik"))
	forge := ticketID(t, createTicket(t, dir, "--label", "forge"))
	createTicket(t, dir, "--label", "docs")

	mustCanvas(t, dir, "pen", "add", "authentik", "--title", "Authentik", "--label", "authentik", "--at", "0,0", "--size", "1200,900")
	mustCanvas(t, dir, "pen", "add", "forge", "--title", "Forges and mirrors", "--label", "forge", "--at", "1300,0", "--size", "1200,900", "--color", "#89ad97")
	mustCanvas(t, dir, "pen", "order", "forge", "authentik")
	mustCanvas(t, dir, "inbox", "--at", "-1400,0")
	mustCanvas(t, dir, "place", auth, "--at", "10,20.5")
	mustCanvas(t, dir, "frame", "add", "sprint", "--title", "Sprint 3", "--at", "0,1000", "--size", "800,400", "--member", forge)

	file := readLayout(t, dir)
	for _, want := range []string{
		`ruleOrder: ["forge", "authentik"]`,
		`inbox: {x: -1400, y: 0}`,
		`  "` + auth + `": {x: 10, y: 20.5}`,
		`"forge": {title: "Forges and mirrors", x: 1300, y: 0, w: 1200, h: 900, color: "#89ad97", pin: {x: 1300, y: 0}, requiredLabels: ["forge"]}`,
		`"sprint": {title: "Sprint 3", x: 0, y: 1000, w: 800, h: 400, color: "#759bcc", members: [` + forge + `]}`,
	} {
		if !strings.Contains(file, want) {
			t.Errorf("layout lacks %q:\n%s", want, file)
		}
	}
	// Three tickets, one pin: nothing wrote a position for the two automatic
	// cards, so the cards map holds exactly the one place wrote.
	if strings.Count(file, "\n  \"TKT-") != 1 {
		t.Errorf("a write other than place put a card on the board:\n%s", file)
	}
	show := mustCanvas(t, dir, "show").stdout
	for _, want := range []string{"1  forge", "2  authentik", "inbox  (-1400, 0)  catches 1", "pinned  1"} {
		if !strings.Contains(show, want) {
			t.Errorf("show lacks %q:\n%s", want, show)
		}
	}
	check := runCLI(t, dir, nil, "check", "--strict")
	if check.code != exitOK {
		t.Fatalf("check --strict on a board the CLI wrote: %s%s", check.stdout, check.stderr)
	}

	// Every write answers with canvas-board, the board after the write.
	env := decode(t, mustCanvas(t, dir, "--json", "release", auth).stdout)
	if env["kind"] != "canvas-board" || env["exists"] != true {
		t.Fatalf("release --json = %v", env)
	}
	if pinned := env["pinned"].([]any); len(pinned) != 0 {
		t.Fatalf("released card still pinned: %v", pinned)
	}
	mustCanvas(t, dir, "pen", "rm", "forge")
	if file := readLayout(t, dir); strings.Contains(file, "forge") || !strings.Contains(file, `ruleOrder: ["authentik"]`) {
		t.Fatalf("pen rm left the pen or its rule:\n%s", file)
	}
}

// A pen for a label outside the allowlist is written and warned about, the
// way create files a ticket under one: the allowlist is advisory, per plan
// 11, and check reports it as label_unknown either way.
func TestCanvasPenAddWarnsOnALabelOutsideTheAllowlist(t *testing.T) {
	dir := newStore(t)
	cfg := filepath.Join(dir, ".tickets", "config.yml")
	raw, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(strings.Replace(string(raw), "labels: []", "labels: [frontend]", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	got := mustCanvas(t, dir, "pen", "add", "fe", "--title", "Frontend", "--label", "frontend", "--at", "0,0", "--size", "1,1")
	if got.stderr != "" {
		t.Fatalf("a listed label warned: %s", got.stderr)
	}
	got = mustCanvas(t, dir, "--json", "pen", "add", "ops", "--title", "Ops", "--label", "ops", "--at", "0,0", "--size", "1,1")
	if !strings.Contains(got.stderr, `pen ops requires "ops", which is not in the config.yml allowlist`) {
		t.Fatalf("stderr = %q", got.stderr)
	}
	if !strings.Contains(readLayout(t, dir), `requiredLabels: ["ops"]`) {
		t.Fatal("the pen was not written")
	}
	check := runCLI(t, dir, nil, "--json", "check")
	if codes := findingCodes(decode(t, check.stdout)["warnings"]); codes != "label_unknown" {
		t.Fatalf("check warnings = %s", codes)
	}
	// A refused write says nothing about labels, because nothing was written
	// for check to warn about.
	refused := runCLI(t, dir, nil, "canvas", "pen", "add", "ops", "--title", "Again", "--label", "ops", "--at", "0,0", "--size", "1,1")
	if refused.code == exitOK || strings.Contains(refused.stderr, "allowlist") {
		t.Fatalf("refused duplicate: code %d, stderr %q", refused.code, refused.stderr)
	}
}

// A write refuses rather than producing a layout check would reject, and
// the file is unchanged after a refusal. Each case is one way to be wrong;
// the assertion is the same for all of them.
func TestCanvasWriteRefusalsLeaveTheFileAlone(t *testing.T) {
	dir := newStore(t)
	auth := ticketID(t, createTicket(t, dir, "--label", "authentik"))
	mustCanvas(t, dir, "pen", "add", "authentik", "--title", "Authentik", "--label", "authentik", "--at", "0,0", "--size", "100,100")
	before := readLayout(t, dir)

	cases := []struct {
		name string
		args []string
		code string
		says string
	}{
		{"duplicate pen", []string{"pen", "add", "authentik", "--title", "T", "--label", "x", "--at", "0,0", "--size", "1,1"}, "validation_failed", "already has a pen authentik"},
		{"zero-size pen", []string{"pen", "add", "empty", "--title", "T", "--label", "x", "--at", "0,0", "--size", "0,10"}, "validation_failed", "geometry"},
		{"bad colour", []string{"pen", "add", "c", "--title", "T", "--label", "x", "--at", "0,0", "--size", "1,1", "--color", "#ffffff"}, "validation_failed", "color"},
		{"pen without a label", []string{"pen", "add", "bare", "--title", "T", "--at", "0,0", "--size", "1,1"}, "usage", "--label"},
		{"order leaves one out", []string{"pen", "order", "nope"}, "validation_failed", "no such pen: nope; left out: authentik"},
		{"order names one twice", []string{"pen", "order", "authentik", "authentik"}, "validation_failed", "named twice"},
		{"rm unknown pen", []string{"pen", "rm", "ghost"}, "validation_failed", "no pen ghost"},
		{"release unpinned", []string{"release", auth}, "validation_failed", "is not pinned"},
		{"place unknown ticket", []string{"place", "TKT-01K3ZZZZZ00000000000000000", "--at", "0,0"}, "ticket_not_found", ""},
		{"place without at", []string{"place", auth}, "usage", "--at"},
		{"flag the word does not take", []string{"place", auth, "--at", "0,0", "--title", "x"}, "usage", "does not take --title"},
		{"read word with a write flag", []string{"show", "--at", "0,0"}, "usage", "takes no --at"},
		{"frame member unknown", []string{"frame", "add", "f", "--title", "F", "--at", "0,0", "--size", "1,1", "--member", "TKT-01K3ZZZZZ00000000000000000"}, "ticket_not_found", ""},
		{"malformed point", []string{"inbox", "--at", "1;2"}, "usage", "X,Y"},
		{"unknown word", []string{"pen", "sort"}, "usage", "not a canvas pen word"},
	}
	for _, c := range cases {
		got := runCLI(t, dir, nil, append([]string{"--json", "canvas"}, c.args...)...)
		if got.code == exitOK {
			t.Errorf("%s: accepted: %s", c.name, got.stdout)
			continue
		}
		env := decode(t, got.stdout)
		e, _ := env["error"].(map[string]any)
		if e["code"] != c.code {
			t.Errorf("%s: code = %v, want %s (%v)", c.name, e["code"], c.code, e["message"])
		}
		if msg, _ := e["message"].(string); c.says != "" && !strings.Contains(msg, c.says) {
			t.Errorf("%s: message %q lacks %q", c.name, msg, c.says)
		}
		if after := readLayout(t, dir); after != before {
			t.Errorf("%s: a refused write changed the file:\n%s", c.name, after)
		}
	}
}

// A board that is valid but not in the form a save writes is the one layout
// finding --fix repairs, per plan 11: a rewrite naming the file, after which
// the same check is silent. A card for a ticket the store lacks is reported
// and left alone, because only a person knows whether the ticket is gone or
// on another branch.
func TestCheckFixRewritesANonCanonicalBoardAndLeavesAStaleCard(t *testing.T) {
	dir := newStore(t)
	id := ticketID(t, createTicket(t, dir))
	writeLayout(t, dir, "schema: 2\nboard: default\ncards:\n  "+id+": {x: 1.004, y: 2}\n  TKT-01K3ZZZZZ00000000000000000: {x: 0, y: 0}\nframes: {}\n")

	got := runCLI(t, dir, nil, "--json", "check")
	env := decode(t, got.stdout)
	codes := findingCodes(env["warnings"])
	if !strings.Contains(codes, "layout_not_canonical") || !strings.Contains(codes, "layout_ticket_missing") {
		t.Fatalf("warnings = %s", codes)
	}

	got = runCLI(t, dir, nil, "--json", "check", "--fix")
	env = decode(t, got.stdout)
	repairs, _ := env["repairs"].([]any)
	if len(repairs) != 1 {
		t.Fatalf("repairs = %v", repairs)
	}
	r := repairs[0].(map[string]any)
	// The store is not in a repository here, so the path is absolute; in one
	// it is repository-relative, per 10.3, as the other fix tests hold.
	if to, _ := r["to"].(string); r["kind"] != "rewrite" || !strings.HasSuffix(to, "/.tickets/canvas/default.yml") || r["from"] != nil {
		t.Fatalf("repair = %v", r)
	}
	if codes := findingCodes(env["warnings"]); strings.Contains(codes, "layout_not_canonical") || !strings.Contains(codes, "layout_ticket_missing") {
		t.Fatalf("after --fix, warnings = %s", codes)
	}
	file := readLayout(t, dir)
	if !strings.HasPrefix(file, "# git-ticket canvas layout") || !strings.Contains(file, `"`+id+`": {x: 1, y: 2}`) || !strings.Contains(file, "TKT-01K3ZZZZZ00000000000000000") {
		t.Fatalf("rewritten board:\n%s", file)
	}
}

func findingCodes(v any) string {
	list, _ := v.([]any)
	var out []string
	for _, f := range list {
		m, _ := f.(map[string]any)
		out = append(out, m["code"].(string))
	}
	return strings.Join(out, ",")
}
