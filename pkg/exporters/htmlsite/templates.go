// Package htmlsite generates multi-page HTML documentation sites from graph data.
package htmlsite

import (
	"embed"
	"html/template"
)

//go:embed templates/*.html templates/*.css
var templateFS embed.FS

// Templates holds the parsed HTML templates.
type Templates struct {
	Index   *template.Template
	Service *template.Template
	CSS     string
}

// LoadTemplates loads and parses all embedded templates.
func LoadTemplates() (*Templates, error) {
	// Load CSS
	cssBytes, err := templateFS.ReadFile("templates/base.css")
	if err != nil {
		return nil, err
	}

	// Parse index template
	indexTmpl, err := template.New("index.html").Parse(mustReadTemplate("templates/index.html"))
	if err != nil {
		return nil, err
	}

	// Parse service template
	serviceTmpl, err := template.New("service.html").Parse(mustReadTemplate("templates/service.html"))
	if err != nil {
		return nil, err
	}

	return &Templates{
		Index:   indexTmpl,
		Service: serviceTmpl,
		CSS:     string(cssBytes),
	}, nil
}

// mustReadTemplate reads a template file or panics.
func mustReadTemplate(name string) string {
	data, err := templateFS.ReadFile(name)
	if err != nil {
		panic("failed to read template " + name + ": " + err.Error())
	}
	return string(data)
}
