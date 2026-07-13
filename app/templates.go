package app

import (
	"html/template"
	"net/http"

	"github.com/monjuik/go-girard/assets"
)

type Templates struct {
	parsed *template.Template
}

type PageData struct {
	Title      string
	ActiveMenu string
	Data       any
}

func NewTemplates() (*Templates, error) {
	parsed, err := template.ParseFS(assets.Templates, "templates/*html")
	if err != nil {
		return nil, err
	}
	return &Templates{
		parsed: parsed,
	}, nil
}

// Render writes directly to the response on purpose: to keep rendering path simple.
// Template execution error may leave client with "200 OK" response, this is fine
func (t *Templates) Render(w http.ResponseWriter, name string, data PageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := t.parsed.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
		return
	}
}
