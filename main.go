package main

import (
	"image/color"
	"log"
	"sync"
	"time"

	owm "github.com/briandowns/openweathermap"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type WeatherTracker struct {
	lastUpdated  time.Time
	lastForecast string
}

func (w *WeatherTracker) queryWeather() (string, error) {
	oc, err := owm.NewOneCall("C", "EN", "1a0abbdb35839424e7bd8acbe577f92a", []string{})
	if err != nil {
		return "", err
	}

	err = oc.OneCallByCoordinates(&owm.Coordinates{
		Latitude:  40.69888,
		Longitude: -73.992659,
	})
	if err != nil {
		return "", err
	}

	var maxPrecipitation float64
	for _, point := range oc.Minutely {
		if point.Precipitation > maxPrecipitation {
			maxPrecipitation = point.Precipitation
		}
	}

	isSnowing := oc.Hourly[0].Snow.OneH > oc.Hourly[0].Rain.OneH ||
		oc.Hourly[1].Snow.OneH > oc.Hourly[1].Rain.OneH
	if maxPrecipitation < 0.5 {
		return "", nil
	}
	if isSnowing {
		return "snow", nil
	} else if maxPrecipitation > 8 {
		return "storm", nil
	} else if maxPrecipitation > 2.5 {
		return "rain", nil
	} else {
		return "drizzle", nil
	}
}

func (w *WeatherTracker) getWeatherEffect() string {
	if time.Since(w.lastUpdated) < time.Minute*15 {
		return w.lastForecast
	}

	result, err := w.queryWeather()
	if err != nil {
		log.Println("error querying weather:", err)
		return ""
	}

	w.lastUpdated = time.Now()
	w.lastForecast = result

	log.Println("got weather:", result)

	return result
}

type LineApp struct {
	ID          string
	Bullet      [][]color.Color
	TrainFilter func(tripData *TripData) bool
	Direction   Direction
	Station     string
	fetcherOnce sync.Once
	fetcher     func() (*SubwayData, error)
	lastData    *SubwayData
	lastDataAt  time.Time
}

func matchingLines(minAway time.Duration, lines ...string) func(tripData *TripData) bool {
	lineMatcher := make(map[string]bool)
	for _, line := range lines {
		lineMatcher[line] = true
	}

	return func(tripData *TripData) bool {
		if lineMatcher[tripData.RouteID] {
			arrival := time.Unix(int64(tripData.EstimatedCurrentStopArrivalTime), 0)
			if time.Until(arrival) < minAway {
				return false
			}

			return true
		}

		return false
	}
}

var lines = []*LineApp{
	{
		ID:          "r_dekalb_north",
		Bullet:      rBullet,
		TrainFilter: matchingLines(2*time.Minute, "R"),
		Direction:   DirectionNorth,
		Station:     "R30",
	},
	{
		ID:          "45_nevins_north",
		Bullet:      fourFiveBullet,
		TrainFilter: matchingLines(4*time.Minute, "4", "5"),
		Direction:   DirectionNorth,
		Station:     "234",
	},
}

const dataCacheDuration = time.Second * 60

func (l *LineApp) fetch() (*SubwayData, error) {
	l.fetcherOnce.Do(func() {
		l.fetcher = DatafetcherFor(l.TrainFilter, l.Direction, l.Station)
	})

	if time.Since(l.lastDataAt) < dataCacheDuration {
		return l.lastData, nil
	}

	data, err := l.fetcher()
	if err == nil {
		l.lastData = data
		l.lastDataAt = time.Now()
		return data, nil
	}

	return nil, err
}

func (l *LineApp) topic(awtrixPrefix string) string {
	return awtrixPrefix + "/custom/" + l.ID
}

func main() {
	// configure these to match your setup
	const (
		mqttBroker   = "tcp://192.168.1.198:1883" // your MQTT broker
		mqttClientID = "subway-time"              // unique client ID
		awtrixPrefix = "awtrix_420508"            // MQTT prefix configured in AWTRIX
		updateEvery  = 15 * time.Second           // how often to refresh
	)

	opts := mqtt.NewClientOptions().
		AddBroker(mqttBroker).
		SetClientID(mqttClientID).
		SetUsername("1lann").
		SetPassword("AEO5ADG2n2Zf0UXZmFKCoQnGv99o-Bt3p5LcUlNVUzGB1y0taYddU3VaOr1ZXmKGs1T9wReOwE8RzKsMV2WR7P4PI17w91TjnEnh23LNq1-YkSm7-qLLNy-YiO593GMYZW_4xOD6YYHXLS1ptACmdYWvf2KWsGZJl7oakNJzgIGgGQCItAemdog")

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT connect error: %v", token.Error())
	}
	log.Println("Connected to MQTT broker:", mqttBroker)

	ticker := time.NewTicker(updateEvery)
	defer ticker.Stop()

	var wt WeatherTracker

	for ; true; <-ticker.C {
		for _, app := range lines {
			func() {
				data, err := app.fetch()
				if err != nil {
					log.Printf("fetch error: %v", err)
					return
				}

				img := SubwayDataToImage(data, app.Bullet)

				var text string
				if len(data.NextTrains) == 0 {
					text = "N/A..."
				}

				payload, err := buildAwtrixImagePayload(img, text, wt.getWeatherEffect())
				if err != nil {
					log.Printf("payload build error: %v", err)
					return
				}

				token := client.Publish(app.topic(awtrixPrefix), 0, false, payload)
				token.Wait()
				if err := token.Error(); err != nil {
					log.Printf("MQTT publish error: %v", err)
				}
			}()
		}
	}
}
