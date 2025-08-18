package monitoring

import (
	"context"
	"fmt"
	"time"

	"github.com/makalin/arcron/internal/config"
	"github.com/makalin/arcron/internal/types"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/sirupsen/logrus"
)

// Use types from the types package
type SystemMetrics = types.SystemMetrics
type DiskIO = types.DiskIO
type NetworkIO = types.NetworkIO
type LoadAvg = types.LoadAvg

// Monitor represents the system monitoring component
type Monitor struct {
	config     *config.Config
	metrics    chan SystemMetrics
	stopChan   chan struct{}
	interval   time.Duration
	isRunning  bool
	lastMetrics *SystemMetrics
}

// New creates a new Monitor instance
func New(cfg *config.Config) (*Monitor, error) {
	return &Monitor{
		config:   cfg,
		metrics:  make(chan SystemMetrics, 100),
		stopChan: make(chan struct{}),
		interval: 5 * time.Second, // Default collection interval
	}, nil
}

// Start starts the monitoring
func (m *Monitor) Start(ctx context.Context) error {
	if m.isRunning {
		return fmt.Errorf("monitor is already running")
	}

	m.isRunning = true
	logrus.Info("Starting system monitoring...")

	go m.collectMetrics(ctx)

	return nil
}

// Stop stops the monitoring
func (m *Monitor) Stop() {
	if !m.isRunning {
		return
	}

	logrus.Info("Stopping system monitoring...")
	close(m.stopChan)
	m.isRunning = false
}

// collectMetrics continuously collects system metrics
func (m *Monitor) collectMetrics(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			metrics, err := m.collectCurrentMetrics()
			if err != nil {
				logrus.Errorf("Failed to collect metrics: %v", err)
				continue
			}

			m.lastMetrics = &metrics
			
			select {
			case m.metrics <- metrics:
				// Metrics sent successfully
			default:
				// Channel is full, skip this metric
				logrus.Warn("Metrics channel is full, skipping metric collection")
			}
		}
	}
}

// collectCurrentMetrics collects current system metrics
func (m *Monitor) collectCurrentMetrics() (SystemMetrics, error) {
	metrics := SystemMetrics{
		Timestamp: time.Now(),
	}

	// Collect CPU usage
	cpuPercent, err := cpu.Percent(0, false)
	if err == nil && len(cpuPercent) > 0 {
		metrics.CPUUsage = cpuPercent[0]
	}

	// Collect memory usage
	if vmstat, err := mem.VirtualMemory(); err == nil {
		metrics.MemoryUsage = vmstat.UsedPercent
	}

	// Collect disk I/O
	if diskIO, err := disk.IOCounters(); err == nil {
		var totalRead, totalWrite uint64
		var totalReadCount, totalWriteCount uint64
		
		for _, io := range diskIO {
			totalRead += io.ReadBytes
			totalWrite += io.WriteBytes
			totalReadCount += io.ReadCount
			totalWriteCount += io.WriteCount
		}
		
		metrics.DiskIO = DiskIO{
			ReadBytes:  totalRead,
			WriteBytes: totalWrite,
			ReadCount:  totalReadCount,
			WriteCount: totalWriteCount,
		}
	}

	// Collect network I/O
	if netIO, err := net.IOCounters(false); err == nil && len(netIO) > 0 {
		io := netIO[0]
		metrics.NetworkIO = NetworkIO{
			BytesSent:   io.BytesSent,
			BytesRecv:   io.BytesRecv,
			PacketsSent: io.PacketsSent,
			PacketsRecv: io.PacketsRecv,
		}
	}

	// Collect load average (Linux only)
	if load, err := getLoadAverage(); err == nil {
		metrics.LoadAvg = load
	}

	return metrics, nil
}

// getLoadAverage gets system load average (Linux specific)
func getLoadAverage() (LoadAvg, error) {
	// This is a simplified implementation
	// In a real implementation, you'd read from /proc/loadavg
	return LoadAvg{
		Load1:  0.0,
		Load5:  0.0,
		Load15: 0.0,
	}, nil
}

// GetMetrics returns the metrics channel
func (m *Monitor) GetMetrics() <-chan SystemMetrics {
	return m.metrics
}

// GetLastMetrics returns the last collected metrics
func (m *Monitor) GetLastMetrics() *SystemMetrics {
	return m.lastMetrics
}

// GetStatus returns the current status of the monitor
func (m *Monitor) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"running": m.isRunning,
		"interval": m.interval.String(),
	}
	
	if m.lastMetrics != nil {
		status["last_collection"] = m.lastMetrics.Timestamp
		status["cpu_usage"] = m.lastMetrics.CPUUsage
		status["memory_usage"] = m.lastMetrics.MemoryUsage
	}
	
	return status
}

// SetInterval sets the metrics collection interval
func (m *Monitor) SetInterval(interval time.Duration) {
	m.interval = interval
}
