package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresService struct {
	pool *pgxpool.Pool
	cfg  *Config
}

func NewPostgresService(cfg *Config) *PostgresService {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.InsertTimeout)
	defer cancel()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	return &PostgresService{
		pool: pool,
		cfg:  cfg,
	}
}

func (s *PostgresService) Close() {
	s.pool.Close()
}

func (s *PostgresService) Start(ctx context.Context, messages <-chan *SensorMessage, wg *sync.WaitGroup) {
	numWorkers := s.cfg.DBWorkers
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-messages:
					if !ok {
						return
					}
					s.handleMessage(msg)
				}
			}
		}()
	}
}

func (s *PostgresService) handleMessage(m *SensorMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.InsertTimeout)
	defer cancel()
	deviceID, locationID, err := s.getDevice(ctx, m.Identifier)
	if err != nil {
		log.Println("device lookup failed:", err)
		log.Printf("Identifier: %s\n", m.Identifier)
		return
	}
	sensorID, readingTypeID, err := s.getSensorForReading(ctx, deviceID, m.ReadingTypeName)
	if err != nil {
		log.Println("sensor lookup failed:", err)
		log.Printf("DeviceID: %d, Reading: %s\n", deviceID, m.ReadingTypeName)
		return
	}
	err = s.insertSensorData(
		ctx,
		m.Time,
		deviceID,
		locationID,
		sensorID,
		readingTypeID,
		m.Value,
	)
	if err != nil {
		log.Println("insert failed:", err)
	}
}

func (s *PostgresService) getDevice(ctx context.Context, identifier string) (int, int, error) {
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

func (s *PostgresService) getSensorForReading(ctx context.Context, deviceID int, readingType string) (int, int, error) {
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

func (s *PostgresService) insertSensorData(
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
