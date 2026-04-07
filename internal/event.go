package internal

import (
	"bionhart.com/elapsed/internal/db"
	"fmt"
	"log"
	"maps"
	"math"
	"slices"
	"time"
)

type Event struct {
	Id                    int
	Title                 string
	Frequency             string
	HowLongTimeUnixMillis int64
	Occurrences           []Occurrence
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

var (
	localTZ, _ = time.LoadLocation("America/Los_Angeles")

	freqSpecs = map[string]FreqSpec{
		"daily": FreqSpec{
			Name:                "daily",
			GoodThreshold:       20,
			BorderlineThreshold: 28,
			ExpiredThreshold:    36,
		},
		"every-other-day": FreqSpec{
			Name:                "every-other-day",
			GoodThreshold:       48,
			BorderlineThreshold: 60,
			ExpiredThreshold:    72,
		},
		"weekly": FreqSpec{
			Name:                "weekly",
			GoodThreshold:       168,
			BorderlineThreshold: 192,
			ExpiredThreshold:    214,
		},
		"monthly": FreqSpec{
			Name:                "monthly",
			GoodThreshold:       744,
			BorderlineThreshold: 800,
			ExpiredThreshold:    900,
		},
		"quarterly": FreqSpec{
			Name:                "quarterly",
			GoodThreshold:       2160,
			BorderlineThreshold: 2400,
			ExpiredThreshold:    2700,
		},
		"bi-annually": FreqSpec{
			Name:                "bi-annually",
			GoodThreshold:       4416,
			BorderlineThreshold: 4600,
			ExpiredThreshold:    4800,
		},
		"annually": FreqSpec{
			Name:                "annually",
			GoodThreshold:       8760,
			BorderlineThreshold: 9500,
			ExpiredThreshold:    1100,
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
	case "weekly", "monthly", "quarterly", "bi-annually", "annually":
		return fmt.Sprintf("%d days ago", int(math.Round(d.Hours()/24)))
	default:
		return fmt.Sprintf("%d hours ago", int(math.Round(d.Hours())))
	}
}

func (e Event) TimelinessIndicator() string {
	switch e.TimelinessStatus() {
	case "just-done":
		return "⭐"
	case "good":
		return "🟩"
	case "borderline":
		return "🟨"
	case "expired":
		return "🟧"
	case "abandoned":
		return "🟥"
	default:
		log.Printf("Unknown TimelinessStatus: %s", e.TimelinessStatus())
		return ""
	}
}

func (e Event) TimelinessStatus() string {
	d := time.Since(time.UnixMilli(e.HowLongTimeUnixMillis))

	if d.Minutes() < 30 {
		return "just-done"
	}

	spec, ok := freqSpecs[e.Frequency]

	if !ok {
		log.Printf("Unknown frequency: %s", e.Frequency)
		return ""
	}

	if d.Hours() < spec.GoodThreshold {
		return "good"
	} else if d.Hours() < spec.BorderlineThreshold {
		return "borderline"
	} else if d.Hours() < spec.ExpiredThreshold {
		return "expired"
	} else {
		return "abandoned"
	}
}

func GetEvents() map[string]*[]Event {
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
		if err := eventRows.Scan(&e.Id, &e.Title, &e.Frequency); err != nil {
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
	slices.SortFunc(allEvents, func(a, b *Event) int { return a.Id - b.Id })

	result := make(map[string]*[]Event)

	for _, event := range allEvents {
		freqEvents, ok := result[event.Frequency]
		if !ok {
			newFreqEvents := make([]Event, 0)
			freqEvents = &newFreqEvents
		}

		newSlice := append(*freqEvents, *event)
		freqEvents = &newSlice
		result[event.Frequency] = freqEvents
	}

	return result
}
