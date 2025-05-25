package main

import (
	"time"

	"github.com/rs/zerolog"
)

type EphemeralSessionExecutor struct {
	stockfishPath  string
	commandTimeout time.Duration
	logger         zerolog.Logger
}

func NewEphemeralSessionExecutor(
	stockfishPath string,
	commandTimeout time.Duration,
	logger zerolog.Logger,
) *EphemeralSessionExecutor {
	return &EphemeralSessionExecutor{
		stockfishPath:  stockfishPath,
		commandTimeout: commandTimeout,
		logger:         logger.With().Str("executor", "ephemeral").Logger(),
	}
}

func (e *EphemeralSessionExecutor) Execute(
	command string,
	clientSessionID string,
	timeout time.Duration,
) (string, []string, error) {
	ephemeralSessionID := "stdio-ephemeral"
	e.logger.Debug().Str("command", command).Msg("Creating ephemeral session")

	session, err := createEphemeralStockfishSession(e.stockfishPath, e.logger)
	if err != nil {
		e.logger.Error().Err(err).Msg("Failed to create ephemeral Stockfish session")
		return ephemeralSessionID, nil, err
	}
	defer session.close()

	responses, err := session.executeCommand(command, e.commandTimeout)
	return ephemeralSessionID, responses, err
}
