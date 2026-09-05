// Package view is the git-ticket TUI application: the views that read
// and write tickets, built on the generic pieces in package tui. The
// split is deliberate. Package tui knows nothing about tickets, which
// is what lets it grow toward a shared terva-sh TUI library, and this
// package knows nothing about escape parsing or diffing, which is what
// keeps a view readable.
package view

import (
	"fmt"
	"strings"

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
	tickets []*ticket.Ticket // everything the lister returned
	shown   []*ticket.Ticket // what the filter lets through
	short   map[string]string
	err     error
	relist  Lister
	acts    Actions
	// msg is the transient footer line: the outcome of the last action,
	// or the re-present notice after a lost race. It stands until the
	// next action replaces it, because a message that clears on the
	// next keypress is a message nobody finished reading.
	msg string

	// filterText is the filter line as typed; filterEditing is whether
	// the prompt has the keyboard. The parsed form is derived on every
	// change rather than stored, because two representations of one
	// filter is how they drift.
	filterText    string
	filterEditing bool

	// order indexes listOrders. The o key cycles it, and the header
	// names it, because an invisible sort mode reads as a broken list.
	order int

	// paletteIdx indexes palettes, the row-color mode the p key
	// cycles, per the TKT-01M1S022 session. paletteLocked pins it to
	// off when NO_COLOR is set, and p says so instead of cycling.
	paletteIdx    int
	paletteLocked bool
}

// listOrders is the sort vocabulary of plan 8, in the order o cycles
// through it. The names are the exact `list --sort` tokens and the
// apply functions are the ticket package's own, so the third surface
// teaches the same rule as the other two rather than a variant. A nil
// apply is the store's ID order, which needs no re-sort.
var listOrders = []struct {
	name  string
	apply func([]*ticket.Ticket)
}{
	{"id", nil},
	{"due_on", ticket.SortByDueOn},
	{"priority", ticket.SortByPriority},
	{"updated_at", ticket.SortByUpdated},
	{"status", ticket.SortByStatus},
}

// NewListView returns a view over relist and loads it once. Writes go
// through acts; a zero Actions makes the view read-only, and each
// action key says so instead of doing nothing.
func NewListView(relist Lister, acts Actions) *ListView {
	v := &ListView{relist: relist, acts: acts}
	v.paletteIdx, v.paletteLocked = startPalette()
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
	v.short = abbreviateIDs(tickets)
	v.applyFilter(selected)
}

// applyFilter rebuilds the shown slice from the full one and moves the
// selection to keep, when the filter still lets it through.
func (v *ListView) applyFilter(keep string) {
	f := parseFilter(v.filterText)
	order := listOrders[v.order]
	if f.empty() && order.apply == nil {
		v.shown = v.tickets
	} else {
		// Always a fresh slice when filtering or sorting: shown must
		// never share a backing array with tickets, or the sort would
		// quietly reorder the unfiltered view underneath it.
		v.shown = v.shown[:0:0]
		for _, t := range v.tickets {
			if f.match(t) {
				v.shown = append(v.shown, t)
			}
		}
		if order.apply != nil {
			order.apply(v.shown)
		}
	}
	v.list.SetTotal(len(v.shown))
	if keep != "" {
		for i, t := range v.shown {
			if t.ID == keep {
				v.list.SetCursor(i)
				break
			}
		}
	}
}

// SelectedTicket is the ticket under the cursor, or nil for an empty
// list. The detail view takes it as-is; the list keeps ownership.
func (v *ListView) SelectedTicket() *ticket.Ticket {
	if v.list.Total() == 0 || v.list.Cursor() >= len(v.shown) {
		return nil
	}
	return v.shown[v.list.Cursor()]
}

// FilterEditing reports whether the filter prompt has the keyboard,
// so the App routes every key here instead of interpreting Enter and
// the letter keys itself.
func (v *ListView) FilterEditing() bool { return v.filterEditing }

// TicketByID finds a ticket in the full listing, filter or no filter.
// The form's conflict loop uses it to re-arm a revision after a
// reload.
func (v *ListView) TicketByID(id string) *ticket.Ticket {
	for _, t := range v.tickets {
		if t.ID == id {
			return t
		}
	}
	return nil
}

// shortOr is the shortest-unique form of id when the listing knows
// it, and the full id when it does not, which happens when a filter
// hides the ticket a message is about.
func (v *ListView) shortOr(id string) string {
	if s, ok := v.short[id]; ok {
		return s
	}
	return id
}

// SelectedID is the ID under the cursor, or empty for an empty list.
func (v *ListView) SelectedID() string {
	if t := v.SelectedTicket(); t != nil {
		return t.ID
	}
	return ""
}

// HandleKey routes one key. It reports quit when the key ends the
// view: q or ctrl+c, and Esc only when there is no filter to clear
// first, because Esc means "back out one level" and an active filter
// is a level.
func (v *ListView) HandleKey(k tui.Key) (quit bool) {
	if v.filterEditing {
		v.handleFilterKey(k)
		return false
	}
	switch {
	case k.Kind == tui.KeyCtrlC, k.Kind == tui.KeyRune && k.Rune == 'q':
		return true
	case k.Kind == tui.KeyEsc:
		if v.filterText != "" {
			v.filterText = ""
			v.applyFilter(v.SelectedID())
			return false
		}
		return true
	case k.Kind == tui.KeyRune && k.Rune == '/':
		v.filterEditing = true
		v.msg = ""
		return false
	case k.Kind == tui.KeyRune && k.Rune == 'r':
		v.Reload()
		v.msg = ""
		return false
	case k.Kind == tui.KeyRune && k.Rune == 'o':
		v.order = (v.order + 1) % len(listOrders)
		v.applyFilter(v.SelectedID())
		v.msg = ""
		return false
	case k.Kind == tui.KeyRune && k.Rune == 'p':
		if v.paletteLocked {
			v.say("NO_COLOR is set; row colors stay off")
			return false
		}
		v.paletteIdx = (v.paletteIdx + 1) % len(palettes)
		v.say("color: " + palettes[v.paletteIdx].name)
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

// handleFilterKey is the filter prompt: runes append and narrow the
// list on every keystroke, because seeing the rows shrink is how you
// know the token you are typing is the right one. Enter keeps the
// filter and returns the keyboard to the list; Esc drops the filter
// entirely.
func (v *ListView) handleFilterKey(k tui.Key) {
	switch k.Kind {
	case tui.KeyEnter:
		v.filterEditing = false
	case tui.KeyEsc:
		v.filterEditing = false
		v.filterText = ""
		v.applyFilter(v.SelectedID())
	case tui.KeyBackspace:
		if v.filterText != "" {
			r := []rune(v.filterText)
			v.filterText = string(r[:len(r)-1])
			v.applyFilter(v.SelectedID())
		}
	case tui.KeyPaste:
		v.filterText += strings.ReplaceAll(k.Paste, "\n", " ")
		v.applyFilter(v.SelectedID())
	case tui.KeyRune:
		v.filterText += string(k.Rune)
		v.applyFilter(v.SelectedID())
	}
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

	if len(v.shown) == 0 {
		msg := "  No open work."
		if len(v.tickets) > 0 {
			msg = "  Nothing matches the filter."
		}
		out = append(out, "", msg)
		for len(out) < rows-1 {
			out = append(out, "")
		}
		out = append(out, v.footerRow())
		return out
	}

	start, end := v.list.Window(body)
	for i := start; i < end; i++ {
		out = append(out, v.row(i, i == v.list.Cursor()))
	}
	for len(out) < rows-1 {
		out = append(out, "")
	}
	out = append(out, v.footerRow())
	return out
}

// footerRow styles the footer: the filter prompt is live input and
// stays bright, everything else is chrome and dims.
func (v *ListView) footerRow() string {
	if v.filterEditing {
		return "  filter: " + v.filterText + "\x1b[7m \x1b[27m"
	}
	return dim(v.footer())
}

// widths returns the ID, status, type, and priority column widths for
// the current rows. Content-derived, because a fixed ID width is
// exactly the abbreviation mistake the store's own CLI refuses to make.
func (v *ListView) widths() (id, status, typ, prio int) {
	id, status, typ, prio = 2, 6, 4, 8 // header widths: ID, STATUS, TYPE, PRIORITY
	for _, t := range v.shown {
		id = max(id, len(v.short[t.ID]))
		status = max(status, len(t.Status))
		typ = max(typ, len(t.Type))
		prio = max(prio, len(t.Priority))
	}
	return id, status, typ, prio
}

// header and row put TYPE between STATUS and PRIORITY because that is
// the order the detail header prints: "draft spike low" is status,
// type, priority, and the list should not teach a different order than
// the view one keystroke deeper.
func (v *ListView) header() string {
	idW, stW, tyW, prW := v.widths()
	return fmt.Sprintf("  %-*s  %-*s  %-*s  %-*s  %-5s  · sort: %s · color: %s",
		idW, "ID", stW, "STATUS", tyW, "TYPE", prW, "PRIORITY", "TITLE",
		listOrders[v.order].name, palettes[v.paletteIdx].name)
}

func (v *ListView) row(i int, selected bool) string {
	t := v.shown[i]
	idW, stW, tyW, prW := v.widths()
	text := fmt.Sprintf("%-*s  %-*s  %-*s  %-*s  %s", idW, v.short[t.ID], stW, t.Status, tyW, t.Type, prW, t.Priority, t.Title)
	// The selected row takes no palette color, per the TKT-01M1S022
	// constraint: plain reverse-video stays legible over any candidate.
	if selected {
		return "\x1b[7m▸ " + text + "\x1b[27m"
	}
	if c := palettes[v.paletteIdx].color(t); c != "" {
		return "  " + c + text + "\x1b[0m"
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
	if v.filterText != "" {
		return fmt.Sprintf("  %d/%d match · filter: %s · / edit · Esc clear · ? help",
			len(v.shown), len(v.tickets), v.filterText)
	}
	n := len(v.tickets)
	word := "tickets"
	if n == 1 {
		word = "ticket"
	}
	// This line must fit 80 columns with a two-digit count, because the
	// Frame clips it and a clipped hint is an invisible control. The
	// keys that do not fit live on the ? help view, and
	// TestEveryListKeyIsHintedSomewhere holds the union complete.
	return fmt.Sprintf("  %d open %s · Enter open · n new · e edit · / filter · ? help · q quit", n, word)
}

func dim(s string) string { return "\x1b[2m" + s + "\x1b[22m" }

// abbreviateIDs maps each ID to the shortest form that still resolves,
// deferring to ticket.ShortestUnique so the TUI and the CLI cannot
// drift. The rule needs nothing from a ticket but its ID.
//
// The name says IDs because cli.abbreviate is a different operation on
// a different thing, cutting a title to a column width.
func abbreviateIDs(tickets []*ticket.Ticket) map[string]string {
	ids := make([]string, 0, len(tickets))
	for _, t := range tickets {
		ids = append(ids, t.ID)
	}
	return ticket.ShortestUnique(ids)
}
