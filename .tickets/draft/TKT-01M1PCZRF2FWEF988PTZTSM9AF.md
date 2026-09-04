---
schema: 1
id: TKT-01M1PCZRF2FWEF988PTZTSM9AF
title: Publish binaries with goreleaser on both forges
type: task
status: draft
status_reason: null
priority: high
due_on: null
labels:
  - release
  - ci
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-04T14:25:58Z
updated_at: 2026-09-04T14:25:58Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

The README says there is no binary release, so installing means building from source. That makes a Go toolchain a prerequisite for every adopter, which is the largest remaining barrier to another project picking this up. go install serves a Go shop and nobody else.

The pattern to follow is in the sibling terva repository, and most of it deliberately does not transfer.

### What terva does

terva runs a curated public cut. scripts/release.sh builds a public worktree excluding .github, scripts/release-overlay and deployments/fpkg, then apply_overlay copies scripts/release-overlay/.goreleaser.yaml over .goreleaser.yaml and writes .github/workflows into it. Two goreleaser configs exist, and a check-overlay step diffs their shared packaging region in CI. Its .github/workflows/release.yml is upstream code, fork-guarded to another repository so it never fires there.

### Why most of that does not transfer

The mirror here is a plain git push, not a built tree: same commits, same tags, one history, per AGENTS.md and plan 12.2. There is nothing to exclude and no overlay to apply, so there is no curation script, no second config to keep in step, and no drift check to maintain. The tree that ships publicly is the tree.

What transfers is the Forgejo workflow shape. On push of a v tag, contents write permission, runs-on docker with the internal registry golang:1.25-alpine, apk add git, Actions-Mirrors/forgejo-actions-checkout@v6 with fetch-depth 0 so the changelog can read the log, and Actions-Mirrors/goreleaser-action@v6 running release --clean with GORELEASER_FORCE_TOKEN set to gitea and GITEA_TOKEN from a repo secret. The changelog filters are worth copying verbatim, because terva documents a real bug in them: an anchor matching only an unscoped chore never matches a scoped one, so scoped housekeeping commits leak into the notes they exist to be filtered out of.

### The decision this ticket owes

Whether one config serves both forges or two do. The release block differs, gitea for Forgejo and github for the mirror, and goreleaser resolves one release target per run. Two configs here would be near-duplicates with none of the curation reason terva has, and terva pays a CI check to police exactly that drift.

### The version trap, specific to this repository

terva stamps -X main.version at link time. This repository must not. Plan 12.1 says nothing is stamped at link time, that go build already records the module version and the commit, and that runtime/debug reads them back, so there is no version variable to keep in step with a tag. cli/version.go is built on that.

It works today. A build from this tree reports a real pseudo-version rather than devel.

The risk is that it goes quiet under goreleaser. buildVersion falls back to devel and unknown when a build carries no VCS stamps, and a container build where git refuses the checkout for dubious ownership produces exactly that, silently. A released binary answering devel is worse than a failed build, because it ships.

So verify rather than assume: unpack a published archive, run the binary with --version, and require the tag. If VCS stamping cannot survive goreleaser, the fallback is ldflags, which contradicts 12.1 and therefore changes docs/plan.md in the same commit with the reason.

### Scope

linux, darwin and windows on amd64 and arm64, CGO_ENABLED 0 and -trimpath, archives carrying LICENSE and README, and a sha256 checksums file. No container image, no package manager, and no installer script here. The README status callout stops saying there is no binary release.

## Acceptance criteria

- [ ] A v tag publishes archives and a checksums file to a release on both forges
- [ ] A binary unpacked from a published archive reports the tag from --version, never devel
- [ ] The release path is documented where AGENTS.md describes pushing the mirror
- [ ] If ldflags become necessary, docs/plan.md section 12.1 changes in the same commit
