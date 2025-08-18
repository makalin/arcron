package jobs

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/makalin/arcron/internal/config"
	"github.com/makalin/arcron/internal/storage"
	"github.com/makalin/arcron/internal/types"
	"github.com/sirupsen/logrus"
)

// Use types from the types package
type JobStatus = types.JobStatus

// Job represents a single job
type Job struct {
	config config.JobConfig
	status JobStatus
	mutex  sync.RWMutex
}

// Use types from the types package
type JobExecution = types.JobExecution

// Manager manages job execution and tracking
type Manager struct {
	jobs   map[string]*Job
	store  *storage.Storage
	mutex  sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a new Job Manager
func New(jobConfigs []config.JobConfig, store *storage.Storage) (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		jobs:   make(map[string]*Job),
		store:  store,
		ctx:    ctx,
		cancel: cancel,
	}

	// Initialize jobs from config
	for _, jobConfig := range jobConfigs {
		job, err := NewJob(jobConfig)
		if err != nil {
			logrus.Errorf("Failed to create job %s: %v", jobConfig.Name, err)
			continue
		}
		manager.jobs[jobConfig.Name] = job
	}

	return manager, nil
}

// NewJob creates a new Job instance
func NewJob(jobConfig config.JobConfig) (*Job, error) {
	if jobConfig.Name == "" {
		return nil, fmt.Errorf("job name cannot be empty")
	}

	if jobConfig.Command == "" {
		return nil, fmt.Errorf("job command cannot be empty")
	}

	return &Job{
		config: jobConfig,
		status: types.StatusPending,
	}, nil
}

// ExecuteJob executes a job
func (m *Manager) ExecuteJob(job *Job) error {
	execution := &JobExecution{
		ID:        generateExecutionID(),
		JobName:   job.config.Name,
		StartTime: time.Now(),
		Status:    types.StatusRunning,
	}

	// Update job status
	job.setStatus(types.StatusRunning)

	// Store execution start
	if err := m.store.StoreJobExecution(execution); err != nil {
		logrus.Errorf("Failed to store job execution start: %v", err)
	}

	// Execute the command
	output, exitCode, err := m.executeCommand(job.config)

	// Update execution details
	execution.EndTime = time.Now()
	execution.Duration = execution.EndTime.Sub(execution.StartTime).Seconds()
	execution.Output = output
	execution.ExitCode = exitCode

	if err != nil {
		execution.Status = types.StatusFailed
		execution.Error = err.Error()
		job.setStatus(types.StatusFailed)
		logrus.Errorf("Job %s failed: %v", job.config.Name, err)
	} else {
		execution.Status = types.StatusCompleted
		job.setStatus(types.StatusCompleted)
		logrus.Infof("Job %s completed successfully in %.2f seconds", job.config.Name, execution.Duration)
	}

	// Store execution result
	if err := m.store.StoreJobExecution(execution); err != nil {
		logrus.Errorf("Failed to store job execution result: %v", err)
	}

	// Handle retries if needed
	if execution.Status == types.StatusFailed && job.config.Retries > 0 {
		m.handleRetry(job, execution)
	}

	return err
}

// executeCommand executes the job command
func (m *Manager) executeCommand(jobConfig config.JobConfig) (string, int, error) {
	ctx, cancel := context.WithTimeout(m.ctx, jobConfig.Timeout)
	defer cancel()

	// Parse command and arguments
	parts := strings.Fields(jobConfig.Command)
	if len(parts) == 0 {
		return "", -1, fmt.Errorf("empty command")
	}

	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)

	// Set environment variables
	if len(jobConfig.Environment) > 0 {
		env := make([]string, 0, len(jobConfig.Environment))
		for k, v := range jobConfig.Environment {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	exitCode := cmd.ProcessState.ExitCode()

	return string(output), exitCode, err
}

// handleRetry handles job retries
func (m *Manager) handleRetry(job *Job, execution *JobExecution) {
	if execution.RetryCount >= job.config.Retries {
		logrus.Warnf("Job %s exceeded maximum retries (%d)", job.config.Name, job.config.Retries)
		return
	}

	execution.RetryCount++
	execution.Status = types.StatusRetrying
	job.setStatus(types.StatusRetrying)

	// Store retry attempt
	if err := m.store.StoreJobExecution(execution); err != nil {
		logrus.Errorf("Failed to store retry execution: %v", err)
	}

	logrus.Infof("Retrying job %s (attempt %d/%d)", job.config.Name, execution.RetryCount, job.config.Retries)

	// Wait before retry (exponential backoff)
	backoff := time.Duration(execution.RetryCount) * 30 * time.Second
	time.Sleep(backoff)

	// Execute retry
	if err := m.ExecuteJob(job); err != nil {
		logrus.Errorf("Retry attempt %d for job %s failed: %v", execution.RetryCount, job.config.Name, err)
	}
}

// GetJob returns a job by name
func (m *Manager) GetJob(name string) (*Job, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	job, exists := m.jobs[name]
	return job, exists
}

// GetAllJobs returns all jobs
func (m *Manager) GetAllJobs() map[string]*Job {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]*Job)
	for name, job := range m.jobs {
		result[name] = job
	}
	return result
}

// GetJobExecutions returns executions for a specific job
func (m *Manager) GetJobExecutions(jobName string, limit int) ([]*JobExecution, error) {
	return m.store.GetJobExecutions(jobName, limit)
}

// Stop stops the job manager
func (m *Manager) Stop() {
	m.cancel()
}

// setStatus sets the job status
func (j *Job) setStatus(status JobStatus) {
	j.mutex.Lock()
	defer j.mutex.Unlock()
	j.status = status
}

// GetStatus returns the current job status
func (j *Job) GetStatus() JobStatus {
	j.mutex.RLock()
	defer j.mutex.RUnlock()
	return j.status
}

// GetConfig returns the job configuration
func (j *Job) GetConfig() config.JobConfig {
	return j.config
}

// GetName returns the job name
func (j *Job) GetName() string {
	return j.config.Name
}

// GetType returns the job type
func (j *Job) GetType() string {
	return j.config.Type
}

// GetSchedule returns the job schedule
func (j *Job) GetSchedule() string {
	return j.config.Schedule
}

// generateExecutionID generates a unique execution ID
func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}
