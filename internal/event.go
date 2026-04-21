package internal

import (
	"bionhart.com/elapsed/internal/db"
	"fmt"
	"log"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

type Event struct {
	Id                    int
	Title                 string
	Frequency             string
	HowLongTimeUnixMillis int64
	Occurrences           []Occurrence
	Position              int
}

type Occurrence struct {
	EventId        int
	TimeUnixMillis int64
}

type FreqSpec struct {
	Name                string
	GoodThreshold       float64
	BorderlineThreshold float64
	ExpiredThreshold    float64
}

type TimelinessStatus string

const (
	StatusJustDone   TimelinessStatus = "just-done"
	StatusGood       TimelinessStatus = "good"
	StatusBorderline TimelinessStatus = "borderline"
	StatusExpired    TimelinessStatus = "expired"
	StatusAbandoned  TimelinessStatus = "abandoned"
)

var (
	localTZ, _ = time.LoadLocation("America/Los_Angeles")

	freqSpecs = map[string]FreqSpec{
		"daily": {
			Name:                "daily",
			GoodThreshold:       20,
			BorderlineThreshold: 28,
			ExpiredThreshold:    36,
		},
		"every-other-day": {
			Name:                "every-other-day",
			GoodThreshold:       48,
			BorderlineThreshold: 60,
			ExpiredThreshold:    72,
		},
		"2x-per-week": {
			Name:                "2x-per-week",
			GoodThreshold:       80,
			BorderlineThreshold: 110,
			ExpiredThreshold:    140,
		},
		"weekly": {
			Name:                "weekly",
			GoodThreshold:       168,
			BorderlineThreshold: 192,
			ExpiredThreshold:    214,
		},
		"monthly": {
			Name:                "monthly",
			GoodThreshold:       744,
			BorderlineThreshold: 800,
			ExpiredThreshold:    900,
		},
		"quarterly": {
			Name:                "quarterly",
			GoodThreshold:       2160,
			BorderlineThreshold: 2400,
			ExpiredThreshold:    2700,
		},
		"bi-annually": {
			Name:                "bi-annually",
			GoodThreshold:       4416,
			BorderlineThreshold: 4600,
			ExpiredThreshold:    4800,
		},
		"annually": {
			Name:                "annually",
			GoodThreshold:       8760,
			BorderlineThreshold: 9500,
			ExpiredThreshold:    11000,
		},
	}
)

func SortedFreqSpecs() []FreqSpec {
	specs := slices.Collect(maps.Values(freqSpecs))
	slices.SortFunc(specs, func(a, b FreqSpec) int { return int(a.GoodThreshold - b.GoodThreshold) })
	return specs
}

func (e Event) GetDayHistory() DayHistory {
	return DayHistory{Event: e}
}

func (e Event) LastDone() string {
	if e.HowLongTimeUnixMillis == 0 {
		return "no occurrences"
	}

	t := time.UnixMilli(e.HowLongTimeUnixMillis)

	return t.In(localTZ).Format("Mon Jan 2, 3:04 PM")
}

func (e Event) HowLong() string {
	if e.HowLongTimeUnixMillis == 0 {
		return "never"
	}

	d := time.Since(time.UnixMilli(e.HowLongTimeUnixMillis))

	if d.Minutes() < 30 {
		return "just now"
	}

	switch e.Frequency {
	case "daily", "every-other-day":
		return fmt.Sprintf("%d hours ago", int(math.Round(d.Hours())))
	case "2x-per-week", "weekly", "monthly", "quarterly", "bi-annually", "annually":
		return fmt.Sprintf("%d days ago", int(math.Round(d.Hours()/24)))
	default:
		return fmt.Sprintf("%d hours ago", int(math.Round(d.Hours())))
	}
}

func (e Event) TimelinessIndicator() string {
	switch e.GetTimelinessStatus() {
	case StatusJustDone:
		return "⭐"
	case StatusGood:
		return "🟩"
	case StatusBorderline:
		return "🟨"
	case StatusExpired:
		return "🟧"
	case StatusAbandoned:
		return "🟥"
	default:
		log.Printf("Unknown TimelinessStatus: %s", e.GetTimelinessStatus())
		return ""
	}
}

func (e Event) GetTimelinessStatus() TimelinessStatus {
	d := time.Since(time.UnixMilli(e.HowLongTimeUnixMillis))

	if d.Minutes() < 30 {
		return StatusJustDone
	}

	spec, ok := freqSpecs[e.Frequency]

	if !ok {
		log.Printf("Unknown frequency: %s", e.Frequency)
		return ""
	}

	if d.Hours() < spec.GoodThreshold {
		return StatusGood
	} else if d.Hours() < spec.BorderlineThreshold {
		return StatusBorderline
	} else if d.Hours() < spec.ExpiredThreshold {
		return StatusExpired
	} else {
		return StatusAbandoned
	}
}

func GetEvents() map[string][]Event {
	eventRows, err := db.Db.Query("SELECT * FROM events")
	if err != nil {
		log.Fatal(err)
	}
	defer eventRows.Close()

	events := make(map[int]*Event)
	for eventRows.Next() {
		occs := make([]Occurrence, 0)
		e := &Event{
			Occurrences: occs,
		}
		if err := eventRows.Scan(&e.Id, &e.Title, &e.Frequency, &e.Position); err != nil {
			log.Fatal(err)
		}
		events[e.Id] = e
	}

	occRows, err := db.Db.Query(`
SELECT event_id, time_unix_millis
FROM occurrences
ORDER BY event_id, time_unix_millis DESC;
`)

	if err != nil {
		log.Fatal(err)
	}
	defer occRows.Close()

	for occRows.Next() {
		var o Occurrence
		if err := occRows.Scan(&o.EventId, &o.TimeUnixMillis); err != nil {
			log.Fatal(err)
		}

		e, ok := events[o.EventId]
		if !ok {
			log.Fatalf("No event with ID %d", o.EventId)
		}

		if e.HowLongTimeUnixMillis == 0 {
			e.HowLongTimeUnixMillis = o.TimeUnixMillis
		}

		e.Occurrences = append(e.Occurrences, o)
	}

	allEvents := slices.Collect(maps.Values(events))
	result := make(map[string][]Event)

	for _, event := range allEvents {
		freqEvents, ok := result[event.Frequency]
		if !ok {
			freqEvents = make([]Event, 0)
		}

		result[event.Frequency] = append(freqEvents, *event)
	}

	for _, siblingEvents := range result {
		slices.SortFunc(siblingEvents, func(a, b Event) int { return a.Position - b.Position })
	}

	return result
}

func MoveEvent(id string, direction string) {
	idInt, err := strconv.Atoi(id)
	if err != nil {
		log.Fatalf("Failed to convert ID: %s", id)
	}

	allEvents := GetEvents()

	var index int
	var freq string
	for frequency, events := range allEvents {
		freq = frequency
		index = slices.IndexFunc(events, func(e Event) bool { return e.Id == idInt })
		if index != -1 {
			break
		}
	}

	events := allEvents[freq]

	if index <= 0 || (direction == "down" && index == len(events)-1) {
		return
	}

	event := events[index]

	var other Event
	if direction == "down" {
		other = events[index+1]
	} else {
		other = events[index-1]
	}

	builder := strings.Builder{}
	builder.WriteString("BEGIN TRANSACTION;")
	fmt.Fprintf(&builder, "UPDATE events SET position = %d WHERE id = %d;", other.Position, idInt)
	fmt.Fprintf(&builder, "UPDATE events SET position = %d WHERE id = %d;", event.Position, other.Id)
	builder.WriteString("END TRANSACTION;")

	db.Db.Exec(builder.String())
}
