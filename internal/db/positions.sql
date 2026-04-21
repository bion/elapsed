ALTER TABLE events ADD COLUMN position INTEGER;

CREATE TEMP TABLE positions
AS SELECT id, ROW_NUMBER() OVER (PARTITION BY frequency ORDER BY id ASC)
AS new_pos FROM events;

UPDATE events
SET position = (SELECT new_pos FROM positions WHERE positions.id = events.id);
