package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/makalin/arcron/internal/config"
	"github.com/makalin/arcron/internal/jobs"
	"github.com/makalin/arcron/internal/ml"
	"github.com/makalin/arcron/internal/monitoring"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// ScheduledJob represents a job with its scheduling information
type ScheduledJob struct {
	Job         *jobs.Job
	EntryID     cron.EntryID
	NextRun     time.Time
	LastRun     time.Time
	RunCount    int
	Status      string
	Prediction  *ml.Prediction
}

// Scheduler represents the intelligent job scheduler
type Scheduler struct {
	config      *config.Config
	jobManager  *jobs.Manager
	mlEngine    *ml.Engine
	monitor     *monitoring.Monitor
	cron        *cron.Cron
	jobs        map[string]*ScheduledJob
	mutex       sync.RWMutex
	stopChan    chan struct{}
	isRunning   bool
}

// New creates a new Scheduler instance
func New(cfg *config.Config, jobManager *jobs.Manager, mlEngine *ml.Engine, monitor *monitoring.Monitor) (*Scheduler, error) {
	c := cron.New(cron.WithSeconds())

	return &Scheduler{
		config:     cfg,
		jobManager: jobManager,
		mlEngine:   mlEngine,
		monitor:    monitor,
		cron:       c,
		jobs:       make(map[string]*ScheduledJob),
		stopChan:   make(chan struct{}),
	}, nil
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) error {
	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	s.isRunning = true
	logrus.Info("Starting intelligent scheduler...")

	// Start the cron scheduler
	s.cron.Start()

	// Schedule all configured jobs
	if err := s.scheduleJobs(); err != nil {
		return fmt.Errorf("failed to schedule jobs: %v", err)
	}

	// Start the intelligent scheduling loop
	go s.intelligentSchedulingLoop(ctx)

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	if !s.isRunning {
		return
	}

	logrus.Info("Stopping scheduler...")
	s.cron.Stop()
	close(s.stopChan)
	s.isRunning = false
}

// scheduleJobs schedules all configured jobs
func (s *Scheduler) scheduleJobs() error {
	for _, jobConfig := range s.config.Jobs {
		if err := s.scheduleJob(jobConfig); err != nil {
			logrus.Errorf("Failed to schedule job %s: %v", jobConfig.Name, err)
			continue
		}
	}

	logrus.Infof("Scheduled %d jobs", len(s.config.Jobs))
	return nil
}

// scheduleJob schedules a single job
func (s *Scheduler) scheduleJob(jobConfig config.JobConfig) error {
	job, err := jobs.NewJob(jobConfig)
	if err != nil {
		return fmt.Errorf("failed to create job: %v", err)
	}

	// Create scheduled job entry
	scheduledJob := &ScheduledJob{
		Job:      job,
		NextRun:  time.Now(),
		Status:   "scheduled",
		RunCount: 0,
	}

	// Add to cron scheduler with initial schedule
	entryID, err := s.cron.AddFunc(jobConfig.Schedule, func() {
		s.executeJob(scheduledJob)
	})
	if err != nil {
		return fmt.Errorf("failed to add job to cron: %v", err)
	}

	scheduledJob.EntryID = entryID
	s.jobs[jobConfig.Name] = scheduledJob

	logrus.Infof("Scheduled job: %s with schedule: %s", jobConfig.Name, jobConfig.Schedule)
	return nil
}

// intelligentSchedulingLoop continuously monitors and adjusts job schedules
func (s *Scheduler) intelligentSchedulingLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopChan:
			return
		case <-ticker.C:
			s.adjustSchedules()
		}
	}
}

// adjustSchedules adjusts job schedules based on ML predictions
func (s *Scheduler) adjustSchedules() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	currentMetrics := s.monitor.GetLastMetrics()
	if currentMetrics == nil {
		logrus.Debug("No metrics available for schedule adjustment")
		return
	}

	for _, scheduledJob := range s.jobs {
		// Get ML prediction for optimal execution time
		prediction, err := s.mlEngine.PredictOptimalTime(
			scheduledJob.Job.GetName(),
			scheduledJob.Job.GetType(),
			*currentMetrics,
		)
		if err != nil {
			logrus.Errorf("Failed to get prediction for job %s: %v", scheduledJob.Job.GetName(), err)
			continue
		}

		scheduledJob.Prediction = prediction

		// Check if we should adjust the schedule
		if s.shouldAdjustSchedule(scheduledJob, prediction) {
			s.adjustJobSchedule(scheduledJob, prediction)
		}
	}
}

// shouldAdjustSchedule determines if a job schedule should be adjusted
func (s *Scheduler) shouldAdjustSchedule(scheduledJob *ScheduledJob, prediction *ml.Prediction) bool {
	// Don't adjust if the job is currently running
	if scheduledJob.Status == "running" {
		return false
	}

	// Don't adjust if the prediction confidence is too low
	if prediction.Confidence < 0.3 {
		return false
	}

	// Adjust if the predicted optimal time is significantly different from the next run
	timeDiff := prediction.OptimalTime.Sub(scheduledJob.NextRun)
	return timeDiff.Abs() > 5*time.Minute
}

// adjustJobSchedule adjusts a job's schedule based on ML prediction
func (s *Scheduler) adjustJobSchedule(scheduledJob *ScheduledJob, prediction *ml.Prediction) {
	// Remove the current cron entry
	s.cron.Remove(scheduledJob.EntryID)

	// Calculate new delay
	delay := time.Until(prediction.OptimalTime)
	if delay < 0 {
		delay = 1 * time.Minute // Minimum delay
	}

	// Create new cron entry with adjusted timing
	entryID, err := s.cron.AddFunc(fmt.Sprintf("@every %s", delay.String()), func() {
		s.executeJob(scheduledJob)
	})
	if err != nil {
		logrus.Errorf("Failed to adjust schedule for job %s: %v", scheduledJob.Job.GetName(), err)
		return
	}

	// Update the scheduled job
	scheduledJob.EntryID = entryID
	scheduledJob.NextRun = prediction.OptimalTime
	scheduledJob.Status = "adjusted"

	logrus.Infof("Adjusted schedule for job %s: new run time %s (reason: %s)",
		scheduledJob.Job.GetName(), prediction.OptimalTime.Format("15:04:05"), prediction.Reasoning)
}

// executeJob executes a scheduled job
func (s *Scheduler) executeJob(scheduledJob *ScheduledJob) {
	s.mutex.Lock()
	scheduledJob.Status = "running"
	scheduledJob.LastRun = time.Now()
	s.mutex.Unlock()

	logrus.Infof("Executing job: %s", scheduledJob.Job.GetName())

	// Execute the job
	if err := s.jobManager.ExecuteJob(scheduledJob.Job); err != nil {
		logrus.Errorf("Failed to execute job %s: %v", scheduledJob.Job.GetName(), err)
		scheduledJob.Status = "failed"
	} else {
		scheduledJob.Status = "completed"
		scheduledJob.RunCount++
	}

	// Reschedule the job for next run
	s.rescheduleJob(scheduledJob)
}

// rescheduleJob reschedules a job after execution
func (s *Scheduler) rescheduleJob(scheduledJob *ScheduledJob) {
	// Remove the current entry
	s.cron.Remove(scheduledJob.EntryID)

	// Add the job back with its original schedule
	entryID, err := s.cron.AddFunc(scheduledJob.Job.GetSchedule(), func() {
		s.executeJob(scheduledJob)
	})
	if err != nil {
		logrus.Errorf("Failed to reschedule job %s: %v", scheduledJob.Job.GetName(), err)
		return
	}

	scheduledJob.EntryID = entryID
	scheduledJob.Status = "scheduled"
}

// GetStatus returns the current status of the scheduler
func (s *Scheduler) GetStatus() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	jobStatuses := make(map[string]interface{})
	for name, job := range s.jobs {
		jobStatuses[name] = map[string]interface{}{
			"status":    job.Status,
			"next_run":  job.NextRun,
			"last_run":  job.LastRun,
			"run_count": job.RunCount,
		}
	}

	return map[string]interface{}{
		"running":    s.isRunning,
		"jobs_count": len(s.jobs),
		"jobs":       jobStatuses,
	}
}

// GetJobStatus returns the status of a specific job
func (s *Scheduler) GetJobStatus(jobName string) (*ScheduledJob, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	job, exists := s.jobs[jobName]
	return job, exists
}
