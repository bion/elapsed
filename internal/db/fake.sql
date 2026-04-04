INSERT INTO events (title, frequency) VALUES ("bedroom pre filter", "monthly");
INSERT INTO events (title, frequency) VALUES ("bedroom main filter", "semi-annually");
INSERT INTO events (title, frequency) VALUES ("bedroom minisplit filter", "quarterly");

INSERT INTO occurrences (event_id, time_unix_millis) VALUES (1, CAST(unixepoch('now', 'subsec') * 1000 AS INTEGER));
INSERT INTO occurrences (event_id, time_unix_millis) VALUES (2, CAST(unixepoch('now', 'subsec') * 1000 AS INTEGER));
INSERT INTO occurrences (event_id, time_unix_millis) VALUES (2, CAST(unixepoch('now', 'subsec') * 1000 AS INTEGER));


SELECT * FROM (
  SELECT *, ROW_NUMBER()
  OVER (PARTITION BY event_id ORDER BY time_unix_millis DESC) AS rn
  FROM occurrences
)
WHERE rn = 1;
