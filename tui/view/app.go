package view

import (
	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

// App is the state machine over the views: the list is the floor, and
// the detail view or the status picker stacks on top of it. One level
// of stack, held in mutually exclusive fields rather than a slice,
// and it stays that way until a real stack earns the generalization.
type App struct {
	list   *ListView
	detail *DetailView
	picker *StatusPicker
	form   *FormView
}

// NewApp opens the list over relist, writing through acts.
func NewApp(relist Lister, acts Actions) *App {
	return &App{list: NewListView(relist, acts)}
}

// HandleKey routes one key to whichever view is on top.
func (a *App) HandleKey(k tui.Key) (quit bool) {
	if a.detail != nil {
		back, quit := a.detail.HandleKey(k)
		if quit {
			return true
		}
		if back {
			a.detail = nil
		}
		return false
	}
	if a.picker != nil {
		act := a.picker.HandleKey(k)
		if act.Quit {
			return true
		}
		if act.Apply {
			a.picker = nil
			a.list.ApplyStatus(act.Status, act.Reason)
			return false
		}
		if act.Cancel {
			a.picker = nil
		}
		return false
	}
	if a.form != nil {
		act := a.form.HandleKey(k)
		if act.Quit {
			return true
		}
		if act.Cancel {
			a.form = nil
			return false
		}
		if act.Save {
			a.saveForm()
		}
		return false
	}
	// While the filter prompt has the keyboard every key is text, so
	// the shortcuts below must not fire. Enter and Esc are the prompt's
	// own, and the list handles them.
	if a.list.FilterEditing() {
		return a.list.HandleKey(k)
	}
	switch {
	case k.Kind == tui.KeyEnter,
		k.Kind == tui.KeyRune && k.Rune == 'l':
		if t := a.list.SelectedTicket(); t != nil {
			a.detail = NewDetailView(t)
		}
		return false
	case k.Kind == tui.KeyRune && k.Rune == 's':
		if t := a.list.SelectedTicket(); t != nil {
			if a.list.acts.SetStatus == nil {
				a.list.say("status changes are not wired in this host")
				return false
			}
			a.picker = NewStatusPicker(t)
		}
		return false
	case k.Kind == tui.KeyRune && k.Rune == 'n':
		if a.list.acts.Create == nil {
			a.list.say("creating is not wired in this host")
			return false
		}
		a.form = NewCreateForm()
		return false
	case k.Kind == tui.KeyRune && k.Rune == 'e':
		if t := a.list.SelectedTicket(); t != nil {
			if a.list.acts.Edit == nil {
				a.list.say("editing is not wired in this host")
				return false
			}
			a.form = NewEditForm(t)
		}
		return false
	}
	return a.list.HandleKey(k)
}

// saveForm performs the form's write and decides what the form does
// next. Success closes it and reports on the list footer. A stale
// revision is the loop this view exists to close correctly: reload,
// re-arm the form with the revision now on disk, keep the typed text,
// and say so. The write did not happen, and the next Ctrl+S is the
// person's deliberate decision.
func (a *App) saveForm() {
	title, desc := a.form.Values()
	if a.form.ref == "" {
		id, err := a.list.acts.Create(title, desc)
		if err != nil {
			a.form.SetError(err.Error())
			return
		}
		a.form = nil
		a.list.Reload()
		a.list.say("created " + a.list.shortOr(id))
		return
	}
	err := a.list.acts.Edit(a.form.ref, a.form.revision, title, desc)
	switch {
	case err == nil:
		ref := a.form.ref
		a.form = nil
		a.list.Reload()
		a.list.say("saved " + a.list.shortOr(ref))
	case ticket.CodeOf(err) == ticket.CodeStaleRevision:
		a.list.Reload()
		if t := a.list.TicketByID(a.form.ref); t != nil {
			a.form.Refresh(t.Revision)
		}
		a.form.SetError("changed by another writer; your text is kept, Ctrl+S again to replace their version")
	default:
		a.form.SetError(err.Error())
	}
}

// Render draws whichever view is on top.
func (a *App) Render(cols, rows int) []string {
	if a.detail != nil {
		return a.detail.Render(cols, rows)
	}
	if a.picker != nil {
		return a.picker.Render(cols, rows)
	}
	if a.form != nil {
		return a.form.Render(cols, rows)
	}
	return a.list.Render(cols, rows)
}
