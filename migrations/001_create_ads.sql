-- +goose Up
CREATE TABLE ads (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title varchar(200) NOT NULL,
    description varchar(4000) NOT NULL,
    price bigint NOT NULL CHECK (price >=0),
    category_id integer NOT NULL,
    city_id integer NOT NULL,
    lat double precision,
    lon double precision,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS ads;
