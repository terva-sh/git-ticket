package cli

import (
	"fmt"
	"io"
	"runtime"
	"runtime/debug"
	"strings"
)

// buildVersion reports what this binary is, read from what the Go toolchain
// already embedded in it.
//
// Nothing is stamped at link time. `go build` records the module version, the
// commit, and whether the tree was dirty, and runtime/debug reads them back, so
// the justfile needs no ldflags and there is no version variable to keep in step
// with a tag. Plan 12.1 says so.
//
// A binary built from a checkout with no tag reachable reports "devel", which is
// honest rather than a version that does not exist.
func buildVersion() (version, commit, goVersion string, modified bool) {
	version, commit, goVersion = "devel", "unknown", runtime.Version()

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version, commit, goVersion, modified
	}
	if info.GoVersion != "" {
		goVersion = info.GoVersion
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		// Go appends +dirty to the version of a build from a modified tree.
		// modified already carries that, and saying it twice in one line reads
		// like two different facts.
		version = strings.TrimSuffix(v, "+dirty")
	}
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			commit = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	return version, commit, goVersion, modified
}

// writeVersion answers --version. A person gets one line and a host gets the
// envelope, because a host shelling out to the binary needs to know what it is
// talking to and should not have to parse prose to find out.
func writeVersion(w io.Writer, asJSON bool) {
	version, commit, goVersion, modified := buildVersion()
	if asJSON {
		writeJSON(w, versionEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "version",
			Version:       version,
			Commit:        commit,
			Go:            goVersion,
			Modified:      modified,
		})
		return
	}
	short := commit
	if len(short) > 12 {
		short = short[:12]
	}
	dirty := ""
	if modified {
		dirty = ", modified"
	}
	fmt.Fprintf(w, "git-ticket %s (%s, %s%s)\n", version, short, goVersion, dirty)
}
