package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/mehmetfazil/whos-up-there/db"
	"github.com/mehmetfazil/whos-up-there/handlers"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	if err := db.Init(os.Getenv("DB")); err != nil {
		log.Fatalf("Unable to connect to the database: %v\n", err)
	}
	defer db.Close()

	http.HandleFunc("/", handlers.HomeHandler)

	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("PORT environment variable is not set")
	}
	log.Printf("Starting server on :%s...\n", port)
	log.Fatal(http.ListenAndServe(":9000", nil))

}
