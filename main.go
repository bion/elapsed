package main

import (
	"bionhart.com/elapsed/internal"
	"bionhart.com/elapsed/internal/db"
	"html/template"
	"log"
	"net/http"
	"time"
)

type Page struct {
	Events map[string]*[]internal.Event
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

func index(w http.ResponseWriter, r *http.Request) {
	events := internal.GetEvents()

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
