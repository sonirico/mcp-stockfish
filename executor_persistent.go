package main

import (
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type PersistentSessionExecutor struct {
	sessionManager *SessionManager
	logger         zerolog.Logger
}

func NewPersistentSessionExecutor(
	sm *SessionManager,
	logger zerolog.Logger,
) *PersistentSessionExecutor {
	return &PersistentSessionExecutor{
		sessionManager: sm,
		logger:         logger.With().Str("executor", "persistent").Logger(),
	}
}

func (e *PersistentSessionExecutor) Execute(
	command string,
	clientSessionID string,
	timeout time.Duration,
) (string, []string, error) {
	session, err := e.sessionManager.getOrCreateSession(clientSessionID)
	if err != nil {
		e.logger.Error().
			Err(err).
			Str("client_session_id", clientSessionID).
			Msg("Failed to get or create session")
		return clientSessionID, nil, err
	}

	actualSessionID := session.ID
	responses, err := session.executeCommand(command, timeout)

	if strings.TrimSpace(command) == "quit" {
		e.sessionManager.removeSession(actualSessionID)
		e.logger.Info().
			Str("session_id", actualSessionID).
			Msg("Session quit and removed by persistent executor")
	}
	return actualSessionID, responses, err
}
