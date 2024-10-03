package types

import (
	"encoding/json"
	"fmt"
)

type AltBaro struct {
	IsGround bool
	Altitude int
}

func (a *AltBaro) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s == "ground" {
			a.IsGround = true
			a.Altitude = 0
			return nil
		}
		return fmt.Errorf("unexpected string value for alt_baro: %s", s)
	}

	var i int
	if err := json.Unmarshal(data, &i); err == nil {
		a.IsGround = false
		a.Altitude = i
		return nil
	}
	return fmt.Errorf("unexpected value for alt_baro: %s", string(data))
}

type Aircraft struct {
	HexCode            string  `json:"hex"`
	MessageType        string  `json:"type"`
	FlightNumber       string  `json:"flight"`
	Registration       string  `json:"r"`
	AircraftType       string  `json:"t"`
	Description        string  `json:"desc"`
	Operator           string  `json:"ownOp"`
	ManufactureYear    string  `json:"year"`
	BarometricAltitude AltBaro `json:"alt_baro"`
	GroundSpeed        float64 `json:"gs"`
	WindDirection      int     `json:"wd"`
	WindSpeed          int     `json:"ws"`
	TrackAngle         float64 `json:"track"`
	Latitude           float64 `json:"lat"`
	Longitude          float64 `json:"lon"`
	PositionAge        float64 `json:"seen_pos"`
	Distance           float64 `json:"dst"`
	Direction          float64 `json:"dir"`
}

type ApiResponse struct {
	Ac    []Aircraft `json:"ac"`
	Msg   string     `json:"msg"`
	Now   int64      `json:"now"`
	Total int        `json:"total"`
	Ctime int64      `json:"ctime"`
	Ptime int        `json:"ptime"`
}
