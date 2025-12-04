package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type TrainData struct {
	DelayedAmount time.Duration
	Realtime      bool
	ArrivalTime   time.Time
}

type Direction string

const (
	DirectionNorth Direction = "north"
	DirectionSouth Direction = "south"
)

type SubwayData struct {
	Direction  Direction
	NextTrains []*TrainData
}

type TripData struct {
	ID                               string  `json:"id"`
	RouteID                          string  `json:"route_id"`
	Direction                        string  `json:"direction"`
	PreviousStop                     string  `json:"previous_stop"`
	PreviousStopArrivalTime          float64 `json:"previous_stop_arrival_time"`
	UpcomingStop                     string  `json:"upcoming_stop"`
	UpcomingStopArrivalTime          float64 `json:"upcoming_stop_arrival_time"`
	EstimatedUpcomingStopArrivalTime float64 `json:"estimated_upcoming_stop_arrival_time"`
	CurrentStopArrivalTime           float64 `json:"current_stop_arrival_time"`
	EstimatedCurrentStopArrivalTime  float64 `json:"estimated_current_stop_arrival_time"`
	DestinationStop                  string  `json:"destination_stop"`
	DelayedTime                      float64 `json:"delayed_time"`
	ScheduleDiscrepancy              float64 `json:"schedule_discrepancy"`
	IsDelayed                        bool    `json:"is_delayed"`
	IsAssigned                       bool    `json:"is_assigned"`
	Timestamp                        float64 `json:"timestamp"`
}

type ResponseJSON struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	SecondaryName string `json:"secondary_name"`
	UpcomingTrips struct {
		North []*TripData `json:"north"`
		South []*TripData `json:"south"`
	} `json:"upcoming_trips"`
	Timestamp int `json:"timestamp"`
}

func DatafetcherFor(trainFilter func(trip *TripData) bool, direction Direction, station string) func() (*SubwayData, error) {
	return func() (*SubwayData, error) {
		url := fmt.Sprintf("https://api.subwaynow.app/stops/%s", station)

		// Request
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}

		// Required headers for SubwayNow API
		req.Header.Set("accept", "*/*")
		req.Header.Set("origin", "https://lite.subwaynow.app")
		req.Header.Set("referer", "https://lite.subwaynow.app")
		req.Header.Set("user-agent", "Go-http-client/1.1")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("API returned %d", resp.StatusCode)
		}

		// Decode JSON
		var api ResponseJSON
		if err := json.NewDecoder(resp.Body).Decode(&api); err != nil {
			return nil, err
		}

		var trips []*TripData
		switch direction {
		case DirectionNorth:
			trips = api.UpcomingTrips.North
		case DirectionSouth:
			trips = api.UpcomingTrips.South
		default:
			return nil, errors.New("invalid direction")
		}

		// Filter trips by lineName (RouteID prefix matches subway line)
		filtered := make([]*TripData, 0)
		for _, t := range trips {
			if trainFilter(t) { // exact match
				filtered = append(filtered, t)
			}
		}

		// Build SubwayData
		sd := &SubwayData{
			Direction:  direction,
			NextTrains: []*TrainData{},
		}

		// Convert trips → TrainData (limit 2 trains)
		for _, t := range filtered {
			// Use EstimatedCurrentStopArrivalTime as ArrivalTime (UNIX seconds)
			arrival := time.Unix(int64(t.EstimatedCurrentStopArrivalTime), 0)

			train := &TrainData{
				DelayedAmount: time.Duration(t.DelayedTime) * time.Second,
				Realtime:      t.IsAssigned,
				ArrivalTime:   arrival,
			}

			sd.NextTrains = append(sd.NextTrains, train)
		}

		return sd, nil
	}
}
