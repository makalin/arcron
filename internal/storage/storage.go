package storage

import (
	"fmt"
	"time"

	"github.com/makalin/arcron/internal/config"
	"github.com/makalin/arcron/internal/types"
	"gorm.io/gorm"
	"gorm.io/driver/sqlite"
	"github.com/sirupsen/logrus"
)

// Storage represents the data storage layer
type Storage struct {
	db *gorm.DB
}

// New creates a new Storage instance
func New(cfg config.DatabaseConfig) (*Storage, error) {
	var db *gorm.DB
	var err error

	switch cfg.Driver {
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(cfg.DSN), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to open SQLite database: %v", err)
		}
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying database: %v", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxConns)
	sqlDB.SetMaxIdleConns(cfg.MaxConns / 2)

	// Auto-migrate database schema
	if err := db.AutoMigrate(
		&JobExecutionRecord{},
		&SystemMetricsRecord{},
		&MLPredictionRecord{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	logrus.Info("Storage initialized successfully")
	return &Storage{db: db}, nil
}

// JobExecutionRecord represents a job execution record in the database
type JobExecutionRecord struct {
	ID          string    `gorm:"primaryKey"`
	JobName     string    `gorm:"index;not null"`
	StartTime   time.Time `gorm:"not null"`
	EndTime     time.Time
	Duration    float64
	Status      string `gorm:"not null"`
	ExitCode    int
	Output      string `gorm:"type:text"`
	Error       string `gorm:"type:text"`
	RetryCount  int
	Environment string `gorm:"type:text"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SystemMetricsRecord represents system metrics in the database
type SystemMetricsRecord struct {
	ID          uint      `gorm:"primaryKey"`
	Timestamp   time.Time `gorm:"index;not null"`
	CPUUsage    float64
	MemoryUsage float64
	DiskIO      float64
	NetworkIO   float64
	LoadAvg     float64
	CreatedAt   time.Time
}

// MLPredictionRecord represents ML predictions in the database
type MLPredictionRecord struct {
	ID           uint      `gorm:"primaryKey"`
	JobName      string    `gorm:"index;not null"`
	PredictedAt  time.Time `gorm:"not null"`
	OptimalTime  time.Time `gorm:"not null"`
	Confidence   float64
	Reasoning    string `gorm:"type:text"`
	ExpectedLoad float64
	CreatedAt    time.Time
}

// StoreJobExecution stores a job execution record
func (s *Storage) StoreJobExecution(execution *types.JobExecution) error {
	record := &JobExecutionRecord{
		ID:          execution.ID,
		JobName:     execution.JobName,
		StartTime:   execution.StartTime,
		EndTime:     execution.EndTime,
		Duration:    execution.Duration,
		Status:      string(execution.Status),
		ExitCode:    execution.ExitCode,
		Output:      execution.Output,
		Error:       execution.Error,
		RetryCount:  execution.RetryCount,
		Environment: execution.Environment,
	}

	result := s.db.Create(record)
	if result.Error != nil {
		return fmt.Errorf("failed to store job execution: %v", result.Error)
	}

	return nil
}

// GetJobExecutions retrieves job executions for a specific job
func (s *Storage) GetJobExecutions(jobName string, limit int) ([]*types.JobExecution, error) {
	var records []JobExecutionRecord

	query := s.db.Where("job_name = ?", jobName).Order("start_time DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve job executions: %v", err)
	}

	executions := make([]*types.JobExecution, len(records))
	for i, record := range records {
		executions[i] = &types.JobExecution{
			ID:          record.ID,
			JobName:     record.JobName,
			StartTime:   record.StartTime,
			EndTime:     record.EndTime,
			Duration:    record.Duration,
			Status:      types.JobStatus(record.Status),
			ExitCode:    record.ExitCode,
			Output:      record.Output,
			Error:       record.Error,
			RetryCount:  record.RetryCount,
			Environment: record.Environment,
		}
	}

	return executions, nil
}

// StoreSystemMetrics stores system metrics
func (s *Storage) StoreSystemMetrics(metrics *types.SystemMetrics) error {
	record := &SystemMetricsRecord{
		Timestamp:   metrics.Timestamp,
		CPUUsage:    metrics.CPUUsage,
		MemoryUsage: metrics.MemoryUsage,
		DiskIO:      float64(metrics.DiskIO.ReadBytes+metrics.DiskIO.WriteBytes) / 1024 / 1024,
		NetworkIO:   float64(metrics.NetworkIO.BytesSent+metrics.NetworkIO.BytesRecv) / 1024 / 1024,
		LoadAvg:     metrics.LoadAvg.Load1,
	}

	result := s.db.Create(record)
	if result.Error != nil {
		return fmt.Errorf("failed to store system metrics: %v", result.Error)
	}

	return nil
}

// GetSystemMetrics retrieves system metrics within a time range
func (s *Storage) GetSystemMetrics(start, end time.Time, limit int) ([]*types.SystemMetrics, error) {
	var records []SystemMetricsRecord

	query := s.db.Where("timestamp BETWEEN ? AND ?", start, end).Order("timestamp DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve system metrics: %v", err)
	}

	metrics := make([]*types.SystemMetrics, len(records))
	for i, record := range records {
		metrics[i] = &types.SystemMetrics{
			Timestamp:   record.Timestamp,
			CPUUsage:    record.CPUUsage,
			MemoryUsage: record.MemoryUsage,
			DiskIO: types.DiskIO{
				ReadBytes:  uint64(record.DiskIO * 1024 * 1024), // Convert back to bytes
				WriteBytes: 0,
			},
			NetworkIO: types.NetworkIO{
				BytesSent: uint64(record.NetworkIO * 1024 * 1024), // Convert back to bytes
				BytesRecv: 0,
			},
			LoadAvg: types.LoadAvg{
				Load1: record.LoadAvg,
			},
		}
	}

	return metrics, nil
}

// StoreMLPrediction stores an ML prediction
func (s *Storage) StoreMLPrediction(prediction *types.SystemMetrics) error {
	// This is a placeholder - in a real implementation, you'd store actual ML predictions
	// For now, we'll just store the metrics that led to the prediction
	record := &MLPredictionRecord{
		PredictedAt:  time.Now(),
		JobName:      "system_prediction",
		OptimalTime:  time.Now().Add(5 * time.Minute),
		Confidence:   0.7,
		Reasoning:    "System metrics analysis",
		ExpectedLoad: prediction.CPUUsage,
	}

	result := s.db.Create(record)
	if result.Error != nil {
		return fmt.Errorf("failed to store ML prediction: %v", result.Error)
	}

	return nil
}

// GetJobStatistics retrieves statistics for a specific job
func (s *Storage) GetJobStatistics(jobName string) (map[string]interface{}, error) {
	var totalCount int64
	var successCount int64
	var failureCount int64
	var avgDuration float64

	// Get total executions
	if err := s.db.Model(&JobExecutionRecord{}).Where("job_name = ?", jobName).Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count total executions: %v", err)
	}

	// Get successful executions
	if err := s.db.Model(&JobExecutionRecord{}).Where("job_name = ? AND status = ?", jobName, "completed").Count(&successCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count successful executions: %v", err)
	}

	// Get failed executions
	if err := s.db.Model(&JobExecutionRecord{}).Where("job_name = ? AND status = ?", jobName, "failed").Count(&failureCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count failed executions: %v", err)
	}

	// Get average duration
	if err := s.db.Model(&JobExecutionRecord{}).Where("job_name = ? AND status = ?", jobName, "completed").Select("AVG(duration)").Scan(&avgDuration).Error; err != nil {
		return nil, fmt.Errorf("failed to get average duration: %v", err)
	}

	successRate := 0.0
	if totalCount > 0 {
		successRate = float64(successCount) / float64(totalCount) * 100
	}

	return map[string]interface{}{
		"total_executions": totalCount,
		"successful":       successCount,
		"failed":           failureCount,
		"success_rate":     successRate,
		"avg_duration":     avgDuration,
	}, nil
}

// CleanupOldRecords removes old records to prevent database bloat
func (s *Storage) CleanupOldRecords(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)

	// Clean up old job executions
	if err := s.db.Where("created_at < ?", cutoff).Delete(&JobExecutionRecord{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup old job executions: %v", err)
	}

	// Clean up old system metrics
	if err := s.db.Where("created_at < ?", cutoff).Delete(&SystemMetricsRecord{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup old system metrics: %v", err)
	}

	// Clean up old ML predictions
	if err := s.db.Where("created_at < ?", cutoff).Delete(&MLPredictionRecord{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup old ML predictions: %v", err)
	}

	logrus.Infof("Cleaned up records older than %v", olderThan)
	return nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
