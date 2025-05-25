package main

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type StockfishSession struct {
	ID         string
	cmd        *exec.Cmd
	stdin      *bufio.Writer
	stdout     *bufio.Scanner
	stderr     *bufio.Scanner
	lastUsed   time.Time
	mu         sync.RWMutex
	logger     zerolog.Logger
	cancelFunc context.CancelFunc
}

type SessionManager struct {
	sessions      map[string]*StockfishSession
	config        StockfishConfig
	logger        zerolog.Logger
	mu            sync.RWMutex
	stopCleanupCh chan struct{}
	shutdownOnce  sync.Once
}

func newSessionManager(config StockfishConfig, logger zerolog.Logger) *SessionManager {
	sm := &SessionManager{
		sessions:      make(map[string]*StockfishSession),
		config:        config,
		logger:        logger.With().Str("component", ComponentSessionManager).Logger(),
		stopCleanupCh: make(chan struct{}),
	}

	go sm.cleanupRoutine()
	return sm
}

func createEphemeralStockfishSession(
	stockfishPath string,
	logger zerolog.Logger,
) (*StockfishSession, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, stockfishPath)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("ephemeral: failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("ephemeral: failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("ephemeral: failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("ephemeral: failed to start stockfish: %w", err)
	}

	sessionLogger := logger.With().Str("session_id", "ephemeral").Logger()

	session := &StockfishSession{
		ID:         "ephemeral-" + uuid.NewString()[:8],
		cmd:        cmd,
		stdin:      bufio.NewWriter(stdin),
		stdout:     bufio.NewScanner(stdout),
		stderr:     bufio.NewScanner(stderr),
		lastUsed:   time.Now(),
		logger:     sessionLogger,
		cancelFunc: cancel,
	}
	sessionLogger.Debug().Msg("Created ephemeral Stockfish instance")
	return session, nil
}

func (sm *SessionManager) Close() {
	sm.shutdownOnce.Do(func() {
		sm.logger.Info().Msg("Shutting down session manager")

		close(sm.stopCleanupCh)

		sm.mu.Lock()
		defer sm.mu.Unlock()

		for sessionID, session := range sm.sessions {
			session.close()
			delete(sm.sessions, sessionID)
			sm.logger.Debug().Str("session_id", sessionID).Msg("Closed session during shutdown")
		}
		sm.sessions = make(map[string]*StockfishSession)

		sm.logger.Info().Msg("Session manager shutdown complete")
	})
}

func (sm *SessionManager) getOrCreateSession(sessionID string) (*StockfishSession, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sessionID != "" {
		if session, exists := sm.sessions[sessionID]; exists {
			session.mu.Lock()
			session.lastUsed = time.Now()
			session.mu.Unlock()
			return session, nil
		}
	}

	if len(sm.sessions) >= sm.config.MaxSessions {
		return nil, fmt.Errorf("maximum number of sessions (%d) reached", sm.config.MaxSessions)
	}

	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	session, err := sm.createSession(sessionID)
	if err != nil {
		return nil, err
	}

	sm.sessions[sessionID] = session
	sm.logger.Info().Str("session_id", sessionID).Msg("Created new Stockfish session")

	return session, nil
}

func (sm *SessionManager) createSession(sessionID string) (*StockfishSession, error) {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, sm.config.Path)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start stockfish: %w", err)
	}

	session := &StockfishSession{
		ID:         sessionID,
		cmd:        cmd,
		stdin:      bufio.NewWriter(stdin),
		stdout:     bufio.NewScanner(stdout),
		stderr:     bufio.NewScanner(stderr),
		lastUsed:   time.Now(),
		logger:     sm.logger.With().Str("session_id", sessionID).Logger(),
		cancelFunc: cancel,
	}

	return session, nil
}

func (sm *SessionManager) removeSession(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		session.close()
		delete(sm.sessions, sessionID)
		sm.logger.Info().Str("session_id", sessionID).Msg("Removed Stockfish session")
	}
}

func (sm *SessionManager) cleanupRoutine() {
	ticker := time.NewTicker(sm.config.SessionTimeout / 2)
	if sm.config.SessionTimeout < 2*time.Second {
		ticker = time.NewTicker(time.Second)
	}
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.cleanupExpiredSessions()
		case <-sm.stopCleanupCh:
			sm.logger.Debug().Msg("Cleanup routine stopped.")
			return
		}
	}
}

func (sm *SessionManager) cleanupExpiredSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now().UTC()
	for sessionID, session := range sm.sessions {
		session.mu.RLock()
		expired := now.Sub(session.lastUsed) > sm.config.SessionTimeout
		session.mu.RUnlock()

		if expired {
			session.close()
			delete(sm.sessions, sessionID)
			sm.logger.Info().Str("session_id", sessionID).Msg("Cleaned up expired session")
		}
	}
}

func (s *StockfishSession) executeCommand(command string, timeout time.Duration) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastUsed = time.Now()

	if _, err := s.stdin.WriteString(command + "\n"); err != nil {
		return nil, fmt.Errorf("failed to write command: %w", err)
	}
	if err := s.stdin.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush command: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var responses []string
	done := make(chan struct{})
	var scanErr error

	go func() {
		defer close(done)

		for {
			if !s.stdout.Scan() {
				scanErr = s.stdout.Err()
				return
			}

			line := s.stdout.Text()
			responses = append(responses, line)

			if shouldStopReading(command, line) {
				return
			}
		}
	}()

	select {
	case <-done:
		if scanErr != nil {
			return responses, fmt.Errorf("scan error: %w", scanErr)
		}
		return responses, nil
	case <-ctx.Done():
		return responses, fmt.Errorf("command timeout after %v", timeout)
	}
}

func shouldStopReading(command, response string) bool {
	switch {
	case command == StockfishCmdUCI && response == "uciok":
		return true
	case command == StockfishCmdIsReady && response == "readyok":
		return true
	case command == StockfishCmdQuit:
		return true
	case command == StockfishCmdStop:
		return true
	case response == "bestmove" || (len(response) > 8 && response[:8] == "bestmove"):
		return true
	}
	return false
}

func (s *StockfishSession) close() {
	if s.cancelFunc != nil {
		s.cancelFunc()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
}
