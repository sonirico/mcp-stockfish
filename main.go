package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, "Server stopped gracefully.")
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "Server exited.")
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if version != "dev" {
		cfg.Server.Version = version
	}

	log, err := newLogger(cfg.Logging)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	log.Info().
		Str("server_name", cfg.Server.Name).
		Str("version", cfg.Server.Version).
		Str("stockfish_path", cfg.Stockfish.Path).
		Int("max_sessions", cfg.Stockfish.MaxSessions).
		Dur("session_timeout", cfg.Stockfish.SessionTimeout).
		Str("server_mode", cfg.Server.Mode).
		Str("http_host", cfg.Server.Host).
		Int("http_port", cfg.Server.Port).
		Msg("Configuration loaded")

	var executor commandExecutor

	if ServerMode(cfg.Server.Mode) == ServerModeHTTP {
		sessionManager := newSessionManager(cfg.Stockfish, log)
		defer sessionManager.Close()
		executor = NewPersistentSessionExecutor(sessionManager, log)
	} else {
		executor = NewEphemeralSessionExecutor(cfg.Stockfish.Path, cfg.Stockfish.CommandTimeout, log)
	}

	stockfishHandler := newStockfishHandler(executor, log)

	s := server.NewMCPServer(
		cfg.Server.Name,
		cfg.Server.Version,
	)

	stockfishTool := mcp.NewTool(
		"stockfish_command",
		mcp.WithDescription(
			"Execute Stockfish UCI commands. Supports session management for concurrent chess analysis.",
		),
		mcp.WithString(
			"command",
			mcp.Required(),
			mcp.Description(
				"Stockfish UCI command to execute (uci, isready, position, go, stop, quit)",
			),
		),
		mcp.WithString(
			"session_id",
			mcp.Description(
				"Session ID for maintaining state across commands. If not provided, a new session is created.",
			),
		),
	)
	s.AddTool(stockfishTool, stockfishHandler.handle)

	switch ServerMode(cfg.Server.Mode) {
	case ServerModeHTTP:
		return runHTTPServer(s, cfg, log)
	case ServerModeStdio:
		return runStdioServer(s, log)
	default:
		return fmt.Errorf("unsupported server mode: %s", cfg.Server.Mode)
	}
}

func runHTTPServer(s *server.MCPServer, cfg *Config, log zerolog.Logger) error {
	return nil
	//ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	//defer stop()

	//addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	//log.Info().
	//	Str("address", addr).
	//	Bool("cors_enabled", cfg.Server.CORS).
	//	Msg("MCP Stockfish HTTP server starting")

	//httpServer := server.NewStreamableHTTPServer(s)
	//errCh := make(chan error, 1)
	//go func() {
	//	errCh <- httpServer.Start(addr)
	//}()

	//select {
	//case <-ctx.Done():
	//	log.Info().Msg("Shutdown signal received, stopping HTTP server...")
	//	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	//	defer cancel()
	//	if err := httpServer.Shutdown(shutdownCtx); err != nil {
	//		log.Error().Err(err).Msg("HTTP server shutdown error")
	//		return err
	//	}
	//	log.Info().Msg("HTTP server stopped gracefully.")
	//	return context.Canceled
	//case err := <-errCh:
	//	if err != nil {
	//		return fmt.Errorf("HTTP server error: %w", err)
	//	}
	//	return nil
	//}
}

func runStdioServer(s *server.MCPServer, log zerolog.Logger) error {
	log.Info().Msg("MCP Stockfish server initialized, serving on stdio")

	err := server.ServeStdio(s)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, os.ErrClosed) {
			log.Info().Err(err).Msg("Stdio server connection closed by client or pipe.")
			return context.Canceled
		}
		return fmt.Errorf("stdio server error: %w", err)
	}
	log.Info().Msg("Stdio server finished.")
	return nil
}
