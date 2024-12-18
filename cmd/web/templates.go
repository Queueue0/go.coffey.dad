package main

import (
	"html/template"
	"io/fs"
	"net/url"
	"path/filepath"
	"time"

	"coffey.dad/internal/models"
	"coffey.dad/ui"
)

type templateData struct {
	CSRFToken       string
	CurrentPage     string
	Post            models.Post
	Posts           []models.Post
	Tags            []models.Tag
	Filter          string
	NewPost         bool
	Form            any
	Flash           string
	IsAuthenticated bool
	FileNames       []string
}

func humanDate(t time.Time) string {
	l, err := time.LoadLocation("America/New_York")

	var final string
	if err != nil {
		final = t.Format("Jan 02, 2006 at 3:04 PM")
	} else {
		final = t.In(l).Format("Jan 02, 2006 at 3:04 PM")
	}

	return final
}

var functions = template.FuncMap{
	"humanDate":  humanDate,
	"pathEscape": url.PathEscape,
}

func newTemplateCache() (map[string]*template.Template, error) {
	cache := map[string]*template.Template{}

	pages, err := fs.Glob(ui.Files, "html/pages/*.tmpl")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		patterns := []string{
			"html/base.tmpl",
			"html/partials/*.tmpl",
			page,
		}
		ts, err := template.New(name).Funcs(functions).ParseFS(ui.Files, patterns...)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}
