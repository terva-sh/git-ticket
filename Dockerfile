# The published image: git-ticket and git on Alpine, per TKT-01M1SZ1X.
#
# The first use is a pipeline that wants `git ticket` without installing it,
# and it is equally a way to run the tool without installing it at all, which
# is why it takes the plain repository name rather than a -ci suffix.
#
# The build context is not this repository. .github/workflows/release.yml makes
# a directory holding the binary unpacked from the published release archive and
# builds against that, so what ships here is provably the artifact the release
# shipped rather than a second build that could differ from it. Building this
# file against the repository root will fail to find git-ticket, deliberately:
# the root copy is a gitignored `just build` artifact with no sense of its own
# age, and shipping that would be exactly the trap AGENTS.md warns about.
FROM alpine:3.22

# git is the point: git-ticket reads a store out of a repository and resolves
# reference paths against its root. ca-certificates is what makes HTTPS work at
# all, so without it a clone over https and `git ticket self-update` both fail.
# openssh-client is for a pipeline that clones or pushes over SSH. tzdata lets a
# named zone render, since Alpine ships UTC alone. just is here so a pipeline can
# run this repository's own recipes.
RUN apk add --no-cache \
        ca-certificates \
        git \
        just \
        openssh-client \
        tzdata

# The binary and its licence travel together. MIT requires the notice to ship
# with every copy, and an image redistributes the binary exactly as a release
# archive does, which is the reason .goreleaser.yaml puts LICENSE in every
# archive. Both files come out of that same archive.
COPY git-ticket /usr/local/bin/git-ticket
COPY LICENSE /usr/share/doc/git-ticket/LICENSE

# An opt-in exception to git's ownership check, shipped disarmed. git refuses to
# work on a mounted repository owned by another uid, and a CI image usually
# answers that with a blanket `safe.directory '*'` in its system config. That
# weakens the check for everyone, including whoever runs this as a tool rather
# than as a pipeline base, so the exception ships switched off instead.
#
# git reads a GIT_CONFIG_* triple only when GIT_CONFIG_COUNT says how many
# entries exist. With the count absent these two are inert and the image trusts
# nothing. Passing `-e GIT_CONFIG_COUNT=1` arms them, so the opt-in costs one
# variable rather than the three the README used to prescribe.
#
# This is environment rather than an entrypoint hook on purpose. A CI container
# job overrides the entrypoint and delivers each step by exec, so a startup
# script never runs in the setting that most wants the exception. Environment
# survives exec, and this was measured against the published image rather than
# assumed.
#
# The cost is that the image claims slot 0. Setting your own GIT_CONFIG_KEY_0
# overrides the value below, so a caller already using the protocol is not
# silently overruled, but a caller who sets only the count gets this.
ENV GIT_CONFIG_KEY_0="safe.directory"
ENV GIT_CONFIG_VALUE_0="*"

# Git spells a binary named git-ticket on PATH as `git ticket`, so both
# spellings work with nothing further to configure.
#
# No ENTRYPOINT, and a shell as CMD. This is a base image a pipeline runs
# arbitrary steps in, and an ENTRYPOINT of git-ticket would make every other
# command in that pipeline awkward. It runs as root for the same reason: a CI
# workspace is mounted with an ownership this image cannot predict, and a
# non-root default trips on it.
CMD ["sh"]
