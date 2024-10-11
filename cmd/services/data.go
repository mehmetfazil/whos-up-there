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

var (
	conn     *pgx.Conn
	apiUrl   string
	interval = 2 * time.Second
	radius   = 10.0
)

func main() {
	if err := loadConfig(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Ensure the database connection is closed when the program exits
	defer conn.Close(context.Background())

	// Start the data collection
	collectLiveFeed(interval, apiUrl, conn)
}

func collectLiveFeed(interval time.Duration, apiUrl string, conn *pgx.Conn) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if err := fetchAndStore(apiUrl, conn); err != nil {
			log.Printf("Error: %v", err)
		}
	}
}

func fetchAndStore(apiUrl string, conn *pgx.Conn) error {
	resp, err := http.Get(apiUrl)
	if err != nil {
		return fmt.Errorf("error fetching data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Decode only the "ac" and "now" fields from the JSON response
	// https://airplanes.live/rest-api-adsb-data-field-descriptions/
	var response struct {
		Ac  []types.Aircraft `json:"ac"`
		Now int64            `json:"now"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("error decoding JSON: %w", err)
	}

	timestamp := time.UnixMilli(response.Now)

	// Prepare a batch for inserting multiple records
	batch := &pgx.Batch{}
	for _, ac := range response.Ac {
		sql := `INSERT INTO live (
				timestamp, hex_code, message_type, flight_number, registration, aircraft_type, description, operator, manufacture_year,
				barometric_altitude, is_ground, ground_speed, wind_direction, wind_speed, track_angle, latitude, longitude,
				position_age, distance, direction
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9,
				$10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
			)`

		batch.Queue(sql,
			timestamp, ac.HexCode, ac.MessageType, ac.FlightNumber, ac.Registration, ac.AircraftType,
			ac.Description, ac.Operator, ac.ManufactureYear,
			ac.BarometricAltitude.Altitude, ac.BarometricAltitude.IsGround, ac.GroundSpeed,
			ac.WindDirection, ac.WindSpeed, ac.TrackAngle,
			ac.Latitude, ac.Longitude, ac.PositionAge,
			ac.Distance, ac.Direction,
		)
	}

	br := conn.SendBatch(context.Background(), batch)

	// Iterate over the batch results to handle any errors
	for i := 0; i < batch.Len(); i++ {
		_, err := br.Exec()
		if err != nil {
			log.Printf("Error inserting record %d: %v", i, err)
		}
	}

	if err := br.Close(); err != nil {
		return fmt.Errorf("error closing batch: %w", err)
	}

	return nil
}

func loadConfig() error {
	if err := godotenv.Load(); err != nil {
		return fmt.Errorf("error loading .env file: %w", err)
	}

	dbURL := os.Getenv("DB")
	lat := os.Getenv("LAT")
	long := os.Getenv("LONG")

	if dbURL == "" || lat == "" || long == "" {
		return fmt.Errorf("missing required environment variables")
	}

	apiUrl = fmt.Sprintf("https://api.airplanes.live/v2/point/%s/%s/%f", lat, long, radius)

	var err error
	conn, err = pgx.Connect(context.Background(), dbURL)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	return nil
}
