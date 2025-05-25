package main

type ServerMode string

const (
	ServerModeHTTP  ServerMode = "http"
	ServerModeStdio ServerMode = "stdio"
)

type LogFormat string

const (
	LogFormatJSON    LogFormat = "json"
	LogFormatConsole LogFormat = "console"
)

type LogOutput string

const (
	LogOutputStdout LogOutput = "stdout"
	LogOutputStderr LogOutput = "stderr"
)

const (
	StockfishCmdQuit      = "quit"
	StockfishCmdUCI       = "uci"
	StockfishCmdIsReady   = "isready"
	StockfishCmdStop      = "stop"
	StockfishCmdPosition  = "position"
	StockfishCmdGo        = "go"
	StockfishCmdSetOption = "setoption"
)

const (
	ComponentSessionManager = "session_manager"
	ComponentHandler        = "handler"
	ExecutorPersistent      = "persistent"
	ExecutorEphemeral       = "ephemeral"
)

const (
	SessionIDStdioEphemeral = "stdio-ephemeral"
)
