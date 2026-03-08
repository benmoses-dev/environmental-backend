package services

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresService struct {
	pool *pgxpool.Pool
}

func NewPostgresService(cfg *Config) *PostgresService {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	return &PostgresService{
		pool: pool,
	}
}

func (s *PostgresService) Close() {
	s.pool.Close()
}

func (s *PostgresService) GetDevice(ctx context.Context, identifier string) (int, int, error) {
	var deviceID int
	var locationID int
	err := s.pool.QueryRow(ctx,
		`SELECT id, location_id
		 FROM devices
		 WHERE identifier = $1`,
		identifier,
	).Scan(&deviceID, &locationID)
	return deviceID, locationID, err
}

func (s *PostgresService) GetSensorForReading(ctx context.Context, deviceID int, readingType string) (int, int, error) {
	var sensorID int
	var readingTypeID int
	err := s.pool.QueryRow(ctx,
		`SELECT dsr.sensor_id, rt.id
		 FROM device_sensor_readings dsr
		 JOIN reading_types rt ON rt.id = dsr.readingtype_id
		 WHERE dsr.device_id = $1
		 AND rt.name = $2`,
		deviceID,
		readingType,
	).Scan(&sensorID, &readingTypeID)
	return sensorID, readingTypeID, err
}

func (s *PostgresService) InsertSensorData(
	ctx context.Context,
	timestamp time.Time,
	deviceID int,
	locationID int,
	sensorID int,
	readingTypeID int,
	value float64,
) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO sensor_data
		(time, device_id, location_id, sensor_id, readingtype_id, value)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		timestamp,
		deviceID,
		locationID,
		sensorID,
		readingTypeID,
		value,
	)
	return err
}
