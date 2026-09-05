---
schema: 1
id: TKT-01M1SZ1XJRBW5KE95JJZ5BH1CK
title: Publish an Alpine container image with git-ticket to ghcr on tags
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels:
  - release
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-05T23:39:26Z
updated_at: 2026-09-05T23:53:09Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

Publish a container image carrying `git-ticket` and `git`, built on
Alpine, to ghcr.io on every `v*` tag, from the GitHub mirror only.

The first use is pipelines that want `git ticket` available without
installing it. It is not only that: the image is also a way to run the
tool without installing it at all, which is why it takes the plain
repository name rather than a `-ci` suffix.

### Settled with the user on 2026-09-05, before the branch

The name is `ghcr.io/terva-sh/git-ticket`.

A `v0.11.2` tag pushes `0.11.2`, `0.11`, and `latest`, so a pipeline
can pin exactly or float on a minor.

linux/amd64 only. goreleaser already builds arm64, so adding it later
is a manifest and not a port, but nobody has asked.

The image is built by a separate job in
`.github/workflows/release.yml` that runs after goreleaser and
downloads the published release archive. The image then provably
contains the exact binary the release shipped, rather than a second
build that could differ.

Contents beyond `git-ticket`: `git`, `ca-certificates`,
`openssh-client`, `tzdata`, and `just`.

No `ENTRYPOINT`, `CMD` is a shell, running as root. A conventional CI
base image that a pipeline runs arbitrary steps in.

### The Forgejo half stays untouched

Only the mirror builds and publishes images. `.forgejo/workflows/`
gets no image job.

That needs the same guard the goreleaser job already carries.
Forgejo Actions also executes `.github/workflows`, and it reports this
repository as `terva-sh/git-ticket` exactly as GitHub does, so a
repository check cannot tell them apart. The existing job compares
`github.server_url` against the public URL instead, and the new job
has to do the same or the Forgejo runner will try it and fail.

### Two things to get right

The workflow needs `packages: write`. It currently declares
`contents: write` only, which is enough for goreleaser and not for
ghcr.

A prerelease must not move `latest` or the minor tag. `.goreleaser.yaml`
sets `prerelease: auto`, so `v1.0.0-rc1` publishes as a prerelease
today, and an image job that blindly tagged `latest` would hand every
floating consumer a release candidate.

## Acceptance criteria

- [x] A v* tag pushed to the mirror publishes ghcr.io/terva-sh/git-ticket tagged with the full version, the minor, and latest, and the image reports that tag from git ticket --version.
- [x] The image is Alpine-based and carries git, ca-certificates, openssh-client, tzdata and just, with no ENTRYPOINT, a shell as CMD, and running as root.
- [x] The git-ticket inside the image is the binary unpacked from the published release archive, not a second build from source.
- [x] Only the GitHub mirror builds and publishes: the image job carries the same github.server_url guard as the goreleaser job, and .forgejo/workflows is unchanged.
- [x] A prerelease tag such as v1.0.0-rc1 publishes its own version tag and does not move latest or the minor tag.
- [x] The release workflow declares packages: write, which it needs for ghcr and did not have.

## Notes

**agent:terva/mieli** at 2026-09-05T23:44:05Z

Built and proven locally with podman, which is available on this
machine while the docker daemon is not.

### What was run

The image was built exactly the way CI will build it: the v0.11.1
linux/amd64 release archive downloaded with `gh release download`,
unpacked into a context directory holding git-ticket, LICENSE and
README.md, and `podman build -f Dockerfile` against that directory
rather than against the repository.

Inside it: `git-ticket v0.11.1 (c15770135cea, go1.25.0)`, git 2.49.1,
just 1.40.0, OpenSSH 10.0p2, `id -u -n` is root, `TZ=Europe/Helsinki
date +%Z` prints EEST, and `/usr/share/doc/git-ticket/LICENSE` is the
21-line MIT text. `podman inspect` reports Cmd `[sh]` with an empty
entrypoint and an empty user. The image is 42.5 MB.

That go1.25.0 is the proof for criterion 3 rather than a detail. This
machine builds with go1.26.2, so a binary reporting 1.25.0 came from
the GitHub runner that built the release, not from here.

A full lifecycle ran inside the container: `git init`, `git ticket
init`, `create`, a status transition to ready, `ready`, and `check
--strict` reporting no problems, with git seeing the store files
afterwards.

`just` is in Alpine's repositories, so the base is `alpine:3.22` with
no third-party repository added.

### Criterion 1 cannot be proven before a tag

It asks that a `v*` tag publish the image to ghcr with three tags. No
tag has been pushed since this was written, so it ships unticked, in
the same shape as TKT-01M1S02QA's curl-from-the-mirror criterion. The
evidence run belongs to the next release's verification: pull
`ghcr.io/terva-sh/git-ticket` at the new version, the minor, and
latest, and check each reports the tag.

What could be proven was. The tag arithmetic was run through the
workflow's exact shell for v0.11.2, v1.0.0, v1.0.0-rc1 and v0.11.10.
The first three produce version, minor and latest; the release
candidate produces its own version alone, which is criterion 5.

### One sharp edge, deliberately not smoothed

git refuses to work on a mounted repository owned by a different uid:
`fatal: detected dubious ownership in repository at '/w'`. Confirmed
by running the image with `--user 12345` against this checkout, where
plain `git status` fails. `git ticket check` survives it, because it
degrades when it cannot resolve a repository root, which makes the
failure quieter rather than absent.

A CI image commonly answers this with `git config --system --add
safe.directory '*'`. That was not added. It weakens a security check
for everyone who runs the image, including the person using it as a
tool rather than as a pipeline base, and that is a decision for the
user rather than a default to slip in. The README documents the
workaround instead, and it was verified verbatim before being
written: the same failing command succeeds with
`GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0=safe.directory
GIT_CONFIG_VALUE_0=/w`.

**agent:terva/mieli** at 2026-09-05T23:53:09Z

Criterion 1 is now proven and ticked. It shipped unticked because it
asked for something no branch can show: that a real `v*` tag reaches
ghcr. v0.11.2 was that tag, and this note records the evidence run.

### The run

v0.11.2 is `c8e13b5beeaccbc95014d4482781cf5cd520a796`, pushed to both
remotes. GitHub run 33999609163 finished with both jobs
`completed success`, `Build and Publish Release` and `Build and
Publish Image`. Forgejo run 331 succeeded with its release job alone
and no image job, which is criterion 4 holding on a real tag rather
than in a workflow file.

All three tags pulled with `podman pull` and no credentials, so the
package is public:

```text
0.11.2   sha256:832abeb47e4a944a78cbfd66e7be20fbf65b8acfaf99e8a6da345098dd2dfc4b
0.11     sha256:832abeb47e4a944a78cbfd66e7be20fbf65b8acfaf99e8a6da345098dd2dfc4b
latest   sha256:832abeb47e4a944a78cbfd66e7be20fbf65b8acfaf99e8a6da345098dd2dfc4b
```

One digest behind all three, which is the tagging scheme working
rather than three separate builds that happen to agree. Each reports
the tag:

```text
git-ticket v0.11.2 (c8e13b5beeac, go1.25.0)
{"version":"v0.11.2","commit":"c8e13b5...","go":"go1.25.0","modified":false}
```

The image also carries git 2.49.1, just 1.40.0, OpenSSH 10.0p2,
tzdata and the CA bundle, and runs as root, which re-proves criterion
2 against the published artifact instead of the local build.

### The Go version is the provenance

`go1.25.0` is worth more than it looks. The Forgejo archive of the
same tag reports `go1.25.12`, because alpine builds with a different
Go patch release, and this machine builds with `go1.26.2`. So the
version string alone distinguishes three possible origins, and the
image matches the GitHub archive exactly. That is criterion 3, that
the binary is the published artifact and not a second build, checked
against the shipped image rather than against the workflow's
intention.

### What could not be read, and why it does not matter

`gh api users/terva-sh/packages/container/git-ticket` and the `orgs/`
form both returned 403, `You need at least read:packages scope to get
a package`. So package visibility was never read through the API.
That is a missing scope on the local `gh` token, not a fact about the
package, and the anonymous pull is the stronger evidence anyway: an
unauthenticated client fetched all three tags, which is what public
means operationally.

The rest of the v0.11.2 verification passed alongside: six assets on
each forge, `sha256sum -c` OK on all five archives on both, the
unpacked linux_amd64 binary reporting the tag with `modified: false`,
and the proxy's `@v/v0.11.2.info` naming
`c8e13b5beeaccbc95014d4482781cf5cd520a796`, equal to
`git rev-parse v0.11.2^{commit}`.

## Summary

Shipped and proven on a real tag. A `v*` tag pushed to the mirror
builds and publishes `ghcr.io/terva-sh/git-ticket`, from an `image`
job in `.github/workflows/release.yml` that runs after goreleaser.

The image is `alpine:3.22` carrying `git-ticket`, `git`,
`ca-certificates`, `openssh-client`, `tzdata` and `just`, with no
entrypoint, a shell as its command, and running as root, so it serves
both as a pipeline base and as a way to run the tool without
installing it. LICENSE ships inside it, because MIT wants the notice
with every copy and an image redistributes the binary exactly as an
archive does.

The binary is not a second build. The job downloads the published
release archive and builds against that, so the image provably
contains the artifact the release shipped, and a final step pulls the
pushed image back and checks it reports the tag.

Tagging is the full version, the minor, and `latest`. A prerelease
publishes its own version alone and never moves `latest`, since
goreleaser already treats `v1.0.0-rc1` as a prerelease and a floating
consumer should not be handed a release candidate.

Only the mirror publishes. The job carries the same
`github.server_url` guard as the goreleaser job, because Forgejo
Actions also executes `.github/workflows` and reports the same owner
and name. `.forgejo/workflows` is unchanged. The job takes job-level
`contents: read` and `packages: write`, the second of which the
workflow did not have.

All six criteria are proven. Five were shown on the branch. The
first needed a tag to exist, and v0.11.2 supplied it: all three tags
pull anonymously and resolve to one digest, each reporting
`v0.11.2 (c8e13b5beeac, go1.25.0)` with `modified: false`. That Go
patch release is also the provenance, since the Forgejo archive of
the same tag reads `go1.25.12` and a local build reads `go1.26.2`,
so the image demonstrably carries the GitHub-published artifact. The
note above records the run.

The image does not set `safe.directory`, so git refuses a mounted
repository owned by another uid. That is a decision left to the user
rather than a default slipped in, and the README carries the verified
workaround.
