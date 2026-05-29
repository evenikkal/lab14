package main

import "time"

type Accident struct {
	ID          string    `json:"id"`
	Date        time.Time `json:"date"`
	Region      string    `json:"region"`
	Type        string    `json:"type"`
	Injured     int       `json:"injured"`
	Dead        int       `json:"dead"`
	CollectedAt time.Time `json:"collected_at"`
}
