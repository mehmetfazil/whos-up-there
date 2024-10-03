package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"
	"github.com/mehmetfazil/whos-up-there/types"
)

var Url string
var Db string

const Radius = 10.0

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	lat := os.Getenv("LAT")
	long := os.Getenv("LONG")
	Db = os.Getenv("DB")

	Url = fmt.Sprintf("https://api.airplanes.live/v2/point/%s/%s/%f", lat, long, Radius)

}

func main() {

	connStr := Db

	// Connect to the database
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			err := fetchAndStore(conn)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		}
	}
}

func fetchAndStore(conn *pgx.Conn) error {

	// Fetch the data
	resp, err := http.Get(Url) // Replace with actual API URL
	if err != nil {
		return fmt.Errorf("error fetching data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var apiResp types.ApiResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&apiResp); err != nil {
		return fmt.Errorf("error decoding JSON: %v", err)
	}

	// Get current timestamp
	timestamp := time.Now()

	// Insert each aircraft into the database
	for _, ac := range apiResp.Ac {
		err := insertAircraft(conn, ac, timestamp)
		if err != nil {
			fmt.Printf("Error inserting aircraft %s: %v\n", ac.HexCode, err)
			// Continue with next aircraft
		}
	}

	return nil
}

func insertAircraft(conn *pgx.Conn, ac types.Aircraft, timestamp time.Time) error {
	// Prepare the SQL statement
	sql := `INSERT INTO live (
        timestamp, hex_code, message_type, flight_number, registration, aircraft_type, description, operator, manufacture_year,
        barometric_altitude, is_ground, ground_speed, wind_direction, wind_speed, track_angle, latitude, longitude,
        position_age, distance, direction
    ) VALUES (
        $1, $2, $3, $4, $5, $6, $7, $8, $9,
        $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
    )`

	_, err := conn.Exec(context.Background(), sql,
		timestamp, ac.HexCode, ac.MessageType, ac.FlightNumber, ac.Registration, ac.AircraftType, ac.Description, ac.Operator, ac.ManufactureYear,
		ac.BarometricAltitude.Altitude, ac.BarometricAltitude.IsGround, ac.GroundSpeed,
		ac.WindDirection, ac.WindSpeed,
		ac.TrackAngle, ac.Latitude, ac.Longitude,
		ac.PositionAge, ac.Distance, ac.Direction,
	)

	return err
}
