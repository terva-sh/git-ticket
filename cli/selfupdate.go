package cli

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// The releases of the public mirror, per plan 12.6. The internal forge
// is never consulted: a binary in the wild has never heard of it.
const (
	defaultReleaseAPI = "https://api.github.com"
	releasePath       = "/repos/terva-sh/git-ticket/releases/latest"
)

// runningVersion is what buildVersion reports, behind a variable so a
// test can be a release. In-tree test binaries report devel, and a
// suite that can only exercise the refusal is no suite.
var runningVersion = func() string {
	v, _, _, _ := buildVersion()
	return v
}

// exitStatusErr carries a specific exit status out of a command that
// has already written its output. Run maps it to the process exit
// code without an error envelope, because the graded bucket of plan
// 10.2 is an answer, not a failure.
type exitStatusErr struct{ code int }

func (e *exitStatusErr) Error() string { return "exit status " + strconv.Itoa(e.code) }

// releaseInfo is the slice of the GitHub release object this command
// reads. Everything else the API sends is ignored.
type releaseInfo struct {
	TagName string         `json:"tag_name"`
	Assets  []releaseAsset `json:"assets"`
}

type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

// selfUpdateEnvelope is the self-update kind of plan 10.7.
type selfUpdateEnvelope struct {
	SchemaVersion int     `json:"schemaVersion"`
	Kind          string  `json:"kind"`
	Current       string  `json:"current"`
	Latest        string  `json:"latest"`
	Gap           string  `json:"gap"`
	Asset         *string `json:"asset"`
	Target        *string `json:"target"`
	Applied       bool    `json:"applied"`
}

// parseSemver reads v0.8.0 or 0.8.0. A prerelease suffix fails the
// parse on purpose: the latest release is never a prerelease per 12.6,
// and a current version carrying one is not a release this command
// can reason from.
func parseSemver(v string) (maj, min, pat int, ok bool) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return 0, 0, 0, false
		}
		nums[i] = n
	}
	return nums[0], nums[1], nums[2], true
}

// gapOf names the highest version component by which latest exceeds
// current: none, patch, minor, or major. A latest at or behind the
// current is none, because offering a downgrade is never an update.
func gapOf(current, latest string) (string, error) {
	cmaj, cmin, cpat, ok := parseSemver(current)
	if !ok {
		return "", fmt.Errorf("cannot parse the running version %q", current)
	}
	lmaj, lmin, lpat, ok := parseSemver(latest)
	if !ok {
		return "", fmt.Errorf("cannot parse the latest release tag %q", latest)
	}
	switch {
	case lmaj > cmaj:
		return "major", nil
	case lmaj < cmaj:
		return "none", nil
	case lmin > cmin:
		return "minor", nil
	case lmin < cmin:
		return "none", nil
	case lpat > cpat:
		return "patch", nil
	default:
		return "none", nil
	}
}

// gapExit maps a gap onto the graded bucket of plan 10.2.
func gapExit(gap string) int {
	switch gap {
	case "patch":
		return 10
	case "minor":
		return 11
	case "major":
		return 12
	default:
		return 0
	}
}

// assetFor names the archive the release pipeline publishes for this
// platform, matching the name_template in .goreleaser.yaml.
func assetFor(tag, goos, goarch string) string {
	ext := "tar.gz"
	if goos == "windows" {
		ext = "zip"
	}
	return fmt.Sprintf("git-ticket_%s_%s_%s.%s", strings.TrimPrefix(tag, "v"), goos, goarch, ext)
}

func runSelfUpdate(ctx *cmdContext, args []string) error {
	var check, dryRun bool
	rest, err := ctx.parseFlags("self-update", args, func(fs *flag.FlagSet) {
		fs.BoolVar(&check, "check", false, "report whether an update exists and exit by the graded bucket")
		fs.BoolVar(&dryRun, "dry-run", false, "also name the asset and target, writing nothing")
	})
	if err != nil {
		return err
	}
	if len(rest) != 0 {
		return usageErr("self-update takes no arguments")
	}
	if check && dryRun {
		return usageErr("--check and --dry-run are the same question at two levels of detail; pick one")
	}

	current := runningVersion()
	if current == "devel" {
		return errors.New("this binary reports devel, which means it was built from a working tree rather than a release; replacing it would destroy that build. Run `just install` from the tree, or `go install github.com/terva-sh/git-ticket/cmd/git-ticket@latest`")
	}

	api := ctx.env.ReleaseAPI
	if api == "" {
		api = defaultReleaseAPI
	}
	client := &http.Client{Timeout: 30 * time.Second}

	rel, err := fetchLatest(client, api, current)
	if err != nil {
		return err
	}
	gap, err := gapOf(current, rel.TagName)
	if err != nil {
		return err
	}

	if check {
		return reportSelfUpdate(ctx, current, rel.TagName, gap, nil, nil)
	}

	target := ctx.env.Executable
	if target == "" {
		target, err = os.Executable()
		if err != nil {
			return fmt.Errorf("cannot resolve this binary's own path: %w", err)
		}
		if resolved, rerr := filepath.EvalSymlinks(target); rerr == nil {
			target = resolved
		}
	}
	asset := assetFor(rel.TagName, runtime.GOOS, runtime.GOARCH)

	if dryRun {
		return reportSelfUpdate(ctx, current, rel.TagName, gap, &asset, &target)
	}

	if gap == "none" {
		return reportSelfUpdate(ctx, current, rel.TagName, gap, &asset, &target)
	}

	if err := applySelfUpdate(client, rel, asset, target); err != nil {
		return err
	}
	if ctx.g.json {
		writeJSON(ctx.out, selfUpdateEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "self-update",
			Current:       current,
			Latest:        rel.TagName,
			Gap:           gap,
			Asset:         &asset,
			Target:        &target,
			Applied:       true,
		})
		return nil
	}
	fmt.Fprintf(ctx.out, "updated %s to %s\n", target, rel.TagName)
	return nil
}

// reportSelfUpdate is both no-op modes and the up-to-date bare run:
// say what was found, then exit by the graded bucket. asset and
// target are nil under --check, which stops before resolving either,
// per plan 10.7.
func reportSelfUpdate(ctx *cmdContext, current, latest, gap string, asset, target *string) error {
	if ctx.g.json {
		writeJSON(ctx.out, selfUpdateEnvelope{
			SchemaVersion: schemaVersion,
			Kind:          "self-update",
			Current:       current,
			Latest:        latest,
			Gap:           gap,
			Asset:         asset,
			Target:        target,
			Applied:       false,
		})
	} else if gap == "none" {
		fmt.Fprintf(ctx.out, "git-ticket %s is up to date (latest release is %s)\n", current, latest)
	} else {
		fmt.Fprintf(ctx.out, "git-ticket %s → %s: a %s update is available; run `git ticket self-update`\n", current, latest, gap)
		if asset != nil && target != nil {
			fmt.Fprintf(ctx.out, "would download %s and replace %s\n", *asset, *target)
		}
	}
	if code := gapExit(gap); code != 0 {
		return &exitStatusErr{code: code}
	}
	return nil
}

// fetchLatest asks the releases API for the newest non-prerelease.
func fetchLatest(client *http.Client, api, current string) (*releaseInfo, error) {
	body, err := fetchURL(client, api+releasePath, current)
	if err != nil {
		return nil, fmt.Errorf("asking for the latest release: %w", err)
	}
	var rel releaseInfo
	if err := json.Unmarshal(body, &rel); err != nil {
		return nil, fmt.Errorf("reading the release answer: %w", err)
	}
	if rel.TagName == "" {
		return nil, errors.New("the releases API answered without a tag_name")
	}
	return &rel, nil
}

// fetchURL is every request this command makes. The User-Agent is
// required by the GitHub API, and naming the version makes a request
// log legible.
func fetchURL(client *http.Client, url, current string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "git-ticket/"+current)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// applySelfUpdate downloads, verifies, and swaps. The sha256 is
// checked before anything on disk moves, per plan 12.6.
func applySelfUpdate(client *http.Client, rel *releaseInfo, asset, target string) error {
	var assetURL, sumsURL string
	for _, a := range rel.Assets {
		switch a.Name {
		case asset:
			assetURL = a.URL
		case "checksums.txt":
			sumsURL = a.URL
		}
	}
	if assetURL == "" {
		return fmt.Errorf("release %s carries no asset for %s/%s (wanted %s)", rel.TagName, runtime.GOOS, runtime.GOARCH, asset)
	}
	if sumsURL == "" {
		return fmt.Errorf("release %s carries no checksums.txt, so nothing can be verified", rel.TagName)
	}

	current := runningVersion()
	archive, err := fetchURL(client, assetURL, current)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", asset, err)
	}
	sums, err := fetchURL(client, sumsURL, current)
	if err != nil {
		return fmt.Errorf("downloading checksums.txt: %w", err)
	}
	if err := verifyChecksum(sums, asset, archive); err != nil {
		return err
	}

	bin, err := extractBinary(asset, archive)
	if err != nil {
		return err
	}
	return replaceExecutable(target, bin)
}

// verifyChecksum holds the downloaded bytes to the line checksums.txt
// carries for this asset.
func verifyChecksum(sums []byte, asset string, archive []byte) error {
	want := ""
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == asset {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("checksums.txt has no line for %s", asset)
	}
	sum := sha256.Sum256(archive)
	if got := hex.EncodeToString(sum[:]); got != want {
		return fmt.Errorf("checksum mismatch for %s: downloaded %s, checksums.txt says %s; refusing to install it", asset, got, want)
	}
	return nil
}

// extractBinary pulls git-ticket out of the archive root, which is
// where .goreleaser.yaml puts it.
func extractBinary(asset string, archive []byte) ([]byte, error) {
	want := "git-ticket"
	if strings.HasSuffix(asset, ".zip") {
		want += ".exe"
		zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			return nil, fmt.Errorf("opening %s: %w", asset, err)
		}
		for _, f := range zr.File {
			if f.Name != want {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
		return nil, fmt.Errorf("%s does not contain %s at its root", asset, want)
	}

	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", asset, err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", asset, err)
		}
		if hdr.Typeflag == tar.TypeReg && filepath.Base(hdr.Name) == want && !strings.Contains(hdr.Name, "/") {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("%s does not contain %s at its root", asset, want)
}

// replaceExecutable swaps the binary atomically: write beside the
// target, then rename. Windows cannot overwrite a running executable,
// so the old one is renamed aside first and left for the OS to let go
// of.
func replaceExecutable(target string, bin []byte) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".git-ticket-new-*")
	if err != nil {
		return fmt.Errorf("cannot write beside %s: %w; check who owns that directory", target, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(bin); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		old := target + ".old"
		os.Remove(old)
		if err := os.Rename(target, old); err != nil {
			return fmt.Errorf("cannot move the running binary aside: %w", err)
		}
	}
	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("cannot replace %s: %w; check who owns it", target, err)
	}
	return nil
}
