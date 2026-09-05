// Lifted from terva's packages/tui (MIT):
//   Copyright (c) 2026 Drew Short (Terva, a hard fork of zot)
//   Copyright (c) 2026 Patric Eckhart

package tui

import "strconv"

// Theme is the minimal slice of terva's Theme that the Markdown
// renderer reads: two 256-color indexes, the styled-text helper, and
// the syntax palette behind HighlightCode. terva's full Theme is
// chat-shaped, with bubbles, spinners, and meters, and none of that
// earns a place here until a view asks for it.
type Theme struct {
	// Accent colors headings, list markers, and inline code.
	Accent int
	// Muted colors blockquote bars and table separators.
	Muted int
	// SyntaxBaseStyle names the chroma style the palette overrides,
	// monokai when empty.
	SyntaxBaseStyle string
	// Syntax is the token palette laid over the base style.
	Syntax SyntaxTheme
}

// SyntaxTheme maps chroma token classes to style entries, in chroma's
// entry syntax: a hex color, optionally followed by bold or italic.
type SyntaxTheme struct {
	Keyword             string
	KeywordConstant     string
	KeywordDeclaration  string
	KeywordNamespace    string
	KeywordReserved     string
	KeywordType         string
	NameBuiltin         string
	NameFunction        string
	NameClass           string
	NameDecorator       string
	LiteralString       string
	LiteralStringEscape string
	LiteralNumber       string
	Comment             string
	CommentPreproc      string
	Operator            string
	Punctuation         string
	Text                string
}

// nordSyntax is terva's default syntax palette, kept byte for byte so
// code reads the same in a ticket as it does in the chat that
// discussed it.
var nordSyntax = SyntaxTheme{
	Keyword:             "#81a1c1 bold",
	KeywordConstant:     "#81a1c1",
	KeywordDeclaration:  "#81a1c1",
	KeywordNamespace:    "#81a1c1",
	KeywordReserved:     "#81a1c1 bold",
	KeywordType:         "#88c0d0",
	NameBuiltin:         "#88c0d0",
	NameFunction:        "#8fbcbb",
	NameClass:           "#a3be8c bold",
	NameDecorator:       "#b48ead",
	LiteralString:       "#a3be8c",
	LiteralStringEscape: "#bf616a",
	LiteralNumber:       "#d08770",
	Comment:             "#616e88 italic",
	CommentPreproc:      "#b48ead",
	Operator:            "#eceff4",
	Punctuation:         "#d8dee9",
	Text:                "#e5e9f0",
}

// DefaultTheme carries terva's dark-theme values for the two color
// slots the renderer uses: soft blue for accents, grey for chrome.
var DefaultTheme = Theme{
	Accent:          111,
	Muted:           244,
	SyntaxBaseStyle: "monokai",
	Syntax:          nordSyntax,
}

// FG256 wraps s in a 256-color foreground and a reset.
func (t Theme) FG256(c int, s string) string {
	return "\x1b[38;5;" + strconv.Itoa(c) + "m" + s + reset
}

// Bold wraps s in SGR bold, closing with 22 rather than a full reset
// so an enclosing color survives.
func Bold(s string) string { return "\x1b[1m" + s + "\x1b[22m" }

// Italic wraps s in SGR italic, closing with 23 for the same reason.
func Italic(s string) string { return "\x1b[3m" + s + "\x1b[23m" }
