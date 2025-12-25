package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config represents the main configuration structure
type Config struct {
	Server   ServerConfig   `yaml:"server" mapstructure:"server"`
	Database DatabaseConfig `yaml:"database" mapstructure:"database"`
	Jobs     []JobConfig    `yaml:"jobs" mapstructure:"jobs"`
	ML       MLConfig       `yaml:"ml" mapstructure:"ml"`
	Logging  LoggingConfig  `yaml:"logging" mapstructure:"logging"`
	Advanced AdvancedConfig `yaml:"advanced" mapstructure:"advanced"`
	Alerts   AlertsConfig   `yaml:"alerts" mapstructure:"alerts"`
	Thresholds ThresholdsConfig `yaml:"thresholds" mapstructure:"thresholds"`
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host         string        `yaml:"host" mapstructure:"host"`
	Port         int           `yaml:"port" mapstructure:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout" mapstructure:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout" mapstructure:"write_timeout"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Driver   string `yaml:"driver" mapstructure:"driver"`
	DSN      string `yaml:"dsn" mapstructure:"dsn"`
	MaxConns int    `yaml:"max_conns" mapstructure:"max_conns"`
}

// JobConfig represents a single job configuration
type JobConfig struct {
	Name        string            `yaml:"name" mapstructure:"name"`
	Command     string            `yaml:"command" mapstructure:"command"`
	Type        string            `yaml:"type" mapstructure:"type"`
	Schedule    string            `yaml:"schedule" mapstructure:"schedule"`
	Timeout     time.Duration     `yaml:"timeout" mapstructure:"timeout"`
	Retries     int               `yaml:"retries" mapstructure:"retries"`
	Environment map[string]string `yaml:"environment" mapstructure:"environment"`
	Priority    int               `yaml:"priority" mapstructure:"priority"`
}

// MLConfig holds machine learning configuration
type MLConfig struct {
	ModelPath     string        `yaml:"model_path" mapstructure:"model_path"`
	TrainingData  string        `yaml:"training_data" mapstructure:"training_data"`
	UpdateInterval time.Duration `yaml:"update_interval" mapstructure:"update_interval"`
	Features      []string      `yaml:"features" mapstructure:"features"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level      string `yaml:"level" mapstructure:"level"`
	Format     string `yaml:"format" mapstructure:"format"`
	OutputFile string `yaml:"output_file" mapstructure:"output_file"`
}

// AdvancedConfig holds advanced configuration
type AdvancedConfig struct {
	MetricsInterval    time.Duration `yaml:"metrics_interval" mapstructure:"metrics_interval"`
	AdjustmentThreshold int          `yaml:"adjustment_threshold" mapstructure:"adjustment_threshold"`
	MaxConcurrentJobs  int          `yaml:"max_concurrent_jobs" mapstructure:"max_concurrent_jobs"`
	JobQueueSize       int          `yaml:"job_queue_size" mapstructure:"job_queue_size"`
	CleanupAfter       time.Duration `yaml:"cleanup_after" mapstructure:"cleanup_after"`
	EnableDashboard    bool         `yaml:"enable_dashboard" mapstructure:"enable_dashboard"`
	DashboardAuth      DashboardAuthConfig `yaml:"dashboard_auth" mapstructure:"dashboard_auth"`
	Prometheus         PrometheusConfig    `yaml:"prometheus" mapstructure:"prometheus"`
	EnableAlerts       bool         `yaml:"enable_alerts" mapstructure:"enable_alerts"`
}

// DashboardAuthConfig holds dashboard authentication configuration
type DashboardAuthConfig struct {
	Enabled  bool   `yaml:"enabled" mapstructure:"enabled"`
	Username string `yaml:"username" mapstructure:"username"`
	Password string `yaml:"password" mapstructure:"password"`
}

// PrometheusConfig holds Prometheus metrics configuration
type PrometheusConfig struct {
	Enabled bool   `yaml:"enabled" mapstructure:"enabled"`
	Path    string `yaml:"path" mapstructure:"path"`
	Port    int    `yaml:"port" mapstructure:"port"`
}

// AlertsConfig holds alerting configuration
type AlertsConfig struct {
	Enabled bool          `yaml:"enabled" mapstructure:"enabled"`
	Email   EmailConfig   `yaml:"email" mapstructure:"email"`
	Slack   SlackConfig   `yaml:"slack" mapstructure:"slack"`
	Webhook WebhookConfig `yaml:"webhook" mapstructure:"webhook"`
}

// EmailConfig holds email alert configuration
type EmailConfig struct {
	Enabled  bool     `yaml:"enabled" mapstructure:"enabled"`
	SMTPHost string   `yaml:"smtp_host" mapstructure:"smtp_host"`
	SMTPPort int      `yaml:"smtp_port" mapstructure:"smtp_port"`
	Username string   `yaml:"username" mapstructure:"username"`
	Password string   `yaml:"password" mapstructure:"password"`
	From     string   `yaml:"from" mapstructure:"from"`
	To       []string `yaml:"to" mapstructure:"to"`
}

// SlackConfig holds Slack alert configuration
type SlackConfig struct {
	Enabled    bool   `yaml:"enabled" mapstructure:"enabled"`
	WebhookURL string `yaml:"webhook_url" mapstructure:"webhook_url"`
	Channel    string `yaml:"channel" mapstructure:"channel"`
	Username   string `yaml:"username" mapstructure:"username"`
}

// WebhookConfig holds webhook alert configuration
type WebhookConfig struct {
	Enabled bool              `yaml:"enabled" mapstructure:"enabled"`
	URL     string            `yaml:"url" mapstructure:"url"`
	Method  string            `yaml:"method" mapstructure:"method"`
	Headers map[string]string `yaml:"headers" mapstructure:"headers"`
}

// ThresholdsConfig holds monitoring thresholds
type ThresholdsConfig struct {
	CPU     ThresholdLevels `yaml:"cpu" mapstructure:"cpu"`
	Memory  ThresholdLevels `yaml:"memory" mapstructure:"memory"`
	Disk    ThresholdLevels `yaml:"disk" mapstructure:"disk"`
	Network ThresholdLevels `yaml:"network" mapstructure:"network"`
}

// ThresholdLevels holds warning and critical thresholds
type ThresholdLevels struct {
	Warning  float64 `yaml:"warning" mapstructure:"warning"`
	Critical float64 `yaml:"critical" mapstructure:"critical"`
}

// Load loads configuration from file
func Load(configPath string) (*Config, error) {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config if it doesn't exist
		if err := createDefaultConfig(configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %v", err)
		}
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// Set defaults for missing values
	setDefaults(&config)

	return &config, nil
}

// createDefaultConfig creates a default configuration file
func createDefaultConfig(configPath string) error {
	// Ensure directory exists
	dir := "config"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	defaultConfig := &Config{
		Server: ServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Database: DatabaseConfig{
			Driver:   "sqlite",
			DSN:      "arcron.db",
			MaxConns: 10,
		},
		Jobs: []JobConfig{
			{
				Name:        "backup",
				Command:     "rsync -av /data /backup",
				Type:        "resource-intensive",
				Schedule:    "0 2 * * *",
				Timeout:     1 * time.Hour,
				Retries:     3,
				Priority:    1,
				Environment: map[string]string{},
			},
			{
				Name:        "logrotate",
				Command:     "logrotate /etc/logrotate.conf",
				Type:        "light",
				Schedule:    "0 0 * * *",
				Timeout:     5 * time.Minute,
				Retries:     1,
				Priority:    5,
				Environment: map[string]string{},
			},
		},
		ML: MLConfig{
			ModelPath:      "models/arcron_model",
			TrainingData:   "data/metrics.csv",
			UpdateInterval: 24 * time.Hour,
			Features:       []string{"cpu_usage", "memory_usage", "io_wait", "network_io"},
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			OutputFile: "logs/arcron.log",
		},
	}

	data, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal default config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write default config: %v", err)
	}

	return nil
}

// setDefaults sets default values for missing configuration
func setDefaults(config *Config) {
	if config.Server.Host == "" {
		config.Server.Host = "localhost"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.ReadTimeout == 0 {
		config.Server.ReadTimeout = 30 * time.Second
	}
	if config.Server.WriteTimeout == 0 {
		config.Server.WriteTimeout = 30 * time.Second
	}

	if config.Database.Driver == "" {
		config.Database.Driver = "sqlite"
	}
	if config.Database.DSN == "" {
		config.Database.DSN = "arcron.db"
	}
	if config.Database.MaxConns == 0 {
		config.Database.MaxConns = 10
	}

	if config.ML.UpdateInterval == 0 {
		config.ML.UpdateInterval = 24 * time.Hour
	}
	if len(config.ML.Features) == 0 {
		config.ML.Features = []string{"cpu_usage", "memory_usage", "io_wait", "network_io"}
	}

	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.Format == "" {
		config.Logging.Format = "json"
	}

	// Advanced defaults
	if config.Advanced.MetricsInterval == 0 {
		config.Advanced.MetricsInterval = 5 * time.Second
	}
	if config.Advanced.AdjustmentThreshold == 0 {
		config.Advanced.AdjustmentThreshold = 5
	}
	if config.Advanced.MaxConcurrentJobs == 0 {
		config.Advanced.MaxConcurrentJobs = 10
	}
	if config.Advanced.JobQueueSize == 0 {
		config.Advanced.JobQueueSize = 100
	}
	if config.Advanced.CleanupAfter == 0 {
		config.Advanced.CleanupAfter = 168 * time.Hour // 7 days
	}
	if !config.Advanced.Prometheus.Enabled {
		config.Advanced.Prometheus.Path = "/metrics"
		config.Advanced.Prometheus.Port = 9090
	}
}
