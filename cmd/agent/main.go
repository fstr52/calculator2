package main

import (
	"context"
	"final3/internal/agent"
	"final3/internal/config"
	"final3/internal/logger"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	cfg, err := config.LoadConfig("", "agent")
	if err != nil {
		panic(err)
	}

	orchestratorURL := os.Getenv("ORCHESTRATOR_URL")
	if orchestratorURL != "" {
		cfg.Agent.OrchestratorURL = orchestratorURL
	}

	if err := logger.Init(cfg); err != nil {
		panic(err)
	}

	agent, err := agent.NewAgent(&cfg.Agent)
	if err != nil {
		return
	}

	errChan := make(chan error, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := agent.Run(ctx); err != nil {
			errChan <- err
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		logger.Info("Receiver signal. Shutting down...", "signal", sig)
		cancel()
		wg.Wait()
		logger.Info("Agent shut down gracefully")
		os.Exit(0)
	case err := <-errChan:
		logger.Error("Agent failed", "error", err)
		cancel()
		wg.Wait()
		logger.Info("Agent shut down with error")
		os.Exit(1)
	}
}
