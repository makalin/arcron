package metrics

import (
	"fmt"
	"net/http"
	"time"

	"github.com/makalin/arcron/internal/config"
	"github.com/makalin/arcron/internal/jobs"
	"github.com/makalin/arcron/internal/monitoring"
	"github.com/makalin/arcron/internal/scheduler"
	"github.com/sirupsen/logrus"
)

// Exporter exports Prometheus metrics
type Exporter struct {
	config    *config.Config
	jobManager *jobs.Manager
	scheduler  *scheduler.Scheduler
	monitor    *monitoring.Monitor
	server     *http.Server
}

// NewExporter creates a new Prometheus metrics exporter
func NewExporter(cfg *config.Config, jobManager *jobs.Manager, 
	scheduler *scheduler.Scheduler, monitor *monitoring.Monitor) *Exporter {
	
	return &Exporter{
		config:     cfg,
		jobManager: jobManager,
		scheduler:  scheduler,
		monitor:    monitor,
	}
}

// Start starts the Prometheus metrics server
func (e *Exporter) Start() error {
	if !e.config.Advanced.Prometheus.Enabled {
		return nil
	}

	path := e.config.Advanced.Prometheus.Path
	if path == "" {
		path = "/metrics"
	}

	port := e.config.Advanced.Prometheus.Port
	if port == 0 {
		port = 9090
	}

	mux := http.NewServeMux()
	mux.HandleFunc(path, e.handleMetrics)

	e.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		logrus.Infof("Starting Prometheus metrics server on :%d%s", port, path)
		if err := e.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("Prometheus metrics server error: %v", err)
		}
	}()

	return nil
}

// Stop stops the Prometheus metrics server
func (e *Exporter) Stop() error {
	if e.server != nil {
		return e.server.Close()
	}
	return nil
}

// handleMetrics handles Prometheus metrics requests
func (e *Exporter) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	// System metrics
	metrics := e.monitor.GetLastMetrics()
	if metrics != nil {
		fmt.Fprintf(w, "# HELP arcron_cpu_usage CPU usage percentage\n")
		fmt.Fprintf(w, "# TYPE arcron_cpu_usage gauge\n")
		fmt.Fprintf(w, "arcron_cpu_usage %.2f\n", metrics.CPUUsage)

		fmt.Fprintf(w, "# HELP arcron_memory_usage Memory usage percentage\n")
		fmt.Fprintf(w, "# TYPE arcron_memory_usage gauge\n")
		fmt.Fprintf(w, "arcron_memory_usage %.2f\n", metrics.MemoryUsage)

		fmt.Fprintf(w, "# HELP arcron_load_average System load average\n")
		fmt.Fprintf(w, "# TYPE arcron_load_average gauge\n")
		fmt.Fprintf(w, "arcron_load_average %.2f\n", metrics.LoadAvg.Load1)
	}

	// Job metrics
	allJobs := e.jobManager.GetAllJobs()
	fmt.Fprintf(w, "# HELP arcron_jobs_total Total number of jobs\n")
	fmt.Fprintf(w, "# TYPE arcron_jobs_total gauge\n")
	fmt.Fprintf(w, "arcron_jobs_total %d\n", len(allJobs))

	runningJobs := 0
	for _, job := range allJobs {
		if job.GetStatus() == "running" {
			runningJobs++
		}
	}

	fmt.Fprintf(w, "# HELP arcron_jobs_running Number of running jobs\n")
	fmt.Fprintf(w, "# TYPE arcron_jobs_running gauge\n")
	fmt.Fprintf(w, "arcron_jobs_running %d\n", runningJobs)

	// Scheduler metrics
	schedulerStatus := e.scheduler.GetStatus()
	if jobsCount, ok := schedulerStatus["jobs_count"].(int); ok {
		fmt.Fprintf(w, "# HELP arcron_scheduler_jobs_count Number of scheduled jobs\n")
		fmt.Fprintf(w, "# TYPE arcron_scheduler_jobs_count gauge\n")
		fmt.Fprintf(w, "arcron_scheduler_jobs_count %d\n", jobsCount)
	}

	// Job execution metrics
	for name, job := range allJobs {
		status := job.GetStatus()
		fmt.Fprintf(w, "# HELP arcron_job_status Job status (1=running, 0=not running)\n")
		fmt.Fprintf(w, "# TYPE arcron_job_status gauge\n")
		if status == "running" {
			fmt.Fprintf(w, "arcron_job_status{job=\"%s\"} 1\n", name)
		} else {
			fmt.Fprintf(w, "arcron_job_status{job=\"%s\"} 0\n", name)
		}
	}
}

