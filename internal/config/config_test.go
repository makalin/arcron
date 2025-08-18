package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Test loading config from non-existent file (should create default)
	configPath := "test_config.yaml"
	defer os.Remove(configPath)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check default values
	if cfg.Server.Host != "localhost" {
		t.Errorf("Expected server host to be 'localhost', got '%s'", cfg.Server.Host)
	}

	if cfg.Server.Port != 8080 {
		t.Errorf("Expected server port to be 8080, got %d", cfg.Server.Port)
	}

	if len(cfg.Jobs) != 2 {
		t.Errorf("Expected 2 default jobs, got %d", len(cfg.Jobs))
	}

	// Check first job
	if cfg.Jobs[0].Name != "backup" {
		t.Errorf("Expected first job name to be 'backup', got '%s'", cfg.Jobs[0].Name)
	}

	if cfg.Jobs[0].Type != "resource-intensive" {
		t.Errorf("Expected first job type to be 'resource-intensive', got '%s'", cfg.Jobs[0].Type)
	}
}

func TestConfigDefaults(t *testing.T) {
	// Test with minimal config
	minimalConfig := &Config{
		Jobs: []JobConfig{
			{
				Name:     "test",
				Command:  "echo test",
				Schedule: "0 * * * *",
			},
		},
	}

	setDefaults(minimalConfig)

	// Check that defaults are set
	if minimalConfig.Server.Host != "localhost" {
		t.Errorf("Expected default server host to be 'localhost', got '%s'", minimalConfig.Server.Host)
	}

	if minimalConfig.Server.Port != 8080 {
		t.Errorf("Expected default server port to be 8080, got %d", minimalConfig.Server.Port)
	}

	if minimalConfig.Database.Driver != "sqlite" {
		t.Errorf("Expected default database driver to be 'sqlite', got '%s'", minimalConfig.Database.Driver)
	}
}

func TestJobConfigValidation(t *testing.T) {
	// Test valid job config
	validJob := JobConfig{
		Name:     "valid",
		Command:  "echo hello",
		Type:     "light",
		Schedule: "0 * * * *",
		Timeout:  5 * time.Minute,
		Retries:  3,
		Priority: 1,
	}

	if validJob.Name == "" {
		t.Error("Job name should not be empty")
	}

	if validJob.Command == "" {
		t.Error("Job command should not be empty")
	}

	// Test job with environment variables
	envJob := JobConfig{
		Name:        "env_test",
		Command:     "echo $TEST_VAR",
		Environment: map[string]string{"TEST_VAR": "test_value"},
	}

	if len(envJob.Environment) != 1 {
		t.Errorf("Expected 1 environment variable, got %d", len(envJob.Environment))
	}

	if envJob.Environment["TEST_VAR"] != "test_value" {
		t.Errorf("Expected TEST_VAR to be 'test_value', got '%s'", envJob.Environment["TEST_VAR"])
	}
}
