package handlers

import (
	"html/template"
	"log"
	"net/http"

	"github.com/mehmetfazil/whos-up-there/data"
)

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	flights, err := data.GetFlightStatus(r.Context())
	if err != nil {
		http.Error(w, "Database query error", http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles("web/templates/flights.html"))
	err = tmpl.Execute(w, flights)
	if err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		log.Println("Template execution error:", err)
	}
}
