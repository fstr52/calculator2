package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"final3/internal/config"
	"final3/internal/logger"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Agent struct {
	cfg    *config.AgentConfig
	client *http.Client
}

// Создание нового агента с заданным конфигом
func NewAgent(cfg *config.AgentConfig) (*Agent, error) {
	if cfg == nil {
		logger.Error("Failed to create agent", "error", "config required")
		return nil, fmt.Errorf("config required")
	}

	logger.Info("Creating new agent",
		"orchestrator_url", cfg.OrchestratorURL,
		"computing_power", cfg.ComputingPower)

	return &Agent{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// Запуск агента
func (a *Agent) Run(ctx context.Context) error {
	logger.Info("Starting agent",
		"orchestrator_url", a.cfg.OrchestratorURL,
		"computing_power", a.cfg.ComputingPower)

	workersErrChan := make(chan error, a.cfg.ComputingPower)
	agentErrChan := make(chan error, 1)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i := int64(0); i < a.cfg.ComputingPower; i++ {
		logger.Debug("Starting worker", "worker_id", i)
		go a.worker(ctx, workersErrChan, i)
	}

	go func() {
		for err := range workersErrChan {
			logger.Error("Worker error received", "error", err)

			select {
			case agentErrChan <- err:
				logger.Debug("Error forwarded to agent error channel")
			default:
				logger.Debug("Agent error channel full, dropping error")
			}
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("Received shutdown signal")
		cancel()
		logger.Info("Agent shut down complete")
		return nil
	case err := <-agentErrChan:
		logger.Error("Agent error occurred", "error", err)
		return err
	}
}

type Task struct {
	ID            int           `json:"id"`
	Arg1          float64       `json:"arg1"`
	Arg2          float64       `json:"arg2"`
	Operation     string        `json:"operation"`
	OperationTime time.Duration `json:"operation_time"`
}

func (a *Agent) worker(ctx context.Context, errChan chan error, workerId int64) {
	logger.Info("Worker started", "worker_id", workerId)

	for {
		select {
		case <-ctx.Done():
			logger.Info("Worker stopping due to context done", "worker_id", workerId)
			return
		default:
			logger.Debug("Worker requesting task", "worker_id", workerId)

			req, err := http.NewRequest("GET", a.cfg.OrchestratorURL+"/internal/task", nil)
			if err != nil {
				logger.Error("Error creating request",
					"worker_id", workerId,
					"error", err)
				errChan <- err
				time.Sleep(time.Second)
				continue
			}

			resp, err := a.client.Do(req)
			if err != nil {
				logger.Error("Error sending GET request",
					"worker_id", workerId,
					"url", a.cfg.OrchestratorURL+"/internal/task",
					"error", err)
				errChan <- err
				time.Sleep(time.Second)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				bodyBytes, _ := io.ReadAll(resp.Body)
				resp.Body.Close()

				if resp.StatusCode == http.StatusNotFound {
					logger.Debug("No tasks available",
						"worker_id", workerId,
						"status_code", resp.StatusCode,
						"response", string(bodyBytes))
					time.Sleep(500 * time.Millisecond)
					continue
				}

				logger.Warn("Non-OK status code from orchestrator",
					"worker_id", workerId,
					"status_code", resp.StatusCode,
					"response", string(bodyBytes))
				time.Sleep(500 * time.Millisecond)
				continue
			}

			var task Task
			if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
				logger.Error("Error decoding task response",
					"worker_id", workerId,
					"error", err)
				errChan <- err
				resp.Body.Close()
				continue
			}

			resp.Body.Close()

			logger.Info("Task received",
				"worker_id", workerId,
				"task_id", task.ID,
				"arg1", task.Arg1,
				"operation", task.Operation,
				"arg2", task.Arg2,
				"operation_time_ms", task.OperationTime.Milliseconds())

			var result float64
			var calcErr error

			timer := time.NewTimer(task.OperationTime)

			logger.Debug("Starting task calculation",
				"worker_id", workerId,
				"task_id", task.ID)

			switch task.Operation {
			case "+":
				result = task.Arg1 + task.Arg2
				logger.Debug("Addition performed",
					"worker_id", workerId,
					"task_id", task.ID,
					"result", result)
			case "-":
				result = task.Arg1 - task.Arg2
				logger.Debug("Subtraction performed",
					"worker_id", workerId,
					"task_id", task.ID,
					"result", result)
			case "*":
				result = task.Arg1 * task.Arg2
				logger.Debug("Multiplication performed",
					"worker_id", workerId,
					"task_id", task.ID,
					"result", result)
			case "/":
				if task.Arg2 == 0 {
					logger.Error("Division by zero",
						"worker_id", workerId,
						"task_id", task.ID)
					calcErr = fmt.Errorf("division by zero")
				} else {
					result = task.Arg1 / task.Arg2
					logger.Debug("Division performed",
						"worker_id", workerId,
						"task_id", task.ID,
						"result", result)
				}
			default:
				logger.Error("Unknown operation",
					"worker_id", workerId,
					"task_id", task.ID,
					"operation", task.Operation)
				calcErr = fmt.Errorf("unknown operation: %s", task.Operation)
			}

			logger.Debug("Waiting for operation time to complete",
				"worker_id", workerId,
				"task_id", task.ID,
				"wait_time_ms", task.OperationTime.Milliseconds())

			select {
			case <-ctx.Done():
				logger.Info("Worker stopping during task execution", "worker_id", workerId)
				return
			case <-timer.C:
				logger.Debug("Operation time completed",
					"worker_id", workerId,
					"task_id", task.ID)
			}

			answer := struct {
				ID     int     `json:"id"`
				Result float64 `json:"result"`
				Error  string  `json:"error,omitempty"`
			}{
				ID:     task.ID,
				Result: result,
			}

			if calcErr != nil {
				answer.Error = calcErr.Error()
				logger.Warn("Task calculation error",
					"worker_id", workerId,
					"task_id", task.ID,
					"error", calcErr.Error())
			}

			jsonAnswer, err := json.Marshal(answer)
			if err != nil {
				logger.Error("Error marshaling task result",
					"worker_id", workerId,
					"task_id", task.ID,
					"error", err)
				errChan <- err
				continue
			}

			logger.Debug("Sending task result to orchestrator",
				"worker_id", workerId,
				"task_id", task.ID,
				"result", result,
				"error", answer.Error)

			postReq, err := http.NewRequest("POST", a.cfg.OrchestratorURL+"/internal/task", bytes.NewReader(jsonAnswer))
			if err != nil {
				logger.Error("Error creating POST request for result",
					"worker_id", workerId,
					"task_id", task.ID,
					"error", err)
				errChan <- err
				continue
			}
			postReq.Header.Set("Content-Type", "application/json")

			postResp, err := a.client.Do(postReq)
			if err != nil {
				logger.Error("Error sending POST request with result",
					"worker_id", workerId,
					"task_id", task.ID,
					"error", err)
				errChan <- err
				continue
			}

			bodyBytes, _ := io.ReadAll(postResp.Body)
			postResp.Body.Close()

			if postResp.StatusCode != http.StatusOK {
				logger.Error("POST response status not OK",
					"worker_id", workerId,
					"task_id", task.ID,
					"status_code", postResp.StatusCode,
					"response", string(bodyBytes))
				errChan <- fmt.Errorf("post response status not OK: %d, body: %s", postResp.StatusCode, string(bodyBytes))
				continue
			}

			logger.Info("Task completed successfully",
				"worker_id", workerId,
				"task_id", task.ID,
				"result", result)

			time.Sleep(100 * time.Millisecond)
		}
	}
}
