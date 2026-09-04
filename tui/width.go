// Width helpers lifted from terva's packages/tui (MIT), with the
// inline-image handling left behind because this renderer paints none:
//   Copyright (c) 2026 Drew Short (Terva, a hard fork of zot)
//   Copyright (c) 2026 Patric Eckhart

package tui

import (
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

// VisibleWidth reports the number of terminal cells s occupies, with
// ANSI CSI and OSC escape sequences counted as zero.
func VisibleWidth(s string) int {
	return runewidth.StringWidth(StripANSI(s))
}

// StripANSI removes CSI and OSC escape sequences (see escSeqLenRunes).
// OSC matters because an OSC 8 hyperlink carries a URL that occupies no
// cells: counted as visible text it would make every measurement of a
// linkified line wrong by the length of the link target.
func StripANSI(s string) string {
	var out []rune
	i := 0
	runes := []rune(s)
	for i < len(runes) {
		if n := escSeqLenRunes(runes, i); n > 0 {
			i += n
			continue
		}
		out = append(out, runes[i])
		i++
	}
	return string(out)
}

// TruncateToWidth clips s so its on-screen width doesn't exceed cols
// cells, preserving ANSI CSI and OSC escape sequences (which don't
// consume cells).
//
// Fast path: a byte-length <= cols is a conservative upper bound
// guaranteeing the cell width is also <= cols, so we skip all the
// rune-width math. That covers the vast majority of rows.
func TruncateToWidth(s string, cols int) string {
	if cols <= 0 {
		return s
	}
	if len(s) <= cols {
		return s
	}
	var out strings.Builder
	out.Grow(len(s))
	seen := 0
	// Tracks an OSC 8 hyperlink left open by the bytes we keep. A row cut
	// mid-link would otherwise hand the terminal a link target with no
	// close, which claims every cell drawn after it.
	linkOpen := false
	for i := 0; i < len(s); {
		// CSI or OSC escape sequence: zero-width.
		if n := escSeqLen(s, i); n > 0 {
			seq := s[i : i+n]
			if strings.HasPrefix(seq, "\x1b]8;") {
				linkOpen = !isHyperlinkClose(seq)
			}
			out.WriteString(seq)
			i += n
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		rw := runewidth.RuneWidth(r)
		if seen+rw > cols {
			// Flush any trailing ANSI escapes (resets, erase-to-EOL,
			// a link close) so background colors and cleanup sequences
			// survive.
			for i < len(s) {
				n := escSeqLen(s, i)
				if n == 0 {
					break
				}
				seq := s[i : i+n]
				if strings.HasPrefix(seq, "\x1b]8;") {
					linkOpen = !isHyperlinkClose(seq)
				}
				out.WriteString(seq)
				i += n
			}
			break
		}
		if r == utf8.RuneError && size == 1 {
			// Invalid byte: emit U+FFFD, so a bad byte cannot smear the
			// column arithmetic of everything after it.
			out.WriteRune(utf8.RuneError)
		} else {
			out.WriteString(s[i : i+size])
		}
		seen += rw
		i += size
	}
	if linkOpen {
		out.WriteString("\x1b]8;;\x1b\\")
	}
	return out.String()
}

// isHyperlinkClose reports whether seq is the OSC 8 terminator, the
// form with an empty URI that ends a hyperlink span.
func isHyperlinkClose(seq string) bool {
	body := strings.TrimPrefix(seq, "\x1b]8;")
	body = strings.TrimSuffix(body, "\x1b\\")
	body = strings.TrimSuffix(body, "\x07")
	_, uri, ok := strings.Cut(body, ";")
	return ok && uri == ""
}
