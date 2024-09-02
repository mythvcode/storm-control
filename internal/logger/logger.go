package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	slogmulti "github.com/samber/slog-multi"
)

const Module = "module"

type Logger struct {
	logger *slog.Logger
}

func marshalLevel(level string) (slog.Level, error) {
	if level == "" {
		return slog.LevelDebug, nil
	}
	var l slog.Level
	err := l.UnmarshalText([]byte(level))

	return l, err
}

func Init(logFile string, level string) error {
	parsedLevel, err := marshalLevel(level)
	if err != nil {
		return err
	}

	slogHandlers := make([]slog.Handler, 0, 2)
	var jsonHandler *slog.JSONHandler
	if logFile != "" {
		logFile, err := os.OpenFile(filepath.Clean(logFile), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("failed to initialize log file %w", err)
		}
		jsonHandler = slog.NewJSONHandler(logFile, &slog.HandlerOptions{Level: parsedLevel})
	} else {
		jsonHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: parsedLevel})
	}

	slogHandlers = append(slogHandlers, jsonHandler)

	logger := slog.New(slogmulti.Fanout(slogHandlers...))

	slog.SetDefault(logger)

	return nil
}

func Default() *Logger {
	return &Logger{
		slog.New(slog.NewJSONHandler(os.Stdout,
			&slog.HandlerOptions{Level: slog.LevelInfo})),
	}
}

func GetLogger() *Logger {
	return &Logger{slog.Default()}
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{l.logger.With(args...)}
}

func (l *Logger) Debugf(fsting string, formaters ...any) {
	l.logger.Debug(fmt.Sprintf(fsting, formaters...))
}

func (l *Logger) Infof(fsting string, formaters ...any) {
	l.logger.Info(fmt.Sprintf(fsting, formaters...))
}

func (l *Logger) Warningf(fsting string, formaters ...any) {
	l.logger.Warn(fmt.Sprintf(fsting, formaters...))
}

func (l *Logger) Errorf(fsting string, formaters ...any) {
	l.logger.Error(fmt.Sprintf(fsting, formaters...))
}
