package main

import (
	"image/color"
	"log"
	"os"
	"sync"
	"time"

	owm "github.com/briandowns/openweathermap"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func mustGetEnv(env string) string {
	if value := os.Getenv(env); value != "" {
		return value
	}

	log.Panicln("env is required but is not set:", env)
	panic("should not be reachable")
}

type WeatherTracker struct {
	lastUpdated  time.Time
	lastForecast string
}

func (w *WeatherTracker) queryWeather() (string, error) {
	oc, err := owm.NewOneCall("C", "EN", mustGetEnv("OPENWEATHERMAP_KEY"), []string{})
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
	IsVisible   func(t time.Time) bool
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

func isWeekday(t time.Time) bool {
	d := t.Weekday()
	return d >= time.Monday && d <= time.Friday
}

func minsOfDay(t time.Time) int {
	h, m, _ := t.Clock()
	return h*60 + m
}

func between(mins, startH, startM, endH, endM int) bool {
	return mins >= startH*60+startM && mins < endH*60+endM
}

var lines = []*LineApp{
	{
		ID:          "r_dekalb_north",
		Bullet:      rBullet,
		TrainFilter: matchingLines(2*time.Minute, "R"),
		Direction:   DirectionNorth,
		Station:     "R30",
		IsVisible: func(t time.Time) bool {
			// Weekdays 07:00–10:00
			return isWeekday(t) && between(minsOfDay(t), 7, 0, 10, 0)
		},
	},
	{
		ID:          "45_nevins_north",
		Bullet:      fourFiveBullet,
		TrainFilter: matchingLines(4*time.Minute, "4", "5"),
		Direction:   DirectionNorth,
		Station:     "234",
		IsVisible: func(t time.Time) bool {
			// Weekdays 07:00–10:00
			return isWeekday(t) && between(minsOfDay(t), 7, 0, 10, 0)
		},
	},
	{
		ID:          "q_dekalb_north",
		Bullet:      qBullet,
		TrainFilter: matchingLines(2*time.Minute, "Q"),
		Direction:   DirectionNorth,
		Station:     "R30",
		IsVisible: func(t time.Time) bool {
			// Every day 10:00–21:30
			return between(minsOfDay(t), 10, 0, 21, 30)
		},
	},
	{
		ID:          "b_dekalb_north",
		Bullet:      bBullet,
		TrainFilter: matchingLines(2*time.Minute, "B"),
		Direction:   DirectionNorth,
		Station:     "R30",
		IsVisible: func(t time.Time) bool {
			mins := minsOfDay(t)
			if isWeekday(t) {
				// Weekdays 10:00–14:00
				return between(mins, 10, 0, 14, 0)
			}
			// Weekends 10:00–21:30
			return between(mins, 10, 0, 21, 30)
		},
	},
	{
		ID:          "r_dekalb_south",
		Bullet:      rBullet,
		TrainFilter: matchingLines(2*time.Minute, "R"),
		Direction:   DirectionSouth,
		Station:     "R30",
		// Late nights the R stops serving DeKalb; the N takes over.
		IsVisible: lateNight,
	},
	{
		ID:          "n_dekalb_south",
		Bullet:      nBullet,
		TrainFilter: matchingLines(2*time.Minute, "N"),
		Direction:   DirectionSouth,
		Station:     "R30",
		// The N only stops at DeKalb late nights.
		IsVisible: lateNight,
	},
}

// lateNight: every night 21:30–02:00 (next day).
func lateNight(t time.Time) bool {
	mins := minsOfDay(t)
	return mins >= 21*60+30 || mins < 2*60
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
	var (
		mqttBroker   = mustGetEnv("MQTT_ENDPOINT")      // like "tcp://192.168.1.198:1883"
		mqttClientID = "subway-time"                    // unique client ID
		awtrixPrefix = mustGetEnv("MQTT_AWTRIX_PREFIX") // like "awtrix_420508"
		updateEvery  = 15 * time.Second                 // how often to refresh
	)

	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic("Could not load America/New_York")
	}

	opts := mqtt.NewClientOptions().
		AddBroker(mqttBroker).
		SetClientID(mqttClientID).
		SetUsername(mustGetEnv("MQTT_USERNAME")).
		SetPassword(mustGetEnv("MQTT_PASSWORD"))

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("MQTT connect error: %v", token.Error())
	}
	log.Println("Connected to MQTT broker:", mqttBroker)

	ticker := time.NewTicker(updateEvery)
	defer ticker.Stop()

	var wt WeatherTracker

	for ; true; <-ticker.C {
		now := time.Now().In(loc)
		for _, app := range lines {
			func() {
				if !app.IsVisible(now) {
					// Remove the custom app from AWTRIX
					token := client.Publish(app.topic(awtrixPrefix), 0, false, "")
					token.Wait()
					return
				}

				data, err := app.fetch()
				if err != nil {
					log.Printf("fetch error: %v", err)
					return
				}

				if len(data.NextTrains) == 0 {
					// No trains coming (e.g. line not running now): hide the app.
					token := client.Publish(app.topic(awtrixPrefix), 0, false, "")
					token.Wait()
					return
				}

				img := SubwayDataToImage(data, app.Bullet)
				payload, err := buildAwtrixImagePayload(img, "", wt.getWeatherEffect())
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
