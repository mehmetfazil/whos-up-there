// File: data/flights.go

package data

import (
	"context"
	"log"
	"time"

	"github.com/mehmetfazil/whos-up-there/db"
)

type Flight struct {
	HexCode              string    `json:"HexCode"`
	FlightNumber         string    `json:"FlightNumber"`
	LatestDistance       float64   `json:"LatestDistance"`
	LatestTimestamp      time.Time `json:"LatestTimestamp"`
	MinDistance          float64   `json:"MinDistance"`
	MinDistanceTimestamp time.Time `json:"MinDistanceTimestamp"`
}

const flightQuery = `
WITH latest_flights AS (
    SELECT DISTINCT ON (l.hex_code)
        l.hex_code,
        l.flight_number,
        l.distance AS latest_distance,
        l.timestamp AS latest_timestamp
    FROM
        live l
    WHERE
        l.distance < 5
        AND l.timestamp >= NOW() - INTERVAL '12 hours'
    ORDER BY
        l.hex_code, l.timestamp DESC
),
limited_latest_flights AS (
    SELECT *
    FROM latest_flights
    ORDER BY latest_timestamp DESC
    LIMIT 20
),
min_distance_data AS (
    SELECT DISTINCT ON (l.hex_code)
        l.hex_code,
        l.flight_number,
        l.distance AS min_distance,
        l.timestamp AS min_distance_timestamp
    FROM
        live l
    WHERE
        l.hex_code IN (SELECT hex_code FROM limited_latest_flights)
        AND l.timestamp >= NOW() - INTERVAL '12 hours'
    ORDER BY
        l.hex_code, l.distance ASC, l.timestamp ASC
)
SELECT
    llf.hex_code,
    llf.flight_number,
    llf.latest_distance,
    llf.latest_timestamp,
    md.min_distance,
    md.min_distance_timestamp
FROM
    limited_latest_flights llf
JOIN
    min_distance_data md ON llf.hex_code = md.hex_code
ORDER BY
    llf.latest_timestamp DESC;

`

func GetFlightData(ctx context.Context) ([]Flight, error) {
	rows, err := db.Pool.Query(ctx, flightQuery)
	if err != nil {
		log.Println("Query error:", err)
		return nil, err
	}
	defer rows.Close()

	var flights []Flight

	for rows.Next() {
		var f Flight
		err := rows.Scan(
			&f.HexCode,
			&f.FlightNumber,
			&f.LatestDistance,
			&f.LatestTimestamp,
			&f.MinDistance,
			&f.MinDistanceTimestamp,
		)
		if err != nil {
			log.Println("Row scan error:", err)
			return nil, err
		}
		flights = append(flights, f)
	}

	if err = rows.Err(); err != nil {
		log.Println("Row iteration error:", err)
		return nil, err
	}

	return flights, nil
}
