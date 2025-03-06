package agent

import (
	"context"
	"encoding/json"
	"final3/internal/config"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestAgent(t *testing.T) {
	var mu sync.Mutex
	nextTaskID := 1
	completedTasks := make(map[int]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/task" {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}

		if r.Method == http.MethodGet {
			mu.Lock()
			if nextTaskID <= 5 {
				taskID := nextTaskID
				nextTaskID++
				mu.Unlock()

				task := Task{
					ID:            taskID,
					Arg1:          float64(taskID),
					Arg2:          2.0,
					Operation:     "+",
					OperationTime: 50 * time.Millisecond,
				}
				json.NewEncoder(w).Encode(task)
			} else {
				mu.Unlock()
				http.Error(w, "No tasks available", http.StatusNotFound)
			}
			return
		}

		if r.Method == http.MethodPost {
			var result struct {
				ID     int     `json:"id"`
				Result float64 `json:"result"`
				Error  string  `json:"error,omitempty"`
			}

			if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			expectedResult := float64(result.ID) + 2.0
			if result.Result != expectedResult {
				t.Errorf("Task %d: expected result %f, got %f", result.ID, expectedResult, result.Result)
			}

			mu.Lock()
			completedTasks[result.ID] = true
			mu.Unlock()

			w.WriteHeader(http.StatusOK)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}))
	defer server.Close()

	cfg := &config.AgentConfig{
		OrchestratorURL: server.URL,
		ComputingPower:  2,
	}

	agent, err := NewAgent(cfg)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err = agent.Run(ctx)

	if err != nil && err.Error() != "context deadline exceeded" {
		t.Errorf("Agent.Run returned unexpected error: %v", err)
	}

	mu.Lock()
	completedTaskCount := len(completedTasks)
	mu.Unlock()

	if completedTaskCount != 5 {
		t.Errorf("Expected 5 tasks to be processed, got %d", completedTaskCount)
	}
}
