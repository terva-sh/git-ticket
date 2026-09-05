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

// WrapPlain folds plain text to limit cells per line, breaking at
// spaces and hard-breaking a word wider than the limit. It is for
// prose that carries no escape sequences, which is what a ticket body
// is; styled text wants the ANSI-aware wrap that has not been lifted
// yet. Leading indentation on a line is preserved on its continuation
// lines, so a wrapped checklist item stays visibly one item.
func WrapPlain(s string, limit int) []string {
	if limit <= 0 {
		return []string{s}
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		out = append(out, wrapPlainLine(line, limit)...)
	}
	return out
}

func wrapPlainLine(line string, limit int) []string {
	if runewidth.StringWidth(line) <= limit {
		return []string{line}
	}
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	// A tab in the indent counts as one cell here, which is wrong but
	// stable; ticket bodies indent with spaces.
	indentW := runewidth.StringWidth(indent)
	if indentW >= limit {
		indent, indentW = "", 0
	}

	words := strings.Fields(line)
	var out []string
	cur := indent
	curW := indentW
	first := true
	for _, w := range words {
		ww := runewidth.StringWidth(w)
		for ww > limit-indentW {
			// A word wider than the pane: flush, then split the word.
			if !first {
				out = append(out, cur)
				cur, curW, first = indent, indentW, true
			}
			head, rest := splitAtWidth(w, limit-indentW)
			out = append(out, indent+head)
			w = rest
			ww = runewidth.StringWidth(w)
			if w == "" {
				break
			}
		}
		if w == "" {
			continue
		}
		sep := 1
		if first {
			sep = 0
		}
		if curW+sep+ww > limit {
			out = append(out, cur)
			cur, curW = indent+w, indentW+ww
		} else {
			if !first {
				cur += " "
				curW++
			}
			cur += w
			curW += ww
		}
		first = false
	}
	if cur != "" || len(out) == 0 {
		out = append(out, cur)
	}
	return out
}

// splitAtWidth cuts s at the last rune boundary within limit cells.
func splitAtWidth(s string, limit int) (head, rest string) {
	seen := 0
	for i, r := range s {
		rw := runewidth.RuneWidth(r)
		if seen+rw > limit {
			return s[:i], s[i:]
		}
		seen += rw
	}
	return s, ""
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

// hyperlinkClose is the OSC 8 terminator, and reset drops every SGR
// attribute. Both lifted with the editor's wrap path, which re-opens
// on each wrapped row what the previous row left open.
const (
	hyperlinkClose = "\x1b]8;;\x1b\\"
	reset          = "\x1b[0m"
)

// linkStateAfter scans piece from a starting OSC 8 state and returns
// the opening sequence in effect at the end of piece, or "" when no
// link is open. A wrapped row has to re-open what the previous row
// left open.
func linkStateAfter(state, piece string) string {
	for i := 0; i < len(piece); {
		n := escSeqLen(piece, i)
		if n == 0 {
			i++
			continue
		}
		seq := piece[i : i+n]
		if strings.HasPrefix(seq, "\x1b]8;") {
			if isHyperlinkClose(seq) {
				state = ""
			} else {
				state = seq
			}
		}
		i += n
	}
	return state
}
