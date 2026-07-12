package services

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresService struct {
	pool *pgxpool.Pool
	cfg  *Config
}

func NewPostgresService(cfg *Config) *PostgresService {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.DatabaseTimeout)
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

func (s *PostgresService) Start(ctx context.Context, aggregates chan<- *AggregateMessage, messages <-chan *SensorMessage, wg *sync.WaitGroup) {
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
	wg.Go(func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				s.logAggregates(aggregates)
			}
			time.Sleep(time.Duration(s.cfg.AggregateWindow) * time.Minute)
		}
	})
}

func (s *PostgresService) logAggregates(aggregates chan<- *AggregateMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.DatabaseTimeout)
	defer cancel()
	type AggregateReading struct {
		Room        string
		TypeName    string
		HourAverage float64
		DayAverage  float64
		HourTime    time.Time
		DayTime     time.Time
	}
	rows, err := s.pool.Query(ctx,
		`WITH hour_averages AS (
        SELECT
        location_id,
        readingtype_id,
        AVG(value) as avg,
        EXTRACT(EPOCH FROM MAX(time))::bigint AS hour_time
        FROM sensor_data
        WHERE time > NOW() - INTERVAL '1 hour'
        GROUP BY location_id, readingtype_id
        )
		day_averages AS (
        SELECT
        location_id,
        readingtype_id,
        AVG(value) as avg,
        EXTRACT(EPOCH FROM MAX(time))::bigint AS day_time
        FROM sensor_data
        WHERE time > NOW() - INTERVAL '24 hours'
        GROUP BY location_id, readingtype_id
        )
        averages AS (
        SELECT
        location_id,
        readingtype_id,
        hour_averages.avg as hour_average,
        day_averages.avg as day_average,
        hour_time,
        day_time
        FROM hour_averages
        JOIN day_averages ON hour_averages.location_id = day_averages.location_id
        AND hour_averages.readingtype_id = day_averages.readingtype_id
        )
        SELECT
        locations.name as room,
        reading_types.name as type,
        hour_average,
        day_average,
        hour_time,
        day_time
        FROM averages
        JOIN locations ON locations.id = averages.location_id
        JOIN reading_types ON reading_types.id = averages.readingtype_id`)
	if err != nil {
		log.Println("Aggregate lookup failed:", err)
		return
	}
	defer rows.Close()
	readings, err := pgx.CollectRows(rows, pgx.RowToStructByName[AggregateReading])
	if err != nil {
		log.Println("Failed to collect rows:", err)
		return
	}
	for _, reading := range readings {
		hourlyMsg := &AggregateMessage{
			Time:  reading.HourTime.Unix(),
			Value: reading.HourAverage,
			Name:  "locations/" + reading.Room + "/hourly",
		}
		select {
		case aggregates <- hourlyMsg:
		default:
			log.Println("Hourly aggregate dropped, channel full")
		}
		dailyMsg := &AggregateMessage{
			Time:  reading.DayTime.Unix(),
			Value: reading.DayAverage,
			Name:  "locations/" + reading.Room + "/daily",
		}
		select {
		case aggregates <- dailyMsg:
		default:
			log.Println("Daily aggregate dropped, channel full")
		}
	}
}

func (s *PostgresService) handleMessage(m *SensorMessage) {
	ctx, cancel := context.WithTimeout(context.Background(), s.cfg.DatabaseTimeout)
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
