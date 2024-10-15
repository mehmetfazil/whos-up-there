package data

import (
	"context"
	"log"

	"github.com/mehmetfazil/whos-up-there/db"
)

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
  CASE
    WHEN status LIKE 'Approaching%' OR status = 'At Closest Point' THEN EXTRACT(EPOCH FROM timestamp)
    ELSE NULL
  END ASC,
  CASE
    WHEN status LIKE 'Passed Over%' THEN mins_ago
    ELSE NULL
  END ASC;
`

func GetFlightStatus(ctx context.Context) ([]Flight, error) {
	rows, err := db.Pool.Query(ctx, flightStatusQuery)
	if err != nil {
		log.Println("Query error:", err)
		return nil, err
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
