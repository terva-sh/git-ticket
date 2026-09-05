package view

import "github.com/terva-sh/git-ticket/tui"

// App is the state machine over the views: the list is the floor, and
// the detail view stacks on top of it. One level of stack is a field,
// not a slice, and it stays a field until a third view earns the
// generalization.
type App struct {
	list   *ListView
	detail *DetailView
}

// NewApp opens the list over relist.
func NewApp(relist Lister) *App {
	return &App{list: NewListView(relist)}
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
	switch {
	case k.Kind == tui.KeyEnter,
		k.Kind == tui.KeyRune && k.Rune == 'l':
		if t := a.list.SelectedTicket(); t != nil {
			a.detail = NewDetailView(t)
		}
		return false
	}
	return a.list.HandleKey(k)
}

// Render draws whichever view is on top.
func (a *App) Render(cols, rows int) []string {
	if a.detail != nil {
		return a.detail.Render(cols, rows)
	}
	return a.list.Render(cols, rows)
}
