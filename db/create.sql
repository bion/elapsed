CREATE TABLE IF NOT EXISTS events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL UNIQUE,
    frequency TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS occurrences (
    event_id INTEGER REFERENCES events(id),
    time_unix_millis INTEGER NOT NULL
);
