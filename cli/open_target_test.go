package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReferenceOpenerRefusesUnresolvedOrCredentialTargets(t *testing.T) {
	for _, target := range []string{"relative.md", "https://name:password@example.invalid/x", "https://"} {
		if err := openReferenceTarget(target); err == nil {
			t.Errorf("unsafe target %q was accepted", target)
		}
	}
}

func TestReferenceOpenerReportsDesktopFailure(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("uses the Linux xdg-open command")
	}
	dir := t.TempDir()
	opener := filepath.Join(dir, "xdg-open")
	if err := os.WriteFile(opener, []byte("#!/bin/sh\nexit 37\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if err := openReferenceTarget("https://example.invalid/ticket"); err == nil {
		t.Fatal("desktop opener failed but target was reported open")
	}
}
