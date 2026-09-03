# The fixture corpus

Phase 0 of [`../docs/plan.md`](../docs/plan.md). These files are the executable
half of the format decision: the prose in sections 4 through 11 says what a
ticket is, and this corpus says what that means byte for byte.

Phase 1 consumes it three ways. The round-trip property in 5.3 runs over every
file in `parse/roundtrip/`. The golden tests pin those same files as the exact
bytes the renderer must produce. The `check` implementation in section 11 must
reproduce each recorded expectation.

## Layout

```text
testdata/
├── parse/
│   ├── roundtrip/     one ticket file that must parse and re-render identically
│   └── reject/        one ticket file that must not parse
└── stores/
    └── <case>/
        ├── store/     a whole store, because the finding spans files
        ├── expected.json
        └── ...         anything else stands in for the repository root
```

The split is not cosmetic. `duplicate_id`, `dependency_cycle`, and
`location_mismatch` cannot be expressed in a single file, so they need a
store. Everything else is cheaper to state and read as one file.

The case directory is the repository root for its store, which is what a
`references` path resolves against under 5.5. Only `reference-unresolved` uses
this, with a `docs/present.txt` that resolves beside a `docs/missing.md` that
does not. A test injects that root rather than running `git rev-parse`, so an
expectation never depends on where this repository is checked out.

## Every fixture carries its expectation

Each fixture has a sidecar, including the ones with nothing wrong. The absence
of a file is not an assertion: a fixture with no sidecar could mean "this is
clean" or "somebody forgot", and a reader cannot tell which. An explicit empty
result can only mean the first.

A `parse/` sidecar is `<name>.expected.json`:

```json
{
  "parse": "ok",
  "roundTrip": "identical",
  "errors": [],
  "warnings": []
}
```

A rejected file sets `"parse": "reject"` and `"roundTrip": null`, and names the
code that section 11 requires:

```json
{
  "parse": "reject",
  "roundTrip": null,
  "errors": [{ "code": "schema_unsupported", "file": "schema-2.md", "ticket": null, "field": "schema" }],
  "warnings": []
}
```

A store sidecar is `expected.json` and carries the two collections alone, since
`parse` and `roundTrip` describe a file rather than a store:

```json
{ "errors": [], "warnings": [] }
```

A finding is always these four keys. `file` is relative to `store/` for a store
case and is the fixture's own name for a parse case. `ticket` is null when the
file could not be parsed far enough to know its ID, and `field` is null when the
finding is about the file rather than one field.

A `parse/` sidecar records only what reading one file can find: `parse_error`,
`merge_conflict`, `schema_unsupported`, `unknown_field`, the three enum codes,
and `claim_expired`. The store-scoped codes are out of scope there and belong to
`stores/`, because no single file can exhibit them: `duplicate_id`,
`filename_id_mismatch`, `dependency_missing`, `parent_missing`,
`dependency_cycle`, `parent_cycle`, `location_mismatch`,
`dependency_archived_incomplete`, `label_unknown`, `milestone_unknown`, and
`reference_path_unresolved`.

That scope split is also why a `parse/` fixture is named after its condition
rather than after its ID. A store fixture must be `<id>.md` because
`filename_id_mismatch` is real there. A parse fixture is never checked against
its filename, so `minimal.md` is clearer than a 26-character ULID and costs
nothing.

Findings sort by `file`, then `code`, then `field`, so a test compares the two
lists directly instead of treating them as sets.

## Time is fixed at 2026-09-30T00:00:00Z

`claim_expired` is the one finding that depends on the clock, so a corpus judged
against the real one would start failing on its own. Every expectation here is
recorded against a reference time of `2026-09-30T00:00:00Z`, and a test must
inject that instant rather than read the system clock. Fixtures are authored so
the answer is not close to the boundary: an expired claim is days past, and a
live one either has no expiry or is weeks away.

Absent scalars are `null` and absent collections are `[]`, never omitted. That is
the same rule the JSON contract in section 10 imposes on the tool's own output,
and the corpus should not describe the format in a dialect the format rejects.

## Three conventions worth knowing before you add a fixture

Every store fixture carries an `epics.md`, even the ones with no epic in them,
where it says `No epics.` and nothing else. A store without the file reports
`epics_index_stale`, so a fixture that omits it trips a warning it was not
written to cover. Generate it with `git ticket check --fix` against the fixture
store rather than typing it, since it has to match `renderEpicsIndex` byte for
byte. The one deliberate exception is `epics-index-stale`, whose whole point is
an index that disagrees with the tickets.

The inner directory is `store/` and not `.tickets/`. Real stores are dot-named,
but `go:embed` silently skips any path whose name begins with `.` or `_`, so a
realistic name here would produce an empty corpus at build time and a confusing
set of passing tests. Discovery is not under test in these cases anyway: a test
points `--store` straight at the directory.

Every `.md` file uses LF endings and ends with exactly one newline, because 5.3
makes both part of the format. A fixture saved any other way fails the
round-trip test for a reason that has nothing to do with what it was written to
cover.

## Adding a fixture

Write the `.md`, write the sidecar, and add nothing to a manifest, because there
is no manifest to drift. A test walks the directories.

Name the file after the condition it covers rather than the ticket it contains,
so `dependency-cycle` and not `two-tickets`. Cover one condition per fixture. A
file that trips three codes at once tests the reporting order more than it tests
any of the three.
