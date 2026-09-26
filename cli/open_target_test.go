package cli

import "testing"

func TestReferenceOpenerRefusesUnresolvedOrCredentialTargets(t *testing.T) {
	for _, target := range []string{"relative.md", "https://name:password@example.invalid/x", "https://"} {
		if err := openReferenceTarget(target); err == nil {
			t.Errorf("unsafe target %q was accepted", target)
		}
	}
}
