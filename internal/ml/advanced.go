package ml

import (
	"fmt"
	"math"
	"time"

	"github.com/makalin/arcron/internal/monitoring"
	"github.com/makalin/arcron/internal/storage"
	"github.com/sirupsen/logrus"
)

// SeasonalityDetector detects seasonal patterns in system metrics
type SeasonalityDetector struct {
	store *storage.Storage
}

// NewSeasonalityDetector creates a new seasonality detector
func NewSeasonalityDetector(store *storage.Storage) *SeasonalityDetector {
	return &SeasonalityDetector{
		store: store,
	}
}

// SeasonalPattern represents a detected seasonal pattern
type SeasonalPattern struct {
	Type      string  `json:"type"`       // "daily", "weekly", "monthly"
	Strength  float64 `json:"strength"`   // 0-1, how strong the pattern is
	PeakHours []int   `json:"peak_hours"` // Hours when load is typically high
	LowHours  []int   `json:"low_hours"`  // Hours when load is typically low
	PeakDays  []int   `json:"peak_days"`  // Days of week (0=Sunday) when load is high
	LowDays   []int   `json:"low_days"`   // Days of week when load is low
}

// DetectSeasonality detects seasonal patterns in historical metrics
func (sd *SeasonalityDetector) DetectSeasonality(jobName string, days int) (*SeasonalPattern, error) {
	end := time.Now()
	start := end.Add(-time.Duration(days) * 24 * time.Hour)

	metrics, err := sd.store.GetSystemMetrics(start, end, 10000)
	if err != nil {
		return nil, err
	}

	if len(metrics) < 24 {
		return nil, nil // Not enough data
	}

	// Analyze hourly patterns
	hourlyLoad := make(map[int][]float64)
	dayOfWeekLoad := make(map[int][]float64)

	for _, m := range metrics {
		hour := m.Timestamp.Hour()
		dayOfWeek := int(m.Timestamp.Weekday())

		load := (m.CPUUsage + m.MemoryUsage) / 2.0
		hourlyLoad[hour] = append(hourlyLoad[hour], load)
		dayOfWeekLoad[dayOfWeek] = append(dayOfWeekLoad[dayOfWeek], load)
	}

	// Calculate average load per hour
	hourlyAvg := make(map[int]float64)
	for hour, loads := range hourlyLoad {
		sum := 0.0
		for _, load := range loads {
			sum += load
		}
		hourlyAvg[hour] = sum / float64(len(loads))
	}

	// Calculate average load per day of week
	dayAvg := make(map[int]float64)
	for day, loads := range dayOfWeekLoad {
		sum := 0.0
		for _, load := range loads {
			sum += load
		}
		dayAvg[day] = sum / float64(len(loads))
	}

	// Find peak and low hours
	peakHours := []int{}
	lowHours := []int{}
	overallAvg := 0.0
	for _, avg := range hourlyAvg {
		overallAvg += avg
	}
	overallAvg /= float64(len(hourlyAvg))

	for hour, avg := range hourlyAvg {
		if avg > overallAvg*1.2 {
			peakHours = append(peakHours, hour)
		} else if avg < overallAvg*0.8 {
			lowHours = append(lowHours, hour)
		}
	}

	// Find peak and low days
	peakDays := []int{}
	lowDays := []int{}
	dayOverallAvg := 0.0
	for _, avg := range dayAvg {
		dayOverallAvg += avg
	}
	dayOverallAvg /= float64(len(dayAvg))

	for day, avg := range dayAvg {
		if avg > dayOverallAvg*1.2 {
			peakDays = append(peakDays, day)
		} else if avg < dayOverallAvg*0.8 {
			lowDays = append(lowDays, day)
		}
	}

	// Calculate pattern strength (coefficient of variation)
	variance := 0.0
	for _, avg := range hourlyAvg {
		variance += math.Pow(avg-overallAvg, 2)
	}
	variance /= float64(len(hourlyAvg))
	stdDev := math.Sqrt(variance)
	strength := stdDev / overallAvg
	if strength > 1.0 {
		strength = 1.0
	}

	pattern := &SeasonalPattern{
		Type:      "daily",
		Strength:  strength,
		PeakHours: peakHours,
		LowHours:  lowHours,
		PeakDays:  peakDays,
		LowDays:   lowDays,
	}

	// Determine if weekly pattern is stronger
	if len(peakDays) > 0 || len(lowDays) > 0 {
		dayVariance := 0.0
		for _, avg := range dayAvg {
			dayVariance += math.Pow(avg-dayOverallAvg, 2)
		}
		dayVariance /= float64(len(dayAvg))
		dayStdDev := math.Sqrt(dayVariance)
		dayStrength := dayStdDev / dayOverallAvg
		if dayStrength > strength {
			pattern.Type = "weekly"
			pattern.Strength = dayStrength
		}
	}

	return pattern, nil
}

// AnomalyDetector detects anomalies in system metrics
type AnomalyDetector struct {
	store        *storage.Storage
	baselineMean float64
	baselineStd  float64
	threshold    float64 // Number of standard deviations
}

// NewAnomalyDetector creates a new anomaly detector
func NewAnomalyDetector(store *storage.Storage) *AnomalyDetector {
	return &AnomalyDetector{
		store:     store,
		threshold: 3.0, // 3-sigma rule
	}
}

// Anomaly represents a detected anomaly
type Anomaly struct {
	Type        string    `json:"type"`     // "cpu", "memory", "disk", "network"
	Severity    string    `json:"severity"` // "low", "medium", "high", "critical"
	Value       float64   `json:"value"`
	Expected    float64   `json:"expected"`
	Deviation   float64   `json:"deviation"` // Number of standard deviations
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
}

// DetectAnomalies detects anomalies in current metrics compared to baseline
func (ad *AnomalyDetector) DetectAnomalies(metrics *monitoring.SystemMetrics) ([]*Anomaly, error) {
	// Update baseline if needed
	if err := ad.updateBaseline(); err != nil {
		logrus.Warnf("Failed to update baseline: %v", err)
	}

	anomalies := []*Anomaly{}

	// Check CPU usage
	if anomaly := ad.checkMetric("cpu", metrics.CPUUsage, ad.baselineMean, ad.baselineStd); anomaly != nil {
		anomalies = append(anomalies, anomaly)
	}

	// Check memory usage
	if anomaly := ad.checkMetric("memory", metrics.MemoryUsage, ad.baselineMean, ad.baselineStd); anomaly != nil {
		anomalies = append(anomalies, anomaly)
	}

	// Check disk I/O
	diskIO := float64(metrics.DiskIO.ReadBytes+metrics.DiskIO.WriteBytes) / 1024 / 1024 // MB
	if anomaly := ad.checkMetric("disk", diskIO, ad.baselineMean, ad.baselineStd); anomaly != nil {
		anomalies = append(anomalies, anomaly)
	}

	// Check network I/O
	netIO := float64(metrics.NetworkIO.BytesSent+metrics.NetworkIO.BytesRecv) / 1024 / 1024 // MB
	if anomaly := ad.checkMetric("network", netIO, ad.baselineMean, ad.baselineStd); anomaly != nil {
		anomalies = append(anomalies, anomaly)
	}

	return anomalies, nil
}

// checkMetric checks if a metric value is anomalous
func (ad *AnomalyDetector) checkMetric(metricType string, value, mean, std float64) *Anomaly {
	if std == 0 {
		return nil // No baseline yet
	}

	deviation := (value - mean) / std

	if math.Abs(deviation) < ad.threshold {
		return nil // Not anomalous
	}

	severity := "low"
	if math.Abs(deviation) >= 4.0 {
		severity = "critical"
	} else if math.Abs(deviation) >= 3.5 {
		severity = "high"
	} else if math.Abs(deviation) >= 3.0 {
		severity = "medium"
	}

	description := ""
	if deviation > 0 {
		description = fmt.Sprintf("%s usage is %.1f%% above normal (%.1f standard deviations)",
			metricType, (value - mean), deviation)
	} else {
		description = fmt.Sprintf("%s usage is %.1f%% below normal (%.1f standard deviations)",
			metricType, (mean - value), math.Abs(deviation))
	}

	return &Anomaly{
		Type:        metricType,
		Severity:    severity,
		Value:       value,
		Expected:    mean,
		Deviation:   deviation,
		Timestamp:   time.Now(),
		Description: description,
	}
}

// updateBaseline updates the baseline statistics from historical data
func (ad *AnomalyDetector) updateBaseline() error {
	end := time.Now()
	start := end.Add(-7 * 24 * time.Hour) // Last 7 days

	metrics, err := ad.store.GetSystemMetrics(start, end, 10000)
	if err != nil {
		return err
	}

	if len(metrics) < 10 {
		return nil // Not enough data
	}

	// Calculate mean and standard deviation of combined load
	loads := make([]float64, len(metrics))
	for i, m := range metrics {
		loads[i] = (m.CPUUsage + m.MemoryUsage) / 2.0
	}

	mean := 0.0
	for _, load := range loads {
		mean += load
	}
	mean /= float64(len(loads))

	variance := 0.0
	for _, load := range loads {
		variance += math.Pow(load-mean, 2)
	}
	variance /= float64(len(loads))
	std := math.Sqrt(variance)

	ad.baselineMean = mean
	ad.baselineStd = std

	return nil
}

// LSTMPredictor uses LSTM-like approach for time series prediction
type LSTMPredictor struct {
	store      *storage.Storage
	windowSize int
}

// NewLSTMPredictor creates a new LSTM predictor
func NewLSTMPredictor(store *storage.Storage) *LSTMPredictor {
	return &LSTMPredictor{
		store:      store,
		windowSize: 24, // 24 hours of data
	}
}

// PredictNextHour predicts the system load for the next hour
func (lp *LSTMPredictor) PredictNextHour() (float64, error) {
	end := time.Now()
	start := end.Add(-time.Duration(lp.windowSize) * time.Hour)

	metrics, err := lp.store.GetSystemMetrics(start, end, lp.windowSize*2)
	if err != nil {
		return 0, err
	}

	if len(metrics) < 10 {
		return 50.0, nil // Default prediction
	}

	// Simple moving average with exponential weighting
	weights := make([]float64, len(metrics))
	totalWeight := 0.0

	for i := range metrics {
		// Exponential weighting: more recent = higher weight
		weight := math.Exp(float64(i) * 0.1)
		weights[i] = weight
		totalWeight += weight
	}

	prediction := 0.0
	for i, m := range metrics {
		load := (m.CPUUsage + m.MemoryUsage) / 2.0
		prediction += load * (weights[i] / totalWeight)
	}

	// Add trend component
	if len(metrics) >= 2 {
		recent := (metrics[0].CPUUsage + metrics[0].MemoryUsage) / 2.0
		older := (metrics[len(metrics)-1].CPUUsage + metrics[len(metrics)-1].MemoryUsage) / 2.0
		trend := (recent - older) / float64(len(metrics))
		prediction += trend
	}

	// Apply seasonal adjustment
	hour := time.Now().Hour()
	seasonalAdjustment := lp.getSeasonalAdjustment(hour)
	prediction = prediction * seasonalAdjustment

	return prediction, nil
}

// getSeasonalAdjustment returns seasonal adjustment factor for a given hour
func (lp *LSTMPredictor) getSeasonalAdjustment(hour int) float64 {
	// Simple sinusoidal pattern: lower load at night (0-6), higher during day (9-17)
	if hour >= 0 && hour < 6 {
		return 0.7 // 30% reduction
	} else if hour >= 9 && hour < 17 {
		return 1.2 // 20% increase
	}
	return 1.0 // Normal
}
