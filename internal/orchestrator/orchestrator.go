package orchestrator

import (
	"context"
	"final3/internal/config"
	"final3/internal/logger"
	"final3/internal/models"
	"final3/pkg/parser"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type ExpressionStatus string

const (
	StatusCreated    ExpressionStatus = "just_created"
	StatusInQueue    ExpressionStatus = "in_queue"
	StatusInProgress ExpressionStatus = "in_progress"
	StatusDone       ExpressionStatus = "done"
	StatusError      ExpressionStatus = "error"
)

type Expression struct {
	ID     int32
	IdMap  map[int]*models.Node
	Status ExpressionStatus
	Err    error
	Result float64
	mu     sync.Mutex
}

// Создание нового выражения для заданного оркестратора
func (o *Orchestrator) NewExpression() *Expression {
	o.mu.Lock()
	id := o.PrevExpressionID + 1
	o.PrevExpressionID++
	o.mu.Unlock()

	logger.Debug("Creating new expression",
		"expression_id", id)

	return &Expression{
		ID:     id,
		IdMap:  make(map[int]*models.Node),
		Status: StatusCreated,
		Err:    nil,
	}
}

type Orchestrator struct {
	Queue            []*Expression
	PrevExpressionID int32
	mu               sync.Mutex
	DataBase         *DataBase
	Config           *config.OrchestratorConfig
}

type DataBase struct {
	ExpressionList map[int32]*Expression
	mu             sync.Mutex
}

// Создание новой базы данных
func NewDatabase() *DataBase {
	logger.Debug("Creating new database")
	return &DataBase{
		ExpressionList: make(map[int32]*Expression),
	}
}

// Создание нового оркестратора
func NewOrchestrator(cfg *config.Config) *Orchestrator {
	logger.Info("Initializing orchestrator",
		"port", cfg.Orchestrator.Port,
		"time_addition_ms", cfg.Orchestrator.TimeAdditionMS,
		"time_subtraction_ms", cfg.Orchestrator.TimeSubtractionMS,
		"time_multiplications_ms", cfg.Orchestrator.TimeMultiplicationsMS,
		"time_divisions_ms", cfg.Orchestrator.TimeDivisionsMS)

	return &Orchestrator{
		Queue:    make([]*Expression, 0),
		DataBase: NewDatabase(),
		Config:   &cfg.Orchestrator,
	}
}

// Запуск оркестратора
func (o *Orchestrator) RunOrchestration(ctx context.Context) error {
	logger.Info("Starting orchestration service")

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/calculate", o.CalculateHandler)
	mux.HandleFunc("/api/v1/expressions", o.ExpressionsListHandler)
	mux.HandleFunc("/api/v1/expressions/", o.GetExpressionByIDHandler)
	mux.HandleFunc("/internal/task", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			o.GetTaskHandler(w, r)
		} else if r.Method == http.MethodPost {
			o.PostTaskHandler(w, r)
		} else {
			logger.Warn("Wrong method for /internal/task",
				"method", r.Method,
				"remote_addr", r.RemoteAddr)
			http.Error(w, `{"error":"Wrong Method"}`, http.StatusMethodNotAllowed)
		}
	})

	port := fmt.Sprintf("%d", o.Config.Port)
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	serverError := make(chan error, 1)

	logger.Info("Starting HTTP server", "port", port)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", "error", err)
			serverError <- err
		} else {
			serverError <- nil
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("Context done, shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("Server shutdown error", "error", err)
			return fmt.Errorf("server shutdown error: %w", err)
		}

		err := <-serverError
		if err != nil {
			logger.Error("Server error during shutdown", "error", err)
			return fmt.Errorf("server error during shutdown: %w", err)
		}

		logger.Info("Server shutdown complete")
		return nil
	case err := <-serverError:
		if err != nil {
			logger.Error("Server error", "error", err)
			return fmt.Errorf("server error: %w", err)
		}
		logger.Info("Server stopped")
		return nil
	}
}

// Подготовка полученного выражения к обработке
func (o *Orchestrator) prepareInput(input string) (*Expression, error) {
	logger.Debug("Preparing input expression", "input", input)

	levelMap, maxLevel, err := parser.ParseExpression(input)
	if err != nil {
		logger.Error("Failed to parse expression",
			"input", input,
			"error", err)
		return nil, err
	}

	logger.Debug("Expression parsed successfully",
		"max_level", maxLevel,
		"levels_count", len(levelMap))

	prevID := 0
	expr := o.NewExpression()

	for level := 0; level <= maxLevel; level++ {
		nodes := levelMap[level]
		logger.Debug("Processing level",
			"level", level,
			"nodes_count", len(nodes),
			"expression_id", expr.ID)

		for _, node := range nodes {
			curID := prevID + 1
			if _, ok := expr.IdMap[curID]; ok {
				logger.Error("Node ID collision",
					"node_id", curID,
					"expression_id", expr.ID)
				return nil, fmt.Errorf("почему уже есть?")
			}

			if node.Type == models.Number {
				node.Status = models.StatusDone
				logger.Debug("Node is a number, marking as done",
					"node_id", curID,
					"value", node.Value)
			} else {
				node.Status = models.StatusInQueue
				logger.Debug("Node is an operation, marking as in queue",
					"node_id", curID,
					"operation", node.Value)
			}

			expr.IdMap[curID] = node
			prevID++
		}
	}

	logger.Info("Input preparation completed",
		"expression_id", expr.ID,
		"nodes_count", len(expr.IdMap))

	o.DataBase.mu.Lock()
	o.DataBase.ExpressionList[expr.ID] = expr
	o.DataBase.mu.Unlock()

	logger.Debug("Expression added to database",
		"expression_id", expr.ID)

	return expr, nil
}
