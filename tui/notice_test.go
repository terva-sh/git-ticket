package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoticeAgreesWithTheHeaders holds NOTICE and the per-file
// attribution headers to each other, both ways. A file whose header
// says it was lifted or adapted from terva must be named in NOTICE,
// and a file NOTICE names must exist and carry the header. Attribution
// that drifts from the code is worse than none, because it reads as
// authoritative and is wrong.
func TestNoticeAgreesWithTheHeaders(t *testing.T) {
	notice, err := os.ReadFile(filepath.Join("..", "NOTICE"))
	if err != nil {
		t.Fatalf("reading NOTICE: %v", err)
	}
	text := string(notice)

	if !strings.Contains(text, "Copyright (c) 2026 Drew Short (Terva, a hard fork of zot)") ||
		!strings.Contains(text, "Copyright (c) 2026 Patric Eckhart") {
		t.Error("NOTICE does not carry both copyright lines from terva's LICENSE")
	}
	if !strings.Contains(text, "Permission is hereby granted") {
		t.Error("NOTICE does not reproduce the MIT permission text")
	}

	// The files NOTICE names, in its indented list.
	named := map[string]bool{}
	for _, line := range strings.Split(text, "\n") {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "tui/") && strings.HasSuffix(l, ".go") {
			named[l] = true
		}
	}
	if len(named) == 0 {
		t.Fatal("NOTICE names no files; the indented tui/ list is missing")
	}

	// The files whose headers claim derivation.
	marked := map[string]bool{}
	for _, glob := range []string{"*.go", "tuitest/*.go"} {
		paths, err := filepath.Glob(glob)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range paths {
			if strings.HasSuffix(p, "_test.go") {
				continue
			}
			src, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			head := string(src)
			if len(head) > 2048 {
				head = head[:2048]
			}
			if strings.Contains(head, "terva") &&
				strings.Contains(head, "Copyright (c) 2026 Drew Short") {
				marked[filepath.ToSlash(filepath.Join("tui", p))] = true
			}
		}
	}

	for f := range marked {
		if !named[f] {
			t.Errorf("%s carries a terva attribution header but NOTICE does not name it", f)
		}
	}
	for f := range named {
		if !marked[f] {
			t.Errorf("NOTICE names %s but the file has no terva attribution header, or does not exist", f)
		}
	}
}
