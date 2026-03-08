# IoT Sensor Architecture

## 1. Entities

- **Device**
  - Unique identifier
  - Assigned to a **Location**
  - Maps to multiple Sensors over time (many-to-many via device_sensors table)

- **Location**
  - Name, coordinates, or description
  - Optional metadata

- **Sensor**
  - Physical device (BME280, BME680, SCD41, etc.)
  - Can provide multiple reading types

- **ReadingType**
  - Measurement type / metric (temperature, humidity, CO2, VOC, etc.)
  - Essentially defines **what the sensor measures**

- **DeviceSensorReading**
  - Mapping table: device -> sensor -> reading type
  - Enforces **one sensor per reading type per device at a time**

- **SensorData**
  - Tall time-series table with:
    - timestamp
    - device_id
    - location_id
    - sensor_id
    - readingtype_id
    - value
  - Hypertable in TimescaleDB for efficient storage

---

## 2. Database Schema

```sql
-- Locations
CREATE TABLE locations (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

-- Devices
CREATE TABLE devices (
    id SERIAL PRIMARY KEY,
    identifier TEXT UNIQUE NOT NULL,
    location_id INT REFERENCES locations(id)
);

-- Sensors
CREATE TABLE sensors (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

-- Reading types
CREATE TYPE reading_type_enum AS ENUM ('temperature','humidity','pressure','voc','co2');

CREATE TABLE reading_types (
    id SERIAL PRIMARY KEY,
    name reading_type_enum NOT NULL
);

-- Device-Sensor-Reading mapping
CREATE TABLE device_sensor_readings (
    id SERIAL PRIMARY KEY,
    device_id INT REFERENCES devices(device_id),
    sensor_id INT REFERENCES sensors(sensor_id),
    readingtype_id INT REFERENCES reading_types(readingtype_id),
    UNIQUE(device_id, readingtype_id)
);

-- Sensor Data (tall table)
CREATE TABLE sensor_data (
    time TIMESTAMPTZ NOT NULL,
    device_id INT NOT NULL REFERENCES devices(device_id),
    location_id INT NOT NULL REFERENCES locations(location_id),
    sensor_id INT NOT NULL REFERENCES sensors(sensor_id),
    readingtype_id INT NOT NULL REFERENCES reading_types(readingtype_id),
    value DOUBLE PRECISION NOT NULL
);

SELECT create_hypertable('sensor_data','time');
