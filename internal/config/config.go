package config

import "fmt"

type OrchestratorConfig struct {
	Port                  int   `yaml:"port" env:"ORCHESTRATOR_PORT"`
	TimeAdditionMS        int64 `yaml:"time_addition_ms" env:"TIME_ADDITION_MS"`
	TimeSubtractionMS     int64 `yaml:"time_subtraction_ms" env:"TIME_SUBTRACTION_MS"`
	TimeMultiplicationsMS int64 `yaml:"time_multiplications_ms" env:"TIME_MULTIPLICATIONS_MS"`
	TimeDivisionsMS       int64 `yaml:"time_divisions_ms" env:"TIME_DIVISIONS_MS"`
}

type AgentConfig struct {
	OrchestratorURL string `yaml:"orchestrator_url" env:"ORCHESTRATOR_URL"`
	ComputingPower  int64  `yaml:"computing_power" env:"COMPUTING_POWER"` // Количество запускаемых горутин для каждого агента
}

type Config struct {
	Orchestrator OrchestratorConfig `yaml:"orchestrator"`
	Agent        AgentConfig        `yaml:"agent"`

	Logging struct {
		ToFile   bool   `yaml:"to_file" env:"TO_FILE"`
		Dir      string `yaml:"logging_dir" env:"LOGGING_DIR"`       // Дирректория для логирования
		Format   string `yaml:"logging_format" env:"LOGGING_FORMAT"` // Формат логирования, поддерживаемые форматы: json, текстовый (по стандарту - текстовый)
		MaxSize  int    `yaml:"logging_file_max_size" env:"LOGGING_FILE_MAX_SIZE"`
		MaxFiles int    `yaml:"logging_max_filex" env:"LOGGING_MAX_FILES"`
	} `yaml:"logging" env:"LOGGING"`
}

func NewDefaultConfig() *Config {
	cfg := &Config{}

	cfg.Orchestrator.Port = 8080
	cfg.Orchestrator.TimeAdditionMS = 5000
	cfg.Orchestrator.TimeSubtractionMS = 5000
	cfg.Orchestrator.TimeMultiplicationsMS = 10000
	cfg.Orchestrator.TimeDivisionsMS = 10000

	cfg.Agent.OrchestratorURL = "http://localhost:8080"
	cfg.Agent.ComputingPower = 5

	cfg.Logging.ToFile = false
	cfg.Logging.Format = "json"
	cfg.Logging.MaxSize = 10
	cfg.Logging.MaxFiles = 3

	return cfg
}

func (c *Config) Validate() error {
	if c.Orchestrator.Port <= 0 || c.Orchestrator.Port > 65535 {
		return fmt.Errorf("invalid orchestrator port: %d", c.Orchestrator.Port)
	}

	if c.Orchestrator.TimeAdditionMS <= 0 {
		return fmt.Errorf("invalid time additions: %d", c.Orchestrator.TimeAdditionMS)
	}

	if c.Orchestrator.TimeDivisionsMS <= 0 {
		return fmt.Errorf("invalid time divisions: %d", c.Orchestrator.TimeDivisionsMS)
	}

	if c.Orchestrator.TimeMultiplicationsMS <= 0 {
		return fmt.Errorf("invalid time multiplications: %d", c.Orchestrator.TimeMultiplicationsMS)
	}

	if c.Orchestrator.TimeSubtractionMS <= 0 {
		return fmt.Errorf("invalid time substractions: %d", c.Orchestrator.TimeSubtractionMS)
	}

	if c.Agent.ComputingPower <= 0 {
		return fmt.Errorf("invalid computing power: %d", c.Agent.ComputingPower)
	}

	if len(c.Agent.OrchestratorURL) == 0 {
		return fmt.Errorf("invalid orchestrator URL: %s", c.Agent.OrchestratorURL)
	}

	return nil
}
