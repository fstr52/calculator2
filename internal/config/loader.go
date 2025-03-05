package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Загрузка конфига
//
// configPath - путь к файлу конфигурации
//
// configType - тип приложения (оркестратор или агент)
func LoadConfig(configPath string, configType string) (*Config, error) {
	path := FindConfigFile(configPath, configType)
	fmt.Printf("Loading %s config from: %s\n", configType, path)

	config := NewDefaultConfig()

	if path != "" {
		fmt.Printf("Loading %s config from: %s\n", configType, path)
		if err := loadFromFile(config, path); err != nil {
			return nil, fmt.Errorf("loading from file: %v", err)
		}
	}

	if err := LoadFromEnv(config); err != nil {
		return nil, fmt.Errorf("loading from env: %v", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config (validating error): %v", err)
	}

	return config, nil
}

// Загрузка конфигурации из файла
func loadFromFile(config *Config, path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if err := yaml.NewDecoder(file).Decode(config); err != nil {
		return err
	}

	return nil
}

// Нахождения фалйга конфигурации по заданному пути, в случае ненахода будет производиться поиск
// по стандартным путям нахождения
func FindConfigFile(providedPath string, configType string) string {
	if providedPath != "" {
		absPath, err := filepath.Abs(providedPath)
		if err == nil {
			if _, err := os.Stat(absPath); err == nil {
				fmt.Printf("Found config at: %s\n", absPath)
				return absPath
			}
		}
	}

	var fileNames []string
	switch configType {
	case "orchestrator":
		fileNames = []string{"orchestrator.yml", "orchestrator.yaml", "config.yml", "config.yaml"}
	case "agent":
		fileNames = []string{"agent.yml", "agent.yaml", "agents.yml", "agents.yaml"}
	default:
		fileNames = []string{"config.yml", "config.yaml"}
	}

	basePaths := []string{
		"/app/configs/",
		"./configs/",
		filepath.Join(os.Getenv("HOME"), ".config/myapp/"),
		"/etc/myapp/",
	}

	for _, basePath := range basePaths {
		for _, fileName := range fileNames {
			path := filepath.Join(basePath, fileName)
			fmt.Printf("Checking path: %s\n", path)
			if _, err := os.Stat(path); err == nil {
				absPath, err := filepath.Abs(path)
				if err == nil {
					fmt.Printf("Found config at: %s\n", absPath)
					return absPath
				}
			}
		}
	}

	fmt.Println("Config file not found, returning empty path")
	return ""
}
