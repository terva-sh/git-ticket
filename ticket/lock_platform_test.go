package ticket

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestOneLockImplementationPerPlatform holds the build-tag partition to being
// exclusive and total: every target compiles exactly one definition of
// tryFlock.
//
// This test exists because the partition failed silently for the whole life of
// the project. lock_other.go was tagged !unix, Windows matched it, and every
// mutation on Windows failed with lock_timeout while the suite stayed green on
// linux and macOS. Compiling for one platform proves nothing about the tags of
// another, so this asks the toolchain which file each target actually takes.
func TestOneLockImplementationPerPlatform(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("the go toolchain is not on PATH")
	}

	want := map[string]string{
		"windows": "lock_windows.go",
		"linux":   "lock_unix.go",
		"darwin":  "lock_unix.go",
		"plan9":   "lock_other.go",
	}

	for goos, file := range want {
		t.Run(goos, func(t *testing.T) {
			cmd := exec.Command("go", "list", "-f", `{{join .GoFiles "\n"}}`, ".")
			// CGO_ENABLED=0 because a cross-target list must not depend on a
			// C toolchain for the target being present.
			cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH=amd64", "CGO_ENABLED=0")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go list for %s: %v\n%s", goos, err, out)
			}

			var got []string
			for _, name := range strings.Fields(string(out)) {
				if strings.HasPrefix(name, "lock_") && strings.HasSuffix(name, ".go") {
					got = append(got, name)
				}
			}
			if len(got) != 1 || got[0] != file {
				t.Errorf("%s compiles %v, want exactly [%s]", goos, got, file)
			}
		})
	}
}
