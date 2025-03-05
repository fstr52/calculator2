package logger

import (
	"context"
	"final3/internal/config"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	instance *slog.Logger
	once     sync.Once
)

func Init(cfg *config.Config) error {
	var err error
	once.Do(func() {
		instance, err = newLogger(cfg)
	})
	return err
}

// Создание нового логера с заданным конфигом
func newLogger(cfg *config.Config) (*slog.Logger, error) {
	var handler slog.Handler

	if cfg.Logging.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
	}

	if cfg.Logging.ToFile {
		fileHandler, err := createFileHandler(cfg)
		if err != nil {
			return nil, err
		}

		handler = newMultiHandler(handler, fileHandler)
	}

	logger := slog.New(handler)

	slog.SetDefault(logger)

	logger.Info("Logger initialized",
		"format", cfg.Logging.Format,
		"to_file", cfg.Logging.ToFile,
	)
	return logger, nil
}

// Создание хэндлера для логера с записью в файл
func createFileHandler(cfg *config.Config) (slog.Handler, error) {
	logDir := cfg.Logging.Dir
	if logDir == "" {
		logDir = "./logs"
		println("Log directory is empty, using default:", logDir)
	}

	if err := os.MkdirAll(logDir, 0755); err != nil {
		println("Error creating directory:", err.Error())
		return nil, err
	}

	fileName := filepath.Join(cfg.Logging.Dir, time.Now().Format("2006-01-02"+".log"))
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	if cfg.Logging.Format == "json" {
		return slog.NewJSONHandler(file, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}), nil
	}

	return slog.NewTextHandler(file, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}), nil
}

type multiHandler struct {
	handlers []slog.Handler
}

// Создание нового мульти-хендлера
func newMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if err := handler.Handle(ctx, r.Clone()); err != nil {
			return err
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}

func Debug(msg string, args ...any) {
	getLogger().Debug(msg, args...)
}

func Info(msg string, args ...any) {
	getLogger().Info(msg, args...)
}

func Warn(msg string, args ...any) {
	getLogger().Warn(msg, args...)
}

func Error(msg string, args ...any) {
	getLogger().Error(msg, args...)
}

func WithGroup(name string) *slog.Logger {
	return getLogger().WithGroup(name)
}

func With(args ...any) *slog.Logger {
	return getLogger().With(args...)
}

func getLogger() *slog.Logger {
	if instance == nil {
		instance = slog.Default()
	}
	return instance
}
