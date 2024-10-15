package main

import (
	"embed"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/mehmetfazil/whos-up-there/db"
	"github.com/mehmetfazil/whos-up-there/handlers"
)

//go:embed web/dist/*
var content embed.FS

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	if err := db.Init(os.Getenv("DB")); err != nil {
		log.Fatalf("Unable to connect to the database: %v\n", err)
	}
	defer db.Close()

	http.Handle("/web/dist/", http.FileServer(http.FS(content)))

	http.HandleFunc("/", handlers.HomeHandler)
	http.HandleFunc("/api/flights", handlers.FlightsAPIHandler)

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("Port is not defined")
	}

	log.Printf("Starting server on :%s...\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
