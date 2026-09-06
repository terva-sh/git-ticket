package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var kindLiteral = regexp.MustCompile(`Kind:\s+"([a-z-]+)"`)

// TestEnvelopeKindsMatchTheSource closes the hole that let two kinds ship
// undeclared.
//
// envelopeKinds is written out by hand because the kinds are a contract with
// consumers rather than a fact about a Go type, and TestEveryEmittedKindIsPublished
// holds it to what the commands answer. That guard is only as complete as its
// table, which is the failure it has now had twice: `version` shipped in v0.4.0
// absent from both the list and the table, so the two hand-written lists agreed
// with each other and not with the plan, and `self-update` shipped the same way
// and stayed undeclared for four releases. A consumer validating an envelope
// against the published list would have rejected a legitimate answer.
//
// This reads the kind literals out of the source instead, so a new kind is
// caught whether or not anybody remembers to exercise it. It is the technique
// TestGitCommandsAreReadOnly already uses on exec.Command for the same reason:
// the property is about what the code contains, not about what one test runs.
func TestEnvelopeKindsMatchTheSource(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	emitted := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(".", name))
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range kindLiteral.FindAllStringSubmatch(string(data), -1) {
			emitted[m[1]] = name
		}
	}
	if len(emitted) == 0 {
		t.Fatal("found no kind literals in the package, so this guard is measuring nothing")
	}

	published := map[string]bool{}
	for _, k := range envelopeKinds {
		published[k] = true
	}

	for kind, file := range emitted {
		if !published[kind] {
			t.Errorf("%s emits kind %q, which envelopeKinds does not publish", file, kind)
		}
	}
	for kind := range published {
		if _, ok := emitted[kind]; !ok {
			t.Errorf("envelopeKinds publishes %q, which no file in this package emits", kind)
		}
	}
}
