package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
)

var pool *pgxpool.Pool
var connStr string
var port string

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	connStr = os.Getenv("DB")
	port = os.Getenv("PORT")

}

func connectDatabase() {
	var err error
	pool, err = pgxpool.Connect(context.Background(), connStr)
	if err != nil {
		panic(err)
	}
}

func getLatestFlight(w http.ResponseWriter, r *http.Request) {
	// Prepare the SQL query to get the latest row
	query := `
        SELECT hex_code
        FROM live
        ORDER BY timestamp DESC
        LIMIT 1
    `

	row := pool.QueryRow(context.Background(), query)

	var result string

	err := row.Scan(
		&result,
	)
	if err != nil {
		return
	}

	w.Write([]byte(result))
}

func main() {
	connectDatabase()
	defer pool.Close()

	r := chi.NewRouter()
	r.Get("/", getLatestFlight)

	http.ListenAndServe(":"+port, r)
}
