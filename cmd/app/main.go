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
WITH flights_within_5 AS (
  SELECT DISTINCT hex_code
  FROM live
  WHERE
    timestamp >= NOW() - INTERVAL '30 minutes'
    AND distance < 5
),
flight_data AS (
  SELECT *
  FROM live
  WHERE
    timestamp >= NOW() - INTERVAL '30 minutes'
    AND hex_code IN (SELECT hex_code FROM flights_within_5)
),
min_distance_data AS (
  SELECT
    md.hex_code,
    md.min_distance,
    MIN(fd.timestamp) AS min_distance_timestamp
  FROM (
    SELECT
      hex_code,
      MIN(distance) AS min_distance
    FROM
      flight_data
    GROUP BY
      hex_code
  ) md
  JOIN flight_data fd
    ON md.hex_code = fd.hex_code AND fd.distance = md.min_distance
  GROUP BY
    md.hex_code,
    md.min_distance
),
flight_with_movement AS (
  SELECT
    fd.*,
    LAG(distance) OVER (PARTITION BY hex_code ORDER BY timestamp) AS previous_distance
  FROM flight_data fd
),
latest_positions AS (
  SELECT *
  FROM (
    SELECT
      *,
      ROW_NUMBER() OVER (PARTITION BY hex_code ORDER BY timestamp DESC) AS rn
    FROM flight_with_movement
  ) sub
  WHERE rn = 1
),
flight_status AS (
  SELECT
    lp.*,
    md.min_distance,
    md.min_distance_timestamp,
    EXTRACT(EPOCH FROM NOW() - md.min_distance_timestamp) / 60 AS mins_ago,
    CASE
      WHEN lp.distance = md.min_distance THEN 'At Closest Point'
      WHEN lp.previous_distance IS NOT NULL AND lp.distance > md.min_distance AND lp.distance > lp.previous_distance THEN
        'Passed Over ' || ROUND(EXTRACT(EPOCH FROM NOW() - md.min_distance_timestamp) / 60)::INT || ' mins ago'
      WHEN lp.previous_distance IS NOT NULL AND lp.distance < lp.previous_distance THEN 'Approaching'
      ELSE 'Unknown'
    END AS status
  FROM latest_positions lp
  JOIN min_distance_data md ON lp.hex_code = md.hex_code
)
SELECT
  flight_number,
  registration,
  aircraft_type,
  operator,
  status,
  distance,
  TO_CHAR(timestamp, 'HH24:MI:SS') AS time
FROM flight_status
WHERE status != 'Unknown'
ORDER BY
  CASE
    WHEN status LIKE 'Approaching%' THEN 1
    WHEN status = 'At Closest Point' THEN 2
    WHEN status LIKE 'Passed Over%' THEN 3
    ELSE 4
  END,
  -- For 'Approaching' and 'At Closest Point', order by timestamp ascending
  CASE
    WHEN status LIKE 'Approaching%' OR status = 'At Closest Point' THEN EXTRACT(EPOCH FROM timestamp)
    ELSE NULL
  END ASC,
  -- For 'Passed Over' flights, order by mins_ago ascending
  CASE
    WHEN status LIKE 'Passed Over%' THEN mins_ago
    ELSE NULL
  END ASC;

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
