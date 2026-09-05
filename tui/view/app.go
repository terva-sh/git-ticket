package view

import (
	"fmt"

	"github.com/terva-sh/git-ticket/ticket"
	"github.com/terva-sh/git-ticket/tui"
)

// App is the state machine over the views: the list is the floor, and
// the other views stack on top of it. The detail views are a real
// stack since TKT-01M1RPZ0, because following links dives epic to
// child to dependency and Esc unwinds one level at a time, with the
// list as the floor. The pickers, the form, and the help page stay
// single fields: nothing stacks on top of them but more detail.
type App struct {
	list    *ListView
	details []*DetailView
	links   *LinkPicker
	picker  *StatusPicker
	tmpl    *TemplatePicker
	form    *FormView
	help    *HelpView
}

// top is the detail view under the keyboard, nil when the list has it.
func (a *App) top() *DetailView {
	if len(a.details) == 0 {
		return nil
	}
	return a.details[len(a.details)-1]
}

// NewApp opens the list over relist, writing through acts.
func NewApp(relist Lister, acts Actions) *App {
	return &App{list: NewListView(relist, acts)}
}

// HandleKey routes one key to whichever view is on top.
func (a *App) HandleKey(k tui.Key) (quit bool) {
	if a.links != nil {
		act := a.links.HandleKey(k)
		if act.Quit {
			return true
		}
		if act.Cancel {
			a.links = nil
			return false
		}
		if act.Open != nil {
			a.links = nil
			a.details = append(a.details, NewDetailView(act.Open))
		}
		return false
	}
	if top := a.top(); top != nil {
		// y and t are handled here rather than in the view, because
		// the copy and links actions live on Actions and the detail
		// view deliberately holds no write surface.
		if k.Kind == tui.KeyRune && k.Rune == 'y' {
			a.copyDetail()
			return false
		}
		if k.Kind == tui.KeyRune && k.Rune == 't' {
			a.openLinks()
			return false
		}
		back, quit := top.HandleKey(k)
		if quit {
			return true
		}
		if back {
			// One level at a time: the dive unwinds the way it went
			// down, and the list is the floor.
			a.details = a.details[:len(a.details)-1]
		}
		return false
	}
	if a.help != nil {
		back, quit := a.help.HandleKey(k)
		if quit {
			return true
		}
		if back {
			a.help = nil
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
	if a.tmpl != nil {
		act := a.tmpl.HandleKey(k)
		if act.Quit {
			return true
		}
		switch {
		case act.Blank:
			a.tmpl = nil
			a.form = NewCreateForm()
		case act.Choose != nil:
			a.tmpl = nil
			a.form = NewCreateFormFrom(*act.Choose)
		case act.Cancel:
			a.tmpl = nil
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
			a.details = append(a.details, NewDetailView(t))
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
		// The template picker fronts the form only when the store
		// defines templates, per plan 4.2: a store without them keeps
		// the one-keystroke path to a blank form.
		if a.list.acts.Templates != nil {
			choices, err := a.list.acts.Templates()
			if err != nil {
				a.list.say("templates failed: " + err.Error())
				return false
			}
			if len(choices) > 0 {
				a.tmpl = NewTemplatePicker(choices)
				return false
			}
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
	case k.Kind == tui.KeyRune && k.Rune == '?':
		a.help = &HelpView{}
		return false
	}
	return a.list.HandleKey(k)
}

// copyDetail puts the open ticket's stored body on the clipboard and
// flashes the outcome on the detail footer, naming the path it took
// and the byte count, so a copy that failed reads differently from a
// copy that landed.
func (a *App) copyDetail() {
	top := a.top()
	if a.list.acts.Copy == nil {
		top.say("copying is not wired in this host")
		return
	}
	via, n, err := a.list.acts.Copy(top.t.ID)
	if err != nil {
		top.say("copy failed: " + err.Error())
		return
	}
	top.say(fmt.Sprintf("copied %s (%d bytes) via %s", a.list.shortOr(top.t.ID), n, via))
}

// openLinks raises the link picker over the top detail view, or says
// why it cannot: nil-ness is the feature detection, same as every
// other action.
func (a *App) openLinks() {
	top := a.top()
	if a.list.acts.Links == nil {
		top.say("linked tickets are not wired in this host")
		return
	}
	links, err := a.list.acts.Links(top.t.ID)
	if err != nil {
		top.say("links failed: " + err.Error())
		return
	}
	a.links = NewLinkPicker(top.t, links)
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
		id, err := a.list.acts.Create(title, desc, a.form.template)
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
	if a.help != nil {
		return a.help.Render(cols, rows)
	}
	if a.links != nil {
		return a.links.Render(cols, rows)
	}
	if top := a.top(); top != nil {
		return top.Render(cols, rows)
	}
	if a.picker != nil {
		return a.picker.Render(cols, rows)
	}
	if a.tmpl != nil {
		return a.tmpl.Render(cols, rows)
	}
	if a.form != nil {
		return a.form.Render(cols, rows)
	}
	return a.list.Render(cols, rows)
}
