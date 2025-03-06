package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configPath  string
		configType  string
		setupFunc   func() (string, func())
		expectError bool
	}{
		{
			name:       "ValidOrchestratorConfig",
			configPath: "",
			configType: "orchestrator",
			setupFunc: func() (string, func()) {
				dir, err := os.MkdirTemp("", "config-test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				configDir := filepath.Join(dir, "configs")
				err = os.Mkdir(configDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create configs dir: %v", err)
				}

				configPath := filepath.Join(configDir, "orchestrator.yml")
				configContent := `
orchestrator:
  port: 8080
  time_addition_ms: 100
  time_subtraction_ms: 100
  time_multiplications_ms: 200
  time_divisions_ms: 200
`
				err = os.WriteFile(configPath, []byte(configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}

				oldDir, err := os.Getwd()
				if err != nil {
					t.Fatalf("Failed to get current dir: %v", err)
				}

				err = os.Chdir(dir)
				if err != nil {
					t.Fatalf("Failed to change dir: %v", err)
				}

				return configPath, func() {
					os.Chdir(oldDir)
					os.RemoveAll(dir)
				}
			},
			expectError: false,
		},
		{
			name:       "ValidAgentConfig",
			configPath: "",
			configType: "agent",
			setupFunc: func() (string, func()) {
				dir, err := os.MkdirTemp("", "config-test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				configDir := filepath.Join(dir, "configs")
				err = os.Mkdir(configDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create configs dir: %v", err)
				}

				configPath := filepath.Join(configDir, "agent.yml")
				configContent := `
agent:
  orchestrator_url: "http://localhost:8080"
  poll_interval_ms: 1000
  operation_types: ["+", "-", "*", "/"]
`
				err = os.WriteFile(configPath, []byte(configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}

				oldDir, err := os.Getwd()
				if err != nil {
					t.Fatalf("Failed to get current dir: %v", err)
				}

				err = os.Chdir(dir)
				if err != nil {
					t.Fatalf("Failed to change dir: %v", err)
				}

				return configPath, func() {
					os.Chdir(oldDir)
					os.RemoveAll(dir)
				}
			},
			expectError: false,
		},
		{
			name:       "ExplicitConfigPath",
			configPath: "explicit_config.yml",
			configType: "orchestrator",
			setupFunc: func() (string, func()) {
				dir, err := os.MkdirTemp("", "config-test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				configPath := filepath.Join(dir, "explicit_config.yml")
				configContent := `
orchestrator:
  port: 8080
  time_addition_ms: 100
  time_subtraction_ms: 100
  time_multiplications_ms: 200
  time_divisions_ms: 200
`
				err = os.WriteFile(configPath, []byte(configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}

				return configPath, func() {
					os.RemoveAll(dir)
				}
			},
			expectError: false,
		},
		{
			name:       "InvalidConfigYaml",
			configPath: "",
			configType: "orchestrator",
			setupFunc: func() (string, func()) {
				dir, err := os.MkdirTemp("", "config-test")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				configDir := filepath.Join(dir, "configs")
				err = os.Mkdir(configDir, 0755)
				if err != nil {
					t.Fatalf("Failed to create configs dir: %v", err)
				}

				configPath := filepath.Join(configDir, "orchestrator.yml")
				configContent := `
orchestrator:
  port: "not-a-number"
  time_addition_ms: 100
  time_subtraction_ms: 100
  time_multiplications_ms: 200
  time_divisions_ms: 200
`
				err = os.WriteFile(configPath, []byte(configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}

				oldDir, err := os.Getwd()
				if err != nil {
					t.Fatalf("Failed to get current dir: %v", err)
				}

				err = os.Chdir(dir)
				if err != nil {
					t.Fatalf("Failed to change dir: %v", err)
				}

				return configPath, func() {
					os.Chdir(oldDir)
					os.RemoveAll(dir)
				}
			},
			expectError: true,
		},
		{
			name:       "NonExistentConfig",
			configPath: "does_not_exist.yml",
			configType: "orchestrator",
			setupFunc: func() (string, func()) {
				return "", func() {}
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, cleanup := tt.setupFunc()
			defer cleanup()

			if tt.configPath == "explicit_config.yml" {
				tt.configPath = configPath
			}

			config, err := LoadConfig(tt.configPath, tt.configType)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && config == nil {
				t.Error("Expected config to be returned, got nil")
			}

			if !tt.expectError && tt.configType == "orchestrator" && config != nil {
				if config.Orchestrator.Port == 0 {
					t.Error("Expected orchestrator port to be set")
				}
			}

			if !tt.expectError && tt.configType == "agent" && config != nil {
				if config.Agent.OrchestratorURL == "" {
					t.Error("Expected agent orchestrator URL to be set")
				}
			}
		})
	}
}
