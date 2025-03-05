package config

import (
	"os"
	"strconv"
)

// Загрузка конфигурационных переменных из окружения
func LoadFromEnv(config *Config) error {
	if env := os.Getenv("ORCHESTRATOR_PORT"); env != "" {
		if val, err := strconv.Atoi(env); err == nil {
			config.Orchestrator.Port = val
		}
	}

	if env := os.Getenv("TIME_ADDITION_MS"); env != "" {
		if val, err := strconv.ParseInt(env, 10, 64); err == nil {
			config.Orchestrator.TimeAdditionMS = val
		}
	}

	if env := os.Getenv("TIME_SUBTRACTION_MS"); env != "" {
		if val, err := strconv.ParseInt(env, 10, 64); err == nil {
			config.Orchestrator.TimeSubtractionMS = val
		}
	}

	if env := os.Getenv("TIME_MULTIPLICATIONS_MS"); env != "" {
		if val, err := strconv.ParseInt(env, 10, 64); err == nil {
			config.Orchestrator.TimeMultiplicationsMS = val
		}
	}

	if env := os.Getenv("TIME_DIVISIONS_MS"); env != "" {
		if val, err := strconv.ParseInt(env, 10, 64); err == nil {
			config.Orchestrator.TimeDivisionsMS = val
		}
	}

	if env := os.Getenv("ORCHESTRATOR_URL"); env != "" {
		config.Agent.OrchestratorURL = env
	}

	if env := os.Getenv("COMPUTING_POWER"); env != "" {
		if val, err := strconv.ParseInt(env, 10, 64); err == nil {
			config.Agent.ComputingPower = val
		}
	}

	if env := os.Getenv("TO_FILE"); env != "" {
		config.Logging.ToFile = env == "true"
	}

	if env := os.Getenv("LOGGING_DIR"); env != "" {
		config.Logging.Dir = env
	}

	return nil
}
