package ticket

import (
	"errors"
	"fmt"
	"strings"
)

func asTicketError(err error, target **Error) bool { return errors.As(err, target) }

// diffLines reports the first line where two renderings differ, with a little
// context. A full diff of a 40-line frontmatter buries the one line that
// matters.
func diffLines(want, got string) string {
	w := strings.Split(want, "\n")
	g := strings.Split(got, "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		var wl, gl string
		if i < len(w) {
			wl = w[i]
		}
		if i < len(g) {
			gl = g[i]
		}
		if wl == gl {
			continue
		}
		var b strings.Builder
		fmt.Fprintf(&b, "first difference at line %d\n", i+1)
		for j := max(0, i-2); j < i; j++ {
			fmt.Fprintf(&b, "  %s\n", w[j])
		}
		fmt.Fprintf(&b, "- want: %q\n", wl)
		fmt.Fprintf(&b, "+ got:  %q\n", gl)
		return b.String()
	}
	return "no line differs, but the strings are not equal"
}
