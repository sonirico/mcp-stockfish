package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog"
)

type commandExecutor interface {
	Execute(command string, clientSessionID string, timeout time.Duration) (string, []string, error)
}

type StockfishHandler struct {
	executor commandExecutor
	logger   zerolog.Logger
}

type CommandResult struct {
	Status    string   `json:"status"`
	SessionID string   `json:"session_id"`
	Command   string   `json:"command"`
	Response  []string `json:"response"`
	Error     string   `json:"error,omitempty"`
}

func newStockfishHandler(executor commandExecutor, logger zerolog.Logger) *StockfishHandler {
	return &StockfishHandler{
		executor: executor,
		logger:   logger.With().Str("component", "handler").Logger(),
	}
}

func (h *StockfishHandler) handle(
	ctx context.Context,
	request mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	command, err := request.RequireString("command")
	if err != nil {
		h.logger.Error().Err(err).Msg("Missing command parameter")
		return mcp.NewToolResultError("Missing 'command' parameter"), nil
	}

	sessionID := request.GetString("session_id", "")

	h.logger.Info().
		Str("command", command).
		Str("client_session_id", sessionID).
		Msg("Received Stockfish command request")

	if err := h.validateCommand(command); err != nil {
		h.logger.Warn().
			Err(err).
			Str("command", command).
			Msg("Invalid command")
		return mcp.NewToolResultError(fmt.Sprintf("Invalid command: %s", err.Error())), nil
	}

	actualSessionID, responses, execErr := h.executor.Execute(command, sessionID, 0)

	result := CommandResult{
		SessionID: actualSessionID,
		Command:   command,
		Response:  responses,
	}

	if execErr != nil {
		result.Status = "error"
		result.Error = execErr.Error()
		h.logger.Error().
			Err(execErr).
			Str("command", command).
			Str("actual_session_id", actualSessionID).
			Msg("Command execution failed")
	} else {
		result.Status = "success"
		h.logger.Debug().
			Str("command", command).
			Str("actual_session_id", actualSessionID).
			Int("response_lines", len(responses)).
			Msg("Command executed successfully")
	}

	jsonBytes, err := json.Marshal(result)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to marshal response")
		return mcp.NewToolResultError("Failed to marshal result to JSON"), nil
	}

	return mcp.NewToolResultText(string(jsonBytes)), nil
}

func (h *StockfishHandler) validateCommand(command string) error {
	cmd := strings.TrimSpace(strings.ToLower(command))

	validCommands := []string{
		StockfishCmdUCI, StockfishCmdIsReady, StockfishCmdQuit, StockfishCmdStop,
	}

	validPrefixes := []string{
		StockfishCmdPosition + " startpos", StockfishCmdPosition + " fen", // Note: "position" itself is not a full command
		StockfishCmdGo, StockfishCmdGo + " depth", StockfishCmdGo + " movetime",
		StockfishCmdSetOption,
	}

	// Check exact matches
	for _, valid := range validCommands {
		if cmd == valid {
			return nil
		}
	}

	// Check prefix matches
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(cmd, prefix) {
			return nil
		}
	}

	return fmt.Errorf("unsupported command: %s", command)
}
