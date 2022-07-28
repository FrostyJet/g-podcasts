
-- +migrate Up
CREATE TABLE IF NOT EXISTS podcasts
(
    id SERIAL NOT NULL PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    poster TEXT,
    date_created TIMESTAMPTZ NOT NULL,
    date_updated TIMESTAMPTZ
);


-- +migrate Down
DROP TABLE IF EXISTS podcasts CASCADE;