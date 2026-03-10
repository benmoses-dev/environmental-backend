CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;

CREATE TABLE IF NOT EXISTS locations (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS devices (
    id SERIAL PRIMARY KEY,
    identifier TEXT UNIQUE NOT NULL,
    location_id INT REFERENCES locations(id)
);

CREATE TABLE IF NOT EXISTS sensors (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TYPE reading_type_enum AS ENUM ('temperature','humidity','pressure');

CREATE TABLE IF NOT EXISTS reading_types (
    id SERIAL PRIMARY KEY,
    name reading_type_enum NOT NULL
);

CREATE TABLE IF NOT EXISTS device_sensor_readings (
    id SERIAL PRIMARY KEY,
    device_id INT REFERENCES devices(id),
    sensor_id INT REFERENCES sensors(id),
    readingtype_id INT REFERENCES reading_types(id),
    UNIQUE(device_id, readingtype_id)
);

CREATE TABLE IF NOT EXISTS sensor_data (
    time TIMESTAMPTZ NOT NULL,
    device_id INT NOT NULL REFERENCES devices(id),
    location_id INT NOT NULL REFERENCES locations(id),
    sensor_id INT NOT NULL REFERENCES sensors(id),
    readingtype_id INT NOT NULL REFERENCES reading_types(id),
    value DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('sensor_data','time', if_not_exists => TRUE);
