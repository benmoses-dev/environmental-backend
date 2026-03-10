package services

import "time"

type SensorMessage struct {
	Time            time.Time
	Value           float64
	ReadingTypeName string
	Identifier      string
}

type Device struct {
	ID         int
	Identifier string
	LocationID int
}

type Location struct {
	ID   int
	Name string
}

type Sensor struct {
	ID   int
	Name string
}

type ReadingType struct {
	ID   int
	Name string
}

type DeviceSensorReading struct {
	DeviceID      int
	SensorID      int
	ReadingTypeID int
}

type SensorData struct {
	Time          time.Time
	DeviceID      int
	LocationID    int
	SensorID      int
	ReadingTypeID int
	Value         float64
}
