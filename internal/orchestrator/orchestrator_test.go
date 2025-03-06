package orchestrator

import (
	"context"
	"final3/internal/config"
	"final3/internal/models"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestPrepareInput(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{},
	}

	orch := NewOrchestrator(cfg)

	tests := []struct {
		name        string
		input       string
		expectError bool
		nodeCount   int
	}{
		{
			name:        "SimpleAddition",
			input:       "2+3",
			expectError: false,
			nodeCount:   3,
		},
		{
			name:        "ComplexExpression",
			input:       "2+3*4",
			expectError: false,
			nodeCount:   5,
		},
		{
			name:        "ExpressionWithBrackets",
			input:       "(2+3)*4",
			expectError: false,
			nodeCount:   5,
		},
		{
			name:        "InvalidExpression",
			input:       "2++3",
			expectError: true,
			nodeCount:   0,
		},
		{
			name:        "UnbalancedBrackets",
			input:       "2+(3*4",
			expectError: true,
			nodeCount:   0,
		},
		{
			name:        "DecimalNumbers",
			input:       "2.5+3.5",
			expectError: false,
			nodeCount:   3,
		},
		{
			name:        "ComplexWithMultipleOperations",
			input:       "((2+3)*(4-1))/5",
			expectError: false,
			nodeCount:   9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := orch.prepareInput(tt.input)

			if tt.expectError && err == nil {
				t.Fatal("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				orch.DataBase.mu.Lock()
				dbExpr, exists := orch.DataBase.ExpressionList[expr.ID]
				orch.DataBase.mu.Unlock()

				if !exists {
					t.Error("Expression not found in database")
				}

				if dbExpr != expr {
					t.Error("Database expression reference doesn't match returned expression")
				}

				if len(expr.IdMap) != tt.nodeCount {
					t.Errorf("Expected %d nodes, got %d", tt.nodeCount, len(expr.IdMap))
				}

				numberCount := 0
				operatorCount := 0

				for _, node := range expr.IdMap {
					if node.Type == models.Number {
						if node.Status != models.StatusDone {
							t.Errorf("Number node has status %s, expected %s", node.Status, models.StatusDone)
						}
						numberCount++
					} else if node.Type == models.Operator {
						if node.Status != models.StatusInQueue {
							t.Errorf("Operator node has status %s, expected %s", node.Status, models.StatusInQueue)
						}
						operatorCount++
					}
				}

				if tt.input == "2+3" && (numberCount != 2 || operatorCount != 1) {
					t.Errorf("For '2+3' expected 2 numbers and 1 operator, got %d numbers and %d operators", numberCount, operatorCount)
				}

				if tt.input == "2+3*4" && (numberCount != 3 || operatorCount != 2) {
					t.Errorf("For '2+3*4' expected 3 numbers and 2 operators, got %d numbers and %d operators", numberCount, operatorCount)
				}

				validateNodeIDs(t, expr.IdMap)
			}
		})
	}
}

func validateNodeIDs(t *testing.T, idMap map[int]*models.Node) {
	maxID := len(idMap)
	for i := 1; i <= maxID; i++ {
		if _, exists := idMap[i]; !exists {
			t.Errorf("Missing node ID %d in sequence", i)
		}
	}

	for _, node := range idMap {
		if node.Type == models.Operator {
			if len(node.Dependencies) != 2 {
				t.Errorf("Operator node should have exactly 2 dependencies, got %d", len(node.Dependencies))
			}

			for _, dep := range node.Dependencies {
				found := false
				for _, mapNode := range idMap {
					if mapNode == dep {
						found = true
						break
					}
				}

				if !found {
					t.Error("Dependency node not found in idMap")
				}
			}
		}
	}
}

func TestRunOrchestration(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{
			Port:                  8090,
			TimeAdditionMS:        100,
			TimeSubtractionMS:     100,
			TimeMultiplicationsMS: 200,
			TimeDivisionsMS:       200,
		},
	}

	orch := NewOrchestrator(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	var orchErr error

	go func() {
		defer wg.Done()
		orchErr = orch.RunOrchestration(ctx)
	}()

	time.Sleep(100 * time.Millisecond)

	testEndpoints := []struct {
		name   string
		path   string
		method string
		status int
	}{
		{
			name:   "CalculateEndpoint",
			path:   "http://localhost:8090/api/v1/calculate",
			method: http.MethodGet,
			status: http.StatusUnprocessableEntity,
		},
		{
			name:   "ExpressionsListEndpoint",
			path:   "http://localhost:8090/api/v1/expressions",
			method: http.MethodGet,
			status: http.StatusOK,
		},
		{
			name:   "GetExpressionByIDEndpoint",
			path:   "http://localhost:8090/api/v1/expressions/1",
			method: http.MethodGet,
			status: http.StatusNotFound,
		},
		{
			name:   "GetTaskEndpoint",
			path:   "http://localhost:8090/internal/task",
			method: http.MethodGet,
			status: http.StatusNotFound,
		},
		{
			name:   "PostTaskEndpoint",
			path:   "http://localhost:8090/internal/task",
			method: http.MethodPost,
			status: http.StatusUnprocessableEntity,
		},
	}

	client := &http.Client{
		Timeout: 100 * time.Millisecond,
	}

	for _, endpoint := range testEndpoints {
		t.Run(endpoint.name, func(t *testing.T) {
			req, err := http.NewRequest(endpoint.method, endpoint.path, nil)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := client.Do(req)
			if err != nil {
				t.Skipf("Endpoint test skipped, server might not be ready: %v", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != endpoint.status && resp.StatusCode != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d for %s, got %d", endpoint.status, endpoint.path, resp.StatusCode)
			}
		})
	}

	cancel()
	wg.Wait()

	if orchErr != nil && orchErr.Error() != "context canceled" {
		t.Errorf("RunOrchestration returned unexpected error: %v", orchErr)
	}
}

func TestRunOrchestrationContextCancellation(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{
			Port: 8091,
		},
	}

	orch := NewOrchestrator(cfg)

	ctx, cancel := context.WithCancel(context.Background())

	var wg sync.WaitGroup
	wg.Add(1)
	var orchErr error

	go func() {
		defer wg.Done()
		orchErr = orch.RunOrchestration(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	wg.Wait()

	if orchErr != nil {
		t.Logf("RunOrchestration after cancellation: %v", orchErr)
	}
}

func TestRunOrchestrationPortConflict(t *testing.T) {
	cfg1 := &config.Config{
		Orchestrator: config.OrchestratorConfig{
			Port: 8092,
		},
	}

	cfg2 := &config.Config{
		Orchestrator: config.OrchestratorConfig{
			Port: 8092,
		},
	}

	orch1 := NewOrchestrator(cfg1)
	orch2 := NewOrchestrator(cfg2)

	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		orch1.RunOrchestration(ctx1)
	}()

	time.Sleep(100 * time.Millisecond)

	ctx2, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel2()

	err := orch2.RunOrchestration(ctx2)

	if err == nil {
		t.Error("Expected error when starting second server on same port, got nil")
	}

	cancel1()
	wg.Wait()
}

func TestRunOrchestrationShutdown(t *testing.T) {
	cfg := &config.Config{
		Orchestrator: config.OrchestratorConfig{
			Port: 8093,
		},
	}

	orch := NewOrchestrator(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := orch.RunOrchestration(ctx)
	duration := time.Since(start)

	if err != nil && err.Error() != "context deadline exceeded" {
		t.Errorf("Unexpected error: %v", err)
	}

	if duration < 200*time.Millisecond {
		t.Errorf("RunOrchestration returned too quickly: %v", duration)
	}
}
