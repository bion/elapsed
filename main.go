package main

import (
	"bionhart.com/elapsed/internal/db"
	"fmt"
	"html/template"
	"log"
	"maps"
	"math"
	"net/http"
	"slices"
	"time"
)

type Event struct {
	Id                    int
	Title                 string
	Frequency             string
	HowLongTimeUnixMillis int64
}

type Occurrence struct {
	EventId        int
	TimeUnixMillis int64
}

type Page struct {
	Events map[string]*[]Event
}

type FreqSpec struct {
	GoodThreshold       float64
	BorderlineThreshold float64
	ExpiredThreshold    float64
}

var (
	localTZ, _ = time.LoadLocation("America/Los_Angeles")

	freqSpecs = map[string]FreqSpec{
		"daily": FreqSpec{
			GoodThreshold:       24,
			BorderlineThreshold: 36,
			ExpiredThreshold:    48,
		},
		"every-other-day": FreqSpec{
			GoodThreshold:       48,
			BorderlineThreshold: 60,
			ExpiredThreshold:    72,
		},
		"weekly": FreqSpec{
			GoodThreshold:       168,
			BorderlineThreshold: 192,
			ExpiredThreshold:    214,
		},
		"monthly": FreqSpec{
			GoodThreshold:       744,
			BorderlineThreshold: 800,
			ExpiredThreshold:    900,
		},
		"quarterly": FreqSpec{
			GoodThreshold:       2160,
			BorderlineThreshold: 2400,
			ExpiredThreshold:    2700,
		},
		"bi-annually": FreqSpec{
			GoodThreshold:       4416,
			BorderlineThreshold: 4600,
			ExpiredThreshold:    4800,
		},
		"annually": FreqSpec{
			GoodThreshold:       8760,
			BorderlineThreshold: 9500,
			ExpiredThreshold:    1100,
		},
	}
)

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

func getEvents() map[string]*[]Event {
	eventRows, err := db.Db.Query("SELECT * FROM events")
	if err != nil {
		log.Fatal(err)
	}
	defer eventRows.Close()

	events := make(map[int]*Event)
	for eventRows.Next() {
		e := &Event{}
		if err := eventRows.Scan(&e.Id, &e.Title, &e.Frequency); err != nil {
			log.Fatal(err)
		}
		events[e.Id] = e
	}

	occRows, err := db.Db.Query(`
SELECT event_id, time_unix_millis FROM (
  SELECT event_id, time_unix_millis, ROW_NUMBER()
  OVER (PARTITION BY event_id ORDER BY time_unix_millis DESC) AS rn
  FROM occurrences
)
WHERE rn = 1;
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

		e.HowLongTimeUnixMillis = o.TimeUnixMillis
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

func newOne(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("web/newOne.html")
	if err != nil {
		log.Fatal(err)
	}

	t.Execute(w, nil)
}

func create(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	frequency := r.FormValue("frequency")

	log.Printf("Create Event: title=%s, frequency=%s", title, frequency)

	_, err := db.Db.Exec("INSERT INTO events (title, frequency) VALUES (?, ?)", title, frequency)
	if err != nil {
		log.Fatal(err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func occur(w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")
	timeUnixMillis := time.Now().UnixMilli()

	_, err := db.Db.Exec("INSERT INTO occurrences VALUES (?, ?)", id, timeUnixMillis)
	if err != nil {
		log.Fatal(err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func filterEvents(events *[]*Event, frequency string) *[]*Event {
	result := make([]*Event, 0)

	for _, event := range *events {
		if event.Frequency == frequency {
			result = append(result, event)
		}
	}

	return &result
}

func index(w http.ResponseWriter, r *http.Request) {
	events := getEvents()

	p := &Page{
		Events: events,
	}

	t, err := template.ParseFiles("web/index.html")
	if err != nil {
		log.Fatal(err)
	}

	err = t.Execute(w, p)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	db.Init()
	
	http.HandleFunc("/new", newOne)
	http.HandleFunc("/create", create)
	http.HandleFunc("/occur", occur)
	http.HandleFunc("/", index)

	log.Print("starting server on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

	defer db.Teardown()
}
