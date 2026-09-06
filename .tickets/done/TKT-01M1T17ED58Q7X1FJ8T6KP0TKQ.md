---
schema: 1
id: TKT-01M1T17ED58Q7X1FJ8T6KP0TKQ
title: Give the image a one-variable opt-in for safe.directory
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
created_at: 2026-09-06T00:17:24Z
updated_at: 2026-09-06T00:20:09Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

The published image refuses to work on a mounted repository owned by a
different uid: `fatal: detected dubious ownership in repository at
'/w'`. TKT-01M1SZ1X shipped it that way on purpose, declining the
blanket `git config --system safe.directory '*'` that a CI image
usually reaches for, because that weakens a security check for
everyone who runs the image as a tool rather than as a pipeline base.
The README documents a per-invocation workaround instead, three
environment variables long.

The user has since ruled that the exception should exist behind an
opt-in rather than not at all. This ticket builds that opt-in.

### The mechanism, and why this one

git reads a `GIT_CONFIG_*` triple only when `GIT_CONFIG_COUNT` says
how many entries there are. So the image bakes the key and the value
and leaves the count out:

```dockerfile
ENV GIT_CONFIG_KEY_0=safe.directory
ENV GIT_CONFIG_VALUE_0=*
```

With no count they are inert and the image trusts nothing. Supplying
`-e GIT_CONFIG_COUNT=1` turns them on, which makes the opt-in one
variable instead of three.

The alternative was an entrypoint script reading a variable of our
own naming, such as `GIT_TICKET_TRUST_WORKSPACE=1`. It reads better
and it was rejected, because it does not work where it is most
needed. A CI container job does not run the image's entrypoint: the
runner starts the container with the entrypoint overridden and
delivers each step by `exec`, so a startup hook never executes. This
was measured rather than assumed, against the published v0.11.2
image, and the variable was present inside the container while git
still refused. Plain environment survives `exec`; startup code does
not.

Baked keys also keep the deliberate no-ENTRYPOINT decision of
TKT-01M1SZ1X intact, and cost two lines with no new machinery.

### What it costs

The opt-in is spelled in git's vocabulary, so `GIT_CONFIG_COUNT=1` in
a compose file does not announce itself as trusting the workspace the
way a named variable would. The README has to carry that meaning.

The image also quietly claims slot 0 of git's config-env protocol.
Someone who sets the count intending their own key 0, and does not
set it, inherits `safe.directory` instead. Setting their own
`GIT_CONFIG_KEY_0` overrides the baked value, so the collision is
narrow, but it is real and the README should say so.

## Acceptance criteria

- [x] With no GIT_CONFIG_COUNT the image still refuses a mounted repository owned by another uid, so it trusts nothing by default.
- [x] Under podman run, -e GIT_CONFIG_COUNT=1 makes both git and git ticket work on that repository.
- [x] The same one-variable opt-in works in a CI container-job shape, where the entrypoint is overridden and the command arrives by exec.
- [x] The Dockerfile gains no ENTRYPOINT and no startup script, keeping the no-entrypoint decision of TKT-01M1SZ1X intact.
- [x] The README documents the one-variable opt-in, what it means, and that the image claims slot 0 of the GIT_CONFIG_* protocol.

## Notes

**agent:terva/mieli** at 2026-09-06T00:20:09Z

Built as described, two `ENV` lines and a README rewrite. Every
criterion was checked against a real image built from the v0.11.2
release archive, not against a derived test image.

### The measurement that chose the mechanism

Four candidates were run against the published v0.11.2 image before
anything was written. The entrypoint hook is the one worth recording,
because it looks right and fails quietly.

A script reading a variable of our own naming, such as
`GIT_TICKET_TRUST_WORKSPACE=1`, works under `podman run`: as root it
writes system config, and as a non-root uid that write fails and an
exported `GIT_CONFIG_*` fallback carries it. Then the CI shape was
tried, a container created with `--entrypoint tail` and the step
delivered by `podman exec`, which is how a container job runs. The
variable was present inside the container, the hook never executed,
and git refused with `detected dubious ownership`. A startup hook is
absent in exactly the setting that most wants the exception.

Environment survives `exec`, so the baked-keys form was measured in
the same four shapes and worked in all of them.

### Evidence on the shipped Dockerfile

Built with `podman build -f Dockerfile -t git-ticket-optin:local
/tmp/imgctx`, the context being the unpacked
`git-ticket_0.11.2_linux_amd64.tar.gz`.

```text
env in image:  GIT_CONFIG_KEY_0=safe.directory
               GIT_CONFIG_VALUE_0=*
               (no GIT_CONFIG_COUNT)

run, no count            exit 128  detected dubious ownership
run + count, git         exit 0    7191d820...
run + count, git ticket  exit 0    lists drafts
exec + count, git        exit 0    7191d820...
exec + count, git ticket exit 0    lists drafts
exec, no count           exit 128  detected dubious ownership
Entrypoint []            Cmd [sh]
```

The `--user 12345` flag produces the ownership mismatch in every row
above. The quoting in `ENV GIT_CONFIG_VALUE_0="*"` does not leak into
the value, which the `env` output confirms.

### The slot-0 collision, measured rather than assumed

The README claims a caller's own `GIT_CONFIG_KEY_0` overrides the
baked one. Running with `GIT_CONFIG_COUNT=1
GIT_CONFIG_KEY_0=user.name GIT_CONFIG_VALUE_0=someone` returned
`someone` from `git config user.name` and then failed with
`detected dubious ownership`, exit 128. So the caller's entry wins
and `safe.directory` is genuinely gone, which is the behaviour worth
warning about and now the documented one.

## Summary

Shipped. The image now carries the ownership exception switched off,
and one variable arms it.

`ENV GIT_CONFIG_KEY_0=safe.directory` and `ENV GIT_CONFIG_VALUE_0=*`
sit in the Dockerfile with no `GIT_CONFIG_COUNT` beside them. git
reads a `GIT_CONFIG_*` triple only when the count says how many
entries exist, so with the count absent the pair is inert and the
image trusts nothing by default. Passing `-e GIT_CONFIG_COUNT=1`
turns it on, which replaces the three-variable incantation the README
used to prescribe.

The mechanism was chosen by measurement, not by taste. An entrypoint
script reading a variable of our own naming reads better and does not
work where it is needed: a CI container job overrides the entrypoint
and delivers each step by `exec`, so the hook never runs. That was
reproduced against the published v0.11.2 image, with the variable
present inside the container and git still refusing. Plain
environment survives `exec`, which is why this form is environment.

It also leaves the no-ENTRYPOINT decision of TKT-01M1SZ1X standing.
The image still has `Entrypoint []` and `Cmd [sh]`, so it remains a
base a pipeline runs arbitrary steps in.

Two costs are documented rather than hidden. The opt-in is spelled in
git's vocabulary, so the README says to read `GIT_CONFIG_COUNT=1` on
this image as trusting the mounted repository. And the image claims
slot 0: a caller who sets their own `GIT_CONFIG_KEY_0` overrides the
baked value and loses the exception, which was confirmed by running
it and watching git refuse.

All five criteria were checked against an image built from the
v0.11.2 release archive, in both the `run` and the `exec` shapes,
with and without the count.
