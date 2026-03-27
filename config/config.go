// Package config provides application configuration loading from YAML files
// and environment variables.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level application configuration.
type Config struct {
	API          APIConfig          `yaml:"api"`
	DB           DBConfig           `yaml:"db"`
	PayloadStore PayloadStoreConfig `yaml:"payload_store"`
	Broker       BrokerConfig       `yaml:"broker"`
	Jobs         map[string]FlowJobsConfig `yaml:"jobs"`
	Metrics      MetricsConfig      `yaml:"metrics"`
}

// APIConfig holds HTTP server settings.
type APIConfig struct {
	HttpPort int `yaml:"http_port"`
}

// DBConfig holds MySQL database connection settings.
type DBConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Name         string `yaml:"name"`
	User         string `yaml:"user"`
	Password     string `yaml:"password"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

// DSN returns a MySQL data source name string.
func (d DBConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		d.User, d.Password, d.Host, d.Port, d.Name)
}

// PayloadStoreConfig holds payload storage settings.
// Type can be "redis" or "memory".
type PayloadStoreConfig struct {
	Type string `yaml:"type"`
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// Address returns the host:port string for the payload store.
func (p PayloadStoreConfig) Address() string {
	return fmt.Sprintf("%s:%d", p.Host, p.Port)
}

// BrokerConfig holds NATS broker settings.
type BrokerConfig struct {
	URL string `yaml:"url"`
}

// FlowJobsConfig holds the three job configurations for a single flow.
type FlowJobsConfig struct {
	Awaiting  *JobParams `yaml:"awaiting"`
	Scheduled *JobParams `yaml:"scheduled"`
	Suspended *JobParams `yaml:"suspended"`
}

// JobParams holds settings for a single scheduled job.
type JobParams struct {
	IntervalMs     int `yaml:"interval_ms"`
	StartupDelayMs int `yaml:"startup_delay_ms"`
	BatchSize      int `yaml:"batch_size"`
	MaxItems       int `yaml:"max_items"`
	MaxFails       int `yaml:"max_fails"`
	MaxRuntimeMs   int `yaml:"max_runtime_ms"`
}

// MetricsConfig holds Prometheus metrics settings.
type MetricsConfig struct {
	Enabled bool `yaml:"enabled"`
}

// Load reads the YAML configuration file at the given path and returns a
// populated Config. Environment variables can override values after loading
// (not yet implemented — extend as needed).
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	applyDefaults(cfg)

	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.API.HttpPort == 0 {
		cfg.API.HttpPort = 8080
	}
	if cfg.DB.Host == "" {
		cfg.DB.Host = "localhost"
	}
	if cfg.DB.Port == 0 {
		cfg.DB.Port = 3306
	}
	if cfg.DB.MaxOpenConns == 0 {
		cfg.DB.MaxOpenConns = 80
	}
	if cfg.DB.MaxIdleConns == 0 {
		cfg.DB.MaxIdleConns = 10
	}
	if cfg.PayloadStore.Type == "" {
		cfg.PayloadStore.Type = "memory"
	}
	if cfg.Broker.URL == "" {
		cfg.Broker.URL = "nats://localhost:4222"
	}
}
