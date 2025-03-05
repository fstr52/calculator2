package orchestrator

import (
	"encoding/json"
	"final3/internal/logger"
	"final3/internal/models"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (o *Orchestrator) CalculateHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Received calculate request",
		"remote_addr", r.RemoteAddr,
		"method", r.Method)

	if r.Method != http.MethodPost {
		logger.Warn("Wrong method for calculate",
			"method", r.Method,
			"remote_addr", r.RemoteAddr)
		http.Error(w, "Wrong method, expected POST", http.StatusUnprocessableEntity)
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		logger.Warn("Wrong content-type",
			"content_type", r.Header.Get("Content-Type"),
			"remote_addr", r.RemoteAddr)
		http.Error(w, "Wrong content-type, expected JSON", http.StatusUnprocessableEntity)
		return
	}
	defer r.Body.Close()

	var userRequest struct {
		Expression string `json:"expression"`
	}

	err := json.NewDecoder(r.Body).Decode(&userRequest)
	if err != nil {
		logger.Error("Failed to decode request body",
			"error", err,
			"remote_addr", r.RemoteAddr)
		panic(err)
	}

	logger.Info("Processing calculation request",
		"expression", userRequest.Expression)

	expr, err := o.prepareInput(userRequest.Expression)
	if err != nil {
		logger.Error("Failed to prepare input",
			"expression", userRequest.Expression,
			"error", err)
		http.Error(w, "Invalid expression", http.StatusUnprocessableEntity)
		return
	}

	if expr == nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	expr.Status = StatusInQueue
	logger.Debug("Expression status updated",
		"expression_id", expr.ID,
		"status", string(StatusInQueue))

	o.mu.Lock()
	o.Queue = append(o.Queue, expr)
	logger.Debug("Expression added to queue",
		"expression_id", expr.ID,
		"queue_length", len(o.Queue))
	o.mu.Unlock()

	o.DataBase.mu.Lock()
	o.DataBase.ExpressionList[expr.ID] = expr
	logger.Debug("Expression added to database",
		"expression_id", expr.ID)
	o.DataBase.mu.Unlock()

	var response = struct {
		ID     int32            `json:"id"`
		Status ExpressionStatus `json:"status"`
	}{
		ID:     expr.ID,
		Status: expr.Status,
	}

	logger.Info("Calculation request processed successfully",
		"expression_id", expr.ID,
		"status", string(expr.Status))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (o *Orchestrator) GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Received get task request",
		"remote_addr", r.RemoteAddr,
		"method", r.Method)

	if r.Method != http.MethodGet {
		logger.Warn("Wrong method for get task",
			"method", r.Method,
			"remote_addr", r.RemoteAddr)
		http.Error(w, "Wrong method, expected GET", http.StatusUnprocessableEntity)
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.Queue) == 0 {
		logger.Debug("No tasks available in queue")
		http.Error(w, "No tasks available now (no expressions in queue)", http.StatusNotFound)
		return
	}
	expr := o.Queue[0]
	logger.Debug("Found expression in queue",
		"expression_id", expr.ID)

	expr.mu.Lock()
	defer expr.mu.Unlock()

	expr.Status = StatusInProgress
	logger.Debug("Expression status updated",
		"expression_id", expr.ID,
		"status", string(StatusInProgress))

	for id, task := range expr.IdMap {
		if task.Type != models.Operator || task.Status != models.StatusInQueue {
			continue
		}

		if len(task.Dependencies) < 2 {
			logger.Debug("Task has insufficient dependencies",
				"task_id", id,
				"dependencies_count", len(task.Dependencies))
			continue
		}

		dep1 := task.Dependencies[0]
		dep2 := task.Dependencies[1]

		if dep1.Type != models.Number || dep2.Type != models.Number {
			logger.Debug("Task dependencies are not ready",
				"task_id", id,
				"dep1_type", dep1.Type,
				"dep2_type", dep2.Type)
			continue
		}

		task.Status = models.StatusAtWorker
		logger.Debug("Task status updated",
			"task_id", id,
			"status", task.Status)

		arg1, err := strconv.ParseFloat(dep1.Value, 64)
		if err != nil {
			logger.Error("Error parsing arg1",
				"value", dep1.Value,
				"error", err)
			fmt.Printf("Error parsing arg1: %v\n", err)
			continue
		}

		arg2, err := strconv.ParseFloat(dep2.Value, 64)
		if err != nil {
			logger.Error("Error parsing arg2",
				"value", dep2.Value,
				"error", err)
			fmt.Printf("Error parsing arg2: %v\n", err)
			continue
		}

		logger.Info("Sending task to worker",
			"task_id", id,
			"expression_id", expr.ID,
			"arg1", arg1,
			"operation", task.Value,
			"arg2", arg2)

		var operationTime time.Duration

		switch task.Value {
		case "+":
			operationTime = time.Duration(o.Config.TimeAdditionMS) * time.Millisecond
		case "-":
			operationTime = time.Duration(o.Config.TimeSubtractionMS) * time.Millisecond
		case "*":
			operationTime = time.Duration(o.Config.TimeMultiplicationsMS) * time.Millisecond
		case "/":
			operationTime = time.Duration(o.Config.TimeDivisionsMS) * time.Millisecond
		}

		logger.Debug("Operation time calculated",
			"operation", task.Value,
			"time_ms", operationTime.Milliseconds())

		taskToSend := struct {
			ID            int           `json:"id"`
			Arg1          float64       `json:"arg1"`
			Arg2          float64       `json:"arg2"`
			Operation     string        `json:"operation"`
			OperationTime time.Duration `json:"operation_time"`
		}{
			ID:            id,
			Arg1:          arg1,
			Arg2:          arg2,
			Operation:     task.Value,
			OperationTime: operationTime,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(taskToSend)
		return
	}

	logger.Debug("No eligible tasks found")
	http.Error(w, "No tasks to do", 404)
}

func (o *Orchestrator) PostTaskHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Received post task result",
		"remote_addr", r.RemoteAddr,
		"method", r.Method)

	if r.Method != http.MethodPost {
		logger.Warn("Wrong method for post task",
			"method", r.Method,
			"remote_addr", r.RemoteAddr)
		http.Error(w, "Wrong method, expected POST", http.StatusUnprocessableEntity)
		return
	}

	if r.Header.Get("Content-Type") != "application/json" {
		logger.Warn("Wrong content-type",
			"content_type", r.Header.Get("Content-Type"),
			"remote_addr", r.RemoteAddr)
		http.Error(w, "Wrong content-type, expected JSON", http.StatusUnprocessableEntity)
		return
	}
	defer r.Body.Close()

	var postReq struct {
		ID     int     `json:"id"`
		Result float64 `json:"result"`
		Error  string  `json:"error,omitempty"`
	}

	err := json.NewDecoder(r.Body).Decode(&postReq)
	if err != nil {
		logger.Error("Failed to decode request body",
			"error", err,
			"remote_addr", r.RemoteAddr)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	logger.Info("Received task result",
		"task_id", postReq.ID,
		"result", postReq.Result,
		"error", postReq.Error)

	o.mu.Lock()
	if len(o.Queue) == 0 {
		logger.Warn("Received result but queue is empty",
			"task_id", postReq.ID)
		o.mu.Unlock()
		http.Error(w, "No expressions in queue", 404)
		return
	}
	expr := o.Queue[0]
	o.mu.Unlock()

	expr.mu.Lock()
	defer expr.mu.Unlock()

	stringResult := fmt.Sprintf("%.5f", postReq.Result)
	logger.Debug("Processing task result",
		"task_id", postReq.ID,
		"expression_id", expr.ID,
		"result", stringResult)

	completedNode := expr.IdMap[postReq.ID]
	if completedNode == nil {
		logger.Error("Node not found",
			"task_id", postReq.ID,
			"expression_id", expr.ID)
		http.Error(w, "Node not found", http.StatusNotFound)
		return
	}

	completedNode.Value = stringResult
	completedNode.Status = models.StatusDone
	completedNode.Type = models.Number

	logger.Debug("Node updated",
		"task_id", postReq.ID,
		"status", completedNode.Status,
		"type", completedNode.Type)

	if postReq.Error != "" {
		logger.Error("Task error reported",
			"task_id", postReq.ID,
			"expression_id", expr.ID,
			"error", postReq.Error)

		o.mu.Lock()
		defer o.mu.Unlock()
		expr.Status = StatusError
		expr.Err = fmt.Errorf(postReq.Error)

		logger.Debug("Expression status updated",
			"expression_id", expr.ID,
			"status", string(StatusError))

		o.Queue = o.Queue[1:]
		logger.Debug("Expression removed from queue",
			"queue_length", len(o.Queue))

		o.DataBase.mu.Lock()
		o.DataBase.ExpressionList[expr.ID].Status = StatusError
		o.DataBase.mu.Unlock()

		logger.Debug("Expression status updated in database",
			"expression_id", expr.ID,
			"status", string(StatusError))
		return
	}

	maxLevel := 0
	for _, node := range expr.IdMap {
		if node.Level > maxLevel {
			maxLevel = node.Level
		}
	}

	logger.Debug("Checking if expression is complete",
		"expression_id", expr.ID,
		"max_level", maxLevel)

	allMaxLevelDone := true
	for _, node := range expr.IdMap {
		if node.Level == maxLevel && node.Status != models.StatusDone {
			allMaxLevelDone = false
			logger.Debug("Found unfinished node at max level",
				"node_level", node.Level,
				"node_status", node.Status)
			break
		}
	}

	if allMaxLevelDone {
		logger.Info("All nodes at max level are done, expression is complete",
			"expression_id", expr.ID)

		for _, node := range expr.IdMap {
			if node.Level == maxLevel {
				result, err := strconv.ParseFloat(node.Value, 64)
				if err == nil {
					expr.Result = result
					expr.Status = StatusDone

					logger.Info("Expression result calculated",
						"expression_id", expr.ID,
						"result", result)

					o.DataBase.mu.Lock()
					o.DataBase.ExpressionList[expr.ID].Status = StatusDone
					o.DataBase.mu.Unlock()

					logger.Debug("Expression status updated in database",
						"expression_id", expr.ID,
						"status", string(StatusDone))

					o.mu.Lock()
					if len(o.Queue) > 0 && o.Queue[0] == expr {
						o.Queue = o.Queue[1:]
						logger.Debug("Expression removed from queue",
							"queue_length", len(o.Queue))
					}
					o.mu.Unlock()

					logger.Info("Expression completed",
						"expression_id", expr.ID,
						"result", expr.Result)
				} else {
					logger.Error("Failed to parse final result",
						"value", node.Value,
						"error", err)
				}
				break
			}
		}
	} else {
		expr.Status = StatusInQueue
		logger.Debug("Expression not yet complete, returning to queue",
			"expression_id", expr.ID)

		o.DataBase.mu.Lock()
		o.DataBase.ExpressionList[expr.ID].Status = StatusInQueue
		o.DataBase.mu.Unlock()

		logger.Debug("Expression status updated in database",
			"expression_id", expr.ID,
			"status", string(StatusInQueue))
	}

	w.WriteHeader(http.StatusOK)
	logger.Debug("Task result processed successfully",
		"task_id", postReq.ID)
}

func (o *Orchestrator) ExpressionsListHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Received expressions list request",
		"remote_addr", r.RemoteAddr,
		"method", r.Method)

	if r.Method != http.MethodGet {
		logger.Warn("Wrong method for expressions list",
			"method", r.Method,
			"remote_addr", r.RemoteAddr)
		http.Error(w, "Wrong method, expected GET", http.StatusUnprocessableEntity)
		return
	}

	o.DataBase.mu.Lock()
	defer o.DataBase.mu.Unlock()

	logger.Debug("Preparing expressions list",
		"expressions_count", len(o.DataBase.ExpressionList))

	type sendStruct struct {
		ID     int32            `json:"id"`
		Status ExpressionStatus `json:"status"`
		Result float64          `json:"result"`
		Error  string           `json:"error,omitempty"`
	}

	ids := make([]int32, 0, len(o.DataBase.ExpressionList))

	for id := range o.DataBase.ExpressionList {
		ids = append(ids, id)
	}

	sort.Slice(ids, func(i, j int) bool {
		return ids[i] < ids[j]
	})

	logger.Debug("Sorted expression IDs",
		"ids_count", len(ids))

	exprList := make([]sendStruct, 0, len(ids))
	for _, id := range ids {
		expr, ok := o.DataBase.ExpressionList[id]
		if !ok {
			logger.Warn("Expression not found in database",
				"expression_id", id)
			continue
		}

		sendExpr := sendStruct{
			ID:     expr.ID,
			Status: expr.Status,
			Result: expr.Result,
		}

		if expr.Err != nil {
			sendExpr.Error = expr.Err.Error()
			logger.Debug("Including error in expression data",
				"expression_id", id,
				"error", expr.Err.Error())
		}

		exprList = append(exprList, sendExpr)
	}

	sendJsonList := struct {
		Expressions []sendStruct `json:"expressions"`
	}{
		Expressions: exprList,
	}

	logger.Info("Sending expressions list",
		"expressions_count", len(exprList))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sendJsonList)
}

func (o *Orchestrator) GetExpressionByIDHandler(w http.ResponseWriter, r *http.Request) {
	logger.Debug("Received get expression by ID request",
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"method", r.Method)

	if r.Method != http.MethodGet {
		logger.Warn("Wrong method for get expression by ID",
			"method", r.Method,
			"remote_addr", r.RemoteAddr)
		http.Error(w, "Wrong method, expected GET", http.StatusUnprocessableEntity)
		return
	}

	url := r.URL.Path
	stringID, ok := strings.CutPrefix(url, "/api/v1/expressions/:")
	if !ok {
		logger.Error("Failed to extract ID from URL",
			"url", url)
		http.Error(w, "Failed to cut ID", 500)
		return
	}

	logger.Debug("Extracted ID from URL",
		"string_id", stringID)

	id, err := strconv.ParseInt(stringID, 10, 32)
	if err != nil {
		logger.Error("Failed to parse ID",
			"string_id", stringID,
			"error", err)
		http.Error(w, "Failed to parse ID", 500)
		return
	}

	logger.Debug("Parsed ID",
		"id", id)

	o.DataBase.mu.Lock()
	expr, ok := o.DataBase.ExpressionList[int32(id)]
	if !ok {
		logger.Warn("Expression not found in database",
			"expression_id", id)
		http.Error(w, "Failed to find expression", 404)
		o.DataBase.mu.Unlock()
		return
	}
	o.DataBase.mu.Unlock()

	logger.Debug("Found expression in database",
		"expression_id", id,
		"status", string(expr.Status))

	sendExpression := struct {
		ID     int32            `json:"id"`
		Status ExpressionStatus `json:"status"`
		Result float64          `json:"result"`
	}{
		ID:     expr.ID,
		Status: expr.Status,
		Result: expr.Result,
	}

	logger.Info("Sending expression details",
		"expression_id", expr.ID,
		"status", string(expr.Status),
		"result", expr.Result)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sendExpression)
}
