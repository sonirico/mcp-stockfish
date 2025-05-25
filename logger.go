package main

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
)

func newLogger(config LoggingConfig) (zerolog.Logger, error) {
	level, err := zerolog.ParseLevel(config.Level)
	if err != nil {
		return zerolog.Logger{}, fmt.Errorf("invalid log level: %w", err)
	}

	var writer *os.File
	switch LogOutput(config.Output) {
	case LogOutputStdout:
		writer = os.Stdout
	case LogOutputStderr:
		writer = os.Stderr
	default:
		writer = os.Stderr
	}

	zerolog.SetGlobalLevel(level)

	var logger zerolog.Logger
	switch LogFormat(config.Format) {
	case LogFormatJSON:
		logger = zerolog.New(writer).With().Timestamp().Logger()
	case LogFormatConsole:
		consoleWriter := zerolog.ConsoleWriter{
			Out:        writer,
			TimeFormat: time.RFC3339,
		}
		logger = zerolog.New(consoleWriter).With().Timestamp().Logger()
	default:
		consoleWriter := zerolog.ConsoleWriter{
			Out:        writer,
			TimeFormat: time.RFC3339,
		}
		logger = zerolog.New(consoleWriter).With().Timestamp().Logger()
	}

	return logger, nil
}
