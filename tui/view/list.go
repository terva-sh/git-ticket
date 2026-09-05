// Package view is the git-ticket TUI application: the views that read
// and write tickets, built on the generic pieces in package tui. The
// split is deliberate. Package tui knows nothing about tickets, which
// is what lets it grow toward a shared terva-sh TUI library, and this
// package knows nothing about escape parsing or diffing, which is what
// keeps a view readable.
package view

import (
	"fmt"
	"sort"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

// Lister fetches the tickets to show. The list view calls it once at
// start and again on every reload, which is the change-detection story
// the TKT-01M1QBS9 spike settled: no watch API, re-list and diff by
// revision, because a re-list of a store this size is trivially cheap.
type Lister func() ([]*ticket.Ticket, error)

// ListView is the open-work list: one row per ticket, a cursor, and a
// footer that says what the keys are. It holds the data and the
// rendering; the generic cursor and scroll live in tui.List.
type ListView struct {
	list    tui.List
	tickets []*ticket.Ticket
	short   map[string]string
	err     error
	relist  Lister
	acts    Actions
	// msg is the transient footer line: the outcome of the last action,
	// or the re-present notice after a lost race. It stands until the
	// next action replaces it, because a message that clears on the
	// next keypress is a message nobody finished reading.
	msg string
}

// NewListView returns a view over relist and loads it once. Writes go
// through acts; a zero Actions makes the view read-only, and each
// action key says so instead of doing nothing.
func NewListView(relist Lister, acts Actions) *ListView {
	v := &ListView{relist: relist, acts: acts}
	v.Reload()
	return v
}

// Reload re-runs the lister. The selection follows the ticket, not the
// row: if the ticket under the cursor survives the reload it stays
// selected at its new position, because the position was never the
// thing the user chose. A lister error keeps the previous rows and
// surfaces in the footer, so a transient failure does not blank the
// screen.
func (v *ListView) Reload() {
	selected := v.SelectedID()
	tickets, err := v.relist()
	v.err = err
	if err != nil {
		return
	}
	v.tickets = tickets
	v.short = shortestUnique(tickets)
	v.list.SetTotal(len(tickets))
	if selected != "" {
		for i, t := range tickets {
			if t.ID == selected {
				v.list.SetCursor(i)
				break
			}
		}
	}
}

// SelectedTicket is the ticket under the cursor, or nil for an empty
// list. The detail view takes it as-is; the list keeps ownership.
func (v *ListView) SelectedTicket() *ticket.Ticket {
	if v.list.Total() == 0 || v.list.Cursor() >= len(v.tickets) {
		return nil
	}
	return v.tickets[v.list.Cursor()]
}

// SelectedID is the ID under the cursor, or empty for an empty list.
func (v *ListView) SelectedID() string {
	if t := v.SelectedTicket(); t != nil {
		return t.ID
	}
	return ""
}

// HandleKey routes one key. It reports quit when the key ends the
// view: q, Esc, or ctrl+c.
func (v *ListView) HandleKey(k tui.Key) (quit bool) {
	switch {
	case k.Kind == tui.KeyCtrlC, k.Kind == tui.KeyEsc,
		k.Kind == tui.KeyRune && k.Rune == 'q':
		return true
	case k.Kind == tui.KeyRune && k.Rune == 'r':
		v.Reload()
		v.msg = ""
		return false
	case k.Kind == tui.KeyRune && k.Rune == 'c':
		v.claim()
		return false
	case k.Kind == tui.KeyRune && k.Rune == 'u':
		v.release()
		return false
	}
	v.list.HandleKey(k)
	return false
}

// say puts a line in the footer until the next action replaces it.
func (v *ListView) say(msg string) { v.msg = msg }

// claim claims the selected ticket for the wired actor.
func (v *ListView) claim() {
	t := v.SelectedTicket()
	if t == nil {
		return
	}
	if v.acts.Claim == nil {
		v.say("claiming is not wired in this host")
		return
	}
	v.afterWrite("claimed "+v.short[t.ID], v.acts.Claim(t.ID, t.Revision))
}

// release drops the claim on the selected ticket.
func (v *ListView) release() {
	t := v.SelectedTicket()
	if t == nil {
		return
	}
	if v.acts.Release == nil {
		v.say("releasing is not wired in this host")
		return
	}
	v.afterWrite("released "+v.short[t.ID], v.acts.Release(t.ID, t.Revision))
}

// ApplyStatus moves the selected ticket to status. The App calls it
// when the status picker resolves.
func (v *ListView) ApplyStatus(status, reason string) {
	t := v.SelectedTicket()
	if t == nil {
		return
	}
	if v.acts.SetStatus == nil {
		v.say("status changes are not wired in this host")
		return
	}
	v.afterWrite(v.short[t.ID]+" is now "+status, v.acts.SetStatus(t.ID, t.Revision, status, reason))
}

// afterWrite is every action's landing: reload so the rows show what
// the store now holds, then say what happened. A lost race is the case
// this exists for, per the TKT-01M1QBS9 spike: stale_revision means
// another writer got there first, so the reload re-presents their
// version and the message says to look again. The write did NOT
// happen, and pretending otherwise is how a TUI overwrites somebody.
func (v *ListView) afterWrite(did string, err error) {
	v.Reload()
	switch {
	case err == nil:
		v.say(did)
	case ticket.CodeOf(err) == ticket.CodeStaleRevision:
		v.say("changed by another writer; reloaded, try again")
	default:
		v.say("error: " + err.Error())
	}
}

// Render lays the view out for a cols x rows terminal: a header row,
// the windowed list, and a footer. The Frame clips each row to cols,
// so Render only decides content, never width.
func (v *ListView) Render(cols, rows int) []string {
	if rows < 3 {
		rows = 3
	}
	body := rows - 2 // header and footer

	out := make([]string, 0, rows)
	out = append(out, dim(v.header()))

	if len(v.tickets) == 0 {
		out = append(out, "", "  No open work.")
		for len(out) < rows-1 {
			out = append(out, "")
		}
		out = append(out, dim(v.footer()))
		return out
	}

	start, end := v.list.Window(body)
	for i := start; i < end; i++ {
		out = append(out, v.row(i, i == v.list.Cursor()))
	}
	for len(out) < rows-1 {
		out = append(out, "")
	}
	out = append(out, dim(v.footer()))
	return out
}

// widths returns the ID, status, and priority column widths for the
// current rows. Content-derived, because a fixed ID width is exactly
// the abbreviation mistake the store's own CLI refuses to make.
func (v *ListView) widths() (id, status, prio int) {
	id, status, prio = 2, 6, 8 // header widths: ID, STATUS, PRIORITY
	for _, t := range v.tickets {
		id = max(id, len(v.short[t.ID]))
		status = max(status, len(t.Status))
		prio = max(prio, len(t.Priority))
	}
	return id, status, prio
}

func (v *ListView) header() string {
	idW, stW, prW := v.widths()
	return fmt.Sprintf("  %-*s  %-*s  %-*s  %s", idW, "ID", stW, "STATUS", prW, "PRIORITY", "TITLE")
}

func (v *ListView) row(i int, selected bool) string {
	t := v.tickets[i]
	idW, stW, prW := v.widths()
	text := fmt.Sprintf("%-*s  %-*s  %-*s  %s", idW, v.short[t.ID], stW, t.Status, prW, t.Priority, t.Title)
	if selected {
		return "\x1b[7m▸ " + text + "\x1b[27m"
	}
	return "  " + text
}

func (v *ListView) footer() string {
	if v.err != nil {
		return "  error: " + v.err.Error() + " · r retry · q quit"
	}
	if v.msg != "" {
		return "  " + v.msg
	}
	n := len(v.tickets)
	word := "tickets"
	if n == 1 {
		word = "ticket"
	}
	return fmt.Sprintf("  %d open %s · Enter open · s status · c claim · u release · r reload · q quit", n, word)
}

func dim(s string) string { return "\x1b[2m" + s + "\x1b[22m" }

// shortestUnique maps each ID to the fewest characters that still
// resolve to it, never fewer than eight. It mirrors the CLI's rule in
// cli/commands.go, which is git's rule for object hashes: a ULID opens
// with ten characters of timestamp, so tickets created in the same
// millisecond are identical that far in, and a fixed width would print
// one abbreviation on two rows.
func shortestUnique(tickets []*ticket.Ticket) map[string]string {
	const abbrevLen = 8
	bodies := make([]string, 0, len(tickets))
	for _, t := range tickets {
		bodies = append(bodies, ticket.NormalizeRef(t.ID))
	}
	sorted := append([]string{}, bodies...)
	sort.Strings(sorted)

	// In sorted order the longest prefix an ID shares with any other is
	// shared with one of its two neighbours, so the pairs are enough.
	need := make(map[string]int, len(sorted))
	for i, b := range sorted {
		n := abbrevLen
		if i > 0 {
			n = max(n, commonPrefixLen(b, sorted[i-1])+1)
		}
		if i+1 < len(sorted) {
			n = max(n, commonPrefixLen(b, sorted[i+1])+1)
		}
		need[b] = n
	}

	out := make(map[string]string, len(tickets))
	for i, t := range tickets {
		b := bodies[i]
		n := min(need[b], len(b))
		out[t.ID] = ticket.IDPrefix + b[:n]
	}
	return out
}

func commonPrefixLen(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	return n
}
