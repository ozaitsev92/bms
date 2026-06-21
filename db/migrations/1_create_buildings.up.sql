CREATE TABLE IF NOT EXISTS building (
    id      SERIAL      PRIMARY KEY,
    name    VARCHAR(255) NOT NULL UNIQUE,
    address TEXT        NOT NULL
);