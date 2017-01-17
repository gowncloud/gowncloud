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
	if templates, err = templates.ParseFiles("index.html"); err != nil {
		log.Panicf("Unable to parse templates , err=%s", err)
	}
}

// Render a template given a model
func renderTemplate(w http.ResponseWriter, tmpl string, p interface{}) {
	err := templates.ExecuteTemplate(w, tmpl, p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
