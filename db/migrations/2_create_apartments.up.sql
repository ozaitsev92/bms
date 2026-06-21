CREATE TABLE IF NOT EXISTS apartment (
    id          SERIAL      PRIMARY KEY,
    building_id INTEGER     NOT NULL REFERENCES building(id) ON DELETE CASCADE,
    number      VARCHAR(50) NOT NULL,
    floor       INTEGER     NOT NULL,
    sq_meters   INTEGER     NOT NULL,

    CONSTRAINT uq_apartment_building_number UNIQUE (building_id, number)
);

CREATE INDEX idx_apartment_building_id ON apartment(building_id);