
-- +migrate Up
CREATE TABLE IF NOT EXISTS tracks
(
    id SERIAL NOT NULL PRIMARY KEY,
    podcast_id INTEGER NOT NULL REFERENCES podcasts(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    date_created TIMESTAMPTZ NOT NULL,
    date_updated TIMESTAMPTZ
);


-- +migrate Down
DROP TABLE IF EXISTS tracks CASCADE;