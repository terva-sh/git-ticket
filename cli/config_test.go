package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setAllowlists rewrites config.yml with the given allowlists, which is the
// only way a store expresses one. `init` writes both empty.
func setAllowlists(t *testing.T, dir string, labels, milestones []string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("schema: 1\nactors:\n  - id: human:sothr\n    name: Drew Short\n")
	for _, section := range []struct {
		name   string
		values []string
	}{{"labels", labels}, {"milestones", milestones}} {
		if len(section.values) == 0 {
			b.WriteString(section.name + ": []\n")
			continue
		}
		b.WriteString(section.name + ":\n")
		for _, v := range section.values {
			b.WriteString("  - " + v + "\n")
		}
	}
	b.WriteString("defaults:\n  type: task\n  priority: normal\n  claim_expiry: null\n")
	b.WriteString("lock:\n  timeout: 10s\n")

	path := filepath.Join(dir, ".tickets", "config.yml")
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatalf("write config.yml: %v", err)
	}
}

// allowlistOf reads one published allowlist out of the config envelope.
func allowlistOf(t *testing.T, dir, key string) ([]string, bool) {
	t.Helper()
	got := runCLI(t, dir, nil, "--json", "config")
	if got.code != exitOK {
		t.Fatalf("config exited %d: %s%s", got.code, got.stdout, got.stderr)
	}
	envelope := decode(t, got.stdout)
	if envelope["kind"] != "config" {
		t.Fatalf("kind = %v, want config", envelope["kind"])
	}
	list, ok := envelope[key].(map[string]any)
	if !ok {
		t.Fatalf("no %s object in %v", key, envelope)
	}
	raw, _ := list["values"].([]any)
	values := make([]string, 0, len(raw))
	for _, v := range raw {
		s, _ := v.(string)
		values = append(values, s)
	}
	enforced, ok := list["enforced"].(bool)
	if !ok {
		t.Fatalf("%s carries no enforced flag: %v", key, list)
	}
	return values, enforced
}

// warningCodes returns every warning code check reports for the store.
func warningCodes(t *testing.T, dir string) []string {
	t.Helper()
	envelope := decode(t, runCLI(t, dir, nil, "--json", "check").stdout)
	raw, _ := envelope["warnings"].([]any)
	out := make([]string, 0, len(raw))
	for _, w := range raw {
		f, _ := w.(map[string]any)
		code, _ := f["code"].(string)
		out = append(out, code)
	}
	return out
}

func has(codes []string, want string) bool {
	for _, c := range codes {
		if c == want {
			return true
		}
	}
	return false
}

// TestConfigPublishesTheAllowlists is the ticket. The legal set had to be read
// out of config.yml, because no command would say it.
func TestConfigPublishesTheAllowlists(t *testing.T) {
	dir := newStore(t)
	setAllowlists(t, dir, []string{"ci", "format"}, []string{"v1.2"})

	labels, enforced := allowlistOf(t, dir, "labels")
	if !enforced {
		t.Error("labels are listed, so the allowlist is enforced")
	}
	if strings.Join(labels, " ") != "ci format" {
		t.Errorf("labels = %v, want the configured set", labels)
	}

	milestones, enforced := allowlistOf(t, dir, "milestones")
	if !enforced || strings.Join(milestones, " ") != "v1.2" {
		t.Errorf("milestones = %v enforced=%v, want [v1.2] enforced", milestones, enforced)
	}
}

// TestConfigSaysAnEmptyAllowlistPermitsEverything is the distinction plan 10.6
// exists for. An empty list permits everything, per 4.1, so a consumer reading
// a bare [] the obvious way gets it exactly backwards.
func TestConfigSaysAnEmptyAllowlistPermitsEverything(t *testing.T) {
	dir := newStore(t)

	for _, key := range []string{"labels", "milestones"} {
		values, enforced := allowlistOf(t, dir, key)
		if enforced {
			t.Errorf("%s: enforced with nothing listed", key)
		}
		// Section 10: an absent collection is [] and never null.
		if values == nil || len(values) != 0 {
			t.Errorf("%s values = %v, want an empty array", key, values)
		}
	}

	// The human form has to carry the same meaning, since that is what a person
	// reads. An empty line there would say the opposite of what it means.
	got := runCLI(t, dir, nil, "config")
	if got.code != exitOK {
		t.Fatalf("config exited %d: %s", got.code, got.stderr)
	}
	if !strings.Contains(got.stdout, "any value is permitted") {
		t.Errorf("the human form does not say an empty allowlist permits everything:\n%s", got.stdout)
	}
}

// TestConfigAgreesWithWhatCheckEnforces is what makes publishing worth
// anything. A published set that check disagreed with would be worse than no
// published set, because a consumer would trust it.
func TestConfigAgreesWithWhatCheckEnforces(t *testing.T) {
	t.Run("a listed label passes and an unlisted one warns", func(t *testing.T) {
		dir := newStore(t)
		setAllowlists(t, dir, []string{"ci", "format"}, nil)

		published, enforced := allowlistOf(t, dir, "labels")
		if !enforced {
			t.Fatal("the allowlist should be enforced")
		}
		makeTicket(t, dir, "Carries a published label", "--label", published[0])
		if codes := warningCodes(t, dir); has(codes, "label_unknown") {
			t.Errorf("a label config publishes as legal raised label_unknown: %v", codes)
		}

		makeTicket(t, dir, "Carries an unpublished label", "--label", "docs")
		if codes := warningCodes(t, dir); !has(codes, "label_unknown") {
			t.Errorf("a label outside the published set did not warn: %v", codes)
		}
	})

	t.Run("nothing listed means nothing warns", func(t *testing.T) {
		dir := newStore(t)
		if _, enforced := allowlistOf(t, dir, "labels"); enforced {
			t.Fatal("a fresh store enforces no allowlist")
		}
		makeTicket(t, dir, "Carries any label at all", "--label", "docs")
		if codes := warningCodes(t, dir); has(codes, "label_unknown") {
			t.Errorf("enforced is false, so no label should warn: %v", codes)
		}
	})
}

// TestConfigPublishesTheDefaultsAndTheLock covers the rest of 10.6. Each is
// per-store and each was reachable only by opening config.yml.
func TestConfigPublishesTheDefaultsAndTheLock(t *testing.T) {
	dir := newStore(t)
	envelope := decode(t, runCLI(t, dir, nil, "--json", "config").stdout)

	defaults, ok := envelope["defaults"].(map[string]any)
	if !ok {
		t.Fatalf("no defaults in %v", envelope)
	}
	if defaults["type"] != "task" || defaults["priority"] != "normal" {
		t.Errorf("defaults = %v, want what create falls back to", defaults)
	}
	// A claim does not expire on its own, and null says that. A zero duration
	// would read as "expires immediately".
	if defaults["claimExpiry"] != nil {
		t.Errorf("claimExpiry = %v, want null when a claim does not expire", defaults["claimExpiry"])
	}

	// The store falls back to DefaultLockTimeout when config.yml does not say,
	// so publishing the raw zero would report 0s for a store that waits 10.
	lock, _ := envelope["lock"].(map[string]any)
	if lock["timeout"] != "10s" {
		t.Errorf("lock timeout = %v, want the effective 10s", lock["timeout"])
	}
}

// TestConfigNeedsAStore is the line between this command and schema. schema
// answers before init because it describes the binary; this describes a store,
// so with no store there is nothing true to say.
func TestConfigNeedsAStore(t *testing.T) {
	got := runCLI(t, t.TempDir(), nil, "--json", "config")
	if got.code == exitOK {
		t.Fatalf("config answered outside a store: %s", got.stdout)
	}
	if code := errCode(t, got); code != "store_not_found" {
		t.Errorf("code = %v, want store_not_found", code)
	}

	// schema still answers there, which is the property that kept these apart.
	if s := runCLI(t, t.TempDir(), nil, "--json", "schema"); s.code != exitOK {
		t.Errorf("schema stopped answering without a store: %s", s.stderr)
	}
}

func TestConfigTakesNoArguments(t *testing.T) {
	dir := newStore(t)
	got := runCLI(t, dir, nil, "--json", "config", "extra")
	if got.code != exitError {
		t.Fatal("a stray argument should be refused")
	}
	if code := errCode(t, got); code != codeUsage {
		t.Errorf("code = %v, want %s", code, codeUsage)
	}
}
