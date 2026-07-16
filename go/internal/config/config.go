package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	OpenAI    OpenAIConfig    `yaml:"openai"`
	LLM       LLMConfig       `yaml:"llm"`
	Embedding EmbeddingConfig `yaml:"embedding"`
	Vector    VectorConfig    `yaml:"vector"`
	Database  DatabaseConfig  `yaml:"database"`
	Engine    EngineConfig    `yaml:"engine"`
}

type ServerConfig struct {
	Addr       string        `yaml:"addr"`
	SessionTTL time.Duration `yaml:"session_ttl"`
}

type OpenAIConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
}

type LLMConfig struct {
	BaseURL     string  `yaml:"base_url"`
	APIKey      string  `yaml:"api_key"`
	Model       string  `yaml:"model"`
	Temperature float64 `yaml:"temperature"`
	MaxTokens   int     `yaml:"max_tokens"`
}

type EmbeddingConfig struct {
	BaseURL   string `yaml:"base_url"`
	APIKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	Dimension int    `yaml:"dimension"`
}

type VectorConfig struct {
	Path     string         `yaml:"path"`
	NResults NResultsConfig `yaml:"n_results"`
}

type NResultsConfig struct {
	DDL           int `yaml:"ddl"`
	Documentation int `yaml:"documentation"`
	SQL           int `yaml:"sql"`
}

type DatabaseConfig struct {
	Driver  string `yaml:"driver"`
	DSN     string `yaml:"dsn"`
	Dialect string `yaml:"dialect"`
}

type EngineConfig struct {
	AllowLLMToSeeData bool   `yaml:"allow_llm_to_see_data"`
	AutoTrain         bool   `yaml:"auto_train"`
	Visualize         bool   `yaml:"visualize"`
	Language          string `yaml:"language"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	expanded := os.ExpandEnv(string(data))
	cfg := Default()
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.applyDefaults()
	return cfg, nil
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Addr:       ":8080",
			SessionTTL: 30 * time.Minute,
		},
		OpenAI: OpenAIConfig{
			BaseURL: "https://api.openai.com/v1",
		},
		LLM: LLMConfig{
			Model:       "gpt-4o",
			Temperature: 0.7,
			MaxTokens:   14000,
		},
		Embedding: EmbeddingConfig{
			Model:     "text-embedding-3-small",
			Dimension: 1536,
		},
		Vector: VectorConfig{
			Path: "./data/chromem",
			NResults: NResultsConfig{
				DDL:           10,
				Documentation: 10,
				SQL:           10,
			},
		},
		Engine: EngineConfig{
			AutoTrain: true,
			Visualize: true,
		},
	}
}

func (c *Config) applyDefaults() {
	d := Default()
	if c.Server.Addr == "" {
		c.Server.Addr = d.Server.Addr
	}
	if c.Server.SessionTTL == 0 {
		c.Server.SessionTTL = d.Server.SessionTTL
	}
	if c.OpenAI.BaseURL == "" {
		c.OpenAI.BaseURL = d.OpenAI.BaseURL
	}
	if c.LLM.Model == "" {
		c.LLM.Model = d.LLM.Model
	}
	if c.LLM.Temperature == 0 {
		c.LLM.Temperature = d.LLM.Temperature
	}
	if c.LLM.MaxTokens == 0 {
		c.LLM.MaxTokens = d.LLM.MaxTokens
	}
	if c.Embedding.Model == "" {
		c.Embedding.Model = d.Embedding.Model
	}
	if c.Embedding.Dimension == 0 {
		c.Embedding.Dimension = d.Embedding.Dimension
	}
	if c.Vector.Path == "" {
		c.Vector.Path = d.Vector.Path
	}
	if c.Vector.NResults.DDL == 0 {
		c.Vector.NResults.DDL = d.Vector.NResults.DDL
	}
	if c.Vector.NResults.Documentation == 0 {
		c.Vector.NResults.Documentation = d.Vector.NResults.Documentation
	}
	if c.Vector.NResults.SQL == 0 {
		c.Vector.NResults.SQL = d.Vector.NResults.SQL
	}
}

func (c *Config) LLMBaseURL() string {
	if strings.TrimSpace(c.LLM.BaseURL) != "" {
		return strings.TrimRight(c.LLM.BaseURL, "/")
	}
	return strings.TrimRight(c.OpenAI.BaseURL, "/")
}

func (c *Config) LLMAPIKey() string {
	if strings.TrimSpace(c.LLM.APIKey) != "" {
		return c.LLM.APIKey
	}
	return c.OpenAI.APIKey
}

func (c *Config) EmbeddingBaseURL() string {
	if strings.TrimSpace(c.Embedding.BaseURL) != "" {
		return strings.TrimRight(c.Embedding.BaseURL, "/")
	}
	return strings.TrimRight(c.OpenAI.BaseURL, "/")
}

func (c *Config) EmbeddingAPIKey() string {
	if strings.TrimSpace(c.Embedding.APIKey) != "" {
		return c.Embedding.APIKey
	}
	return c.OpenAI.APIKey
}

func (c *Config) SQLDialect() string {
	if strings.TrimSpace(c.Database.Dialect) != "" {
		return c.Database.Dialect
	}
	switch strings.ToLower(c.Database.Driver) {
	case "mysql":
		return "MySQL"
	case "duckdb":
		return "DuckDB"
	default:
		return "SQL"
	}
}
