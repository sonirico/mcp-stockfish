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
	executor = NewEphemeralSessionExecutor(cfg.Stockfish.Path, cfg.Stockfish.CommandTimeout, log)

	stockfishHandler := newStockfishHandler(executor, log)

	s := server.NewMCPServer(
		cfg.Server.Name,
		cfg.Server.Version,
	)

	stockfishTool := mcp.NewTool(
		"chess_engine",
		mcp.WithDescription(`
Advanced chess analysis using Stockfish engine via UCI (Universal Chess Interface).
Analyzes positions, finds best moves, evaluates positions. Returns structured results.

MOVE NOTATION: Use algebraic notation (e2e4, g1f3, e1g1 for castling, e7e8q for promotion)
EVALUATION: Centipawns (100 = 1 pawn), positive = White advantage, negative = Black advantage

═══ UCI COMMAND REFERENCE ═══

ENGINE CONTROL:
┌─ uci           → Initialize engine, get info & options
├─ isready       → Check if engine ready for commands  
├─ quit          → Shutdown engine
└─ stop          → Stop current analysis immediately

POSITION SETUP:
┌─ position startpos                    → Initial chess position
├─ position startpos moves e2e4 e7e5    → Initial + move sequence
├─ position fen [FEN]                   → Custom position from FEN
└─ position fen [FEN] moves [MOVES]     → Custom FEN + additional moves

ANALYSIS COMMANDS:
┌─ go depth [N]       → Analyze N plies deep (typical: 15-25)
├─ go movetime [MS]   → Analyze for N milliseconds (typical: 3000-10000)
├─ go infinite       → Analyze until stopped (use 'stop' to end)
├─ go wtime [MS] btime [MS]  → Analysis with time controls
└─ go nodes [N]      → Analyze exactly N nodes

ENGINE OPTIONS (setoption name [NAME] value [VALUE]):
┌─ Hash [1-32768]         → Memory in MB (default: 16)
├─ Threads [1-128]        → CPU threads (default: 1)  
├─ MultiPV [1-500]        → Show N best lines (default: 1)
├─ Skill Level [0-20]     → Engine strength (20=strongest)
├─ Move Overhead [0-5000] → Time buffer in ms
├─ Slow Mover [10-1000]   → Time usage factor
└─ Contempt [-100-100]    → Draw tendency

TYPICAL WORKFLOWS:
1. Quick analysis:    "position startpos moves e2e4" → "go movetime 3000"
2. Deep analysis:     "position startpos" → "go depth 25" 
3. Custom position:   "position fen r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq -" → "go depth 20"
4. Multi-line:        "setoption name MultiPV value 3" → "position startpos" → "go depth 18"
5. Weaker play:       "setoption name Skill Level value 10" → "position startpos" → "go depth 15"

EXAMPLES:
• "uci" → Get engine info
• "position startpos moves e2e4 e7e5 g1f3 b8c6" → Set position  
• "go depth 18" → Deep analysis
• "setoption name Hash value 512" → Increase memory
• "setoption name MultiPV value 5" → Show top 5 moves
		`),
		mcp.WithString(
			"command",
			mcp.Required(),
			mcp.Description(`
Exact UCI command to execute. Use commands from the reference above.

QUICK REFERENCE:
• uci, isready, quit, stop
• position startpos [moves MOVE_LIST]  
• position fen FEN_STRING [moves MOVE_LIST]
• go depth N | go movetime MS | go infinite
• setoption name OPTION_NAME value VALUE

MOVE FORMAT: e2e4 e7e5 g1f3 (algebraic notation)
EXAMPLES: "position startpos moves e2e4", "go depth 15", "setoption name Hash value 256"
			`),
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
	// TODO: waiting for https://github.com/mark3labs/mcp-go/pull/331
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
