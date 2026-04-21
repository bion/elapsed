package main

import (
	"bionhart.com/elapsed/internal"
	"bionhart.com/elapsed/internal/db"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
)

type EventFormPageModel struct {
	Event      internal.Event
	Endpoint   string
	SubmitText string
}

func newOne(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("web/eventForm.html")
	if err != nil {
		log.Fatal(err)
	}

	t.Execute(w, &EventFormPageModel{
		Endpoint:   "/create",
		SubmitText: "create",
	})
}

func edit(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("web/eventForm.html")
	if err != nil {
		log.Fatal(err)
	}

	eventId := r.PathValue("id")
	event := &internal.Event{}
	err = db.Db.QueryRow("SELECT id, title, frequency FROM events WHERE id = ?", eventId).Scan(&event.Id, &event.Title, &event.Frequency)
	if err != nil {
		log.Fatal(err)
	}

	t.Execute(w, &EventFormPageModel{
		Event:      *event,
		Endpoint:   fmt.Sprintf("/events/%d", event.Id),
		SubmitText: "update",
	})
}

func update(w http.ResponseWriter, r *http.Request) {
	eventId := r.PathValue("id")
	title := r.FormValue("title")
	frequency := r.FormValue("frequency")
	_, err := db.Db.Exec("UPDATE events SET title = ?, frequency = ? WHERE id = ?", title, frequency, eventId)
	if err != nil {
		log.Fatal(err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func create(w http.ResponseWriter, r *http.Request) {
	title := r.FormValue("title")
	frequency := r.FormValue("frequency")

	log.Printf("Create Event: title=%s, frequency=%s", title, frequency)

	events, ok := internal.GetEvents()[frequency]
	var position int
	if ok {
		position = len(events) + 1
	} else {
		position = 1
	}

	_, err := db.Db.Exec("INSERT INTO events (title, frequency, position) VALUES (?, ?, ?)", title, frequency, position)
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

func up(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	internal.MoveEvent(id, "up")

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func down(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	internal.MoveEvent(id, "down")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type IndexPageModel struct {
	Events          map[string][]internal.Event
	SortedFreqSpecs []internal.FreqSpec
}

func index(w http.ResponseWriter, r *http.Request) {
	events := internal.GetEvents()

	p := &IndexPageModel{
		Events:          events,
		SortedFreqSpecs: internal.SortedFreqSpecs(),
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

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		log.Printf(
			"%s %s %s %s",
			r.Method,
			r.URL.Path,
			fmt.Sprintf("%dms", time.Since(start).Milliseconds()),
			r.RemoteAddr)
	})
}

func main() {
	db.Init()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /new", newOne)
	mux.HandleFunc("GET /events/{id}/edit", edit)
	mux.HandleFunc("POST /events/{id}", update)
	mux.HandleFunc("POST /events/{id}/up", up)
	mux.HandleFunc("POST /events/{id}/down", down)
	mux.HandleFunc("POST /create", create)
	mux.HandleFunc("POST /occur", occur)
	mux.HandleFunc("GET /", index)

	log.Print("starting server on port 8080")
	log.Fatal(http.ListenAndServe(":8080", logRequest(mux)))

	defer db.Teardown()
}
