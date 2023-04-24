// Package utils implements some utility functions.
package utils

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
)

// GetDefaultLogger returns a default zapr logger.
func GetDefaultLogger(logLevel string) logr.Logger {
	cfg := zap.Config{
		Encoding:    "json",
		OutputPaths: []string{"stdout"},
		EncoderConfig: zapcore.EncoderConfig{
			MessageKey:    "message",
			CallerKey:     "file",
			LevelKey:      "level",
			TimeKey:       "time",
			NameKey:       "logger",
			StacktraceKey: "stacktrace",

			LineEnding:     zapcore.DefaultLineEnding,
			EncodeCaller:   zapcore.ShortCallerEncoder,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeName:     zapcore.FullNameEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
		},
	}

	switch logLevel {
	case "error":
		cfg.Development = false
		cfg.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case "debug":
		cfg.Development = true
		cfg.Level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	default:
		cfg.Development = true
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}

	zapLog, err := cfg.Build()
	if err != nil {
		log.Fatalf("Error while initializing zapLogger: %v", err)
	}

	return zapr.NewLogger(zapLog)
}
