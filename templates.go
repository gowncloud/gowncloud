package main

import (
	"html/template"
	"log"
	"net/http"
	"strings"
)

var (
	templateMap = template.FuncMap{
		"Upper": func(s string) string {
			return strings.ToUpper(s)
		},
	}
	templates *template.Template
)

// Parse all of the bindata templates
func init() {
	var err error
	htmlData, err := Asset("index.html")
	if err != nil {
		log.Panicf("Unable to load templates , err=%s", err)
		return
	}
	templates, err = template.New("index.html").Parse(string(htmlData))
	if err != nil {
		log.Panicf("Unable to parse templates , err=%s", err)
		return
	}
}

// Render a template given a model
func renderTemplate(w http.ResponseWriter, tmpl string, p interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
