package main

import (
	"context"
	"html/template"
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

type Flight struct {
	FlightNumber string
	Registration string
	AircraftType string
	Operator     string
	Status       string
	Distance     float64
	Time         string
}

const flightStatusQuery = `
-- [Insert the SQL query here as shown above]
`

func getFlightStatuses(w http.ResponseWriter, r *http.Request) {
	rows, err := pool.Query(context.Background(), flightStatusQuery)
	if err != nil {
		http.Error(w, "Database query error", http.StatusInternalServerError)
		log.Println("Query error:", err)
		return
	}
	defer rows.Close()

	var flights []Flight

	for rows.Next() {
		var f Flight
		err := rows.Scan(
			&f.FlightNumber,
			&f.Registration,
			&f.AircraftType,
			&f.Operator,
			&f.Status,
			&f.Distance,
			&f.Time,
		)
		if err != nil {
			http.Error(w, "Error scanning row", http.StatusInternalServerError)
			log.Println("Row scan error:", err)
			return
		}
		flights = append(flights, f)
	}

	if err = rows.Err(); err != nil {
		http.Error(w, "Row iteration error", http.StatusInternalServerError)
		log.Println("Row iteration error:", err)
		return
	}

	tmpl := template.Must(template.New("flightTable").Parse(htmlTemplate))
	err = tmpl.Execute(w, flights)
	if err != nil {
		http.Error(w, "Template execution error", http.StatusInternalServerError)
		log.Println("Template execution error:", err)
		return
	}
}

const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>Flight Statuses</title>
    <style>
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            border: 1px solid #aaa;
            padding: 8px;
            text-align: left;
        }
        th {
            background-color: #ddd;
        }
    </style>
</head>
<body>
    <h1>Flight Statuses</h1>
    <table>
        <thead>
            <tr>
                <th>Flight Number</th>
                <th>Registration</th>
                <th>Aircraft Type</th>
                <th>Operator</th>
                <th>Status</th>
                <th>Distance</th>
                <th>Time</th>
            </tr>
        </thead>
        <tbody>
            {{range .}}
            <tr>
                <td>{{.FlightNumber}}</td>
                <td>{{.Registration}}</td>
                <td>{{.AircraftType}}</td>
                <td>{{.Operator}}</td>
                <td>{{.Status}}</td>
                <td>{{printf "%.2f" .Distance}}</td>
                <td>{{.Time}}</td>
            </tr>
            {{end}}
        </tbody>
    </table>
</body>
</html>
`

func main() {
	connectDatabase()
	defer pool.Close()

	r := chi.NewRouter()
	r.Get("/", getFlightStatuses)

	log.Println("Server starting on port", port)
	http.ListenAndServe(":"+port, r)
}
