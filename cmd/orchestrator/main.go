package main

import (
	"context"
	"final3/internal/config"
	"final3/internal/logger"
	"final3/internal/orchestrator"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	cfg, err := config.LoadConfig("", "orchestrator")
	if err != nil {
		panic(err)
	}

	if err := logger.Init(cfg); err != nil {
		panic(err)
	}

	o := orchestrator.NewOrchestrator(cfg)
	errChan := make(chan error, 1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		if err := o.RunOrchestration(ctx); err != nil {
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
		logger.Info("Orchestrator shut down gracefully")
		os.Exit(0)
	case err := <-errChan:
		logger.Error("Orchestrator failed", "error", err)
		cancel()
		wg.Wait()
		logger.Info("Orchestrator shut down with error")
		os.Exit(1)
	}
}
