package ml

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/makalin/arcron/internal/config"
	"github.com/makalin/arcron/internal/monitoring"

	"github.com/sirupsen/logrus"
)

// Prediction represents a job execution prediction
type Prediction struct {
	JobName       string    `json:"job_name"`
	OptimalTime   time.Time `json:"optimal_time"`
	Confidence    float64   `json:"confidence"`
	Reasoning     string    `json:"reasoning"`
	ExpectedLoad  float64   `json:"expected_load"`
}

// FeatureVector represents the input features for ML prediction
type FeatureVector struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskIO      float64 `json:"disk_io"`
	NetworkIO   float64 `json:"network_io"`
	LoadAvg     float64 `json:"load_avg"`
	HourOfDay   float64 `json:"hour_of_day"`
	DayOfWeek   float64 `json:"day_of_week"`
}

// Engine represents the machine learning engine
type Engine struct {
	config       config.MLConfig
	model        *SimpleMLModel
	stopChan     chan struct{}
	isRunning    bool
	lastTraining time.Time
}

// SimpleMLModel represents a simplified ML model
type SimpleMLModel struct {
	weights     []float64
	featureMean []float64
	featureStd  []float64
	trained     bool
}

// New creates a new ML Engine instance
func New(cfg config.MLConfig) (*Engine, error) {
	model := &SimpleMLModel{
		weights:     make([]float64, 8), // 8 features
		featureMean: make([]float64, 8),
		featureStd:  make([]float64, 8),
		trained:     false,
	}

	return &Engine{
		config:    cfg,
		model:     model,
		stopChan:  make(chan struct{}),
	}, nil
}

// Start starts the ML engine
func (e *Engine) Start(ctx context.Context) error {
	if e.isRunning {
		return fmt.Errorf("ML engine is already running")
	}

	e.isRunning = true
	logrus.Info("Starting ML engine...")

	// Initialize with simple heuristics if no model exists
	if !e.model.trained {
		e.initializeHeuristics()
	}

	go e.periodicTraining(ctx)

	return nil
}

// Stop stops the ML engine
func (e *Engine) Stop() {
	if !e.isRunning {
		return
	}

	logrus.Info("Stopping ML engine...")
	close(e.stopChan)
	e.isRunning = false
}

// PredictOptimalTime predicts the optimal execution time for a job
func (e *Engine) PredictOptimalTime(jobName, jobType string, currentMetrics monitoring.SystemMetrics) (*Prediction, error) {
	if !e.model.trained {
		return e.predictWithHeuristics(jobName, jobType, currentMetrics)
	}

	features := e.extractFeatures(currentMetrics)
	prediction := e.model.predict(features)

	// Convert prediction to time
	optimalTime := time.Now().Add(time.Duration(prediction) * time.Minute)

	return &Prediction{
		JobName:      jobName,
		OptimalTime:  optimalTime,
		Confidence:   0.7, // Placeholder confidence
		Reasoning:    fmt.Sprintf("ML model prediction based on %d features", len(features)),
		ExpectedLoad: prediction,
	}, nil
}

// predictWithHeuristics predicts using simple heuristics
func (e *Engine) predictWithHeuristics(jobName, jobType string, metrics monitoring.SystemMetrics) (*Prediction, error) {
	var delay time.Duration
	var reasoning string

	switch jobType {
	case "resource-intensive":
		// For resource-intensive jobs, wait for low system load
		if metrics.CPUUsage > 80 || metrics.MemoryUsage > 80 {
			delay = 30 * time.Minute
			reasoning = "High system load detected, delaying resource-intensive job"
		} else if metrics.CPUUsage > 60 || metrics.MemoryUsage > 60 {
			delay = 15 * time.Minute
			reasoning = "Moderate system load, slight delay for resource-intensive job"
		} else {
			delay = 5 * time.Minute
			reasoning = "Low system load, minimal delay for resource-intensive job"
		}
	case "light":
		// For light jobs, minimal delay
		if metrics.CPUUsage > 90 || metrics.MemoryUsage > 90 {
			delay = 10 * time.Minute
			reasoning = "Very high system load, delaying light job"
		} else {
			delay = 1 * time.Minute
			reasoning = "System load acceptable for light job"
		}
	default:
		delay = 5 * time.Minute
		reasoning = "Unknown job type, using default delay"
	}

	optimalTime := time.Now().Add(delay)

	return &Prediction{
		JobName:      jobName,
		OptimalTime:  optimalTime,
		Confidence:   0.5, // Lower confidence for heuristics
		Reasoning:    reasoning,
		ExpectedLoad: float64(delay.Minutes()),
	}, nil
}

// extractFeatures extracts features from system metrics
func (e *Engine) extractFeatures(metrics monitoring.SystemMetrics) []float64 {
	now := time.Now()
	
	features := []float64{
		metrics.CPUUsage,
		metrics.MemoryUsage,
		float64(metrics.DiskIO.ReadBytes+metrics.DiskIO.WriteBytes) / 1024 / 1024, // MB
		float64(metrics.NetworkIO.BytesSent+metrics.NetworkIO.BytesRecv) / 1024 / 1024, // MB
		metrics.LoadAvg.Load1,
		float64(now.Hour()),
		float64(now.Weekday()),
	}

	return features
}

// initializeHeuristics initializes the model with simple heuristics
func (e *Engine) initializeHeuristics() {
	// Simple weights based on domain knowledge
	e.model.weights = []float64{
		-0.1,  // CPU usage (negative: prefer lower)
		-0.1,  // Memory usage (negative: prefer lower)
		-0.05, // Disk I/O (negative: prefer lower)
		-0.05, // Network I/O (negative: prefer lower)
		-0.1,  // Load average (negative: prefer lower)
		0.0,   // Hour of day (neutral)
		0.0,   // Day of week (neutral)
	}

	e.model.trained = true
	logrus.Info("ML model initialized with heuristics")
}

// periodicTraining performs periodic model training
func (e *Engine) periodicTraining(ctx context.Context) {
	ticker := time.NewTicker(e.config.UpdateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopChan:
			return
		case <-ticker.C:
			if err := e.trainModel(); err != nil {
				logrus.Errorf("Failed to train model: %v", err)
			}
		}
	}
}

// trainModel trains the ML model with collected data
func (e *Engine) trainModel() error {
	// This is a simplified training implementation
	// In a real implementation, you'd use actual training data
	logrus.Debug("Training ML model...")
	
	// For now, just update the last training time
	e.lastTraining = time.Now()
	
	return nil
}

// GetStatus returns the current status of the ML engine
func (e *Engine) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"running":       e.isRunning,
		"model_trained": e.model.trained,
		"last_training": e.lastTraining,
		"features":      len(e.model.weights),
	}
}

// predict makes a prediction using the trained model
func (m *SimpleMLModel) predict(features []float64) float64 {
	if !m.trained || len(features) != len(m.weights) {
		return 0.0
	}

	var prediction float64
	for i, feature := range features {
		prediction += feature * m.weights[i]
	}

	// Apply sigmoid activation and scale to reasonable range
	prediction = 1.0 / (1.0 + math.Exp(-prediction))
	return prediction * 60.0 // Scale to minutes
}
