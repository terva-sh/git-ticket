---
schema: 1
id: TKT-01M1QMPSPKET9ZPJD7G39594C2
title: Render detail sections as styled Markdown with highlighted code
type: task
status: done
status_reason: null
priority: normal
due_on: null
labels: []
assignees: []
milestone: null
parent: null
dependencies: []
blocks_on: none
references: []
claim: null
archive: null
created_at: 2026-09-05T02:00:07Z
updated_at: 2026-09-05T02:05:38Z
created_by:
  id: agent:terva/mieli
  name: ""
updated_by:
  id: agent:terva/mieli
  name: ""
extensions: {}
---

## Description

The detail view renders every section as plain wrapped text, and
`DetailView.build` has held a named slot for styled rendering since the
view shipped in v0.7.0. Fill it: lift terva's Markdown renderer so
headings, inline styles, lists, blockquotes, tables, and fenced code
render styled, with code highlighted through chroma.

The lift, from `../terva/packages/tui` under MIT with the same notice
the earlier lifts carry:

`RenderMarkdown` from markdown.go, adapted to a minimal local Theme.
The full terva Theme is chat-shaped (bubbles, spinners, meters); the
renderer reads exactly four things from it: `Accent`, `Muted`,
`FG256`, and `HighlightCode`. The minimal Theme carries those, the
`SyntaxTheme` palette, and the chroma style builder, and nothing else.

`HighlightCode` from highlight.go with its memoising cache, the lexer
chooser, and the background stripper. `LanguageFromPath` stays behind:
the detail view has fence language hints and no file paths.

Wrapping is already solved: the editor lift carries
`WrapANSILineKeepStyle`, so `build` renders each section through
`RenderMarkdown` and wraps each styled line with it, replacing
`WrapPlain` for section text.

New dependency: `github.com/alecthomas/chroma/v2`. Module graph
pruning keeps it out of any consumer that imports only `ticket` and
`cli`, the same argument that admitted the terminal stack.

Tests assert through the vt10x grid or StripANSI as the suite already
does: a `###` heading renders bold and accented, a fenced Go block
comes back colored, a table aligns, and long prose still wraps to the
width. StripANSI of the styled render must equal the prose a reader
would type, because plan 14's "reads correctly as written" holds for
the styled view too.

## Acceptance criteria

- [x] Headings, lists, blockquotes, tables, and inline styles render styled in the detail view
- [x] Fenced code with a language hint renders through chroma highlighting
- [x] StripANSI of the styled render preserves the section prose, and long lines still wrap to the width

## Notes

**agent:terva/mieli** at 2026-09-05T02:05:37Z

Built and all three criteria are ticked.

The lift landed as three files. tui/theme.go is the minimal Theme: the
two color slots the renderer reads (Accent 111, Muted 244, terva's dark
values), the SyntaxTheme palette (nord, byte for byte), and FG256, Bold,
Italic. tui/highlight.go is HighlightCode with its memoising cache, the
lexer chooser, the chroma style builder, and the background stripper,
minus LanguageFromPath, which no caller here has a file path for.
tui/markdown.go is RenderMarkdown with the table renderer, minus
FlushLeftSentinel, which existed for terva callers that predate its
current fence rendering.

DetailView.build now renders each section through RenderMarkdown and
folds each styled line with WrapANSILineKeepStyle, which the editor lift
already carried, so a colored span that crosses the fold keeps its color
on the continuation row. The one visible change to existing content:
Markdown list items render with the bullet, so a checklist reads
"• [ ]" rather than "- [ ]", and the section test moved with it.

Attribution grew alongside, at the user's direction, because the
borrowing is now substantial: NOTICE at the repository root reproduces
terva's full MIT text with both copyright lines (terva, and zot through
the fork), names every derived file, and TestNoticeAgreesWithTheHeaders
holds the file list and the per-file headers to each other in both
directions. tui/list.go had adapted terva's dialogs picker pattern
without a header and gained one in the audit. The README License
section now points at NOTICE.

New dependency: github.com/alecthomas/chroma/v2 v2.27.0. Module graph
pruning keeps it out of consumers that import only ticket and cli.

Seven new tests: five on the renderer (heading bold and accented, block
kinds styled, a Go fence through chroma with StripANSI equal to the
code, a bare fence accent-colored, table columns aligned) and two on
the wiring (styled sections reach the detail rows, long prose still
wraps under the width).

## Summary

Shipped. The detail view renders every section as styled Markdown:
headings bold and accented, lists bulleted, blockquotes barred, tables
aligned, inline code accented, and fenced code highlighted through
chroma with terva's nord palette. The lift is tui/theme.go,
tui/highlight.go, and tui/markdown.go, wired in DetailView.build
through WrapANSILineKeepStyle so styled lines fold without losing
color. StripANSI of the styled render preserves the written prose, per
plan 14. Attribution shipped alongside: NOTICE reproduces terva's full
MIT license and names every derived file, TestNoticeAgreesWithTheHeaders
keeps that list honest, and the README credits terva and zot.
