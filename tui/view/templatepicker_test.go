package view

import (
	"strings"
	"testing"

	"github.com/terva-sh/git-ticket/tui"
)

// These tests hold the create flow to plan 4.2: the template picker
// fronts the form only when the store defines templates, the blank
// path stays one keystroke deep, and the chosen template rides to the
// store with the save.

func newTemplateApp(t *testing.T, choices []TemplateChoice, create func(string, string, string) (string, error)) *App {
	t.Helper()
	a := NewApp(fixed(tktA), Actions{
		Create:    create,
		Templates: func() ([]TemplateChoice, error) { return choices, nil },
	})
	a.list.Reload()
	return a
}

func TestCreateWithNoTemplatesOpensTheBareForm(t *testing.T) {
	a := newTemplateApp(t, nil, func(string, string, string) (string, error) { return "", nil })
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'n'})
	if a.form == nil || a.tmpl != nil {
		t.Fatalf("n with no templates: form=%v picker=%v, want the bare form", a.form != nil, a.tmpl != nil)
	}
}

func TestCreateWithTemplatesOffersThePicker(t *testing.T) {
	choices := []TemplateChoice{{Name: "bug", Description: "Steps to reproduce:"}}
	a := newTemplateApp(t, choices, func(string, string, string) (string, error) { return "", nil })
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'n'})
	if a.tmpl == nil {
		t.Fatal("n with templates did not open the picker")
	}
	body := strings.Join(renderApp(a, 90, 10), "\n")
	if !strings.Contains(body, "a blank ticket") || !strings.Contains(body, "template: bug") {
		t.Fatalf("picker body:\n%s", body)
	}

	// The blank row is first, so Enter alone is the old one-keystroke
	// path to an empty form.
	a.HandleKey(tui.Key{Kind: tui.KeyEnter})
	if a.form == nil || a.form.template != "" {
		t.Fatalf("the blank row did not open a bare form")
	}
}

// TestTemplateChoiceSeedsTheFormAndTheSave: choosing a template
// prefills the description editor, names the template in the header,
// and passes the name through Create.
func TestTemplateChoiceSeedsTheFormAndTheSave(t *testing.T) {
	var gotTemplate, gotDesc string
	choices := []TemplateChoice{{Name: "bug", Description: "Steps to reproduce:"}}
	a := newTemplateApp(t, choices, func(_, desc, tmpl string) (string, error) {
		gotDesc, gotTemplate = desc, tmpl
		return "TKT-01NEWZZZZZZZZZZZZZZZZZZZZZ", nil
	})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'n'})
	a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: 'j'}) // move to the template row
	a.HandleKey(tui.Key{Kind: tui.KeyEnter})

	if a.form == nil || a.form.template != "bug" {
		t.Fatalf("the template did not reach the form")
	}
	body := strings.Join(renderApp(a, 90, 14), "\n")
	if !strings.Contains(body, "from template: bug") {
		t.Fatalf("the form header does not name the template:\n%s", body)
	}
	if !strings.Contains(body, "Steps to reproduce:") {
		t.Fatalf("the description editor is not prefilled:\n%s", body)
	}

	// Type a title and save: the template name rides along.
	for _, r := range "Clock skew" {
		a.HandleKey(tui.Key{Kind: tui.KeyRune, Rune: r})
	}
	a.HandleKey(tui.Key{Kind: tui.KeyCtrlS})
	if gotTemplate != "bug" {
		t.Fatalf("Create got template %q, want bug", gotTemplate)
	}
	if !strings.Contains(gotDesc, "Steps to reproduce:") {
		t.Fatalf("Create got description %q, want the prefilled skeleton", gotDesc)
	}
}
