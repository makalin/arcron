package types

import (
	"time"
)

// JobStatus represents the status of a job execution
type JobStatus string

const (
	StatusPending   JobStatus = "pending"
	StatusRunning   JobStatus = "running"
	StatusCompleted JobStatus = "completed"
	StatusFailed    JobStatus = "failed"
	StatusRetrying  JobStatus = "retrying"
)

// JobExecution represents a single job execution
type JobExecution struct {
	ID          string    `json:"id"`
	JobName     string    `json:"job_name"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Duration    float64   `json:"duration"`
	Status      JobStatus `json:"status"`
	ExitCode    int       `json:"exit_code"`
	Output      string    `json:"output"`
	Error       string    `json:"error"`
	RetryCount  int       `json:"retry_count"`
	Environment string    `json:"environment"`
}

// SystemMetrics represents collected system metrics
type SystemMetrics struct {
	Timestamp   time.Time `json:"timestamp"`
	CPUUsage    float64   `json:"cpu_usage"`
	MemoryUsage float64   `json:"memory_usage"`
	DiskIO      DiskIO    `json:"disk_io"`
	NetworkIO   NetworkIO `json:"network_io"`
	LoadAvg     LoadAvg   `json:"load_avg"`
}

// DiskIO represents disk I/O metrics
type DiskIO struct {
	ReadBytes  uint64  `json:"read_bytes"`
	WriteBytes uint64  `json:"write_bytes"`
	ReadCount  uint64  `json:"read_count"`
	WriteCount uint64  `json:"write_count"`
	IOUtil     float64 `json:"io_util"`
}

// NetworkIO represents network I/O metrics
type NetworkIO struct {
	BytesSent    uint64 `json:"bytes_sent"`
	BytesRecv    uint64 `json:"bytes_recv"`
	PacketsSent  uint64 `json:"packets_sent"`
	PacketsRecv  uint64 `json:"packets_recv"`
	Connections  int    `json:"connections"`
}

// LoadAvg represents system load average
type LoadAvg struct {
	Load1  float64 `json:"load_1"`
	Load5  float64 `json:"load_5"`
	Load15 float64 `json:"load_15"`
}

// Prediction represents a job execution prediction
type Prediction struct {
	JobName       string    `json:"job_name"`
	OptimalTime   time.Time `json:"optimal_time"`
	Confidence    float64   `json:"confidence"`
	Reasoning     string    `json:"reasoning"`
	ExpectedLoad  float64   `json:"expected_load"`
}
